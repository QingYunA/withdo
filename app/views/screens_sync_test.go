package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The Account screens (M11, v42): every state of Settings > Account and the two sidebar states, in light and
// dark, English and Chinese (the -zh suffix). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSettingsSync

func syncScreenStates() []struct {
	name  string
	snap  data.SyncSnapshot
	setup func(r *settingsRig)
} {
	now := fixedNow()
	// The signed-out card of a PocketBase server: Sign in | Sign up, Verify, Forgot password.
	signIn := signedOutCodeSync()
	signIn.GoogleAvailable = true
	email := signIn
	signInWrong := signIn
	signInWrong.SignInFailure = data.SignInFailureCredentials
	signUp := signIn
	signUp.SignInStep = data.SignInStepSignUp
	signUpTaken := signUp
	signUpTaken.SignInFailure = data.SignInFailureEmailTaken
	googleWaiting := signIn
	googleWaiting.Phase, googleWaiting.SignInStage = data.SyncSigningIn, data.SignInStageGoogle
	googleFailed := signIn
	googleFailed.SignInFailure = data.SignInFailureGoogleFailed
	verify := signIn
	verify.SignInStep, verify.SignInEmail = data.SignInStepVerify, "cyan@example.com"
	verify.ResendAt = now.Add(24 * time.Second)
	verifyChecking := verify
	verifyChecking.Phase, verifyChecking.SignInStage = data.SyncSigningIn, data.SignInStageChecking
	verifyWrong := verify
	verifyWrong.ResendAt = now.Add(-time.Second)
	verifyWrong.SignInFailure = data.SignInFailureWrongCode
	verifyExpired := verify
	verifyExpired.SignInFailure = data.SignInFailureCodeExpired
	tooMany := verify
	tooMany.SignInFailure = data.SignInFailureTooMany
	tooMany.ResendAt = now.Add(-time.Second)
	resetEmail := signIn
	resetEmail.SignInStep = data.SignInStepResetEmail
	resetCode := signIn
	resetCode.SignInStep, resetCode.SignInEmail = data.SignInStepResetCode, "cyan@example.com"
	resetCode.ResendAt = now.Add(24 * time.Second)
	both := email
	both.Phase, both.Linked, both.Account = data.SyncBothHaveTasks, true, "Cyan 云辰"
	both.Devices = []data.SyncDevice{
		{ID: "phone", Name: "Pixel 8", LastSeen: now.Add(-3 * time.Minute)},
		{ID: "mac", Name: "Cyan 的 MacBook Pro", LastSeen: now.Add(-time.Minute), This: true},
	}
	both.Conflict = &data.SyncConflict{LocalTasks: 128, LocalProjects: 6, AccountDevices: 2}
	uploading := email
	uploading.Phase, uploading.Linked, uploading.Account = data.SyncUploading, true, "Cyan 云辰"
	uploading.SeedDone, uploading.SeedTotal, uploading.Pending = 1240, 3512, 2272
	synced := syncedSync()
	synced.Devices = []data.SyncDevice{
		{ID: "phone", Name: "Pixel 8", LastSeen: now.Add(-3 * time.Minute)},
		{ID: "mac", Name: "Cyan 的 MacBook Pro", LastSeen: now.Add(-10 * time.Second), This: true},
		{ID: "mini", Name: "公司的 Mac mini", LastSeen: now.Add(-24 * time.Hour)},
	}
	failed := synced
	failed.Phase = data.SyncFailed
	failed.LastError = "the server did not answer"
	failed.Failure = data.SyncFailureNoAnswer
	failed.RetryAt = now.Add(2 * time.Minute)
	failed.Pending = 3
	offline := synced
	offline.Phase = data.SyncOffline
	offline.Pending = 3
	wrongDevice := synced
	wrongDevice.Phase, wrongDevice.Failure = data.SyncWrongAccount, data.SyncFailureWrongAccount
	wrongDevice.LastError = "sync: this device is linked to another account (status 403)"
	wrongDevice.Pending = 204
	otherSheet := both
	otherSheet.Phase, otherSheet.Conflict = data.SyncOtherAccountSheet, nil
	otherSheet.Account = "Bee"
	otherSheet.OtherAccount = &data.SyncOtherAccount{PreviousEmail: "ada@example.com", AccountEmail: "bee@example.com"}
	googleLink := synced
	googleLink.GoogleLink = data.GoogleLinkAvailable
	googleLinked := synced
	googleLinked.GoogleLink = data.GoogleLinkLinked
	return []struct {
		name  string
		snap  data.SyncSnapshot
		setup func(r *settingsRig)
	}{
		{"signin", signIn, nil},
		{"signin-filled", email, func(r *settingsRig) {
			r.s.syncLogin, r.s.syncPassword = "cyan@example.com", "hunter2hunter2"
		}},
		{"signin-wrong", signInWrong, func(r *settingsRig) { r.s.syncLogin = "cyan@example.com" }},
		{"signup", signUp, nil},
		{"signup-filled", signUp, func(r *settingsRig) {
			r.s.syncLogin, r.s.syncPassword = "cyan@example.com", "correct horse battery"
		}},
		{"signup-taken", signUpTaken, func(r *settingsRig) {
			r.s.syncLogin, r.s.syncPassword = "cyan@example.com", "correct horse battery"
		}},
		{"google-waiting", googleWaiting, nil},
		{"google-failed", googleFailed, nil},
		{"verify", verify, func(r *settingsRig) { r.s.syncCode = "48" }},
		{"verify-checking", verifyChecking, func(r *settingsRig) { r.s.syncCode, r.s.syncCodeTried = "482913", "482913" }},
		{"verify-wrong", verifyWrong, nil},
		{"verify-expired", verifyExpired, nil},
		{"verify-too-many", tooMany, nil},
		{"reset-email", resetEmail, func(r *settingsRig) { r.s.syncLogin = "cyan@example.com" }},
		{"reset-code", resetCode, func(r *settingsRig) {
			r.s.syncCode, r.s.syncCodeTried, r.s.syncPassword = "481207", "481207", "correct horse battery"
		}},
		{"both", both, nil},
		{"uploading", uploading, nil},
		{"synced", synced, nil},
		{"failed", failed, nil},
		{"offline", offline, nil},
		{"wrong-device", wrongDevice, nil},
		{"other-account-sheet", otherSheet, nil},
		{"google-link", googleLink, nil},
		{"google-linked", googleLinked, nil},
	}
}

func TestScreensSettingsSync(t *testing.T) {
	screensTest(t)
	ready := data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow()}
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh-" + a.suffix
			}
			// One app for every state: the pane only reads it, and a demo store per picture is slow.
			app := settingsApp(t, ready)
			for _, st := range syncScreenStates() {
				app.Sync = &fakeSync{snap: st.snap}
				settingsShot(t, "settings-account-"+st.name+sfx+".png", app, a.dark, zh, PaneAccount, st.setup)
			}
			// The server address while it is being edited.
			app.Sync = &fakeSync{snap: signedOutCodeSync()}
			settingsShot(t, "settings-account-change"+sfx+".png", app, a.dark, zh, PaneAccount, func(r *settingsRig) {
				change := "Use your own server"
				if zh {
					change = "使用自己的服务器"
				}
				mustNil(t, r.tt.Click(change))
				r.tt.Frame()
			})

			// The sidebar is quiet while sync works, and carries the red cloud when it needs the
			// user: after a long failure, or after signing out.
			app.Sync = &fakeSync{snap: syncedSync()}
			if zh {
				zhShot(t, "settings-account-sidebar"+sfx+".png", app, a.dark, nil)
			} else {
				mainShot(t, "settings-account-sidebar"+sfx+".png", app, a.dark, nil)
			}

			failure := syncedSync()
			failure.Phase = data.SyncFailed
			failure.FailedSince = fixedNow().Add(-16 * time.Minute)
			app.Sync = &fakeSync{snap: failure}
			if zh {
				zhShot(t, "settings-account-sidebar-alert"+sfx+".png", app, a.dark, nil)
			} else {
				mainShot(t, "settings-account-sidebar-alert"+sfx+".png", app, a.dark, nil)
			}
		}
	}
}
