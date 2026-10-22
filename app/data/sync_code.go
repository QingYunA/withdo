package data

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

const (
	// CodeResendAfter is how long after a code was sent the next one may be asked for.
	CodeResendAfter = 30 * time.Second
	// CodeValidFor is how long a code works; the server's 600 s, kept here so an old code is not
	// even sent.
	CodeValidFor = 10 * time.Minute
)

// codeState is the signed-out card's code flow (M13 P2): the screen, the address, the id of the
// code that was mailed and when. It lives in syncTask, under Real.mu, and is cleared when a
// sign-in finishes or is cancelled.
type codeState struct {
	step     SignInStep
	email    string
	otpID    string
	sentAt   time.Time
	resendAt time.Time
	cfg      syncclient.Config
	// probed is the server whose /v1/config was read for passwordOnly; probing is a read in flight.
	probed       string
	probing      bool
	passwordOnly bool
}

// SyncSetSignInStep moves the signed-out card between its screens (to the password screen and back,
// or back to the email with the address kept). It clears the last failure.
func (r *Real) SyncSetSignInStep(step SignInStep) {
	r.mu.Lock()
	if r.sync.signingIn {
		r.mu.Unlock()
		return
	}
	r.sync.code.step = step
	r.sync.signInFailure = SignInFailureNone
	r.sync.status.LastError = ""
	r.mu.Unlock()
	r.syncChanged()
}

// SyncRequestCode mails a 6-digit code to email (the server makes the account if there is none) and
// moves the card to the code step.
func (r *Real) SyncRequestCode(email string) {
	email = strings.TrimSpace(email)
	if !r.beginCodeWork(SignInStageSending) {
		return
	}
	go r.requestCode(email, false)
}

// SyncResendCode mails a new code to the address of the code step. It does nothing before
// ResendAt.
func (r *Real) SyncResendCode() {
	r.mu.Lock()
	email := r.sync.code.email
	early := r.Now().Before(r.sync.code.resendAt)
	r.mu.Unlock()
	if email == "" || early || !r.beginCodeWork(SignInStageSending) {
		return
	}
	go r.requestCode(email, false)
}

// SyncSignInWithCode checks the code. A code sent more than ten minutes ago is not tried: a new
// one is mailed at once and the failure is SignInFailureCodeExpired. On success it continues as the
// password sign-in does.
func (r *Real) SyncSignInWithCode(code string) {
	code = strings.TrimSpace(code)
	r.mu.Lock()
	expired := r.sync.code.otpID != "" && r.Now().Sub(r.sync.code.sentAt) > CodeValidFor
	email := r.sync.code.email
	r.mu.Unlock()
	if expired {
		if r.beginCodeWork(SignInStageSending) {
			go r.requestCode(email, true)
		}
		return
	}
	if !r.beginCodeWork(SignInStageChecking) {
		return
	}
	go r.signInWithCode(code)
}

// SyncRequestPasswordReset mails a password reset link and moves the card to its "check your
// email" step.
func (r *Real) SyncRequestPasswordReset(email string) {
	email = strings.TrimSpace(email)
	if !r.beginCodeWork(SignInStageSending) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cfg, source, err := r.codeSource(ctx)
		if err == nil {
			err = source.RequestPasswordReset(ctx, cfg.Server, email)
		}
		if err != nil {
			r.failSignIn(err)
			return
		}
		r.mu.Lock()
		r.sync.signingIn = false
		r.sync.signInStage = SignInStageNone
		r.sync.code.step, r.sync.code.email = SignInStepResetSent, email
		r.mu.Unlock()
		r.syncChanged()
	}()
}

// beginCodeWork marks a request or check as running; it is false when one already is or sync is
// not set up here.
func (r *Real) beginCodeWork(stage SignInStage) bool {
	if r.syncPath == "" || r.opts.Secrets == nil {
		r.setSyncError(errors.New("sync is not set up on this Mac"))
		return false
	}
	r.mu.Lock()
	if r.sync.signingIn {
		r.mu.Unlock()
		return false
	}
	r.loadSyncLocked()
	r.sync.signingIn = true
	r.sync.signInStage = stage
	r.sync.conflict = nil
	r.sync.status.LastError = ""
	r.sync.failedSince = time.Time{}
	r.sync.signInFailure = SignInFailureNone
	r.mu.Unlock()
	r.syncChanged()
	return true
}

// codeSource is the server's config and its token source as a CodeSignIn; the config is fetched
// once per code flow. A server that does not sign in with PocketBase has no codes.
func (r *Real) codeSource(ctx context.Context) (syncclient.Config, syncclient.CodeSignIn, error) {
	r.mu.Lock()
	cfg, server := r.sync.code.cfg, r.sync.server
	r.mu.Unlock()
	if cfg.Server == "" || cfg.Server != server {
		public, err := syncclient.FetchConfig(ctx, server, nil)
		if err != nil {
			return cfg, nil, err
		}
		cfg = syncclient.Config{Server: server}
		if err := cfg.ApplyPublic(public); err != nil {
			return cfg, nil, err
		}
		r.mu.Lock()
		r.sync.code.cfg = cfg
		r.mu.Unlock()
	}
	source, ok := r.accountSource(cfg).(syncclient.CodeSignIn)
	if !ok {
		return cfg, nil, syncclient.ErrSignInUnsupported
	}
	return cfg, source, nil
}

// requestCode is SyncRequestCode and SyncResendCode off the UI thread. expired marks a re-send
// because the code in hand was too old.
func (r *Real) requestCode(email string, expired bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cfg, source, err := r.codeSource(ctx)
	var otpID string
	if err == nil {
		otpID, err = source.RequestCode(ctx, cfg.Server, email)
	}
	if err != nil {
		r.failSignIn(err)
		return
	}
	now := r.Now()
	r.mu.Lock()
	r.sync.signingIn = false
	r.sync.signInStage = SignInStageNone
	r.sync.code.step, r.sync.code.email, r.sync.code.otpID = SignInStepCode, email, otpID
	r.sync.code.sentAt, r.sync.code.resendAt = now, now.Add(CodeResendAfter)
	if expired {
		r.sync.signInFailure = SignInFailureCodeExpired
	}
	r.mu.Unlock()
	r.syncChanged()
}

// signInWithCode is SyncSignInWithCode off the UI thread.
func (r *Real) signInWithCode(code string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	r.mu.Lock()
	otpID := r.sync.code.otpID
	r.mu.Unlock()
	cfg, source, err := r.codeSource(ctx)
	if err == nil {
		if otpID == "" {
			err = syncclient.ErrInvalidCode
		} else {
			err = source.SignInWithCode(ctx, cfg.Server, otpID, code)
		}
	}
	if err != nil {
		r.failSignIn(err)
		return
	}
	r.finishSignIn(ctx, cfg.Server, cfg, r.accountSource(cfg))
}

// SyncCheckServer reads the server's /v1/config once, to learn whether it signs in with emailed
// codes. A server that is not PocketBase keeps the old account-and-password card
// (SyncSnapshot.PasswordOnly); one that does not answer leaves the card as it is.
func (r *Real) SyncCheckServer() {
	r.mu.Lock()
	r.loadSyncLocked()
	server := r.sync.server
	if r.sync.code.probing || r.sync.code.probed == server {
		r.mu.Unlock()
		return
	}
	r.sync.code.probing = true
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		public, err := syncclient.FetchConfig(ctx, server, nil)
		r.mu.Lock()
		r.sync.code.probing = false
		if err == nil {
			r.sync.code.probed = server
			r.sync.code.passwordOnly = public.Auth != syncclient.AuthPocketBase
		}
		r.mu.Unlock()
		if err == nil {
			r.syncChanged()
		}
	}()
}
