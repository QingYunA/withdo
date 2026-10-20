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
	name string
	snap data.SyncSnapshot
} {
	now := fixedNow()
	signedOut := data.SyncSnapshot{Phase: data.SyncSignedOut, Server: "https://todo.sri-ai.xyz"}
	signing := signedOut
	signing.Phase = data.SyncSigningIn
	both := signedOut
	both.Phase, both.Linked, both.Account = data.SyncBothHaveTasks, true, "Cyan 云辰"
	both.Devices = []data.SyncDevice{
		{ID: "phone", Name: "Pixel 8", LastSeen: now.Add(-3 * time.Minute)},
		{ID: "mac", Name: "Cyan 的 MacBook Pro", LastSeen: now.Add(-time.Minute), This: true},
	}
	both.Conflict = &data.SyncConflict{LocalTasks: 128, LocalProjects: 6, AccountDevices: 2}
	uploading := signedOut
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
		name string
		snap data.SyncSnapshot
	}{
		{"signed-out", signedOut},
		{"signing-in", signing},
		{"both", both},
		{"uploading", uploading},
		{"synced", synced},
		{"failed", failed},
		{"offline", offline},
	}
}

func TestScreensSettingsSync(t *testing.T) {
	screensTest(t)
	ready := data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow()}
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh" + a.suffix
			}
			// One app for every state: the pane only reads it, and a demo store per picture is slow.
			app := settingsApp(t, ready)
			for _, st := range syncScreenStates() {
				app.Sync = &fakeSync{snap: st.snap}
				settingsShot(t, "settings-sync-"+st.name+sfx+".png", app, a.dark, zh, PaneSync, nil)
			}
			// The server address while it is being edited.
			app.Sync = &fakeSync{snap: signedOutSync()}
			settingsShot(t, "settings-sync-change"+sfx+".png", app, a.dark, zh, PaneSync, func(r *settingsRig) {
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
				zhShot(t, "settings-sync-sidebar"+sfx+".png", app, a.dark, nil)
			} else {
				mainShot(t, "settings-sync-sidebar"+sfx+".png", app, a.dark, nil)
			}

			failure := syncedSync()
			failure.Phase = data.SyncFailed
			failure.FailedSince = fixedNow().Add(-16 * time.Minute)
			app.Sync = &fakeSync{snap: failure}
			if zh {
				zhShot(t, "settings-sync-sidebar-alert"+sfx+".png", app, a.dark, nil)
			} else {
				mainShot(t, "settings-sync-sidebar-alert"+sfx+".png", app, a.dark, nil)
			}
		}
	}
}
