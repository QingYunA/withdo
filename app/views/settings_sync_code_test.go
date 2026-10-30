package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func signUpTabSync() data.SyncSnapshot {
	snap := signedOutCodeSync()
	snap.SignInStep = data.SignInStepSignUp
	return snap
}

func verifyStepSync() data.SyncSnapshot {
	snap := signedOutCodeSync()
	snap.SignInStep, snap.SignInEmail = data.SignInStepVerify, "cyan@example.com"
	snap.ResendAt = fixedNow().Add(24 * time.Second)
	return snap
}

func resetCodeStepSync() data.SyncSnapshot {
	snap := signedOutCodeSync()
	snap.SignInStep, snap.SignInEmail = data.SignInStepResetCode, "cyan@example.com"
	snap.ResendAt = fixedNow().Add(24 * time.Second)
	return snap
}

func TestSyncSignedOutCardOpensOnSignIn(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	has(t, r.tt, "Sign in", "Sign up", "Email", "Password", "Show", "Forgot password?",
		"Don't have an account?", "Keys and model settings stay on this Mac.")
	hasNot(t, r.tt, "Create account", "Use a code instead", "Use a password instead", "Email or username")
	if fs.checks == 0 {
		t.Error("the pane did not ask what kind of server it talks to")
	}
}

func TestSyncSignInEnterSubmitsAndTabGoesEmailPasswordButton(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	// The email has the keyboard when the pane opens; Enter with no password only moves on.
	r.tt.Type("cyan@example.com")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if fs.signIns != 0 {
		t.Fatalf("Enter on the email alone signed in")
	}
	r.tt.Type("hunter22")
	r.tt.Frame()
	if r.s.syncLogin != "cyan@example.com" || r.s.syncPassword != "hunter22" {
		t.Fatalf("typed %q / %q: Enter on the email should move to the password", r.s.syncLogin, r.s.syncPassword)
	}
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if fs.signIns != 1 || fs.login != "cyan@example.com" || fs.password != "hunter22" {
		t.Fatalf("sign-ins %d with %q / %q", fs.signIns, fs.login, fs.password)
	}
}

func TestSyncSignInTabOrder(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	r.tt.Type("cyan@example.com")
	r.tt.Key(0, ui.KeyTab)
	r.tt.Frame()
	r.tt.Type("hunter22")
	r.tt.Frame()
	if r.s.syncLogin != "cyan@example.com" || r.s.syncPassword != "hunter22" {
		t.Fatalf("after Tab the password should have the keyboard: %q / %q", r.s.syncLogin, r.s.syncPassword)
	}
	// Tab again reaches the button, and Enter presses it.
	r.tt.Key(0, ui.KeyTab)
	r.tt.Frame()
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if fs.signIns != 1 {
		t.Fatalf("Tab, Tab, Enter signed in %d times, want 1 (email, password, button)", fs.signIns)
	}
}

func TestSyncSignInButtonWaitsForBothFields(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	_ = r.tt.Click("Sign in") // the tab; the button is the second "Sign in" and is disabled
	r.tt.Type("cyan@example.com")
	r.tt.Frame()
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if fs.signIns != 0 {
		t.Fatal("signed in without a password")
	}
}

func TestSyncTabsKeepTheEmail(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	r.tt.Type("cyan@example.com")
	r.tt.Frame()
	if err := r.tt.Click("Sign up"); err != nil { // the tab comes first
		t.Fatal(err)
	}
	r.tt.Frame()
	if fs.snap.SignInStep != data.SignInStepSignUp {
		t.Fatalf("step %q", fs.snap.SignInStep)
	}
	has(t, r.tt, "Create account", "At least 8 characters.", "By creating an account you agree to the",
		"Terms of Service", "Privacy Policy")
	hasNot(t, r.tt, "Forgot password?", "Don't have an account?")
	if r.s.syncLogin != "cyan@example.com" {
		t.Fatalf("the email was lost: %q", r.s.syncLogin)
	}
	// And back, with the email still there.
	if err := r.tt.Click("Sign in"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if fs.snap.SignInStep != data.SignInStepSignIn || r.s.syncLogin != "cyan@example.com" {
		t.Fatalf("back on Sign in: step %q email %q", fs.snap.SignInStep, r.s.syncLogin)
	}
}

func TestSyncTabsTakeTheArrowKeys(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	if err := r.tt.Click("Sign in"); err != nil { // focus the switch
		t.Fatal(err)
	}
	r.tt.Key(0, ui.KeyRight)
	r.tt.Frame()
	if fs.snap.SignInStep != data.SignInStepSignUp {
		t.Fatalf("Right: step %q, want Sign up", fs.snap.SignInStep)
	}
	r.tt.Key(0, ui.KeyLeft)
	r.tt.Frame()
	if fs.snap.SignInStep != data.SignInStepSignIn {
		t.Fatalf("Left: step %q, want Sign in", fs.snap.SignInStep)
	}
}

func TestSyncSwapLinkPicksSignUpWithTheEmail(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	r.tt.Type("cyan@example.com")
	r.tt.Frame()
	if err := r.tt.Click("Don't have an account? Sign up"); err != nil {
		t.Fatal(err)
	}
	if fs.snap.SignInStep != data.SignInStepSignUp || r.s.syncLogin != "cyan@example.com" {
		t.Fatalf("step %q email %q", fs.snap.SignInStep, r.s.syncLogin)
	}
}

func TestSyncSignUpNeedsAnEmailAndEightCharacters(t *testing.T) {
	r, fs := syncPane(t, signUpTabSync())
	r.tt.Type("cyan@example.com")
	r.tt.Key(0, ui.KeyTab)
	r.tt.Type("short")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if len(fs.signUps) != 0 {
		t.Fatalf("a 5 character password signed up: %v", fs.signUps)
	}
	r.tt.Type("er pass")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if len(fs.signUps) != 1 || fs.signUps[0] != "cyan@example.com|shorter pass" || fs.langs[0] != "en" {
		t.Fatalf("sign-ups %v langs %v", fs.signUps, fs.langs)
	}
}

func TestSyncCreateAccountButton(t *testing.T) {
	r, fs := syncPane(t, signUpTabSync())
	r.tt.Type("cyan@example.com")
	r.tt.Key(0, ui.KeyTab)
	r.tt.Type("a long password")
	r.tt.Frame()
	if err := r.tt.Click("Create account"); err != nil {
		t.Fatal(err)
	}
	if len(fs.signUps) != 1 {
		t.Fatalf("sign-ups %v", fs.signUps)
	}
}

func TestSyncShowTogglesThePassword(t *testing.T) {
	r, _ := syncPane(t, signUpTabSync())
	has(t, r.tt, "Show")
	if err := r.tt.Click("Show"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	has(t, r.tt, "Hide")
	hasNot(t, r.tt, "Show")
	if !r.s.syncShowPassword {
		t.Fatal("Show did not show the password")
	}
}

func TestPasswordStrengthSteps(t *testing.T) {
	for password, want := range map[string]int{
		"":                      0,
		"short":                 0,
		"lowercas":              1, // 8 characters, one kind
		"12345678":              1,
		"短短短短短短短短":              1,
		"alllowercase":          2, // from 12 characters
		"abcdefgh1":             2, // two kinds
		"Lowercase1":            3, // three kinds from 10 characters
		"correct horse battery": 3,
		"Correct-Horse-9":       4,
	} {
		if got := passwordStrength(password); got != want {
			t.Errorf("passwordStrength(%q) = %d, want %d", password, got, want)
		}
	}
}

func TestSyncEmailTakenSaysSoAndLinksToSignIn(t *testing.T) {
	snap := signUpTabSync()
	snap.SignInFailure = data.SignInFailureEmailTaken
	snap.SignInEmail = "cyan@example.com"
	r, fs := syncPane(t, snap)
	r.s.syncLogin = "cyan@example.com"
	r.tt.Frame()
	has(t, r.tt, "This email already has an account.")
	if err := r.tt.Click("This email already has an account. Sign in"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if fs.snap.SignInStep != data.SignInStepSignIn || r.s.syncLogin != "cyan@example.com" {
		t.Fatalf("step %q email %q: the link should pick Sign in with the email kept", fs.snap.SignInStep, r.s.syncLogin)
	}
	has(t, r.tt, "Forgot password?")
}

func TestSyncInvalidEmailLine(t *testing.T) {
	snap := signUpTabSync()
	snap.SignInFailure = data.SignInFailureInvalidEmail
	r, _ := syncPane(t, snap)
	has(t, r.tt, "That doesn't look like an email")
}

func TestSyncWrongPasswordEmptiesTheField(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	r.tt.Type("cyan@example.com")
	r.tt.Key(0, ui.KeyTab)
	r.tt.Type("hunter22")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	fs.snap.Phase, fs.snap.SignInStage = data.SyncSigningIn, data.SignInStageChecking
	r.tt.Frame()
	has(t, r.tt, "Checking your account…")
	fs.snap.Phase, fs.snap.SignInStage, fs.snap.SignInFailure = data.SyncSignedOut, data.SignInStageNone, data.SignInFailureCredentials
	r.tt.Frame()
	has(t, r.tt, "Wrong email or password")
	if r.s.syncPassword != "" || r.s.syncLogin != "cyan@example.com" {
		t.Fatalf("the password kept %q / the email %q", r.s.syncPassword, r.s.syncLogin)
	}
	// The cursor is back in the password: the button waits until it holds something.
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if fs.signIns != 1 {
		t.Fatalf("Enter on an empty password signed in again (%d)", fs.signIns)
	}
	r.tt.Type("hunter23")
	r.tt.Frame()
	if r.s.syncPassword != "hunter23" {
		t.Fatalf("typing after a wrong password went to %q", r.s.syncPassword)
	}
}

func TestSyncForgotPasswordFlow(t *testing.T) {
	r, fs := syncPane(t, signedOutCodeSync())
	r.tt.Type("cyan@example.com")
	r.tt.Frame()
	if err := r.tt.Click("Forgot password?"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if fs.snap.SignInStep != data.SignInStepResetEmail {
		t.Fatalf("step %q", fs.snap.SignInStep)
	}
	has(t, r.tt, "Reset your password", "Send code", "Back to sign in")
	if r.s.syncLogin != "cyan@example.com" {
		t.Fatalf("the email was not kept: %q", r.s.syncLogin)
	}
	if err := r.tt.Click("Send code"); err != nil {
		t.Fatal(err)
	}
	if len(fs.requestedCodes) != 1 || fs.requestedCodes[0] != "cyan@example.com" {
		t.Fatalf("reset codes %v", fs.requestedCodes)
	}

	// The code step: six digits do not send anything; the password does, with Enter.
	fs.snap = resetCodeStepSync()
	r.tt.Frame()
	has(t, r.tt, "Reset your password", "The code went to cyan@example.com", "New password", "Set and sign in",
		"At least 8 characters.", "Back to sign in")
	r.tt.Type("482913")
	r.tt.Frame()
	if len(fs.codes) != 0 || len(fs.resetCodes) != 0 {
		t.Fatalf("six digits sent %v / %v", fs.codes, fs.resetCodes)
	}
	r.tt.Type("a new password")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if len(fs.resetCodes) != 1 || fs.resetCodes[0] != "482913|a new password" {
		t.Fatalf("resets %v: the sixth digit should move on to the new password", fs.resetCodes)
	}
	if err := r.tt.Click("Back to sign in"); err != nil {
		t.Fatal(err)
	}
	if fs.snap.SignInStep != data.SignInStepSignIn {
		t.Fatalf("step %q", fs.snap.SignInStep)
	}
}

func TestSyncResetNeedsTheWholeCodeAndEightCharacters(t *testing.T) {
	r, fs := syncPane(t, resetCodeStepSync())
	r.tt.Type("4829")
	r.tt.Frame()
	r.s.syncPassword = "a new password"
	r.tt.Frame()
	if err := r.tt.Click("Set and sign in"); err != nil {
		t.Fatal(err)
	}
	if len(fs.resetCodes) != 0 {
		t.Fatalf("four digits reset: %v", fs.resetCodes)
	}
}

func TestSyncVerifyStepShowsTheAddressAndTheCountdown(t *testing.T) {
	r, _ := syncPane(t, verifyStepSync())
	has(t, r.tt, "Verify your email", "We sent a 6-digit code to cyan@example.com",
		"It checks itself once complete", "Use a different email", "Didn't get it? Resend in 0:24",
		"An account can't sign in until its email is verified.")
	hasNot(t, r.tt, "Resend code", "Continue with Google", "Create account", "Use a password instead",
		"Keys and model settings stay on this Mac.")

	fs := &fakeSync{snap: verifyStepSync()}
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
	r, fs := syncPane(t, verifyStepSync())
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
	r, fs := syncPane(t, verifyStepSync())
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
	r, fs := syncPane(t, verifyStepSync())
	r.tt.SetClipboard("482 913")
	r.tt.Key(ui.Cmd, ui.KeyV)
	r.tt.Frame()
	if len(fs.codes) != 1 || fs.codes[0] != "482913" {
		t.Fatalf("a pasted code sent %v (typed %q)", fs.codes, r.s.syncCode)
	}
}

func TestSyncUseADifferentEmailGoesBackToSignUpWithTheAddress(t *testing.T) {
	r, fs := syncPane(t, verifyStepSync())
	if err := r.tt.Click("Use a different email"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if fs.snap.SignInStep != data.SignInStepSignUp {
		t.Fatalf("step %q, want Sign up", fs.snap.SignInStep)
	}
	if r.s.syncLogin != "cyan@example.com" {
		t.Errorf("the email was not kept: %q", r.s.syncLogin)
	}
	has(t, r.tt, "Create account")
}

func TestSyncResetCodeStepChangesEmailToTheResetEmailStep(t *testing.T) {
	r, fs := syncPane(t, resetCodeStepSync())
	if err := r.tt.Click("Use a different email"); err != nil {
		t.Fatal(err)
	}
	if fs.snap.SignInStep != data.SignInStepResetEmail || r.s.syncLogin != "cyan@example.com" {
		t.Fatalf("step %q email %q", fs.snap.SignInStep, r.s.syncLogin)
	}
}

func TestSyncFailureLines(t *testing.T) {
	cases := map[data.SignInFailure]string{
		data.SignInFailureWrongCode:   "That code isn't right",
		data.SignInFailureCodeExpired: "That code expired; we sent a new one",
		data.SignInFailureTooMany:     "Too many tries; wait a minute",
		data.SignInFailureLocked:      "This account is deactivated; ask the admin",
		data.SignInFailureNetwork:     "the server did not answer",
	}
	for failure, line := range cases {
		snap := verifyStepSync()
		snap.SignInFailure = failure
		r, _ := syncPane(t, snap)
		if failure == data.SignInFailureNetwork {
			line = r.s.say.SettingsSyncFailedNoAnswer()
		}
		has(t, r.tt, line)
	}

	snap := signedOutCodeSync()
	snap.SignInFailure = data.SignInFailureCredentials
	r, _ := syncPane(t, snap)
	has(t, r.tt, "Wrong email or password")

	snap = signUpTabSync()
	snap.SignInFailure = data.SignInFailureWeakPassword
	r, _ = syncPane(t, snap)
	has(t, r.tt, "At least 8 characters.")
}

func TestSyncWrongCodeEmptiesTheCells(t *testing.T) {
	r, fs := syncPane(t, verifyStepSync())
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

func TestSyncBusyLines(t *testing.T) {
	for stage, line := range map[data.SignInStage]string{
		data.SignInStageSending:     "Sending the code…",
		data.SignInStageCreating:    "Creating your account…",
		data.SignInStageChecking:    "Checking your account…",
		data.SignInStageDownloading: "Downloading your tasks…",
	} {
		snap := signUpTabSync()
		snap.Phase, snap.SignInStage = data.SyncSigningIn, stage
		r, _ := syncPane(t, snap)
		has(t, r.tt, line)
	}
}

func TestSyncTermsLinksOpenTheWebsite(t *testing.T) {
	r, _ := syncPane(t, signUpTabSync())
	if err := r.tt.Click("Terms of Service"); err != nil {
		t.Fatal(err)
	}
	if err := r.tt.Click("Privacy Policy"); err != nil {
		t.Fatal(err)
	}
	if len(r.rec.urls) != 2 || r.rec.urls[0] != termsURL || r.rec.urls[1] != privacyURL {
		t.Fatalf("opened %v", r.rec.urls)
	}
}

func TestSyncSendingAndDownloadingLines(t *testing.T) {
	snap := signedOutCodeSync()
	snap.Phase, snap.SignInStage = data.SyncSigningIn, data.SignInStageSending
	r, fs := syncPane(t, snap)
	has(t, r.tt, "Sending the code…")
	fs.snap.SignInStep, fs.snap.SignInEmail = data.SignInStepVerify, "cyan@example.com"
	fs.snap.SignInStage = data.SignInStageDownloading
	r.tt.Frame()
	has(t, r.tt, "Downloading your tasks…")
}
