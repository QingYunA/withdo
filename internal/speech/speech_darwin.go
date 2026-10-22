//go:build darwin

package speech

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// The darwin recorder is pure Go: purego calls the Objective-C runtime without cgo, which is how
// mygo itself talks to AppKit and why the release build is CGO_ENABLED=0. AVAudioEngine wants a
// block for the input tap; objc.NewBlock makes it.

var (
	loadOnce sync.Once
	loadErr  error
)

func load() error {
	loadOnce.Do(func() {
		if _, err := purego.Dlopen("/System/Library/Frameworks/AVFoundation.framework/AVFoundation",
			purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
			loadErr = err
		}
	})
	return loadErr
}

func sel(name string) objc.SEL { return objc.RegisterName(name) }

func class(name string) objc.ID { return objc.ID(objc.GetClass(name)) }

func nsString(s string) objc.ID {
	return class("NSString").Send(sel("stringWithUTF8String:"), s)
}

// AVAuthorizationStatus.
const (
	avNotDetermined, avRestricted, avDenied, avAuthorized = 0, 1, 2, 3
)

const avMediaTypeAudio = "soun" // AVMediaTypeAudio

type darwinRecorder struct {
	mu     sync.Mutex
	engine objc.ID
	tap    objc.Block
	pcm    *pcm
}

func newRecorder() Recorder { return &darwinRecorder{} }

func (r *darwinRecorder) Available() (bool, string) {
	if err := load(); err != nil {
		return false, "AVFoundation: " + err.Error()
	}
	return true, ""
}

func (r *darwinRecorder) Authorization() Authorization {
	if load() != nil {
		return AuthUnknown
	}
	switch objc.Send[int](class("AVCaptureDevice"), sel("authorizationStatusForMediaType:"), nsString(avMediaTypeAudio)) {
	case avAuthorized:
		return AuthAuthorized
	case avDenied, avRestricted:
		return AuthDenied
	}
	return AuthNotDetermined
}

func (r *darwinRecorder) RequestAuthorization(done func(Authorization)) {
	finish := func() {
		if done != nil {
			done(r.Authorization())
		}
	}
	if load() != nil || r.Authorization() != AuthNotDetermined {
		finish()
		return
	}
	// The prompt needs NSMicrophoneUsageDescription in the app's Info.plist (macos.infoPlist in
	// mygo.json), or macOS ends the process.
	var block objc.Block
	block = objc.NewBlock(func(_ objc.Block, _ bool) {
		block.Release()
		finish()
	})
	class("AVCaptureDevice").Send(sel("requestAccessForMediaType:completionHandler:"), nsString(avMediaTypeAudio), block)
}

func (r *darwinRecorder) Start(onLevel func(float32)) error {
	if ok, _ := r.Available(); !ok {
		return ErrUnavailable
	}
	if r.Authorization() != AuthAuthorized {
		return ErrNotAuthorized
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.engine != 0 {
		return ErrBusy
	}

	engine := class("AVAudioEngine").Send(sel("new"))
	input := engine.Send(sel("inputNode"))
	var session *pcm
	var mono []float32
	tap := objc.NewBlock(func(_ objc.Block, buffer, _ objc.ID) {
		frames := int(objc.Send[uint32](buffer, sel("frameLength")))
		format := buffer.Send(sel("format"))
		rate := objc.Send[float64](format, sel("sampleRate"))
		channels := int(objc.Send[uint32](format, sel("channelCount")))
		data := objc.Send[unsafe.Pointer](buffer, sel("floatChannelData"))
		if frames == 0 || rate <= 0 || channels == 0 || data == nil {
			return
		}
		if session == nil { // the first buffer tells the hardware's rate
			session = newPCM(rate)
			r.mu.Lock()
			if r.engine == engine {
				r.pcm = session
			}
			r.mu.Unlock()
		}
		// floatChannelData is float * const *: one pointer per channel (non-interleaved).
		ptrs := unsafe.Slice((**float32)(data), channels)
		chans := make([][]float32, channels)
		for i, p := range ptrs {
			chans[i] = unsafe.Slice(p, frames)
		}
		mono = mixToMono(mono, chans)
		level := session.write(mono)
		if onLevel != nil {
			onLevel(level)
		}
	})
	// format nil: the tap takes the format of the hardware, whatever its rate; pcm converts.
	input.Send(sel("installTapOnBus:bufferSize:format:block:"), uintptr(0), uint32(1024), uintptr(0), tap)
	engine.Send(sel("prepare"))
	var nsErr objc.ID
	if !objc.Send[bool](engine, sel("startAndReturnError:"), unsafe.Pointer(&nsErr)) {
		input.Send(sel("removeTapOnBus:"), uintptr(0))
		tap.Release()
		engine.Send(sel("release"))
		return ErrUnavailable
	}
	r.engine, r.tap = engine, tap
	return nil
}

// stopEngine takes the running recording out of r and ends its audio. The caller holds no lock.
func (r *darwinRecorder) stopEngine() (*pcm, bool) {
	r.mu.Lock()
	engine, tap, session := r.engine, r.tap, r.pcm
	r.engine, r.tap, r.pcm = 0, 0, nil
	r.mu.Unlock()
	if engine == 0 {
		return nil, false
	}
	engine.Send(sel("stop"))
	engine.Send(sel("inputNode")).Send(sel("removeTapOnBus:"), uintptr(0))
	// The runtime keeps a block alive until its last call returns, so releasing ours is safe.
	tap.Release()
	engine.Send(sel("release"))
	return session, true
}

func (r *darwinRecorder) Stop() ([]byte, error) {
	session, ok := r.stopEngine()
	if !ok {
		return nil, ErrNoAudio
	}
	if session == nil { // stopped before the first buffer
		return nil, ErrNoAudio
	}
	return session.finish()
}

func (r *darwinRecorder) Cancel() {
	if session, _ := r.stopEngine(); session != nil {
		session.cancel()
	}
}
