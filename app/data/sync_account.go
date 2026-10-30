package data

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// SyncOtherAccount is what the sheet "This Mac was linked to another account" shows: the account
// this Mac was synced with (empty when sync.json no longer remembers it) and the one that just
// signed in. Nothing synced while it is open.
type SyncOtherAccount struct {
	PreviousEmail string
	AccountEmail  string
}

// GoogleLinkState is the Google row of Settings > Account.
type GoogleLinkState int

// The states of the row. GoogleLinkHidden is the default: signed out, a server without Google, or
// not asked yet.
const (
	GoogleLinkHidden GoogleLinkState = iota
	GoogleLinkAvailable
	GoogleLinkLinked
	// GoogleLinkWaiting is the browser open on Google's consent page, the app waiting.
	GoogleLinkWaiting
)

// GoogleLinkFailure is why the last "Link Google" did not link.
type GoogleLinkFailure string

// The failures of linking Google.
const (
	GoogleLinkFailureNone GoogleLinkFailure = ""
	// GoogleLinkFailureUsed is a Google account that already belongs to another account.
	GoogleLinkFailureUsed    GoogleLinkFailure = "used"
	GoogleLinkFailureFailed  GoogleLinkFailure = "failed"
	GoogleLinkFailureNetwork GoogleLinkFailure = "network"
)

// googleLinkTask is the Google row's state under Real.mu.
type googleLinkTask struct {
	state   GoogleLinkState
	failure GoogleLinkFailure
	probing bool
	cancel  context.CancelFunc
	gen     int
}

// SyncUseOldAccount is the sheet's first choice: sign out of the account that just signed in, and
// go back to the email card with the old address kept for the person to continue with.
func (r *Real) SyncUseOldAccount() {
	r.mu.Lock()
	other := r.sync.other
	r.mu.Unlock()
	if other == nil {
		return
	}
	r.SyncCancel()
	r.mu.Lock()
	r.sync.code.step, r.sync.code.email = SignInStepSignIn, other.PreviousEmail
	r.mu.Unlock()
	r.syncChanged()
}

// SyncMoveToAccount is the sheet's second choice: forget the old account's link of this Mac and
// continue as a new device of the account that just signed in. The Mac's tasks stay; they go
// through the usual link, which asks the both-have-tasks question when the account has tasks too.
func (r *Real) SyncMoveToAccount() {
	r.mu.Lock()
	other := r.sync.other
	client := r.sync.client
	source, ok := r.sync.tokens.(syncclient.AccountSource)
	cfg, server := r.sync.cfg, r.sync.server
	if other == nil || client == nil || !ok {
		r.mu.Unlock()
		return
	}
	r.sync.other = nil
	r.sync.signingIn = true
	r.sync.signInStage = SignInStageDownloading
	r.mu.Unlock()
	r.syncChanged()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if err := client.Unlink(ctx); err != nil {
			r.failSignIn(err)
			return
		}
		r.mu.Lock()
		r.sync.linked, r.sync.deviceID = false, ""
		r.sync.seedTotal, r.sync.pending = 0, 0
		r.mu.Unlock()
		r.finishSignIn(ctx, server, cfg, source)
	}()
}

// rememberAccount stores the signed-in account's email as the one this database is linked to, so a
// later sign-in as someone else can name it.
func (r *Real) rememberAccount(client *syncclient.Client) {
	r.mu.Lock()
	email := r.sync.account.Email
	r.mu.Unlock()
	_ = client.RememberAccount(context.Background(), email)
}

// rememberAccountOnce does rememberAccount for the first successful sync of a run: a sync that
// worked proves the signed-in account owns this device.
func (r *Real) rememberAccountOnce() {
	r.mu.Lock()
	if r.sync.remembered || !r.sync.hasAccount || r.sync.cfg.Server == "" {
		r.mu.Unlock()
		return
	}
	r.sync.remembered = true
	server := r.sync.cfg.Server
	r.mu.Unlock()
	if client, err := syncclient.New(r.core, syncclient.Options{BaseURL: server}); err == nil {
		r.rememberAccount(client)
	}
}

// rememberBeforeSignOut keeps the email of a linked Mac across a sign-out, unless the sign-out is
// from an account that does not own this Mac (it must not become the remembered one).
func (r *Real) rememberBeforeSignOut() {
	r.mu.Lock()
	skip := !r.sync.linked || !r.sync.hasAccount || r.sync.status.State == syncclient.StateWrongAccount ||
		r.sync.other != nil || r.sync.cfg.Server == ""
	server := r.sync.cfg.Server
	r.mu.Unlock()
	if skip {
		return
	}
	client, err := syncclient.New(r.core, syncclient.Options{BaseURL: server})
	if err != nil || client.RememberedAccount(context.Background()) != "" {
		return
	}
	r.rememberAccount(client)
}

// SyncRefreshGoogleLink asks the server whether it offers Google and whether the account has a
// Google identity, for the Google row. It is called when the pane opens.
func (r *Real) SyncRefreshGoogleLink() {
	r.mu.Lock()
	r.loadSyncLocked()
	cfg := r.sync.cfg
	if !r.sync.hasAccount || !cfg.SignsInWithAccount() || r.opts.Secrets == nil || r.sync.googleLink.probing ||
		r.sync.googleLink.state == GoogleLinkWaiting {
		r.mu.Unlock()
		return
	}
	r.sync.googleLink.probing = true
	r.mu.Unlock()
	go func() {
		defer func() {
			r.mu.Lock()
			r.sync.googleLink.probing = false
			r.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		source := r.accountSource(cfg)
		state := GoogleLinkHidden
		if signIn, ok := source.(syncclient.GoogleSignIn); ok {
			if offered, err := signIn.GoogleAvailable(ctx, cfg.Server); err == nil && offered {
				state = GoogleLinkAvailable
			}
		}
		if linker, ok := source.(syncclient.GoogleLinker); ok && state == GoogleLinkAvailable {
			linked, err := linker.GoogleLinked(ctx, cfg.Server)
			switch {
			case err != nil:
				state = GoogleLinkHidden
			case linked:
				state = GoogleLinkLinked
			}
		}
		r.mu.Lock()
		if r.sync.googleLink.state != GoogleLinkWaiting {
			r.sync.googleLink.state = state
		}
		r.mu.Unlock()
		r.syncChanged()
	}()
}

// SyncLinkGoogle opens the system browser on Google's consent page, through open, and waits for
// the answer. On success the account has a Google identity; a Google account that belongs to
// another account is GoogleLinkFailureUsed. SyncCancelGoogleLink drops the wait.
func (r *Real) SyncLinkGoogle(open func(url string)) {
	r.mu.Lock()
	cfg := r.sync.cfg
	if !r.sync.hasAccount || r.sync.googleLink.state != GoogleLinkAvailable || r.opts.Secrets == nil {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.sync.googleLink.gen++
	gen := r.sync.googleLink.gen
	r.sync.googleLink.cancel = cancel
	r.sync.googleLink.state = GoogleLinkWaiting
	r.sync.googleLink.failure = GoogleLinkFailureNone
	r.mu.Unlock()
	r.syncChanged()
	go func() {
		defer cancel()
		var err error
		linker, ok := r.accountSource(cfg).(syncclient.GoogleLinker)
		if !ok {
			err = syncclient.ErrSignInUnsupported
		} else {
			err = linker.LinkGoogle(ctx, cfg.Server, func(url string) error {
				open(url)
				return nil
			})
		}
		r.mu.Lock()
		if r.sync.googleLink.gen != gen {
			r.mu.Unlock()
			return
		}
		r.sync.googleLink.cancel = nil
		switch {
		case err == nil:
			r.sync.googleLink.state = GoogleLinkLinked
		case errors.Is(err, syncclient.ErrGoogleCancelled):
			r.sync.googleLink.state = GoogleLinkAvailable
		default:
			r.sync.googleLink.state = GoogleLinkAvailable
			r.sync.googleLink.failure = googleLinkFailureKind(err)
		}
		r.mu.Unlock()
		r.syncChanged()
	}()
}

// SyncCancelGoogleLink drops a Google link that waits for the browser.
func (r *Real) SyncCancelGoogleLink() {
	r.mu.Lock()
	cancel := r.sync.googleLink.cancel
	if r.sync.googleLink.state == GoogleLinkWaiting {
		r.sync.googleLink.gen++
		r.sync.googleLink.state = GoogleLinkAvailable
		r.sync.googleLink.cancel = nil
	}
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	r.syncChanged()
}

// googleLinkFailureKind classifies why a Google link failed.
func googleLinkFailureKind(err error) GoogleLinkFailure {
	switch {
	case errors.Is(err, syncclient.ErrGoogleAlreadyLinked):
		return GoogleLinkFailureUsed
	case errors.Is(err, syncclient.ErrGoogleFailed):
		return GoogleLinkFailureFailed
	case signInNetworkError(err):
		return GoogleLinkFailureNetwork
	}
	return GoogleLinkFailureFailed
}

// emailsEqual compares two addresses the way a mailbox does.
func emailsEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
