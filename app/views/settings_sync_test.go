package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// fakeSync is the SyncService of the view tests and the design screens: it records what the pane
// asked for and shows the snapshot the test set.
type fakeSync struct {
	snap data.SyncSnapshot

	signIns, merges, uses, cancels, signOuts, syncNows, retries, refreshes int
	login, password                                                        string
	server                                                                 string
	setErr                                                                 error
}

func (f *fakeSync) SyncSnapshot() data.SyncSnapshot { return f.snap }
func (f *fakeSync) SyncSignIn(login, password string) {
	f.signIns++
	f.login, f.password = login, password
}
func (f *fakeSync) SyncMerge()      { f.merges++ }
func (f *fakeSync) SyncUseAccount() { f.uses++ }
func (f *fakeSync) SyncCancel() {
	f.cancels++
	f.snap.Conflict = nil
	f.snap.Phase = data.SyncSignedOut
}
func (f *fakeSync) SyncSignOut()        { f.signOuts++ }
func (f *fakeSync) SyncNow()            { f.syncNows++ }
func (f *fakeSync) SyncRetry()          { f.retries++ }
func (f *fakeSync) SyncRefreshDevices() { f.refreshes++ }
func (f *fakeSync) SyncSetServer(server string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.server, f.snap.Server = server, server
	return nil
}

func signedOutSync() data.SyncSnapshot {
	return data.SyncSnapshot{Phase: data.SyncSignedOut, Server: "https://todo.sri-ai.xyz"}
}

func syncedSync() data.SyncSnapshot {
	now := fixedNow()
	return data.SyncSnapshot{
		Phase: data.SyncSynced, Server: "https://todo.sri-ai.xyz", Account: "Cyan 云辰",
		Linked: true, LastSynced: now.Add(-10 * time.Second), ThisDeviceID: "mac",
		Devices: []data.SyncDevice{
			{ID: "phone", Name: "Pixel 8", LastSeen: now.Add(-3 * time.Minute)},
			{ID: "mac", Name: "Cyan 的 MacBook Pro", LastSeen: now.Add(-10 * time.Second), This: true},
			{ID: "mini", Name: "公司的 Mac mini", LastSeen: now.Add(-24 * time.Hour)},
		},
	}
}

// syncPane opens Settings > Sync over a service showing snap.
func syncPane(t *testing.T, snap data.SyncSnapshot) (*settingsRig, *fakeSync) {
	t.Helper()
	fs := &fakeSync{snap: snap}
	r := newSettingsRig(t, func(a *App) { a.Sync = fs })
	r.s.Show(PaneSync)
	r.tt.Frame()
	return r, fs
}

func TestSyncPaneSignedOut(t *testing.T) {
	r, fs := syncPane(t, signedOutSync())
	has(t, r.tt, "Sync with your phone and other computers", "Email or username", "Password",
		"Sign in and sync", "Server", "todo.sri-ai.xyz", "Change")
	has(t, r.tt, "Accounts are made by the server's admin. Keys and model settings stay on this Mac.")
	hasNot(t, r.tt, "The sign-in page opens in the browser.", "Finish signing in in your browser.")
	if fs.refreshes == 0 {
		t.Error("opening the pane did not ask for the devices")
	}
	// The account field has the keyboard when the pane opens, and Enter moves to the password.
	r.tt.Type("cyan@example.com")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	r.tt.Type("hunter2")
	r.tt.Frame()
	if r.s.syncLogin != "cyan@example.com" || r.s.syncPassword != "hunter2" {
		t.Fatalf("typed account %q, password %q", r.s.syncLogin, r.s.syncPassword)
	}
	if err := r.tt.Click("Sign in and sync"); err != nil {
		t.Fatal(err)
	}
	if fs.signIns != 1 || fs.login != "cyan@example.com" || fs.password != "hunter2" {
		t.Errorf("Sign in and sync called %d times with account %q, password %q", fs.signIns, fs.login, fs.password)
	}
}

func TestSyncPaneSigningInDisablesTheFields(t *testing.T) {
	snap := signedOutSync()
	snap.Phase = data.SyncSigningIn
	r, _ := syncPane(t, snap)
	has(t, r.tt, "Signing in…", "Email or username", "Password")
	hasNot(t, r.tt, "Sign in and sync")
	r.tt.Type("cyan@example.com")
	if r.s.syncLogin != "" {
		t.Errorf("a disabled field took text: %q", r.s.syncLogin)
	}
}

func TestSyncPaneSignInFailures(t *testing.T) {
	cases := map[data.SignInFailure]string{
		data.SignInFailureCredentials:  "Wrong account or password",
		data.SignInFailureTooMany:      "Too many tries; wait a few minutes",
		data.SignInFailureSecondFactor: "This account uses two-step verification, which in-app sign-in does not support yet",
		data.SignInFailureLocked:       "This account is deactivated; ask the admin",
		data.SignInFailureUnsupported:  "This server does not support signing in from the app",
		data.SignInFailureNetwork:      "the server did not answer",
		data.SignInFailureOther:        "Couldn't sign in. Try again.",
	}
	// One app for every kind: each newSettingsRig seeds a demo database, which costs minutes under
	// the race detector.
	r, fs := syncPane(t, signedOutSync())
	for failure, want := range cases {
		fs.snap.SignInFailure = failure
		r.tt.Frame()
		has(t, r.tt, want)
	}
}

func TestSyncPaneCredentialsErrorKeepsTheAccount(t *testing.T) {
	r, fs := syncPane(t, signedOutSync())
	r.tt.Type("cyan@example.com")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	r.tt.Type("wrong")
	r.tt.Frame()
	fs.snap.Phase = data.SyncSigningIn
	r.tt.Frame()
	fs.snap.Phase = data.SyncSignedOut
	fs.snap.SignInFailure = data.SignInFailureCredentials
	r.tt.Frame()
	if r.s.syncLogin != "cyan@example.com" {
		t.Errorf("the account was lost: %q", r.s.syncLogin)
	}
	if r.s.syncPassword != "" {
		t.Errorf("the password was kept: %q", r.s.syncPassword)
	}
	// The focus came back to the password: typing lands there.
	r.tt.Type("second try")
	if r.s.syncPassword != "second try" {
		t.Errorf("typing after the error went to %q", r.s.syncPassword)
	}
}

func TestSyncPaneBothHaveTasksMerge(t *testing.T) {
	snap := signedOutSync()
	snap.Phase = data.SyncBothHaveTasks
	snap.Linked, snap.Account = true, "Cyan 云辰"
	snap.Devices = []data.SyncDevice{{ID: "other", Name: "Cyan 的 MacBook Pro"}}
	snap.Conflict = &data.SyncConflict{LocalTasks: 128, LocalProjects: 6, AccountDevices: 2}
	r, fs := syncPane(t, snap)
	has(t, r.tt, "This Mac and your account both have tasks",
		"The account already has data from 2 devices. This Mac has 128 tasks and 6 projects.",
		"Merge into the account", "Recommended", "Use the account", "Cancel", "Continue")
	if err := r.tt.Click("Continue"); err != nil {
		t.Fatal(err)
	}
	if fs.merges != 1 || fs.uses != 0 {
		t.Errorf("Continue merged %d, used the account %d", fs.merges, fs.uses)
	}
}

func TestSyncPaneBothHaveTasksUseTheAccount(t *testing.T) {
	snap := signedOutSync()
	snap.Phase = data.SyncBothHaveTasks
	snap.Linked, snap.Account = true, "Cyan 云辰"
	snap.Conflict = &data.SyncConflict{LocalTasks: 128, LocalProjects: 6, AccountDevices: 2}
	r, fs := syncPane(t, snap)
	if err := r.tt.Click("Use the account"); err != nil {
		t.Fatal(err)
	}
	if err := r.tt.Click("Continue"); err != nil {
		t.Fatal(err)
	}
	if fs.uses != 1 || fs.merges != 0 {
		t.Errorf("Continue merged %d, used the account %d", fs.merges, fs.uses)
	}
}

func TestSyncPaneBothHaveTasksCancel(t *testing.T) {
	snap := signedOutSync()
	snap.Phase = data.SyncBothHaveTasks
	snap.Linked, snap.Account = true, "Cyan 云辰"
	snap.Conflict = &data.SyncConflict{LocalTasks: 128, LocalProjects: 6, AccountDevices: 2}
	r, fs := syncPane(t, snap)
	if err := r.tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if fs.cancels != 1 {
		t.Errorf("Cancel called %d times", fs.cancels)
	}
}

func TestSyncPaneUploadingShowsTheCountAndTheBar(t *testing.T) {
	snap := signedOutSync()
	snap.Phase = data.SyncUploading
	snap.Linked, snap.Account = true, "Cyan 云辰"
	snap.SeedDone, snap.SeedTotal, snap.Pending = 1240, 3512, 2272
	r, _ := syncPane(t, snap)
	has(t, r.tt, "Uploading this Mac's data",
		"1,240 / 3,512 · other devices wait until it is done",
		"You can close this window; the upload goes on in the background.")
	if got := syncBarFilled(1240, 3512); got != 14 {
		t.Errorf("the bar lights %d cells of %d, want 14", got, syncBarCells)
	}
	if got := syncBarFilled(0, 3512); got != 0 {
		t.Errorf("an empty upload lights %d cells", got)
	}
	if got := syncBarFilled(3512, 3512); got != syncBarCells {
		t.Errorf("a finished upload lights %d cells", got)
	}
}

func TestSyncPaneSyncedListsDevicesWithThisMacFirst(t *testing.T) {
	r, fs := syncPane(t, syncedSync())
	has(t, r.tt, "Cyan 云辰", "Signed in · 3 devices", "Sign out", "This Mac")
	has(t, r.tt, "Synced", "just now", "Sync now", "Devices", "Cyan 的 MacBook Pro", "Pixel 8", "公司的 Mac mini")
	if err := r.tt.Click("Sync now"); err != nil {
		t.Fatal(err)
	}
	mac, ok := r.tt.Find("Cyan 的 MacBook Pro")
	if !ok {
		t.Fatal("no this Mac row")
	}
	phone, ok := r.tt.Find("Pixel 8")
	if !ok || phone.Y < mac.Y {
		t.Errorf("this Mac is not first: Mac at %.0f, phone at %.0f", mac.Y, phone.Y)
	}
	if err := r.tt.Click("Sign out"); err != nil {
		t.Fatal(err)
	}
	if fs.syncNows != 1 || fs.signOuts != 1 {
		t.Errorf("Sync now called %d times, Sign out %d", fs.syncNows, fs.signOuts)
	}
}

func TestSyncPaneFailedRetries(t *testing.T) {
	snap := syncedSync()
	snap.Phase = data.SyncFailed
	snap.LastError = "the server did not answer"
	snap.Failure = data.SyncFailureNoAnswer
	snap.RetryAt = fixedNow().Add(2 * time.Minute)
	snap.Pending = 3
	r, fs := syncPane(t, snap)
	has(t, r.tt, "Sync failed: the server did not answer", "Retrying in 2 min",
		"3 changes wait on this Mac and are not lost", "Retry")
	hasNot(t, r.tt, "Sync now")
	if err := r.tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	if fs.retries != 1 {
		t.Errorf("Retry called %d times", fs.retries)
	}
}

func TestSyncPaneOfflineHasNoButton(t *testing.T) {
	snap := syncedSync()
	snap.Phase = data.SyncOffline
	snap.Pending = 3
	r, _ := syncPane(t, snap)
	has(t, r.tt, "Offline · changes sync when you are back online (3 waiting)")
	hasNot(t, r.tt, "Retry", "Sync now", "Sync failed")
}

func TestSyncPaneChangeServer(t *testing.T) {
	r, fs := syncPane(t, signedOutSync())
	if err := r.tt.Click("Change"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	r.tt.Key(ui.Cmd, ui.KeyA)
	r.tt.Type("https://sync.example.com/")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if fs.server != "https://sync.example.com/" {
		t.Errorf("saved %q", fs.server)
	}
	if r.tt.HasText("todo.sri-ai.xyz") {
		t.Error("the footer kept the old server")
	}

	// Esc keeps the old server.
	if err := r.tt.Click("Change"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	r.tt.Key(ui.Cmd, ui.KeyA)
	r.tt.Type("https://other.example.com")
	r.tt.Key(0, ui.KeyEscape)
	r.tt.Frame()
	if fs.server != "https://sync.example.com/" {
		t.Errorf("Esc saved %q", fs.server)
	}
	has(t, r.tt, "sync.example.com")
}

func TestSidebarSyncCloud(t *testing.T) {
	app, _, rec := newTestApp(t)
	failed := syncedSync()
	failed.Phase = data.SyncFailed
	failed.FailedSince = fixedNow().Add(-16 * time.Minute)
	fs := &fakeSync{snap: failed}
	app.Sync = fs
	tt, _, _ := newTesterFor(app, 1100)
	if !tt.HasText("Sync needs attention") {
		t.Fatalf("no cloud after 16 minutes of failure: %q", tt.Texts())
	}
	mustNil(t, tt.Click("Settings"))
	if len(rec.panes) != 1 || rec.panes[0] != PaneSync {
		t.Errorf("Settings opened on %v", rec.panes)
	}

	// Five minutes of failure is still quiet.
	app2, _, rec2 := newTestApp(t)
	failed.FailedSince = fixedNow().Add(-5 * time.Minute)
	app2.Sync = &fakeSync{snap: failed}
	tt2, _, _ := newTesterFor(app2, 1100)
	if tt2.HasText("Sync needs attention") {
		t.Error("the cloud showed after five minutes")
	}
	mustNil(t, tt2.Click("Settings"))
	if len(rec2.panes) != 1 || rec2.panes[0] != PaneGeneral {
		t.Errorf("Settings opened on %v", rec2.panes)
	}

	// A linked Mac that signed out shows it at once.
	app3, _, _ := newTestApp(t)
	out := syncedSync()
	out.Phase, out.Linked, out.Account = data.SyncSignedOut, true, ""
	app3.Sync = &fakeSync{snap: out}
	tt3, _, _ := newTesterFor(app3, 1100)
	if !tt3.HasText("Sync needs attention") {
		t.Error("the cloud did not show after signing out")
	}
}

func TestAReplyReportShowsYou(t *testing.T) {
	app, _ := newEmptyApp(t)
	mustNil(t, app.AddTask(data.CreateTaskInput{Title: "Answer the design question", Source: data.SourceUser}))
	task := app.snap.tasks[0]
	app.Store.(*data.Real).SeedReport(data.Report{
		TaskID: task.ID, Agent: data.You, Kind: data.ReportReply, Summary: "Thanks, that is done.",
		CreatedAt: fixedNow().Add(-time.Minute),
	})
	app.Refresh()
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	still(tt)
	m.OpenTask(task.ID)
	tt.Frame()
	has(t, tt, "You", "Thanks, that is done.")
}
