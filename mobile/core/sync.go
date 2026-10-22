package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// secretAdapter is the Kotlin Secrets as syncclient.Secrets. The phone reports a missing key as
// an error whose text is "not found"; the sync client wants syncclient.ErrSecretNotFound.
type secretAdapter struct{ phone Secrets }

func (a secretAdapter) Get(key string) (string, error) {
	value, err := a.phone.Get(key)
	if err != nil {
		if err.Error() == "not found" {
			return "", syncclient.ErrSecretNotFound
		}
		return "", errors.New("could not read the stored sign-in")
	}
	return value, nil
}

func (a secretAdapter) Set(key, value string) error {
	if err := a.phone.Set(key, value); err != nil {
		return errors.New("could not store the sign-in")
	}
	return nil
}

func (a secretAdapter) Delete(key string) error {
	if err := a.phone.Delete(key); err != nil {
		if err.Error() == "not found" {
			return syncclient.ErrSecretNotFound
		}
		return errors.New("could not delete the stored sign-in")
	}
	return nil
}

// SyncConfigure writes dataDir/sync.json: the server, and either the OIDC issuer and client id or
// (in development) a token already in the file. scopes is space separated and may be empty. It
// stops a running background sync so the next StartSync uses the new settings.
func (c *Core) SyncConfigure(server, issuer, clientID, scopes string) error {
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	if server == "" {
		return errors.New("a sync server is required")
	}
	_ = c.StopSync()
	cfg := syncclient.Config{
		Server: server, Issuer: strings.TrimRight(strings.TrimSpace(issuer), "/"),
		ClientID: strings.TrimSpace(clientID), Scopes: strings.Fields(scopes),
	}
	if err := syncclient.SaveConfigAt(c.syncPath, cfg); err != nil {
		return errors.New("could not save the sync settings")
	}
	return nil
}

// SyncSetServer saves a self-hosted server in dataDir/sync.json. It asks the server's public
// GET /v1/config first, so the file gets the sign-in it names (issuer, client id and scopes)
// rather than a guessed one; an address that is not https, a server that does not answer and a
// server without sign-in are all left unwritten. The address is the one the Sync screen's
// "Server … · Change" edits.
func (c *Core) SyncSetServer(server string) error {
	base, err := syncclient.ServerURL(server)
	if err != nil {
		return errors.New("use an https address, such as https://todo.example.com")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	public, err := syncclient.FetchConfig(ctx, base, nil)
	if err != nil {
		return fmt.Errorf("could not reach the sync server: %w", err)
	}
	if public.Auth != syncclient.AuthPocketBase && (public.Issuer == "" || public.ClientID == "") {
		return fmt.Errorf("the server does not offer sign-in: %w", syncclient.ErrSignInUnsupported)
	}
	_ = c.StopSync()
	cfg, found, err := c.loadConfig()
	if err != nil {
		return err
	}
	if !found {
		cfg = syncclient.Config{Server: base}
	}
	cfg.Server = base
	if err := cfg.ApplyPublic(public); err != nil {
		return err
	}
	if err := syncclient.SaveConfigAt(c.syncPath, cfg); err != nil {
		return errors.New("could not save the sync settings")
	}
	return nil
}

// SignIn signs in with an account and password through the sync server's /v1/sign-in, then links
// this database as a device named deviceName when it is not linked yet. It returns
// {"state":"linked"} once the phone is linked, or {"state":"choose"} when both the phone and the
// account have tasks; the tokens stay stored, so LinkMerge or LinkReplace finishes without a
// second sign-in. Signing in an already linked phone returns {"state":"linked"}. With no issuer in
// sync.json it first reads the server's /v1/config through SyncSetServer.
func (c *Core) SignIn(loginName, password, deviceName string) (string, error) {
	if c.secrets == nil {
		return "", errors.New("this phone has no secure store for the sign-in")
	}
	c.noteCodeFailure(nil)
	c.startLink()
	defer c.finishLink()
	c.setSignInStage(signInStageChecking)
	defer c.setSignInStage(signInStageNone)
	cfg, err := c.signInConfig()
	if err != nil {
		return "", err
	}
	source := c.accountSource(cfg)
	if err := source.SignIn(context.Background(), cfg.Server, loginName, password); err != nil {
		c.noteCodeFailure(err)
		return "", syncError(err)
	}
	return c.finishSignIn(cfg, source, deviceName)
}

// signInConfig is the saved sync settings, with the server's sign-in read first when the phone
// never changed its server and so has none yet. SyncSetServer writes both, so the next sign-in is
// one call.
func (c *Core) signInConfig() (syncclient.Config, error) {
	cfg, found, err := c.loadConfig()
	if err != nil {
		return cfg, err
	}
	if !found || !cfg.SignsInWithAccount() {
		server := data.DefaultSyncServer
		if found && cfg.Server != "" {
			server = cfg.Server
		}
		if err := c.SyncSetServer(server); err != nil {
			return cfg, syncError(err)
		}
		cfg, found, err = c.loadConfig()
		if err != nil {
			return cfg, err
		}
		if !found || !cfg.SignsInWithAccount() {
			return cfg, errors.New("sign-in is not set up; set the server first")
		}
	}
	return cfg, nil
}

// finishSignIn is everything after a token is stored, shared by the password and the code
// sign-in: save the account, then link the phone (or find it linked), answering "linked" or
// "choose".
func (c *Core) finishSignIn(cfg syncclient.Config, source syncclient.AccountSource, deviceName string) (string, error) {
	c.setSignInStage(signInStageDownloading)
	if account, ok := source.Account(); ok {
		cfg.Account = &account
		if err := syncclient.SaveConfigAt(c.syncPath, cfg); err != nil {
			return "", errors.New("could not save the account")
		}
	}
	c.clearCodeState()
	device, linked, err := c.dom.SyncState(context.Background(), "device_id")
	if err != nil {
		return "", errors.New("could not read the link")
	}
	if linked && device != "" {
		return signInLinkedJSON, nil
	}
	client, err := c.newClient(false)
	if err != nil {
		return "", err
	}
	if err := client.Link(context.Background(), strings.TrimSpace(deviceName)); err != nil {
		if errors.Is(err, syncclient.ErrLocalData) {
			return c.signInChoose(context.Background(), client)
		}
		return "", syncError(err)
	}
	return signInLinkedJSON, nil
}

// SignOut forgets the refresh token or PocketBase token, so the next sync is signed out until SignIn runs again. The
// database stays linked to its device.
func (c *Core) SignOut() error {
	_ = c.StopSync()
	c.mu.Lock()
	c.lastStatus = syncclient.Status{State: syncclient.StateSignedOut}
	c.failedSince = time.Time{}
	c.mu.Unlock()
	if c.secrets == nil {
		return nil
	}
	if err := syncclient.DeleteRefreshTokenAt(secretAdapter{c.secrets}, c.dbPath); err != nil {
		return errors.New("could not sign out")
	}
	if err := syncclient.DeletePocketBaseToken(secretAdapter{c.secrets}, c.dbPath); err != nil {
		return errors.New("could not sign out")
	}
	return nil
}

// SyncNow syncs at once. With the background runner working it nudges the runner; without one it
// pushes and pulls once, then refills the id blocks (StartSync, the background loop, must not be
// running at the same time in that case).
func (c *Core) SyncNow() error {
	c.mu.Lock()
	runner := c.runner
	c.mu.Unlock()
	if runner != nil {
		runner.SyncNow()
		return nil
	}
	client, err := c.client()
	if err != nil {
		return err
	}
	if err := client.Sync(context.Background()); err != nil {
		return syncError(err)
	}
	return nil
}

// StartSync starts the background Runner: a nudge on every change, the server's poke, and a five
// minute timer. Starting an already running sync is a no-op.
func (c *Core) StartSync() error {
	c.mu.Lock()
	if c.links > 0 {
		// SignIn, LinkMerge or LinkReplace is still talking to the server. It starts the runner
		// itself when it is done; a runner now would lease ids and refresh the token underneath it.
		c.mu.Unlock()
		return nil
	}
	existing := c.runner
	c.mu.Unlock()
	if existing != nil {
		// Runner.Start is a no-op while it runs, and starts it again after a 401 stopped it.
		return syncError(existing.Start(context.Background()))
	}
	client, err := c.client()
	if err != nil {
		return err
	}
	runner := syncclient.NewRunner(client, syncclient.RunnerOptions{OnStatus: c.runnerStatus})
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		_ = runner.Close()
		return errors.New("the core is closed")
	}
	if c.runner != nil {
		c.mu.Unlock()
		_ = runner.Close()
		return nil
	}
	c.runner = runner
	c.mu.Unlock()
	if err := runner.Start(context.Background()); err != nil {
		c.mu.Lock()
		if c.runner == runner {
			c.runner = nil
		}
		c.mu.Unlock()
		_ = runner.Close()
		return syncError(err)
	}
	return nil
}

// StopSync stops the background Runner and waits for it. StartSync can start a new one.
func (c *Core) StopSync() error {
	c.mu.Lock()
	runner := c.runner
	c.runner = nil
	c.mu.Unlock()
	if runner == nil {
		return nil
	}
	return runner.Close()
}

// SyncStatus returns the Runner's status as JSON, the sync settings the screens need, and the
// first upload's size:
//
//	{"state":"idle","pending":0,"server":"https://todo.sri-ai.xyz","linked":true,"signed_in":true}
//
// state is idle, syncing, offline, error, seeding or signed out; last_synced, last_error,
// retry_at, failed_since and seed_total are left out when empty. linked says this database has a
// device id, signed_in says it still has a token, and server is the address in sync.json (the
// default before one is written). Without a running Runner it is the last status the Runner
// reported, with the outbox count read now.
func (c *Core) SyncStatus() string {
	c.mu.Lock()
	status := c.lastStatus
	c.mu.Unlock()
	if pending, err := c.dom.SyncPending(context.Background()); err == nil {
		status.Pending = pending
	}
	raw, err := c.encodeStatus(status)
	if err != nil {
		return `{"state":"error","pending":0}`
	}
	return raw
}

// The stages of a sign-in, as SyncStatus reports them.
const (
	signInStageSending     = "sending"
	signInStageChecking    = "checking"
	signInStageDownloading = "downloading"
	signInStageGoogle      = "google"
	signInStageNone        = ""
)

// statusJSON is SyncStatus's shape.
type statusJSON struct {
	State       string `json:"state"`
	LastSynced  string `json:"last_synced,omitempty"`
	LastError   string `json:"last_error,omitempty"`
	Pending     int64  `json:"pending"`
	RetryAt     string `json:"retry_at,omitempty"`
	FailedSince string `json:"failed_since,omitempty"`
	SeedTotal   int64  `json:"seed_total,omitempty"`
	Server      string `json:"server,omitempty"`
	Linked      bool   `json:"linked"`
	SignedIn    bool   `json:"signed_in"`
	SignInStage string `json:"sign_in_stage,omitempty"`
}

// encodeStatus encodes one Runner status with the settings the Sync screen reads.
func (c *Core) encodeStatus(status syncclient.Status) (string, error) {
	out := statusJSON{
		State: string(status.State), LastError: status.LastError, Pending: int64(status.Pending),
		Server: data.DefaultSyncServer,
	}
	if !status.LastSynced.IsZero() {
		out.LastSynced = status.LastSynced.UTC().Format(time.RFC3339)
	}
	if !status.RetryAt.IsZero() {
		out.RetryAt = status.RetryAt.UTC().Format(time.RFC3339)
	}
	c.mu.Lock()
	failedSince, signInStage := c.failedSince, c.signInStage
	c.mu.Unlock()
	if !failedSince.IsZero() {
		out.FailedSince = failedSince.UTC().Format(time.RFC3339)
	}
	out.SignInStage = signInStage
	ctx := context.Background()
	if device, linked, err := c.dom.SyncState(ctx, "device_id"); err == nil && linked && device != "" {
		out.Linked = true
	}
	if raw, found, err := c.dom.SyncState(ctx, "seed_total"); err == nil && found && raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			out.SeedTotal = int64(n)
		}
	}
	if cfg, found, err := c.loadConfig(); err == nil && found {
		out.Server = cfg.Server
		out.SignedIn = c.hasCredentials(cfg)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// hasCredentials says whether a config can still authenticate: a development token in the file,
// or a refresh token in the phone's secure store.
func (c *Core) hasCredentials(cfg syncclient.Config) bool {
	if cfg.Token != "" {
		return true
	}
	if !cfg.SignsInWithAccount() || c.secrets == nil {
		return false
	}
	if cfg.Auth == syncclient.AuthPocketBase {
		return syncclient.HasPocketBaseToken(secretAdapter{c.secrets}, c.dbPath)
	}
	_, err := syncclient.LoadRefreshToken(secretAdapter{c.secrets}, c.dbPath)
	return err == nil
}

// runnerStatus stores the Runner's newest status, remembers when a failure started, and tells the
// listener.
func (c *Core) runnerStatus(status syncclient.Status) {
	c.mu.Lock()
	c.lastStatus = status
	switch status.State {
	case syncclient.StateError:
		if c.failedSince.IsZero() {
			c.failedSince = c.nowTime()
		}
	case syncclient.StateOffline, syncclient.StateIdle, syncclient.StateSyncing:
		// A success or a retry in flight is not a failure that turns the header's cloud red.
		c.failedSince = time.Time{}
	}
	l := c.listener
	c.mu.Unlock()
	if l == nil {
		return
	}
	if raw, err := c.encodeStatus(status); err == nil {
		l.OnStatus(raw)
	}
}

// loadConfig reads dataDir/sync.json.
func (c *Core) loadConfig() (syncclient.Config, bool, error) {
	cfg, found, err := syncclient.LoadConfigAt(c.syncPath)
	if err != nil {
		return syncclient.Config{}, false, errors.New("could not read the sync settings")
	}
	return cfg, found, nil
}

// client builds the sync client from sync.json for a linked database.
func (c *Core) client() (*syncclient.Client, error) { return c.newClient(true) }

// newClient builds the sync client; requireLinked is false for SignIn, which links the database
// with the client it is about to make.
func (c *Core) newClient(requireLinked bool) (*syncclient.Client, error) {
	cfg, found, err := c.loadConfig()
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("sync is not set up")
	}
	if requireLinked {
		device, linked, err := c.dom.SyncState(context.Background(), "device_id")
		if err != nil {
			return nil, errors.New("could not read the link")
		}
		if !linked || device == "" {
			return nil, errors.New("this phone is not linked")
		}
	}
	var tokens syncclient.TokenSource
	switch {
	case cfg.SignsInWithAccount():
		if c.secrets == nil {
			return nil, errors.New("this phone has no secure store for the sign-in")
		}
		tokens = c.accountSource(cfg)
	case cfg.Token != "":
		tokens = syncclient.StaticToken(cfg.Token)
	default:
		return nil, errors.New("sign in to sync")
	}
	client, err := syncclient.New(c.dom, syncclient.Options{BaseURL: cfg.Server, TokenSource: tokens})
	if err != nil {
		return nil, errors.New("the sync settings are not valid")
	}
	return client, nil
}

// syncError turns an internal sync failure into a sentence fit to show the user. The raw error is
// logged first, for logcat through gomobile's GoLog; errors carry the request URL, which has no
// token, and request bodies are never logged.
func syncError(err error) error {
	if err == nil {
		return nil
	}
	log.Printf("adhd-todo sync: %v", err)
	switch {
	case errors.Is(err, syncclient.ErrInvalidCredentials):
		return errors.New("wrong account or password")
	case errors.Is(err, syncclient.ErrTooManyAttempts):
		return errors.New("too many tries; wait a few minutes")
	case errors.Is(err, syncclient.ErrSecondFactor):
		return errors.New("this account uses two-step verification, which in-app sign-in does not support yet")
	case errors.Is(err, syncclient.ErrAccountLocked):
		return errors.New("this account is deactivated; ask the admin")
	case errors.Is(err, syncclient.ErrSignInUnsupported):
		return errors.New("this server does not support signing in from the app")
	case errors.Is(err, syncclient.ErrGoogleCancelled):
		return errors.New("Google sign-in was cancelled")
	case errors.Is(err, syncclient.ErrGoogleFailed):
		return errors.New("Google sign-in didn't finish; try again")
	case errors.Is(err, syncclient.ErrSeeding):
		return errors.New("another device is still uploading this workspace; try again later")
	case errors.Is(err, syncclient.ErrLocalData):
		return errors.New("this phone already has tasks of its own")
	case errors.Is(err, syncclient.ErrNotLinked):
		return errors.New("this phone is not linked")
	case errors.Is(err, syncclient.ErrSignedOut), syncclient.IsUnauthorized(err):
		return errors.New("sign in again")
	case errors.Is(err, context.Canceled):
		return errors.New("the sync was cancelled")
	}
	if isNetworkError(err) {
		return errors.New("could not reach the server; check the connection")
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "sign-in timed out"):
		return errors.New("sign-in timed out")
	case strings.Contains(message, "sign-in was refused"):
		return errors.New("sign-in was refused")
	case strings.Contains(message, "discover OIDC issuer"), strings.Contains(message, "sign-in listener"):
		return errors.New("could not reach the sign-in server")
	}
	return errors.New("could not sync; check the connection and try again")
}

// isNetworkError says whether err is the network, rather than the server, refusing the request: a
// name that did not resolve, a dial or another net.Error (a timeout included), or a syscall error
// such as a refused, reset or aborted connection.
func isNetworkError(err error) bool {
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

// setSignInStage moves the sign-in button's text and tells the screen to re-read the status.
func (c *Core) setSignInStage(stage string) {
	c.mu.Lock()
	c.signInStage = stage
	c.mu.Unlock()
	c.notifyChange()
}

// startLink marks a SignIn, LinkMerge or LinkReplace in progress. StartSync does nothing while one
// runs; the link starts the runner itself once it is done.
func (c *Core) startLink() {
	c.mu.Lock()
	c.links++
	c.mu.Unlock()
}

// finishLink clears one startLink.
func (c *Core) finishLink() {
	c.mu.Lock()
	if c.links > 0 {
		c.links--
	}
	c.mu.Unlock()
}

// accountSource is the token source for cfg (PocketBase or OIDC), keyed to this database so another
// profile cannot replace this phone's sign-in.
// The same settings get the same source, so the runner and a Sync now share one refresh.
func (c *Core) accountSource(cfg syncclient.Config) syncclient.AccountSource {
	key := cfg.SourceKey()
	c.tokensMu.Lock()
	defer c.tokensMu.Unlock()
	if c.tokens != nil && c.tokensKey == key {
		return c.tokens
	}
	source := syncclient.NewAccountSource(cfg, secretAdapter{c.secrets}, c.dbPath)
	c.tokens, c.tokensKey = source, key
	return source
}
