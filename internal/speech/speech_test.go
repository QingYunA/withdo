package speech

import "testing"

// The recognizer must answer the read-only questions on every platform without asking the person
// for anything; Start must not listen without authorization.
func TestReadOnlyQuestions(t *testing.T) {
	r := New()
	_, _ = r.Available()
	a := r.Authorization()
	if a != AuthAuthorized {
		if err := r.Start(nil, nil); err == nil {
			r.Cancel()
			t.Fatalf("Start succeeded with authorization %s", a)
		}
	}
}
