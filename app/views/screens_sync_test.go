package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The M11 Sync screens: every state of Settings > Sync and the two sidebar states, in light and
// dark, English and Chinese (the -zh suffix). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSettingsSync

func syncScreenStates() []struct {
	name  string
	snap  data.SyncSnapshot
	setup func(r *settingsRig)
} {
	now := fixedNow()
	// The signed-out card of a PocketBase server: the email, the code, the password, the reset.
	email := signedOutCodeSync()
	email.GoogleAvailable = true
	googleWaiting := email
	googleWaiting.Phase, googleWaiting.SignInStage = data.SyncSigningIn, data.SignInStageGoogle
	googleFailed := email
	googleFailed.SignInFailure = data.SignInFailureGoogleFailed
	code := email
	code.SignInStep, code.SignInEmail = data.SignInStepCode, "cyan@example.com"
	code.ResendAt = now.Add(24 * time.Second)
	codeChecking := code
	codeChecking.Phase, codeChecking.SignInStage = data.SyncSigningIn, data.SignInStageChecking
	codeWrong := code
	codeWrong.ResendAt = now.Add(-time.Second)
	codeWrong.SignInFailure = data.SignInFailureWrongCode
	codeExpired := code
	codeExpired.SignInFailure = data.SignInFailureCodeExpired
	tooMany := code
	tooMany.SignInFailure = data.SignInFailureTooMany
	tooMany.ResendAt = now.Add(-time.Second)
	password := code
	password.SignInStep = data.SignInStepPassword
	resetSent := code
	resetSent.SignInStep = data.SignInStepResetSent
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
	return []struct {
		name  string
		snap  data.SyncSnapshot
		setup func(r *settingsRig)
	}{
		{"email", email, nil},
		{"email-filled", email, func(r *settingsRig) { r.s.syncLogin = "cyan@example.com" }},
		{"google-waiting", googleWaiting, nil},
		{"google-failed", googleFailed, nil},
		{"code", code, func(r *settingsRig) { r.s.syncCode = "48" }},
		{"code-checking", codeChecking, func(r *settingsRig) { r.s.syncCode, r.s.syncCodeTried = "482913", "482913" }},
		{"code-wrong", codeWrong, nil},
		{"code-expired", codeExpired, nil},
		{"code-too-many", tooMany, nil},
		{"password", password, nil},
		{"reset-sent", resetSent, nil},
		{"both", both, nil},
		{"uploading", uploading, nil},
		{"synced", synced, nil},
		{"failed", failed, nil},
		{"offline", offline, nil},
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
				change := "Change"
				if zh {
					change = "更改"
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
