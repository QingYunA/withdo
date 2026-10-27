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
	checks, resends, googles                                               int
	open                                                                   func(string)
	requestedCodes, codes, resets                                          []string
	useOld, moves, googleRefreshes, googleLinks, googleLinkCancels         int
}

func (f *fakeSync) SyncSnapshot() data.SyncSnapshot { return f.snap }
func (f *fakeSync) SyncSignIn(login, password string) {
	f.signIns++
	f.login, f.password = login, password
}
func (f *fakeSync) SyncCheckServer() { f.checks++ }
func (f *fakeSync) SyncRequestCode(email string) {
	f.requestedCodes = append(f.requestedCodes, email)
}
func (f *fakeSync) SyncResendCode() { f.resends++ }
func (f *fakeSync) SyncSignInWithGoogle(open func(url string)) {
	f.googles++
	f.open = open
}
func (f *fakeSync) SyncSignInWithCode(code string) {
	f.codes = append(f.codes, code)
}
func (f *fakeSync) SyncRequestPasswordReset(email string) {
	f.resets = append(f.resets, email)
}
func (f *fakeSync) SyncSetSignInStep(step data.SignInStep) { f.snap.SignInStep = step }
func (f *fakeSync) SyncMerge()                             { f.merges++ }
func (f *fakeSync) SyncUseAccount()                        { f.uses++ }
func (f *fakeSync) SyncCancel() {
	f.cancels++
	f.snap.Conflict = nil
	f.snap.Phase = data.SyncSignedOut
}
func (f *fakeSync) SyncUseOldAccount() {
	f.useOld++
	f.snap.OtherAccount, f.snap.Phase = nil, data.SyncSignedOut
	f.snap.SignInEmail = "old"
}
func (f *fakeSync) SyncMoveToAccount()     { f.moves++ }
func (f *fakeSync) SyncRefreshGoogleLink() { f.googleRefreshes++ }
func (f *fakeSync) SyncLinkGoogle(open func(url string)) {
	f.googleLinks++
	f.open = open
}
func (f *fakeSync) SyncCancelGoogleLink() { f.googleLinkCancels++ }
func (f *fakeSync) SyncSignOut()          { f.signOuts++ }
func (f *fakeSync) SyncNow()              { f.syncNows++ }
func (f *fakeSync) SyncRetry()            { f.retries++ }
func (f *fakeSync) SyncRefreshDevices()   { f.refreshes++ }
func (f *fakeSync) SyncSetServer(server string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.server, f.snap.Server = server, server
	return nil
}

// signedOutCodeSync is the signed-out card of a PocketBase server: the email step.
func signedOutCodeSync() data.SyncSnapshot {
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
	r.s.Show(PaneAccount)
	r.tt.Frame()
	return r, fs
}

func TestSyncPaneBothHaveTasksMerge(t *testing.T) {
	snap := signedOutCodeSync()
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
	snap := signedOutCodeSync()
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
	snap := signedOutCodeSync()
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
	snap := signedOutCodeSync()
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

func TestSyncPaneAccountedListsDevicesWithThisMacFirst(t *testing.T) {
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
	r, fs := syncPane(t, signedOutCodeSync())
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
	if len(rec.panes) != 1 || rec.panes[0] != PaneAccount {
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
	mustNil(t, app.AddTask(data.CreateTaskInput{Title: "Answer the design question", Source: data.SourceAgent}))
	task := app.snap.tasks[0]
	app.Store.(*data.Real).SeedReport(data.Report{
		TaskID: task.ID, Agent: data.You, Kind: data.ReportReply, Summary: "Thanks, that is done.",
		CreatedAt: fixedNow().Add(-time.Minute),
	})
	app.Refresh()
	m := NewMain(app)
	tt := newTester(m.View, 1100, 720)
	still(tt)
	m.OpenTask(task.ID)
	tt.Frame()
	has(t, tt, "You", "Thanks, that is done.")
}

func wrongAccountSnap() data.SyncSnapshot {
	snap := syncedSync()
	snap.Phase, snap.Failure = data.SyncWrongAccount, data.SyncFailureWrongAccount
	snap.LastError = "sync: this device is linked to another account (status 403)"
	snap.Pending = 204
	snap.GoogleLink = data.GoogleLinkAvailable
	return snap
}

// The server's 403 for a device of another account says so, offers Sign out in This Mac's row, and
// has no Retry, no Sync now and no "server did not answer".
func TestSyncPaneWrongAccount(t *testing.T) {
	r, fs := syncPane(t, wrongAccountSnap())
	has(t, r.tt, "This Mac is linked to another account. Sign out, then sign in with that account.",
		"204 changes wait on this Mac and are not lost", "Sign out")
	hasNot(t, r.tt, "Retry", "Sync now", "Sync failed", "the server did not answer", "Google")
	if err := r.tt.Click("Sign out"); err != nil {
		t.Fatal(err)
	}
	if fs.signOuts != 1 {
		t.Errorf("Sign out called %d times", fs.signOuts)
	}
}

// An error of a kind the pane does not know is not worded as the server not answering.
func TestSyncPaneUnknownFailureIsNotNoAnswer(t *testing.T) {
	snap := syncedSync()
	snap.Phase, snap.Failure = data.SyncFailed, data.SyncFailureUnknown
	snap.LastError = "sync server: not found (status 404)"
	r, _ := syncPane(t, snap)
	has(t, r.tt, "Sync failed: something went wrong")
	hasNot(t, r.tt, "did not answer")
}

func otherAccountSnap(previous string) data.SyncSnapshot {
	snap := signedOutCodeSync()
	snap.Phase = data.SyncOtherAccountSheet
	snap.Linked, snap.Account = true, "Bee"
	snap.OtherAccount = &data.SyncOtherAccount{PreviousEmail: previous, AccountEmail: "bee@example.com"}
	return snap
}

func TestSyncPaneOtherAccountSheetNamesBothAndDefaultsToTheOldAccount(t *testing.T) {
	r, fs := syncPane(t, otherAccountSnap("ada@example.com"))
	has(t, r.tt, "This Mac was linked to another account",
		"It was synced with ada@example.com. You just signed in as bee@example.com.",
		"Sign in with ada@example.com instead", "Recommended",
		"Move this Mac to bee@example.com", "Cancel", "Continue")
	if err := r.tt.Click("Continue"); err != nil {
		t.Fatal(err)
	}
	if fs.useOld != 1 || fs.moves != 0 {
		t.Fatalf("Continue chose the old account %d times and moved %d times", fs.useOld, fs.moves)
	}
	if r.s.syncLogin != "ada@example.com" {
		t.Errorf("the email field = %q, want the old address", r.s.syncLogin)
	}
}

func TestSyncPaneOtherAccountMove(t *testing.T) {
	r, fs := syncPane(t, otherAccountSnap("ada@example.com"))
	if err := r.tt.Click("Move this Mac to bee@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := r.tt.Click("Continue"); err != nil {
		t.Fatal(err)
	}
	if fs.moves != 1 || fs.useOld != 0 {
		t.Fatalf("moved %d times, chose the old account %d times", fs.moves, fs.useOld)
	}
}

func TestSyncPaneOtherAccountCancelSignsOut(t *testing.T) {
	r, fs := syncPane(t, otherAccountSnap(""))
	has(t, r.tt, "You just signed in as bee@example.com.", "Sign in with another account")
	if err := r.tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if fs.cancels != 1 {
		t.Errorf("Cancel called %d times", fs.cancels)
	}
}

func TestSyncPaneGoogleRow(t *testing.T) {
	snap := syncedSync()
	snap.GoogleLink = data.GoogleLinkAvailable
	r, fs := syncPane(t, snap)
	has(t, r.tt, "Google", "Link")
	if err := r.tt.Click("Link"); err != nil {
		t.Fatal(err)
	}
	if fs.googleLinks != 1 {
		t.Errorf("Link called %d times", fs.googleLinks)
	}
	if fs.googleRefreshes < 1 {
		t.Error("the pane never asked whether Google is linked")
	}

	snap.GoogleLink = data.GoogleLinkLinked
	r, _ = syncPane(t, snap)
	has(t, r.tt, "Google", "Linked")

	snap.GoogleLink, snap.GoogleLinkFailure = data.GoogleLinkAvailable, data.GoogleLinkFailureUsed
	r, _ = syncPane(t, snap)
	has(t, r.tt, "This Google account is already used by another account", "Link")

	snap.GoogleLink, snap.GoogleLinkFailure = data.GoogleLinkWaiting, data.GoogleLinkFailureNone
	r, fs = syncPane(t, snap)
	has(t, r.tt, "Finish in your browser", "Cancel")
	if err := r.tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if fs.googleLinkCancels != 1 {
		t.Errorf("Cancel called %d times", fs.googleLinkCancels)
	}
}
