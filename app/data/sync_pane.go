package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// DefaultSyncServer is where a Mac syncs until the user changes it. The pane shows only the host.
const DefaultSyncServer = "https://todo.sri-ai.xyz"

// SyncPhase is what Settings > Sync shows.
type SyncPhase int

// The phases of the pane.
const (
	SyncSignedOut SyncPhase = iota
	SyncSigningIn
	SyncSynced
	SyncUploading
	SyncFailed
	SyncOffline
	SyncBothHaveTasks
)

// SyncDevice is one device of the account, as the pane lists it.
type SyncDevice struct {
	ID       string
	Name     string
	LastSeen time.Time
	This     bool
}

// SyncFailure is the kind of a failed sync, so Settings can word it in the user's language.
type SyncFailure int

// Kinds of failure.
const (
	SyncFailureUnknown SyncFailure = iota
	SyncFailureNoAnswer
	SyncFailureAuth
)

// SyncConflict is what the pane asks before it changes this Mac's task ids: how much data each
// side has.
type SyncConflict struct {
	LocalTasks     int
	LocalProjects  int
	AccountDevices int
}

// SyncSnapshot is everything Settings > Sync draws in one frame.
type SyncSnapshot struct {
	Phase        SyncPhase
	Server       string
	Account      string
	AccountEmail string
	Linked       bool
	LastSynced   time.Time
	FailedSince  time.Time
	RetryAt      time.Time
	LastError    string
	Failure      SyncFailure
	Pending      int
	SeedDone     int
	SeedTotal    int
	Devices      []SyncDevice
	ThisDeviceID string
	Conflict     *SyncConflict
}

// The JSON files keyed by task id that a merge must rename, next to the database.
const (
	syncSeenFile    = "seen.json"
	syncDismissed   = "dismissed.json"
	syncRecentCache = "recent-changes.json"
)

// syncTask is the state Settings > Sync keeps about the background runner. Real owns the runner;
// this file adds the pane's view of it and the sign-in it starts.
type syncTask struct {
	loaded      bool
	cfg         syncclient.Config
	server      string
	account     syncclient.Account
	hasAccount  bool
	linked      bool
	status      syncclient.Status
	signingIn   bool
	seeding     bool
	seedTotal   int
	pending     int
	failedSince time.Time
	conflict    *SyncConflict
	deviceID    string
	devices     []SyncDevice
	devicesBusy bool
	tokens      syncclient.TokenSource
	client      *syncclient.Client
}

// SetSyncNotify tells the service where to ask for a redraw after a status change. The shell
// passes its Post, so the Settings window and the sidebar cloud follow.
func (r *Real) SetSyncNotify(fn func()) {
	r.mu.Lock()
	r.syncNotify = fn
	r.mu.Unlock()
}

// SyncSnapshot is what Settings > Sync draws.
func (r *Real) SyncSnapshot() SyncSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loadSyncLocked()
	s := SyncSnapshot{
		Server: r.sync.server, Account: r.sync.account.Name, AccountEmail: r.sync.account.Email,
		Linked: r.sync.linked, LastSynced: r.sync.status.LastSynced, RetryAt: r.sync.status.RetryAt,
		LastError: r.sync.status.LastError, Failure: syncFailureKind(r.sync.status.LastError),
		Pending:   r.sync.pending,
		SeedTotal: r.sync.seedTotal, SeedDone: r.sync.seedTotal - r.sync.pending,
		ThisDeviceID: r.syncDeviceIDLocked(), Devices: append([]SyncDevice(nil), r.sync.devices...),
	}
	if s.Pending < 0 {
		s.Pending = 0
	}
	if s.SeedDone < 0 {
		s.SeedDone = 0
	}
	switch {
	case r.sync.seeding && r.sync.seedTotal > 0:
		s.Phase = SyncUploading
	case r.sync.signingIn:
		s.Phase = SyncSigningIn
	case r.sync.conflict != nil:
		s.Phase = SyncBothHaveTasks
		s.Conflict = &SyncConflict{
			LocalTasks: r.sync.conflict.LocalTasks, LocalProjects: r.sync.conflict.LocalProjects,
			AccountDevices: r.sync.conflict.AccountDevices,
		}
	case !r.sync.linked:
		if r.sync.status.LastError != "" {
			s.Phase = SyncFailed
			s.FailedSince = r.sync.failedSince
		} else {
			s.Phase = SyncSignedOut
		}
	case r.sync.status.State == syncclient.StateIdle:
		s.Phase = SyncSynced
	case r.sync.status.State == syncclient.StateSeeding:
		s.Phase = SyncUploading
	case r.sync.status.State == syncclient.StateSyncing:
		if r.sync.seeding || (r.sync.seedTotal > 0 && r.sync.pending > 0 && r.sync.status.LastSynced.IsZero()) {
			s.Phase = SyncUploading
		} else {
			s.Phase = SyncSynced
		}
	case r.sync.status.State == syncclient.StateError:
		s.Phase = SyncFailed
		s.FailedSince = r.sync.failedSince
	case r.sync.status.State == syncclient.StateOffline:
		s.Phase = SyncOffline
	default:
		s.Phase = SyncSignedOut
	}
	if !s.FailedSince.IsZero() && s.Phase != SyncFailed {
		s.FailedSince = time.Time{}
	}
	return s
}

// syncFailureKind names a runner error in a way the pane can word. An error it does not know is
// shown as it came.
func syncFailureKind(message string) SyncFailure {
	e := strings.ToLower(message)
	switch {
	case e == "":
		return SyncFailureUnknown
	case strings.Contains(e, "401"), strings.Contains(e, "unauthorized"),
		strings.Contains(e, "signed out"), strings.Contains(e, "refresh token"):
		return SyncFailureAuth
	case strings.Contains(e, "500"), strings.Contains(e, "502"), strings.Contains(e, "503"),
		strings.Contains(e, "504"), strings.Contains(e, "no such host"),
		strings.Contains(e, "connection refused"), strings.Contains(e, "connection reset"),
		strings.Contains(e, "timeout"), strings.Contains(e, "timed out"),
		strings.Contains(e, "network is unreachable"), strings.Contains(e, "did not answer"):
		return SyncFailureNoAnswer
	}
	return SyncFailureUnknown
}

// loadSyncLocked reads sync.json and the linked database once. The caller holds r.mu.
func (r *Real) loadSyncLocked() {
	if r.sync.loaded {
		return
	}
	r.sync.loaded = true
	r.sync.server = DefaultSyncServer
	if r.syncPath != "" {
		if cfg, found, err := syncclient.LoadConfigAt(r.syncPath); err == nil && found {
			r.sync.cfg = cfg
			r.sync.server = cfg.Server
			if cfg.Account != nil {
				r.sync.account, r.sync.hasAccount = *cfg.Account, true
			}
		}
	}
	if r.syncPath == "" {
		return
	}
	ctx := context.Background()
	if id, found, err := r.core.SyncState(ctx, "device_id"); err == nil && found && id != "" {
		r.sync.linked, r.sync.deviceID = true, id
	}
	if raw, found, err := r.core.SyncState(ctx, "seed_total"); err == nil && found && raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			r.sync.seedTotal = n
		}
	}
	if n, err := r.core.SyncPending(ctx); err == nil {
		r.sync.pending = n
	}
	// A linked database with credentials is syncing, not signed out, before the runner's first
	// status arrives. Without credentials it is signed out and the sidebar says so.
	if r.sync.linked && r.sync.status.State == "" {
		if r.sync.cfg.Token != "" || (r.sync.cfg.Issuer != "" && r.sync.cfg.Account != nil) {
			r.sync.status.State = syncclient.StateSyncing
		}
	}
}

// syncDeviceIDLocked is this database's device id, empty when it never linked.
func (r *Real) syncDeviceIDLocked() string {
	if !r.sync.linked {
		return ""
	}
	if r.sync.deviceID == "" {
		id, found, err := r.core.SyncState(context.Background(), "device_id")
		if err != nil || !found {
			return ""
		}
		r.sync.deviceID = id
	}
	return r.sync.deviceID
}

// SyncSignIn signs in through the browser, links this database and starts the runner. When both
// sides have tasks it stops at the conflict the pane asks about.
func (r *Real) SyncSignIn() {
	if r.syncPath == "" || r.opts.Secrets == nil || r.opts.SyncOpen == nil {
		r.setSyncError(errors.New("sync is not set up on this Mac"))
		return
	}
	r.mu.Lock()
	if r.sync.signingIn {
		r.mu.Unlock()
		return
	}
	r.loadSyncLocked()
	r.sync.signingIn = true
	r.sync.conflict = nil
	r.sync.status.LastError = ""
	r.sync.failedSince = time.Time{}
	server := r.sync.server
	r.mu.Unlock()
	r.syncChanged()
	go r.signIn(server)
}

// signIn is SyncSignIn off the UI thread.
func (r *Real) signIn(server string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	public, err := syncclient.FetchConfig(ctx, server, nil)
	if err != nil {
		r.failSignIn(err)
		return
	}
	if public.Issuer == "" || public.ClientID == "" {
		r.failSignIn(errors.New("the server does not offer sign-in"))
		return
	}
	tokens := syncclient.NewOIDCSource(public.Issuer, public.ClientID, public.Scopes, r.opts.Secrets)
	if err := tokens.Login(ctx, r.opts.SyncOpen); err != nil {
		r.failSignIn(err)
		return
	}
	account, hasAccount := tokens.Account()
	cfg := syncclient.Config{
		Server: server, Issuer: public.Issuer, ClientID: public.ClientID, Scopes: public.Scopes,
	}
	if hasAccount {
		cfg.Account = &account
	}
	client, err := syncclient.New(r.core, syncclient.Options{BaseURL: server, TokenSource: tokens})
	if err != nil {
		r.failSignIn(err)
		return
	}
	if err := syncclient.SaveConfigAt(r.syncPath, cfg); err != nil {
		r.failSignIn(err)
		return
	}
	r.mu.Lock()
	r.sync.cfg, r.sync.server = cfg, server
	r.sync.account, r.sync.hasAccount = account, hasAccount
	r.sync.tokens, r.sync.client = tokens, client

	r.sync.seeding = false
	r.sync.seedTotal, r.sync.pending = 0, 0
	linked := r.sync.linked
	r.mu.Unlock()
	if linked {
		// The database kept its device when the user signed out: signing in again resumes it.
		r.mu.Lock()
		r.sync.signingIn = false
		r.sync.status = syncclient.Status{State: syncclient.StateSyncing}
		r.sync.failedSince = time.Time{}
		r.mu.Unlock()
		r.startSync(context.Background())
		r.syncChanged()
		return
	}

	stop := make(chan struct{})
	go r.watchSeed(ctx, client, stop)
	err = client.Link(ctx, r.syncDeviceName())
	close(stop)
	r.mu.Lock()
	r.sync.seeding = false
	r.mu.Unlock()
	if errors.Is(err, syncclient.ErrLocalData) {
		r.conflictAfterLink(ctx, client)
		return
	}
	if err != nil {
		r.failSignIn(err)
		return
	}
	r.mu.Lock()
	r.sync.linked = true
	r.sync.signingIn = false
	r.sync.conflict = nil
	r.sync.status = syncclient.Status{State: syncclient.StateSyncing}
	r.sync.failedSince = time.Time{}
	r.mu.Unlock()
	r.startSync(context.Background())
	r.syncChanged()
}

// watchSeed keeps the upload count up to date while Link pushes the first snapshot.
func (r *Real) watchSeed(ctx context.Context, client *syncclient.Client, stop <-chan struct{}) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			total, err := client.SeedTotal(ctx)
			if err != nil || total == 0 {
				continue
			}
			pending, err := client.Pending(ctx)
			if err != nil {
				continue
			}
			r.mu.Lock()
			changed := r.sync.seedTotal != total || r.sync.pending != pending
			r.sync.seedTotal, r.sync.pending = total, pending
			r.mu.Unlock()
			if changed {
				r.syncChanged()
			}
		}
	}
}

// conflictAfterLink asks the pane which side wins when both this Mac and the account have tasks.
func (r *Real) conflictAfterLink(ctx context.Context, client *syncclient.Client) {
	counts, err := r.core.SyncCounts(ctx)
	if err != nil {
		r.failSignIn(err)
		return
	}
	devices, _ := client.Devices(ctx)
	// The device this sign-in just registered has never synced; it is not one of the devices
	// whose data the account already has.
	seen := 0
	for _, d := range devices {
		if d.LastSeenAt != nil {
			seen++
		}
	}
	r.mu.Lock()
	r.sync.signingIn = false
	r.sync.seeding = false
	r.sync.conflict = &SyncConflict{
		LocalTasks: int(counts["tasks"]), LocalProjects: int(counts["projects"]), AccountDevices: seen,
	}
	r.sync.devices = r.syncDevicesLocked(devices)
	r.sync.status.LastError = ""
	r.mu.Unlock()
	r.syncChanged()
}

// SyncMerge merges this Mac's tasks into the account, then renames the ids in the files the app
// keeps by task id.
func (r *Real) SyncMerge() {
	client, ok := r.syncClientForConflict()
	if !ok {
		return
	}
	r.mu.Lock()
	r.sync.signingIn = true
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		ids, err := client.LinkMerge(ctx, r.syncDeviceName())
		if err != nil {
			r.failSignIn(err)
			return
		}
		if err := rewriteSyncTaskIDs(filepath.Dir(r.syncPath), ids); err != nil {
			// The merge is done; a file the app can rebuild is not worth undoing it for.
			r.setSyncError(err)
		}
		r.finishLink(client)
	}()
}

// SyncUseAccount backs this Mac up and replaces its data with the account's.
func (r *Real) SyncUseAccount() {
	client, ok := r.syncClientForConflict()
	if !ok {
		return
	}
	r.mu.Lock()
	r.sync.signingIn = true
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if _, err := client.LinkReplace(ctx, r.syncDeviceName()); err != nil {
			r.failSignIn(err)
			return
		}
		r.finishLink(client)
	}()
}

// finishLink records a successful LinkMerge or LinkReplace and starts the runner.
func (r *Real) finishLink(client *syncclient.Client) {
	r.mu.Lock()
	r.sync.linked = true
	r.sync.signingIn = false
	r.sync.conflict = nil
	r.sync.status = syncclient.Status{State: syncclient.StateSyncing}
	r.sync.failedSince = time.Time{}
	r.sync.client = client
	r.mu.Unlock()
	r.startSync(context.Background())
	r.syncChanged()
}

// SyncCancel abandons a sign-in that found tasks on both sides. The data stays as it was and the
// token is deleted again.
func (r *Real) SyncCancel() {
	r.mu.Lock()
	conflict := r.sync.conflict
	r.sync.conflict = nil
	r.sync.signingIn = false
	r.sync.status.LastError = ""
	r.sync.failedSince = time.Time{}
	tokens := r.sync.tokens
	cfg := r.sync.cfg
	cfg.Account = nil
	r.sync.cfg = cfg
	r.sync.account, r.sync.hasAccount = syncclient.Account{}, false
	r.sync.tokens, r.sync.client = nil, nil
	r.mu.Unlock()
	if conflict == nil {
		return
	}
	if source, ok := tokens.(*syncclient.OIDCSource); ok {
		_ = source.Logout()
	}
	if r.syncPath != "" && cfg.Server != "" {
		_ = syncclient.SaveConfigAt(r.syncPath, cfg)
	}
	r.syncChanged()
}

// SyncSignOut stops the runner and deletes the refresh token. The database stays linked, so the
// next sign-in resumes the same device.
func (r *Real) SyncSignOut() {
	r.mu.Lock()
	runner := r.runner
	r.runner = nil
	cfg := r.sync.cfg
	cfg.Account = nil
	r.sync.cfg = cfg
	r.sync.account, r.sync.hasAccount = syncclient.Account{}, false
	r.sync.signingIn, r.sync.conflict = false, nil
	r.sync.status = syncclient.Status{}
	r.sync.failedSince = time.Time{}
	r.sync.tokens, r.sync.client = nil, nil
	r.sync.devices = nil
	r.mu.Unlock()
	if runner != nil {
		_ = runner.Close()
		r.mu.Lock()
		r.sync.status = syncclient.Status{}
		r.mu.Unlock()
	}
	if cfg.Issuer != "" && r.opts.Secrets != nil {
		_ = syncclient.NewOIDCSource(cfg.Issuer, cfg.ClientID, cfg.Scopes, r.opts.Secrets).Logout()
	}
	if r.syncPath != "" && cfg.Server != "" {
		_ = syncclient.SaveConfigAt(r.syncPath, cfg)
	}
	r.syncChanged()
}

// SyncNow syncs at once, without waiting for the debounce.
func (r *Real) SyncNow() {
	r.mu.Lock()
	runner := r.runner
	linked := r.sync.linked
	r.mu.Unlock()
	if runner == nil && linked {
		r.startSync(context.Background())
		r.mu.Lock()
		runner = r.runner
		r.mu.Unlock()
	}
	if runner != nil {
		runner.SyncNow()
	}
}

// SyncRetry is the button of a failed row: sign in again when the database is not linked, else
// sync now.
func (r *Real) SyncRetry() {
	r.mu.Lock()
	linked := r.sync.linked
	r.mu.Unlock()
	if !linked {
		r.SyncSignIn()
		return
	}
	r.SyncNow()
}

// SyncSetServer saves a self-hosted server. It is the URL the pane's Change edits.
func (r *Real) SyncSetServer(raw string) error {
	server := strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(server)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("the server address must be an http or https URL")
	}
	r.mu.Lock()
	r.loadSyncLocked()
	cfg := r.sync.cfg
	if cfg.Server == "" {
		cfg = syncclient.Config{Server: server}
	}
	cfg.Server = server
	if r.syncPath != "" {
		if err := syncclient.SaveConfigAt(r.syncPath, cfg); err != nil {
			r.mu.Unlock()
			return err
		}
	}
	r.sync.cfg, r.sync.server = cfg, server
	r.mu.Unlock()
	r.syncChanged()
	return nil
}

// SyncRefreshDevices asks the server who else is signed in. It is called when the pane opens and
// after each sync while it is open.
func (r *Real) SyncRefreshDevices() {
	r.mu.Lock()
	if !r.sync.linked || !r.sync.hasAccount || r.sync.devicesBusy {
		r.mu.Unlock()
		return
	}
	r.loadSyncLocked()
	r.sync.devicesBusy = true
	client := r.sync.client
	cfg := r.sync.cfg
	r.mu.Unlock()
	go func() {
		defer func() {
			r.mu.Lock()
			r.sync.devicesBusy = false
			r.mu.Unlock()
		}()
		if client == nil {
			var err error
			client, err = r.syncClientFor(cfg)
			if err != nil {
				return
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		devices, err := client.Devices(ctx)
		if err != nil {
			return
		}
		r.mu.Lock()
		r.sync.devices = r.syncDevicesLocked(devices)
		r.mu.Unlock()
		r.syncChanged()
	}()
}

// syncDevicesLocked marks this Mac and puts it first, as the pane lists it. The caller holds r.mu.
func (r *Real) syncDevicesLocked(devices []syncproto.Device) []SyncDevice {
	this := r.syncDeviceIDLocked()
	out := make([]SyncDevice, 0, len(devices))
	for _, d := range devices {
		item := SyncDevice{ID: d.ID, Name: d.Name, This: d.ID == this}
		if d.LastSeenAt != nil {
			item.LastSeen = *d.LastSeenAt
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].This && !out[j].This })
	return out
}

// syncClientForConflict is the client of the sign-in that found tasks on both sides.
func (r *Real) syncClientForConflict() (*syncclient.Client, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sync.conflict == nil {
		return nil, false
	}
	if r.sync.client != nil {
		return r.sync.client, true
	}
	client, err := r.syncClientFor(r.sync.cfg)
	if err != nil {
		return nil, false
	}
	r.sync.client = client
	return client, true
}

// syncClientFor builds a client from a saved config, with a fresh OIDC source when needed.
func (r *Real) syncClientFor(cfg syncclient.Config) (*syncclient.Client, error) {
	var tokens syncclient.TokenSource
	switch {
	case cfg.Issuer != "":
		if r.opts.Secrets == nil {
			return nil, errors.New("sync has no secrets store")
		}
		tokens = syncclient.NewOIDCSource(cfg.Issuer, cfg.ClientID, cfg.Scopes, r.opts.Secrets)
	case cfg.Token != "":
		tokens = syncclient.StaticToken(cfg.Token)
	default:
		return nil, errors.New("sync has no way to sign in")
	}
	return syncclient.New(r.core, syncclient.Options{BaseURL: cfg.Server, TokenSource: tokens})
}

// failSignIn clears the busy state and says what went wrong.
func (r *Real) failSignIn(err error) {
	r.mu.Lock()
	r.sync.signingIn = false
	r.sync.seeding = false
	r.sync.status.LastError = err.Error()
	r.sync.failedSince = r.Now()
	r.mu.Unlock()
	r.syncChanged()
}

// setSyncError reports an error without stopping the runner.
func (r *Real) setSyncError(err error) {
	r.mu.Lock()
	r.sync.status.LastError = err.Error()
	r.mu.Unlock()
	r.syncChanged()
}

// syncStatusChanged is the runner's OnStatus: remember it and redraw.
func (r *Real) syncStatusChanged(st syncclient.Status) {
	r.mu.Lock()
	r.sync.status = st
	r.sync.pending = st.Pending
	switch st.State {
	case syncclient.StateError:
		if r.sync.failedSince.IsZero() {
			r.sync.failedSince = r.Now()
		}
	case syncclient.StateOffline:
		// Offline is not a failure that puts a red cloud in the sidebar.
		r.sync.failedSince = time.Time{}
	default:
		r.sync.failedSince = time.Time{}
	}
	r.mu.Unlock()
	r.syncChanged()
}

// syncChanged asks the shell for a redraw when it asked for one.
func (r *Real) syncChanged() {
	r.mu.Lock()
	fn := r.syncNotify
	r.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// syncDeviceName is this Mac's name on the server.
func (r *Real) syncDeviceName() string {
	if name, err := os.Hostname(); err == nil && strings.TrimSpace(name) != "" {
		return name
	}
	return "This Mac"
}

// rewriteSyncTaskIDs renames task ids in the three files the app keys by task id. The files are
// beside the database; a missing one is skipped. recent-changes.json is keyed by URL today, but
// the walk also renames any numeric key or task id it finds there.
func rewriteSyncTaskIDs(dir string, ids map[int64]int64) error {
	if len(ids) == 0 {
		return nil
	}
	mapping := make(map[string]string, len(ids))
	for old, id := range ids {
		mapping[strconv.FormatInt(old, 10)] = strconv.FormatInt(id, 10)
	}
	var errs []error
	for _, name := range []string{syncSeenFile, syncDismissed, syncRecentCache} {
		if err := rewriteTaskIDsInFile(filepath.Join(dir, name), mapping); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// rewriteTaskIDsInFile rewrites one JSON file through a temporary one, so a crash leaves the old
// file whole.
func rewriteTaskIDsInFile(path string, mapping map[string]string) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var value any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	value = rewriteTaskIDs(value, "", mapping)
	out, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// rewriteTaskIDs walks a decoded JSON value: numeric object keys are task ids in the two seen
// files, and a value under a task-ish key is a task id elsewhere.
func rewriteTaskIDs(value any, key string, mapping map[string]string) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, child := range v {
			next := k
			if id, ok := mapping[k]; ok {
				if _, err := strconv.Atoi(k); err == nil {
					next = id
				}
			}
			out[next] = rewriteTaskIDs(child, k, mapping)
		}
		return out
	case []any:
		for i, child := range v {
			v[i] = rewriteTaskIDs(child, key, mapping)
		}
		return v
	case json.Number:
		if taskKey(key) {
			if id, ok := mapping[v.String()]; ok {
				return json.Number(id)
			}
		}
		return v
	case string:
		if taskKey(key) {
			if id, ok := mapping[v]; ok {
				return id
			}
		}
		return v
	case float64:
		if taskKey(key) {
			raw := strconv.FormatInt(int64(v), 10)
			if id, ok := mapping[raw]; ok {
				if n, err := strconv.ParseFloat(id, 64); err == nil {
					return n
				}
			}
		}
		return v
	default:
		return value
	}
}

// taskKey says whether a JSON key names a task id, such as "task_id" or "taskId".
func taskKey(key string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "task") && (strings.HasSuffix(k, "id") || k == "task")
}
