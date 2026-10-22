package data

import (
	"context"
	"errors"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
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

// SignInStage is how far the in-app sign-in has got, so the button can say what is happening.
type SignInStage string

// The stages of a sign-in.
const (
	SignInStageNone        SignInStage = ""
	SignInStageChecking    SignInStage = "checking"
	SignInStageDownloading SignInStage = "downloading"
	// SignInStageSending is the email being sent its code.
	SignInStageSending SignInStage = "sending"
	// SignInStageGoogle is the system browser open on Google's consent page, the app waiting.
	SignInStageGoogle SignInStage = "google"
)

// SignInStep is which screen of the signed-out card the Account pane shows (M13 P2).
type SignInStep string

// The steps. SignInStepEmail is the default; the code step follows SyncRequestCode, the password
// step is the user's choice, and the reset step follows SyncRequestPasswordReset.
const (
	SignInStepEmail     SignInStep = ""
	SignInStepCode      SignInStep = "code"
	SignInStepPassword  SignInStep = "password"
	SignInStepResetSent SignInStep = "resetSent"
)

// SignInFailure is why the last in-app sign-in failed, so Settings can word it in the user's
// language without matching the error's text.
type SignInFailure string

// The answers POST /v1/sign-in gives, and the local failures around it.
const (
	SignInFailureNone        SignInFailure = ""
	SignInFailureCredentials SignInFailure = "credentials"
	SignInFailureTooMany     SignInFailure = "tooMany"
	SignInFailureLocked      SignInFailure = "locked"
	SignInFailureUnsupported SignInFailure = "unsupported"
	SignInFailureNetwork     SignInFailure = "network"
	SignInFailureOther       SignInFailure = "other"
	// SignInFailureWrongCode is a code that was not right.
	SignInFailureWrongCode SignInFailure = "wrongCode"
	// SignInFailureCodeExpired is a code older than ten minutes: a new one was mailed at once.
	SignInFailureCodeExpired SignInFailure = "codeExpired"
	// SignInFailureGoogleCancelled is a Google sign-in the person cancelled.
	SignInFailureGoogleCancelled SignInFailure = "googleCancelled"
	// SignInFailureGoogleFailed is a Google sign-in that timed out or was refused.
	SignInFailureGoogleFailed SignInFailure = "googleFailed"
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
	Phase         SyncPhase
	Server        string
	Account       string
	AccountEmail  string
	Linked        bool
	LastSynced    time.Time
	FailedSince   time.Time
	RetryAt       time.Time
	LastError     string
	Failure       SyncFailure
	SignInFailure SignInFailure
	SignInStage   SignInStage
	Pending       int
	SeedDone      int
	SeedTotal     int
	Devices       []SyncDevice
	ThisDeviceID  string
	Conflict      *SyncConflict
	// SignInStep, SignInEmail and ResendAt are the signed-out card's code flow: the screen, the
	// address being signed in, and when Resend code is allowed again (zero outside the code step).
	SignInStep  SignInStep
	SignInEmail string
	ResendAt    time.Time
	// GoogleAvailable is true once the server's auth methods listed a Google provider.
	GoogleAvailable bool
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
	loaded        bool
	cfg           syncclient.Config
	server        string
	account       syncclient.Account
	hasAccount    bool
	linked        bool
	status        syncclient.Status
	signingIn     bool
	signInFailure SignInFailure
	signInStage   SignInStage
	seeding       bool
	seedTotal     int
	pending       int
	failedSince   time.Time
	conflict      *SyncConflict
	deviceID      string
	devices       []SyncDevice
	devicesBusy   bool
	tokens        syncclient.TokenSource
	client        *syncclient.Client
	code          codeState
	// google is the running Google sign-in's cancel func; gen tells a finished one from a newer.
	google    context.CancelFunc
	googleGen int
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
		SignInFailure: r.sync.signInFailure,
		SignInStage:   r.sync.signInStage,
		Pending:       r.sync.pending,
		SeedTotal:     r.sync.seedTotal, SeedDone: r.sync.seedTotal - r.sync.pending,
		SignInStep: r.sync.code.step, SignInEmail: r.sync.code.email, ResendAt: r.sync.code.resendAt,
		GoogleAvailable: r.sync.code.google && r.sync.code.probed == r.sync.server,
		ThisDeviceID:    r.syncDeviceIDLocked(), Devices: append([]SyncDevice(nil), r.sync.devices...),
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
		// An unlinked Mac is always signed out: a failed sign-in belongs in the card's error line,
		// with the fields still there to try again.
		s.Phase = SyncSignedOut
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

// signInFailureKind maps a failed sign-in to the code Settings words in the user's language. The
// five answers of POST /v1/sign-in are sentinels; a network failure is local.
func signInFailureKind(err error) SignInFailure {
	switch {
	case err == nil:
		return SignInFailureNone
	case errors.Is(err, syncclient.ErrInvalidCredentials):
		return SignInFailureCredentials
	case errors.Is(err, syncclient.ErrInvalidCode):
		return SignInFailureWrongCode
	case errors.Is(err, syncclient.ErrGoogleCancelled):
		return SignInFailureGoogleCancelled
	case errors.Is(err, syncclient.ErrGoogleFailed):
		return SignInFailureGoogleFailed
	case errors.Is(err, syncclient.ErrTooManyAttempts):
		return SignInFailureTooMany
	case errors.Is(err, syncclient.ErrAccountLocked):
		return SignInFailureLocked
	case errors.Is(err, syncclient.ErrSignInUnsupported):
		return SignInFailureUnsupported
	case signInNetworkError(err):
		return SignInFailureNetwork
	}
	return SignInFailureOther
}

// signInNetworkError says whether err is the network refusing the request rather than the server
// answering.
func signInNetworkError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	for _, target := range []error{
		syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED, syscall.EPIPE,
		syscall.ETIMEDOUT, syscall.ENETUNREACH, syscall.EHOSTUNREACH,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
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
		if r.sync.cfg.Token != "" || (r.sync.cfg.SignsInWithAccount() && r.sync.cfg.Account != nil) {
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
	r.sync.code = r.sync.code.cleared()
	r.sync.signInFailure = SignInFailureNone
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
	if cfg.SignsInWithAccount() && r.opts.Secrets != nil {
		_ = r.accountSource(cfg).Logout()
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

// SyncRetry is the button of a failed row. A failed sign-in is not a row: the signed-out card
// keeps the fields and its own error line, so Retry only syncs a linked Mac again.
func (r *Real) SyncRetry() {
	r.SyncNow()
}

// SyncSetServer saves a self-hosted server. It is the URL the pane's Change edits.
func (r *Real) SyncSetServer(raw string) error {
	server, err := syncclient.ServerURL(raw)
	if err != nil {
		return err
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

// syncClientFor builds a client from a saved config, with a fresh account source when needed.
func (r *Real) syncClientFor(cfg syncclient.Config) (*syncclient.Client, error) {
	var tokens syncclient.TokenSource
	switch {
	case cfg.SignsInWithAccount():
		if r.opts.Secrets == nil {
			return nil, errors.New("sync has no secrets store")
		}
		tokens = r.accountSource(cfg)
	case cfg.Token != "":
		tokens = syncclient.StaticToken(cfg.Token)
	default:
		return nil, errors.New("sync has no way to sign in")
	}
	return syncclient.New(r.core, syncclient.Options{BaseURL: cfg.Server, TokenSource: tokens})
}

// setSignInStage moves the sign-in button's text and redraws.
func (r *Real) setSignInStage(stage SignInStage) {
	r.mu.Lock()
	r.sync.signInStage = stage
	r.mu.Unlock()
	r.syncChanged()
}

// failSignIn clears the busy state and says what went wrong. The error text goes to LastError for
// the logs; the pane words the classification instead.
func (r *Real) failSignIn(err error) {
	r.mu.Lock()
	r.sync.signingIn = false
	r.sync.signInStage = SignInStageNone
	r.sync.seeding = false
	r.sync.status.LastError = err.Error()
	r.sync.signInFailure = signInFailureKind(err)
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
