//go:build !darwin

package speech

type unavailable struct{}

func newRecognizer() Recognizer { return unavailable{} }

func (unavailable) Available() (bool, string) { return false, "not implemented on this platform" }

func (unavailable) Authorization() Authorization { return AuthUnknown }

func (unavailable) RequestAuthorization(done func(Authorization)) {
	if done != nil {
		done(AuthUnknown)
	}
}

func (unavailable) Start(_, _ func(string)) error { return ErrUnavailable }

func (unavailable) Stop() {}

func (unavailable) Cancel() {}

func (unavailable) LastError() error { return nil }
