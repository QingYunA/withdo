package data

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// The official model on this computer (M15): Settings > Model's first card, the composer's chip
// and the rule for which model answers all read OfficialInfo. The state is the sync session
// (signed in or not), what the server's GET /v1/config says (does it offer the model) and the
// account's turns for today, asked for in the background and kept for a few minutes.

// OfficialPhase is what Settings says about the official model.
type OfficialPhase int

// The phases of the official card.
const (
	// OfficialHidden: nothing to offer. The server does not offer the model (a server of one's
	// own), or it is not known yet.
	OfficialHidden OfficialPhase = iota
	// OfficialSignedOut: it would be there after signing in.
	OfficialSignedOut
	// OfficialReady: signed in, and turns are left today.
	OfficialReady
	// OfficialUsedUp: signed in, and today's turns are used.
	OfficialUsedUp
)

// OfficialInfo is the official card in one frame.
type OfficialInfo struct {
	Phase OfficialPhase
	// Left and Limit are today's turns; Left is -1 until the server has said.
	Left, Limit int
	// ResetsAt is when the count starts again (UTC midnight); zero until known.
	ResetsAt time.Time
	// InUse says the official model is the one that answers: no provider of one's own is in use.
	InUse bool
}

// Offered says the card has the model to show: signed in or not, it is a thing to use or to sign
// in for.
func (o OfficialInfo) Offered() bool { return o.Phase != OfficialHidden }

// OfficialSource is where Settings and the composer learn about the official model. *Real
// implements it from the sync session.
type OfficialSource interface {
	// OfficialInfo is the current state, without InUse (the provider settings know that). It
	// never blocks on the network: it starts a refresh in the background when its answer is old.
	OfficialInfo() OfficialInfo
	// OfficialModel is the model and whether it can answer now: signed in, offered by the server,
	// and turns left. It is the providers.Selector's Official.
	OfficialModel() (providers.Official, bool)
	// RefreshOfficial asks the server again, soon: after a turn, and when Settings opens.
	RefreshOfficial()
	// TestOfficial checks the session and the server, as the Test button does. It does not spend a
	// turn.
	TestOfficial(ctx context.Context) (time.Duration, error)
}

// officialTask is the state behind OfficialInfo.
type officialTask struct {
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

const (
	officialRefreshEvery = 5 * time.Minute
	officialRetryAfter   = 30 * time.Second
	officialRefreshTO    = 15 * time.Second
)

var _ OfficialSource = (*Real)(nil)

// officialSession reads who the sync session is: the server's address and whether an account (or a
// development token) is signed in.
func (r *Real) officialSession() (server string, signedIn bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loadSyncLocked()
	cfg := r.sync.cfg
	return r.sync.server, (cfg.SignsInWithAccount() && cfg.Account != nil) || cfg.Token != ""
}

// OfficialInfo implements OfficialSource.
func (r *Real) OfficialInfo() OfficialInfo {
	server, signedIn := r.officialSession()
	o := &r.llm
	o.mu.Lock()
	if o.server != server || o.signedIn != signedIn { // another server or account: ask again
		o.server, o.signedIn, o.known, o.haveQuota, o.refreshAt = server, signedIn, false, false, time.Time{}
	}
	due := !o.busy && !time.Now().Before(o.refreshAt)
	o.busy = o.busy || due
	info := o.info(time.Now())
	o.mu.Unlock()
	if due {
		go r.refreshOfficial()
	}
	return info
}

// info is the card for the state held. Call with mu held.
func (o *officialTask) info(now time.Time) OfficialInfo {
	info := OfficialInfo{Left: -1}
	switch {
	case o.known && !o.offered:
		return info
	case !o.signedIn:
		// Signed out the server may not have answered yet: the default server is the official one,
		// which offers the model, so the card shows until it says otherwise.
		info.Phase = OfficialSignedOut
		return info
	case !o.known:
		return info
	}
	info.Phase = OfficialReady
	if o.haveQuota {
		info.Left, info.Limit, info.ResetsAt = o.quota.Left, o.quota.Limit, o.quota.ResetsAt
		if o.quota.Left <= 0 && now.Before(o.quota.ResetsAt) {
			info.Phase = OfficialUsedUp
		}
	}
	return info
}

// OfficialModel implements OfficialSource.
func (r *Real) OfficialModel() (providers.Official, bool) {
	info := r.OfficialInfo()
	server, _ := r.officialSession()
	base, err := syncclient.ServerURL(server)
	if err != nil {
		return providers.Official{}, false
	}
	return providers.Official{
		BaseURL: base + "/v1/llm", Token: r.officialToken, SignedOut: syncclient.IsSignedOut,
	}, info.Phase == OfficialReady
}

// officialToken is the session's bearer for one request, from the account source the runner and
// Settings share, so it is renewed the way every other call's is.
func (r *Real) officialToken(ctx context.Context) (string, error) {
	client, err := r.officialClient()
	if err != nil {
		return "", syncclient.ErrSignedOut
	}
	return client.BearerToken(ctx)
}

// officialClient is the sync client of the signed-in session, for its token and the quota.
func (r *Real) officialClient() (*syncclient.Client, error) {
	r.mu.Lock()
	r.loadSyncLocked()
	client, cfg := r.sync.client, r.sync.cfg
	r.mu.Unlock()
	if client != nil {
		return client, nil
	}
	if cfg.Server == "" {
		return nil, syncclient.ErrSignedOut
	}
	return r.syncClientFor(cfg)
}

// RefreshOfficial implements OfficialSource.
func (r *Real) RefreshOfficial() {
	o := &r.llm
	o.mu.Lock()
	start := !o.busy
	o.busy = true
	o.mu.Unlock()
	if start {
		go r.refreshOfficial()
	}
}

// refreshOfficial asks the server what it offers and, signed in, how many turns are left. The
// caller has set busy, which this clears.
func (r *Real) refreshOfficial() {
	o := &r.llm
	ctx, cancel := context.WithTimeout(context.Background(), officialRefreshTO)
	defer cancel()
	server, signedIn := r.officialSession()
	offered, quota, haveQuota, err := r.askOfficial(ctx, server, signedIn)
	o.mu.Lock()
	switch {
	case o.server != server || o.signedIn != signedIn: // the session changed while asking: that answer is old
		o.refreshAt = time.Time{}
	case err != nil:
		o.refreshAt = time.Now().Add(officialRetryAfter)
	default:
		o.known, o.offered, o.quota, o.haveQuota = true, offered, quota, haveQuota
		o.refreshAt = time.Now().Add(officialRefreshEvery)
	}
	o.busy = false
	o.mu.Unlock()
	r.syncChanged() // the cards and the chip follow
}

func (r *Real) askOfficial(ctx context.Context, server string, signedIn bool) (offered bool, quota syncclient.LLMQuota, haveQuota bool, err error) {
	cfg, err := syncclient.FetchConfig(ctx, server, nil)
	if err != nil {
		return false, quota, false, err
	}
	if !cfg.LLM || !signedIn {
		return cfg.LLM, quota, false, nil
	}
	client, err := r.officialClient()
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

// TestOfficial implements OfficialSource: it reads the quota, which proves the session, the server
// and the route without spending a turn.
func (r *Real) TestOfficial(ctx context.Context) (time.Duration, error) {
	client, err := r.officialClient()
	if err != nil {
		return 0, providers.ErrOfficialSignedOut
	}
	start := time.Now()
	if _, err := client.LLMQuota(ctx); err != nil {
		switch {
		case errors.Is(err, syncclient.ErrLLMNotSignedIn):
			return 0, providers.ErrOfficialSignedOut
		case errors.Is(err, syncclient.ErrLLMOff):
			return 0, providers.ErrOfficialOff
		}
		return 0, err
	}
	r.RefreshOfficial()
	return time.Since(start), nil
}
