// Package speech is the system's speech to text, streaming: words arrive while the person is still
// talking. There is no text to speech. macOS uses SFSpeechRecognizer with AVAudioEngine (see
// speech_darwin.go); every other platform returns ErrUnavailable until its recognizer is wired in
// (docs/plans/m12-voice-spike.md).
package speech

import "errors"

// ErrUnavailable is returned by Start where the platform has no recognizer wired in, or the
// recognizer exists but cannot run now (no model for the language, no microphone).
var ErrUnavailable = errors.New("speech recognition is unavailable")

// ErrNotAuthorized is returned by Start when the person has not allowed the microphone or speech
// recognition for the app. Call RequestAuthorization first, and Authorization to read the state.
var ErrNotAuthorized = errors.New("microphone or speech recognition is not allowed")

// Authorization is the permission state of the microphone and speech recognition together.
type Authorization int

const (
	// AuthUnknown: the platform has no such permission, or it could not be read.
	AuthUnknown Authorization = iota
	// AuthNotDetermined: the person has not been asked yet.
	AuthNotDetermined
	// AuthDenied: the person said no, or a policy forbids it. Only System Settings changes that.
	AuthDenied
	// AuthAuthorized: both are allowed.
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

// Recognizer listens to the default microphone for one utterance at a time. Callbacks run on a
// thread of the system, not the UI thread, and are not called after onFinal or after Cancel.
type Recognizer interface {
	// Available says whether Start can work on this machine, and if not, why in a short English
	// phrase for logs (not for the UI).
	Available() (ok bool, reason string)
	// Authorization reads the permission state without asking.
	Authorization() Authorization
	// RequestAuthorization shows the system prompts that are due and calls done with the result,
	// on a system thread. It calls done at once when nothing is left to ask.
	RequestAuthorization(done func(Authorization))
	// Start begins listening. onPartial gets the whole text heard so far, replacing the previous
	// call's text (unsettled words included); onFinal gets the settled text once, after Stop.
	Start(onPartial, onFinal func(text string)) error
	// Stop ends the audio and lets the recognizer settle: onFinal follows within a moment.
	Stop()
	// LastError is why the last utterance ended without text, such as the system's "Siri and
	// Dictation are disabled" (System Settings > Keyboard > Dictation); nil when it did not fail.
	LastError() error
	// Cancel drops the utterance without calling onFinal.
	Cancel()
}

// New returns the recognizer of this platform.
func New() Recognizer { return newRecognizer() }
