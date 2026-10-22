package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

func googleSync() data.SyncSnapshot {
	snap := signedOutCodeSync()
	snap.GoogleAvailable = true
	return snap
}

func TestSyncEmailStepOffersGoogleAboveTheEmailField(t *testing.T) {
	r, fs := syncPane(t, googleSync())
	has(t, r.tt, "Continue with Google", "or", "Continue with email")
	if err := r.tt.Click("Continue with Google"); err != nil {
		t.Fatal(err)
	}
	if fs.googles != 1 {
		t.Fatalf("Continue with Google called %d times", fs.googles)
	}
	// The browser is opened by the shell's own helper, handed to the sign-in.
	fs.open("https://accounts.google.com/auth")
	if len(r.rec.urls) != 1 || r.rec.urls[0] != "https://accounts.google.com/auth" {
		t.Fatalf("opened %v, want Google's page through Act.OpenURL", r.rec.urls)
	}
}

func TestSyncEmailStepHasNoGoogleButtonWithoutAProvider(t *testing.T) {
	r, _ := syncPane(t, signedOutCodeSync())
	hasNot(t, r.tt, "Continue with Google")
	has(t, r.tt, "Continue with email")

	old := signedOutSync()
	old.GoogleAvailable = true // never offered on a password-only server
	r2, _ := syncPane(t, old)
	hasNot(t, r2.tt, "Continue with Google")
}

func TestSyncGoogleWaitingCardCancels(t *testing.T) {
	snap := googleSync()
	snap.Phase, snap.SignInStage = data.SyncSigningIn, data.SignInStageGoogle
	r, fs := syncPane(t, snap)
	has(t, r.tt, "Finish in your browser",
		"Choose your Google account there; this window picks it up by itself.", "Cancel")
	hasNot(t, r.tt, "Continue with email", "Continue with Google")
	if err := r.tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if fs.cancels != 1 {
		t.Fatalf("Cancel called SyncCancel %d times", fs.cancels)
	}
}

func TestSyncGoogleFailuresAreWorded(t *testing.T) {
	for failure, text := range map[data.SignInFailure]string{
		data.SignInFailureGoogleCancelled: "Google sign-in was cancelled",
		data.SignInFailureGoogleFailed:    "Google sign-in didn't finish; try again",
		data.SignInFailureNetwork:         "the server did not answer",
	} {
		snap := googleSync()
		snap.SignInFailure = failure
		r, _ := syncPane(t, snap)
		has(t, r.tt, text, "Continue with Google")
	}
}
