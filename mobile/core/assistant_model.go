package core

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// The official model on the phone: the same rule as app/data/official.go (the desktop's), reading
// the phone's own sync session. Signed in to a server that offers it, with turns left today, the
// composer asks the model; anything else leaves the local rules (ParseQuick) in charge.

// The phases ModelState reports.
const (
	phaseReady     = "ready"      // signed in, offered, turns left: Ask may be called
	phaseSignedOut = "signed_out" // it would be there after signing in
	phaseUsedUp    = "used_up"    // signed in, and today's turns are used
	phaseHidden    = "hidden"     // nothing to offer: the server has no such model, or not known yet
)

const (
	modelRefreshEvery = 5 * time.Minute
	modelRetryAfter   = 30 * time.Second
	modelRefreshTO    = 15 * time.Second
)

// officialState is what ModelState reads: the session, what the server offers and today's turns,
// asked for in the background and kept for a few minutes.
type officialState struct {
	mu        sync.Mutex
	server    string
	signedIn  bool
	known     bool // offered is the server's answer
	offered   bool
	quota     syncclient.LLMQuota
	haveQuota bool
	refreshAt time.Time // no refresh before this
	busy      bool
}

// modelStateJSON is the JSON of ModelState.
type modelStateJSON struct {
	Source string `json:"source"`
	Phase  string `json:"phase"`
	Left   int    `json:"left"`
	Limit  int    `json:"limit"`
}

// ModelState says which model would answer the composer, as JSON, without touching the network
// (it starts a refresh in the background when its answer is old):
//
//	{"source":"official","phase":"ready","left":86,"limit":100}
//
// source is official when Ask may be called and none otherwise (the app uses ParseQuick). phase is
// ready, signed_out (the model would be there after signing in), used_up (today's turns are used;
// it comes back at 0:00 UTC) or hidden (the server has no official model, or it is not known yet).
// left is today's turns left and limit the day's turns; left is -1 until the server has said. The
// listener's OnStatus fires when a refresh finishes, so read it again then.
func (c *Core) ModelState() (string, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return "", errors.New("the core is closed")
	}
	return marshal(c.modelInfo())
}

// modelInfo is the current state, starting a refresh when the held answer is old.
func (c *Core) modelInfo() modelStateJSON {
	server, signedIn, explicit := c.officialSession()
	o := &c.assist.official
	o.mu.Lock()
	if o.server != server || o.signedIn != signedIn { // another server or account: ask again
		o.server, o.signedIn, o.known, o.haveQuota, o.refreshAt = server, signedIn, false, false, time.Time{}
	}
	// A phone that never chose a server and never signed in has nothing to ask: the default server
	// is the official one, which offers the model, so the line says to sign in without a request.
	due := (signedIn || explicit) && !o.busy && !time.Now().Before(o.refreshAt)
	o.busy = o.busy || due
	info := o.info(time.Now())
	o.mu.Unlock()
	if due {
		go c.refreshModel()
	}
	return info
}

// info is the state held, as the phases name it. Call with mu held.
func (o *officialState) info(now time.Time) modelStateJSON {
	info := modelStateJSON{Source: "none", Phase: phaseHidden, Left: -1}
	switch {
	case o.known && !o.offered:
		return info
	case !o.signedIn:
		info.Phase = phaseSignedOut
		return info
	case !o.known:
		return info
	}
	info.Phase = phaseReady
	if o.haveQuota {
		info.Left, info.Limit = o.quota.Left, o.quota.Limit
		if o.quota.Left <= 0 && now.Before(o.quota.ResetsAt) {
			info.Phase = phaseUsedUp
		}
	}
	if info.Phase == phaseReady {
		info.Source = "official"
	}
	return info
}

// officialSession reads who the sync session is: the server (the default before one is chosen),
// whether an account or a development token is signed in, and whether sync.json names a server.
func (c *Core) officialSession() (server string, signedIn, explicit bool) {
	server = data.DefaultSyncServer
	cfg, found, err := c.loadConfig()
	if err != nil || !found {
		return server, false, false
	}
	return cfg.Server, c.hasCredentials(cfg), true
}

// officialModel is the providers.Selector's Official: the server's model and whether it can answer
// now. It never touches the network.
func (c *Core) officialModel() (providers.Official, bool) {
	cfg, found, err := c.loadConfig()
	if err != nil || !found {
		return providers.Official{}, false
	}
	base, err := syncclient.ServerURL(cfg.Server)
	if err != nil {
		return providers.Official{}, false
	}
	return providers.Official{
		BaseURL: base + "/v1/llm", Token: c.officialToken, SignedOut: syncclient.IsSignedOut,
	}, c.modelInfo().Phase == phaseReady
}

// officialToken is the session's bearer for one request, from the account source the sync runner
// shares, so it is renewed the way every other call's is.
func (c *Core) officialToken(ctx context.Context) (string, error) {
	client, err := c.newClient(false)
	if err != nil {
		return "", syncclient.ErrSignedOut
	}
	return client.BearerToken(ctx)
}

// refreshModel asks the server what it offers and, signed in, how many turns are left. The caller
// has set busy, which this clears.
func (c *Core) refreshModel() {
	o := &c.assist.official
	ctx, cancel := context.WithTimeout(context.Background(), modelRefreshTO)
	defer cancel()
	server, signedIn, _ := c.officialSession()
	offered, quota, haveQuota, err := c.askOfficial(ctx, server, signedIn)
	o.mu.Lock()
	switch {
	case o.server != server || o.signedIn != signedIn: // the session changed while asking: that answer is old
		o.refreshAt = time.Time{}
	case err != nil:
		o.refreshAt = time.Now().Add(modelRetryAfter)
	default:
		o.known, o.offered, o.quota, o.haveQuota = true, offered, quota, haveQuota
		o.refreshAt = time.Now().Add(modelRefreshEvery)
	}
	o.busy = false
	o.mu.Unlock()
	c.notifyStatus() // the chip and the Settings row follow
}

func (c *Core) askOfficial(ctx context.Context, server string, signedIn bool) (offered bool, quota syncclient.LLMQuota, haveQuota bool, err error) {
	cfg, err := syncclient.FetchConfig(ctx, server, nil)
	if err != nil {
		return false, quota, false, err
	}
	if !cfg.LLM || !signedIn {
		return cfg.LLM, quota, false, nil
	}
	client, err := c.newClient(false)
	if err != nil {
		return cfg.LLM, quota, false, err
	}
	quota, err = client.LLMQuota(ctx)
	switch {
	case err == nil:
		return true, quota, true, nil
	case errors.Is(err, syncclient.ErrLLMOff):
		return false, quota, false, nil
	case errors.Is(err, syncclient.ErrLLMNotSignedIn):
		// The server refused the session: the model is offered, but nothing is known of the turns.
		return true, quota, false, nil
	}
	return false, quota, false, err
}

// refreshModelSoon asks the server again now: after a turn, whose count changed.
func (c *Core) refreshModelSoon() {
	o := &c.assist.official
	o.mu.Lock()
	start := !o.busy
	o.busy = true
	o.mu.Unlock()
	if start {
		go c.refreshModel()
	}
}

// notifyStatus tells the listener the status JSON again; the model state is read alongside it.
func (c *Core) notifyStatus() {
	c.mu.Lock()
	l, closed := c.listener, c.closed
	c.mu.Unlock()
	if l == nil || closed {
		return
	}
	var raw string
	if raw = c.SyncStatus(); !json.Valid([]byte(raw)) {
		return
	}
	l.OnStatus(raw)
}
