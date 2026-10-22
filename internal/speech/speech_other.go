//go:build !darwin

package speech

// Windows recording is a later change; until then every other platform has no recorder and the UI
// hides the microphone.
type unavailable struct{}

func newRecorder() Recorder { return unavailable{} }

func (unavailable) Available() (bool, string) { return false, "not implemented on this platform" }

func (unavailable) Authorization() Authorization { return AuthUnknown }

func (unavailable) RequestAuthorization(done func(Authorization)) {
	if done != nil {
		done(AuthUnknown)
	}
}

func (unavailable) Start(func(float32)) error { return ErrUnavailable }

func (unavailable) Stop() ([]byte, error) { return nil, ErrUnavailable }

func (unavailable) Cancel() {}
