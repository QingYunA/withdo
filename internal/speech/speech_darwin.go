//go:build darwin

package speech

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// The darwin recognizer is pure Go: purego calls the Objective-C runtime without cgo, which is how
// mygo itself talks to AppKit and why the release build is CGO_ENABLED=0. SFSpeechRecognizer wants
// blocks (the result handler, the audio tap); objc.NewBlock makes them.

var (
	loadOnce sync.Once
	loadErr  error
)

func load() error {
	loadOnce.Do(func() {
		for _, p := range []string{
			"/System/Library/Frameworks/Speech.framework/Speech",
			"/System/Library/Frameworks/AVFoundation.framework/AVFoundation",
		} {
			if _, err := purego.Dlopen(p, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
				loadErr = err
				return
			}
		}
	})
	return loadErr
}

func sel(name string) objc.SEL { return objc.RegisterName(name) }

func class(name string) objc.ID { return objc.ID(objc.GetClass(name)) }

func nsString(s string) objc.ID {
	return class("NSString").Send(sel("stringWithUTF8String:"), s)
}

func goString(s objc.ID) string {
	if s == 0 {
		return ""
	}
	n := objc.Send[uint](s, sel("lengthOfBytesUsingEncoding:"), uint(4)) // NSUTF8StringEncoding
	p := objc.Send[unsafe.Pointer](s, sel("UTF8String"))
	if p == nil || n == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(p), n))
}

// The two frameworks number their statuses differently.
const (
	sfNotDetermined, sfDenied, sfRestricted, sfAuthorized = 0, 1, 2, 3 // SFSpeechRecognizerAuthorizationStatus
	avNotDetermined, avRestricted, avDenied, avAuthorized = 0, 1, 2, 3 // AVAuthorizationStatus
)

var avMediaTypeAudio = "soun" // AVMediaTypeAudio

type darwinRecognizer struct {
	mu      sync.Mutex
	engine  objc.ID
	request objc.ID
	task    objc.ID
	rec     objc.ID
	blocks  []objc.Block
	active  bool
	lastErr error
}

func newRecognizer() Recognizer { return &darwinRecognizer{} }

func (r *darwinRecognizer) Available() (bool, string) {
	if err := load(); err != nil {
		return false, "frameworks: " + err.Error()
	}
	rec := class("SFSpeechRecognizer").Send(sel("new"))
	if rec == 0 {
		return false, "no recognizer for the system language"
	}
	defer rec.Send(sel("release"))
	if !objc.Send[bool](rec, sel("isAvailable")) {
		return false, "recognizer not available now (offline without an on-device model?)"
	}
	return true, ""
}

func (r *darwinRecognizer) Authorization() Authorization {
	if load() != nil {
		return AuthUnknown
	}
	sf := objc.Send[int](class("SFSpeechRecognizer"), sel("authorizationStatus"))
	av := objc.Send[int](class("AVCaptureDevice"), sel("authorizationStatusForMediaType:"), nsString(avMediaTypeAudio))
	switch {
	case sf == sfDenied || sf == sfRestricted || av == avDenied || av == avRestricted:
		return AuthDenied
	case sf == sfAuthorized && av == avAuthorized:
		return AuthAuthorized
	}
	return AuthNotDetermined
}

func (r *darwinRecognizer) RequestAuthorization(done func(Authorization)) {
	finish := func() {
		if done != nil {
			done(r.Authorization())
		}
	}
	if load() != nil || r.Authorization() != AuthNotDetermined {
		finish()
		return
	}
	// Ask for speech first, then for the microphone; both prompts need the usage descriptions in
	// the app's Info.plist (macos.infoPlist in mygo.json), or macOS ends the process.
	var sfBlock, avBlock objc.Block
	avBlock = objc.NewBlock(func(_ objc.Block, _ bool) {
		avBlock.Release()
		finish()
	})
	sfBlock = objc.NewBlock(func(_ objc.Block, status int) {
		sfBlock.Release()
		if status != sfAuthorized {
			finish()
			return
		}
		class("AVCaptureDevice").Send(sel("requestAccessForMediaType:completionHandler:"), nsString(avMediaTypeAudio), avBlock)
	})
	class("SFSpeechRecognizer").Send(sel("requestAuthorization:"), sfBlock)
}

func (r *darwinRecognizer) Start(onPartial, onFinal func(string)) error {
	if ok, _ := r.Available(); !ok {
		return ErrUnavailable
	}
	if r.Authorization() != AuthAuthorized {
		return ErrNotAuthorized
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active {
		return nil
	}
	r.lastErr = nil

	rec := class("SFSpeechRecognizer").Send(sel("new"))
	// The default queue is the main queue, which nothing drains in a command line tool and which
	// the UI thread should not carry anyway.
	rec.Send(sel("setQueue:"), class("NSOperationQueue").Send(sel("new")))

	req := class("SFSpeechAudioBufferRecognitionRequest").Send(sel("new"))
	req.Send(sel("setShouldReportPartialResults:"), true)
	if objc.Send[bool](rec, sel("supportsOnDeviceRecognition")) {
		req.Send(sel("setRequiresOnDeviceRecognition:"), true)
	}

	engine := class("AVAudioEngine").Send(sel("new"))
	input := engine.Send(sel("inputNode"))
	tap := objc.NewBlock(func(_ objc.Block, buffer, _ objc.ID) {
		req.Send(sel("appendAudioPCMBuffer:"), buffer)
	})
	// format nil: the tap takes the format of the hardware, which the recognizer converts.
	input.Send(sel("installTapOnBus:bufferSize:format:block:"), uintptr(0), uint32(1024), uintptr(0), tap)
	engine.Send(sel("prepare"))
	var nsErr objc.ID
	if !objc.Send[bool](engine, sel("startAndReturnError:"), unsafe.Pointer(&nsErr)) {
		input.Send(sel("removeTapOnBus:"), uintptr(0))
		tap.Release()
		return ErrUnavailable
	}

	var settled bool
	handler := objc.NewBlock(func(_ objc.Block, result, err objc.ID) {
		r.mu.Lock()
		if r.request != req { // cancelled or replaced
			r.mu.Unlock()
			return
		}
		r.mu.Unlock()
		if err != 0 {
			r.mu.Lock()
			r.lastErr = errors.New(goString(err.Send(sel("localizedDescription"))))
			r.mu.Unlock()
		}
		var text string
		var final bool
		if result != 0 {
			text = goString(result.Send(sel("bestTranscription")).Send(sel("formattedString")))
			final = objc.Send[bool](result, sel("isFinal"))
		}
		if final || (err != 0 && !settled) {
			settled = true
			r.teardown(req)
			if onFinal != nil {
				onFinal(text)
			}
			return
		}
		if text != "" && onPartial != nil {
			onPartial(text)
		}
	})
	task := rec.Send(sel("recognitionTaskWithRequest:resultHandler:"), req, handler)

	r.rec, r.request, r.engine, r.task = rec, req, engine, task
	r.blocks = []objc.Block{tap, handler}
	r.active = true
	return nil
}

// teardown stops the audio and releases what Start made, when req is still the current request.
func (r *darwinRecognizer) teardown(req objc.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.request != req {
		return
	}
	r.engine.Send(sel("stop"))
	r.engine.Send(sel("inputNode")).Send(sel("removeTapOnBus:"), uintptr(0))
	for _, id := range []objc.ID{r.engine, r.request, r.rec} {
		id.Send(sel("release"))
	}
	// The blocks may still be running on their threads; the runtime keeps a block alive until its
	// last call returns, so releasing ours is safe.
	for _, b := range r.blocks {
		b.Release()
	}
	r.engine, r.request, r.task, r.rec, r.blocks, r.active = 0, 0, 0, 0, nil, false
}

func (r *darwinRecognizer) Stop() {
	r.mu.Lock()
	req, engine := r.request, r.engine
	r.mu.Unlock()
	if req == 0 {
		return
	}
	engine.Send(sel("stop"))
	req.Send(sel("endAudio")) // the final result follows through the handler
}

func (r *darwinRecognizer) Cancel() {
	r.mu.Lock()
	req, task := r.request, r.task
	r.mu.Unlock()
	if req == 0 {
		return
	}
	task.Send(sel("cancel"))
	r.teardown(req)
}

func (r *darwinRecognizer) LastError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastErr
}
