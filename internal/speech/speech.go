// Package speech records the microphone for voice input. It does not recognize anything: the
// recording is a 16 kHz mono PCM16 WAV that the app uploads to the sync server, which sends it to a
// cloud recognizer (docs/plans/m12-voice.md). macOS records with AVAudioEngine (speech_darwin.go);
// every other platform returns ErrUnavailable until its recorder is wired in.
package speech

import (
	"errors"
	"time"
)

// MaxDuration is the longest recording. The recorder stops adding samples after it; the caller
// ends the recording by then.
const MaxDuration = 60 * time.Second

// ErrUnavailable is returned by Start where the platform has no recorder wired in, or the recorder
// cannot run now (no microphone).
var ErrUnavailable = errors.New("recording is unavailable")

// ErrNotAuthorized is returned by Start when the person has not allowed the microphone for the app.
// Call RequestAuthorization first, and Authorization to read the state.
var ErrNotAuthorized = errors.New("microphone access is not allowed")

// ErrBusy is returned by Start while a recording is already running.
var ErrBusy = errors.New("already recording")

// ErrNoAudio is returned by Stop when the recording holds under a fifth of a second of sound.
var ErrNoAudio = errors.New("nothing was recorded")

// Authorization is the permission state of the microphone.
type Authorization int

const (
	// AuthUnknown: the platform has no such permission, or it could not be read.
	AuthUnknown Authorization = iota
	// AuthNotDetermined: the person has not been asked yet.
	AuthNotDetermined
	// AuthDenied: the person said no, or a policy forbids it. Only System Settings changes that.
	AuthDenied
	// AuthAuthorized: the microphone is allowed.
	AuthAuthorized
)

func (a Authorization) String() string {
	switch a {
	case AuthNotDetermined:
		return "not determined"
	case AuthDenied:
		return "denied"
	case AuthAuthorized:
		return "authorized"
	}
	return "unknown"
}

// Recorder records the default microphone, one utterance at a time. Callbacks run on a thread of
// the system, not the UI thread.
type Recorder interface {
	// Available says whether Start can work on this machine, and if not, why in a short English
	// phrase for logs (not for the UI).
	Available() (ok bool, reason string)
	// Authorization reads the permission state without asking.
	Authorization() Authorization
	// RequestAuthorization shows the system prompt if one is due and calls done with the result, on
	// a system thread. It calls done at once when nothing is left to ask.
	RequestAuthorization(done func(Authorization))
	// Start begins recording. onLevel, which may be nil, is called about every 20 ms with the
	// loudness of the last buffer, 0 to 1.
	Start(onLevel func(level float32)) error
	// Stop ends the recording and returns it as a 16 kHz mono PCM16 WAV.
	Stop() ([]byte, error)
	// Cancel drops the recording.
	Cancel()
}

// New returns the recorder of this platform.
func New() Recorder { return newRecorder() }
