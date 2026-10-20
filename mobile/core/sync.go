package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

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

// SignIn signs in with OIDC through the system browser (PKCE on a loopback redirect), then links
// this database as a device named deviceName when it is not linked yet. BrowserOpener.Open gets
// the authorization URL. It returns {"state":"linked"} once the phone is linked, or
// {"state":"choose"} when both the phone and the account have tasks; the tokens stay stored, so
// LinkMerge or LinkReplace finishes without a second sign-in. Signing in an already linked phone
// returns {"state":"linked"}.
func (c *Core) SignIn(opener BrowserOpener, deviceName string) (string, error) {
	if opener == nil {
		return "", errors.New("there is no browser to sign in with")
	}
	if c.secrets == nil {
		return "", errors.New("this phone has no secure store for the sign-in")
	}
	cfg, found, err := c.loadConfig()
	if err != nil {
		return "", err
	}
	if !found || cfg.Issuer == "" || cfg.ClientID == "" {
		return "", errors.New("sign-in is not set up; set the issuer first")
	}
	source := syncclient.NewOIDCSource(cfg.Issuer, cfg.ClientID, cfg.Scopes, secretAdapter{c.secrets})
	if err := source.Login(context.Background(), opener.Open); err != nil {
		return "", syncError(err)
	}
	if account, ok := source.Account(); ok {
		cfg.Account = &account
		if err := syncclient.SaveConfigAt(c.syncPath, cfg); err != nil {
			return "", errors.New("could not save the account")
		}
	}
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
			return signInChooseJSON, nil
		}
		return "", syncError(err)
	}
	return signInLinkedJSON, nil
}

// SignOut forgets the refresh token, so the next sync is signed out until SignIn runs again. The
// database stays linked to its device.
func (c *Core) SignOut() error {
	_ = c.StopSync()
	if c.secrets == nil {
		return nil
	}
	if err := syncclient.DeleteRefreshToken(secretAdapter{c.secrets}); err != nil {
		return errors.New("could not sign out")
	}
	return nil
}

// SyncNow pushes and pulls once, then refills the id blocks. StartSync, the background loop, must
// not be running at the same time.
func (c *Core) SyncNow() error {
	if c.hasRunner() {
		return errors.New("the background sync is running")
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

// SyncStatus returns the Runner's status as JSON:
//
//	{"state":"idle","last_synced":"2026-10-20T09:00:00Z","last_error":"","pending":0}
//
// state is idle, syncing, offline, error, seeding or signed out; last_synced and last_error are
// left out when empty. Without a running Runner it is the last status the Runner reported, with
// the outbox count read now.
func (c *Core) SyncStatus() string {
	c.mu.Lock()
	status := c.lastStatus
	c.mu.Unlock()
	if pending, err := c.dom.SyncPending(context.Background()); err == nil {
		status.Pending = pending
	}
	raw, err := encodeStatus(status)
	if err != nil {
		return `{"state":"error","pending":0}`
	}
	return raw
}

// statusJSON is SyncStatus's shape.
type statusJSON struct {
	State      string `json:"state"`
	LastSynced string `json:"last_synced,omitempty"`
	LastError  string `json:"last_error,omitempty"`
	Pending    int64  `json:"pending"`
}

// encodeStatus encodes one Runner status.
func encodeStatus(status syncclient.Status) (string, error) {
	out := statusJSON{State: string(status.State), LastError: status.LastError, Pending: int64(status.Pending)}
	if !status.LastSynced.IsZero() {
		out.LastSynced = status.LastSynced.UTC().Format(time.RFC3339)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// runnerStatus stores the Runner's newest status and tells the listener.
func (c *Core) runnerStatus(status syncclient.Status) {
	c.mu.Lock()
	c.lastStatus = status
	l := c.listener
	c.mu.Unlock()
	if l == nil {
		return
	}
	if raw, err := encodeStatus(status); err == nil {
		l.OnStatus(raw)
	}
}

// hasRunner says whether a background sync is running.
func (c *Core) hasRunner() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.runner != nil
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
	case cfg.Issuer != "":
		if c.secrets == nil {
			return nil, errors.New("this phone has no secure store for the sign-in")
		}
		tokens = syncclient.NewOIDCSource(cfg.Issuer, cfg.ClientID, cfg.Scopes, secretAdapter{c.secrets})
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

// syncError turns an internal sync failure into a sentence fit to show the user.
func syncError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, syncclient.ErrSeeding):
		return errors.New("another device is still uploading this workspace; try again later")
	case errors.Is(err, syncclient.ErrLocalData):
		return errors.New("this phone already has tasks of its own")
	case errors.Is(err, syncclient.ErrNotLinked):
		return errors.New("this phone is not linked")
	case errors.Is(err, syncclient.ErrSignedOut), syncclient.IsUnauthorized(err):
		return errors.New("sign in again")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return errors.New("the sync was cancelled")
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
