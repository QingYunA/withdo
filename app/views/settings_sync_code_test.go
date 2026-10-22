package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func codeStepSync() data.SyncSnapshot {
	snap := signedOutCodeSync()
	snap.SignInStep, snap.SignInEmail = data.SignInStepCode, "cyan@example.com"
	snap.ResendAt = fixedNow().Add(24 * time.Second)
	return snap
}

func TestSyncEmailStepAsksForACode(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	has(t, r.tt, "Sign in or sign up", "Continue with email", "Keys and model settings stay on this Mac.")
	hasNot(t, r.tt, "Password", "Sign in and sync")
	if fs.checks == 0 {
		t.Error("the pane did not ask what kind of server it talks to")
	}
	// The field has the keyboard when the pane opens; the button waits for an email.
	r.tt.Type("cyan@")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if len(fs.requestedCodes) != 0 {
		t.Fatalf("Enter on half an email asked for a code: %v", fs.requestedCodes)
	}
	r.tt.Type("example.com")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if len(fs.requestedCodes) != 1 || fs.requestedCodes[0] != "cyan@example.com" {
		t.Fatalf("requested codes %v", fs.requestedCodes)
	}
	if fs.signIns != 0 {
		t.Error("the email step signed in with a password")
	}
}

func TestSyncEmailStepButtonAsksForACode(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	r.tt.Type("cyan@example.com")
	r.tt.Frame()
	if err := r.tt.Click("Continue with email"); err != nil {
		t.Fatal(err)
	}
	if len(fs.requestedCodes) != 1 {
		t.Fatalf("requested codes %v", fs.requestedCodes)
	}
}

func TestSyncCodeStepShowsTheAddressAndTheCountdown(t *testing.T) {
	r, _ := syncPane(t, codeStepSync())
	has(t, r.tt, "Check your email", "We sent a 6-digit code to", "cyan@example.com",
		"Use a different email", "Didn't get it? Resend in 0:24", "Use a password instead")
	hasNot(t, r.tt, "Resend code")

	fs := &fakeSync{snap: codeStepSync()}
	fs.snap.ResendAt = fixedNow().Add(-time.Second)
	r2 := newSettingsRig(t, func(a *App) { a.Sync = fs })
	r2.s.Show(PaneAccount)
	r2.tt.Frame()
	has(t, r2.tt, "Resend code")
	hasNot(t, r2.tt, "Resend in")
	if err := r2.tt.Click("Resend code"); err != nil {
		t.Fatal(err)
	}
	if fs.resends != 1 {
		t.Errorf("resends = %d", fs.resends)
	}
}

func TestSyncSixthDigitSendsTheCode(t *testing.T) {
	r, fs := syncPane(t, codeStepSync())
	r.tt.Type("48291")
	r.tt.Frame()
	if len(fs.codes) != 0 {
		t.Fatalf("five digits sent %v", fs.codes)
	}
	r.tt.Type("3")
	r.tt.Frame()
	if len(fs.codes) != 1 || fs.codes[0] != "482913" {
		t.Fatalf("codes %v", fs.codes)
	}
	r.tt.Frame()
	if len(fs.codes) != 1 {
		t.Errorf("the same code was sent again: %v", fs.codes)
	}
}

func TestSyncCodeTakesDigitsOnlyAndBackspaceGoesBack(t *testing.T) {
	r, fs := syncPane(t, codeStepSync())
	r.tt.Type("4a8 2")
	r.tt.Frame()
	if r.s.syncCode != "482" {
		t.Fatalf("code %q", r.s.syncCode)
	}
	r.tt.Key(0, ui.KeyBackspace)
	r.tt.Frame()
	if r.s.syncCode != "48" {
		t.Fatalf("after Backspace %q", r.s.syncCode)
	}
	if len(fs.codes) != 0 {
		t.Errorf("sent %v", fs.codes)
	}
}

func TestSyncPasteFillsAllSixCells(t *testing.T) {
	r, fs := syncPane(t, codeStepSync())
	r.tt.SetClipboard("482 913")
	r.tt.Key(ui.Cmd, ui.KeyV)
	r.tt.Frame()
	if len(fs.codes) != 1 || fs.codes[0] != "482913" {
		t.Fatalf("a pasted code sent %v (typed %q)", fs.codes, r.s.syncCode)
	}
}

func TestSyncCodeStepLinksSwitchSteps(t *testing.T) {
	r, fs := syncPane(t, codeStepSync())
	if err := r.tt.Click("Use a password instead"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	has(t, r.tt, "Sign in with a password", "Password", "Sign in", "Forgot password", "Use a code instead")

	r.tt.Type("hunter2")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if fs.signIns != 1 || fs.login != "cyan@example.com" || fs.password != "hunter2" {
		t.Errorf("password sign-in %d times with %q / %q", fs.signIns, fs.login, fs.password)
	}
	if err := r.tt.Click("Forgot password"); err != nil {
		t.Fatal(err)
	}
	if len(fs.resets) != 1 || fs.resets[0] != "cyan@example.com" {
		t.Errorf("reset mails %v", fs.resets)
	}
	if err := r.tt.Click("Use a code instead"); err != nil {
		t.Fatal(err)
	}
	if len(fs.requestedCodes) != 1 || fs.requestedCodes[0] != "cyan@example.com" {
		t.Errorf("codes %v", fs.requestedCodes)
	}

	// A different email goes back to the first step with the address kept.
	if err := r.tt.Click("Use a different email"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	has(t, r.tt, "Sign in or sign up", "Continue with email")
	if r.s.syncLogin != "cyan@example.com" {
		t.Errorf("the email was not kept: %q", r.s.syncLogin)
	}
}

func TestSyncResetSentStep(t *testing.T) {
	snap := codeStepSync()
	snap.SignInStep = data.SignInStepResetSent
	r, fs := syncPane(t, snap)
	has(t, r.tt, "Reset your password from your email",
		"We sent a link to cyan@example.com. Set a new password in the browser, then come back",
		"Back to sign in", "Use a code instead")
	if err := r.tt.Click("Back to sign in"); err != nil {
		t.Fatal(err)
	}
	if fs.snap.SignInStep != data.SignInStepPassword {
		t.Errorf("step %q", fs.snap.SignInStep)
	}
}

func TestSyncCodeFailureLines(t *testing.T) {
	cases := map[data.SignInFailure]string{
		data.SignInFailureWrongCode:   "That code isn't right",
		data.SignInFailureCodeExpired: "That code expired; we sent a new one",
		data.SignInFailureTooMany:     "Too many tries; wait a minute",
		data.SignInFailureLocked:      "This account is deactivated; ask the admin",
		data.SignInFailureNetwork:     "the server did not answer",
	}
	for failure, line := range cases {
		snap := codeStepSync()
		snap.SignInFailure = failure
		r, _ := syncPane(t, snap)
		if failure == data.SignInFailureNetwork {
			line = r.s.say.SettingsSyncFailedNoAnswer()
		}
		has(t, r.tt, line)
	}

	snap := codeStepSync()
	snap.SignInStep, snap.SignInFailure = data.SignInStepPassword, data.SignInFailureCredentials
	r, _ := syncPane(t, snap)
	has(t, r.tt, "Wrong email or password")
}

func TestSyncWrongCodeEmptiesTheCells(t *testing.T) {
	r, fs := syncPane(t, codeStepSync())
	r.tt.Type("482913")
	r.tt.Frame()
	fs.snap.Phase, fs.snap.SignInStage = data.SyncSigningIn, data.SignInStageChecking
	r.tt.Frame()
	has(t, r.tt, "Checking the code…")
	fs.snap.Phase, fs.snap.SignInStage, fs.snap.SignInFailure = data.SyncSignedOut, data.SignInStageNone, data.SignInFailureWrongCode
	r.tt.Frame()
	has(t, r.tt, "That code isn't right")
	if r.s.syncCode != "" {
		t.Fatalf("the cells kept %q", r.s.syncCode)
	}
	// The first cell has the keyboard again: a new code goes through.
	r.tt.Type("111111")
	r.tt.Frame()
	if len(fs.codes) != 2 || fs.codes[1] != "111111" {
		t.Errorf("codes %v", fs.codes)
	}
}

func TestSyncSendingAndDownloadingLines(t *testing.T) {
	snap := signedOutCodeSync()
	snap.Phase, snap.SignInStage = data.SyncSigningIn, data.SignInStageSending
	r, fs := syncPane(t, snap)
	has(t, r.tt, "Sending the code…")
	fs.snap.SignInStep, fs.snap.SignInEmail = data.SignInStepCode, "cyan@example.com"
	fs.snap.SignInStage = data.SignInStageDownloading
	r.tt.Frame()
	has(t, r.tt, "Downloading your tasks…")
}
