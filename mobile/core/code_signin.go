package core

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// The failure kinds CodeState reports, so the screen can word them without matching error text.
const (
	failureWrongCode     = "wrong_code"
	failureCodeExpired   = "expired"
	failureTooMany       = "too_many"
	failureWrongPassword = "wrong_password"
	failureDeactivated   = "deactivated"
	failureNetwork       = "network"
	failureUnsupported   = "unsupported"
	failureOther         = "other"
)

// codeSignIn is the code step of the sign-in (M13 P2): the address a code was mailed to, the id of
// that code and when it was sent, and why the last attempt failed. Core keeps it under mu.
type codeSignIn struct {
	email   string
	otpID   string
	sentAt  time.Time
	failure string
}

// codeStateJSON is CodeState's shape.
type codeStateJSON struct {
	Step         string `json:"step"`
	Email        string `json:"email"`
	ResendInSecs int    `json:"resend_in"`
	Failure      string `json:"failure"`
}

// CodeState tells the Sync screen where the emailed-code sign-in is:
//
//	{"step":"code","email":"ada@example.com","resend_in":24,"failure":""}
//
// step is "code" once RequestCode mailed a code (and until a sign-in finishes), else "". resend_in
// is the seconds left before ResendCode is allowed, 0 when it is. failure is the kind of the last
// failed attempt of RequestCode, SignInWithCode, ResendCode or SignIn, cleared by the next
// attempt: "wrong_code", "expired" (a new code was mailed), "too_many", "wrong_password",
// "deactivated", "network", "unsupported", "other", or "". The screen re-reads it when the sign-in
// stage changes (SyncStatus's sign_in_stage goes back to "").
func (c *Core) CodeState() (string, error) {
	c.mu.Lock()
	code := c.code
	c.mu.Unlock()
	out := codeStateJSON{Email: code.email, Failure: code.failure}
	if code.otpID != "" {
		out.Step = "code"
		if left := code.sentAt.Add(data.CodeResendAfter).Sub(c.nowTime()); left > 0 {
			out.ResendInSecs = int((left + time.Second - 1) / time.Second)
		}
	}
	return marshal(out)
}

// RequestCode mails a 6-digit sign-in code to email; an email with no account gets one. The code
// step starts here: CodeState then reports step "code".
func (c *Core) RequestCode(email string) error {
	return c.sendCode(strings.TrimSpace(email), false)
}

// ResendCode mails a new code to the address of the code step. It is refused (without a request)
// less than 30 seconds after the last one.
func (c *Core) ResendCode() error {
	c.mu.Lock()
	email, sentAt := c.code.email, c.code.sentAt
	c.mu.Unlock()
	if email == "" {
		return errors.New("ask for a code first")
	}
	if c.nowTime().Before(sentAt.Add(data.CodeResendAfter)) {
		return errors.New("wait a moment before asking for another code")
	}
	return c.sendCode(email, false)
}

// sendCode is the request behind RequestCode, ResendCode and an expired code's replacement.
func (c *Core) sendCode(email string, expired bool) error {
	if c.secrets == nil {
		return errors.New("this phone has no secure store for the sign-in")
	}
	c.noteCodeFailure(nil)
	c.setSignInStage(signInStageSending)
	defer c.setSignInStage(signInStageNone)
	cfg, err := c.signInConfig()
	if err != nil {
		return c.codeFailed(err)
	}
	source, ok := c.accountSource(cfg).(syncclient.CodeSignIn)
	if !ok {
		return c.codeFailed(syncclient.ErrSignInUnsupported)
	}
	otpID, err := source.RequestCode(context.Background(), cfg.Server, email)
	if err != nil {
		return c.codeFailed(err)
	}
	c.mu.Lock()
	c.code = codeSignIn{email: email, otpID: otpID, sentAt: c.nowTime()}
	if expired {
		c.code.failure = failureCodeExpired
	}
	c.mu.Unlock()
	return nil
}

// SignInWithCode checks the code and continues as SignIn does, answering {"state":"linked"} or
// {"state":"choose"}. A code mailed more than ten minutes ago is not tried: a new one is mailed at
// once and the call fails with failure "expired".
func (c *Core) SignInWithCode(code, deviceName string) (string, error) {
	if c.secrets == nil {
		return "", errors.New("this phone has no secure store for the sign-in")
	}
	c.mu.Lock()
	email, otpID, sentAt := c.code.email, c.code.otpID, c.code.sentAt
	c.mu.Unlock()
	if otpID == "" {
		return "", errors.New("ask for a code first")
	}
	if c.nowTime().Sub(sentAt) > data.CodeValidFor {
		if err := c.sendCode(email, true); err != nil {
			return "", err
		}
		return "", errors.New("that code expired; we sent a new one")
	}
	c.noteCodeFailure(nil)
	c.startLink()
	defer c.finishLink()
	c.setSignInStage(signInStageChecking)
	defer c.setSignInStage(signInStageNone)
	cfg, err := c.signInConfig()
	if err != nil {
		return "", c.codeFailed(err)
	}
	source, ok := c.accountSource(cfg).(interface {
		syncclient.AccountSource
		syncclient.CodeSignIn
	})
	if !ok {
		return "", c.codeFailed(syncclient.ErrSignInUnsupported)
	}
	if err := source.SignInWithCode(context.Background(), cfg.Server, otpID, strings.TrimSpace(code)); err != nil {
		if errors.Is(err, syncclient.ErrInvalidCode) {
			c.noteCodeFailure(err)
			return "", errors.New("that code isn't right")
		}
		return "", c.codeFailed(err)
	}
	return c.finishSignIn(cfg, source, deviceName)
}

// RequestPasswordReset mails a link that sets a new password in the browser.
func (c *Core) RequestPasswordReset(email string) error {
	if c.secrets == nil {
		return errors.New("this phone has no secure store for the sign-in")
	}
	cfg, err := c.signInConfig()
	if err != nil {
		return c.codeFailed(err)
	}
	source, ok := c.accountSource(cfg).(syncclient.CodeSignIn)
	if !ok {
		return c.codeFailed(syncclient.ErrSignInUnsupported)
	}
	if err := source.RequestPasswordReset(context.Background(), cfg.Server, strings.TrimSpace(email)); err != nil {
		return c.codeFailed(err)
	}
	return nil
}

// codeFailed records why an attempt failed and returns the sentence for it.
func (c *Core) codeFailed(err error) error {
	c.noteCodeFailure(err)
	if errors.Is(err, syncclient.ErrInvalidCode) {
		return errors.New("that email address isn't valid")
	}
	return syncError(err)
}

// noteCodeFailure keeps the kind of err for CodeState.
func (c *Core) noteCodeFailure(err error) {
	kind := failureKind(err)
	c.mu.Lock()
	c.code.failure = kind
	c.mu.Unlock()
}

// clearCodeState forgets the code step after a sign-in stored its token.
func (c *Core) clearCodeState() {
	c.mu.Lock()
	c.code = codeSignIn{}
	c.mu.Unlock()
}

// failureKind classifies a sign-in error for CodeState.
func failureKind(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, syncclient.ErrInvalidCode):
		return failureWrongCode
	case errors.Is(err, syncclient.ErrInvalidCredentials):
		return failureWrongPassword
	case errors.Is(err, syncclient.ErrTooManyAttempts):
		return failureTooMany
	case errors.Is(err, syncclient.ErrAccountLocked):
		return failureDeactivated
	case errors.Is(err, syncclient.ErrSignInUnsupported):
		return failureUnsupported
	case errors.Is(err, syncclient.ErrGoogleCancelled):
		return failureGoogleCancelled
	case errors.Is(err, syncclient.ErrGoogleFailed):
		return failureGoogleFailed
	case isNetworkError(err):
		return failureNetwork
	}
	return failureOther
}
