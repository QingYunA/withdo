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
	failureEmailTaken    = "email_taken"
	failureInvalidEmail  = "invalid_email"
	failureWeakPassword  = "weak_password"
	failureWrongCode     = "wrong_code"
	failureCodeExpired   = "expired"
	failureTooMany       = "too_many"
	failureWrongPassword = "wrong_password"
	failureDeactivated   = "deactivated"
	failureNetwork       = "network"
	failureUnsupported   = "unsupported"
	failureOther         = "other"
)

// The steps CodeState reports once a code was mailed.
const (
	// stepVerify is the six cells that verify the email of a new account (after SignUp, or after
	// SignIn answered {"state":"verify"}).
	stepVerify = "verify"
	// stepResetCode is the six cells and the new password of Forgot password.
	stepResetCode = "reset_code"
)

// codeSignIn is the code step of the sign-up and of the password reset (v42): the step, the address
// a code was mailed to, the id of that code and when it was sent, and why the last attempt failed.
// Core keeps it under mu.
type codeSignIn struct {
	step    string
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

// CodeState tells the Sync screen where the verify and reset steps are:
//
//	{"step":"verify","email":"ada@example.com","resend_in":24,"failure":""}
//
// step is "verify" once SignUp (or a SignIn that answered "verify") mailed a code, "reset_code" once
// RequestResetCode did, until a sign-in finishes; else "". resend_in is the seconds left before
// ResendCode is allowed, 0 when it is. failure is the kind of the last failed attempt of SignUp,
// RequestResetCode, ResendCode, SignInWithCode, ResetPassword or SignIn, cleared by the next
// attempt: "email_taken", "invalid_email", "weak_password", "wrong_code", "expired" (a new code was
// mailed), "too_many", "wrong_password", "deactivated", "network", "unsupported", "other", or "".
// The screen re-reads it when the sign-in stage changes (SyncStatus's sign_in_stage goes back to
// "").
func (c *Core) CodeState() (string, error) {
	c.mu.Lock()
	code := c.code
	c.mu.Unlock()
	out := codeStateJSON{Email: code.email, Failure: code.failure}
	if code.otpID != "" {
		out.Step = code.step
		if left := code.sentAt.Add(data.CodeResendAfter).Sub(c.nowTime()); left > 0 {
			out.ResendInSecs = int((left + time.Second - 1) / time.Second)
		}
	}
	return marshal(out)
}

// SignUp makes an account with an email and a password (at least 8 characters) and mails its first
// code; CodeState then reports step "verify" and SignInWithCode verifies and signs in. lang is the
// app's language, "en" or "zh-Hans", the language of the code mails. A taken email fails with
// failure "email_taken", a malformed one "invalid_email", a short password "weak_password".
func (c *Core) SignUp(email, password, lang string) error {
	if c.secrets == nil {
		return errors.New("this phone has no secure store for the sign-in")
	}
	email = strings.TrimSpace(email)
	c.noteCodeFailure(nil)
	c.setSignInStage(signInStageCreating)
	defer c.setSignInStage(signInStageNone)
	cfg, err := c.signInConfig()
	if err != nil {
		return c.codeFailed(err)
	}
	source, ok := c.accountSource(cfg).(syncclient.CodeSignIn)
	if !ok {
		return c.codeFailed(syncclient.ErrSignInUnsupported)
	}
	otpID, err := source.SignUp(context.Background(), cfg.Server, email, password, lang)
	if err != nil {
		c.rememberCodeEmail(email)
		return c.codeFailed(err)
	}
	c.codeMailed(email, otpID, stepVerify, false)
	return nil
}

// RequestResetCode mails a 6-digit code for a password reset (Forgot password); CodeState then
// reports step "reset_code" and ResetPassword finishes it. An email with no account gets the same
// answer and no mail.
func (c *Core) RequestResetCode(email string) error {
	return c.sendCode(strings.TrimSpace(email), false, stepResetCode)
}

// ResendCode mails a new code to the address of the verify or reset step. It is refused (without a
// request) less than 30 seconds after the last one.
func (c *Core) ResendCode() error {
	c.mu.Lock()
	step, email, sentAt := c.code.step, c.code.email, c.code.sentAt
	c.mu.Unlock()
	if email == "" || step == "" {
		return errors.New("ask for a code first")
	}
	if c.nowTime().Before(sentAt.Add(data.CodeResendAfter)) {
		return errors.New("wait a moment before asking for another code")
	}
	return c.sendCode(email, false, step)
}

// sendCode is the request behind RequestResetCode, ResendCode and an expired code's replacement.
func (c *Core) sendCode(email string, expired bool, step string) error {
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
		c.rememberCodeEmail(email)
		return c.codeFailed(err)
	}
	c.codeMailed(email, otpID, step, expired)
	return nil
}

// codeMailed records the code that was just mailed.
func (c *Core) codeMailed(email, otpID, step string, expired bool) {
	c.mu.Lock()
	c.code = codeSignIn{step: step, email: email, otpID: otpID, sentAt: c.nowTime()}
	if expired {
		c.code.failure = failureCodeExpired
	}
	c.mu.Unlock()
}

// rememberCodeEmail keeps the address of a failed request, for CodeState.
func (c *Core) rememberCodeEmail(email string) {
	c.mu.Lock()
	c.code.email = email
	c.mu.Unlock()
}

// SignInWithCode checks the code of the verify step: it verifies the email and continues as SignIn
// does, answering {"state":"linked"} or {"state":"choose"}. A code mailed more than ten minutes ago
// is not tried: a new one is mailed at once and the call fails with failure "expired".
func (c *Core) SignInWithCode(code, deviceName string) (string, error) {
	return c.finishWithCode(code, deviceName, func(source codeAccountSource, cfg syncclient.Config, otpID string) error {
		return source.SignInWithCode(context.Background(), cfg.Server, otpID, strings.TrimSpace(code))
	})
}

// ResetPassword sets a new password (at least 8 characters) with the code of the reset step and
// continues as SignIn does, answering {"state":"linked"} or {"state":"choose"}. An old, wrong or
// unknown code fails as SignInWithCode's does ("expired", "wrong_code"); a short password with
// failure "weak_password".
func (c *Core) ResetPassword(code, password, deviceName string) (string, error) {
	return c.finishWithCode(code, deviceName, func(source codeAccountSource, cfg syncclient.Config, otpID string) error {
		return source.ResetPassword(context.Background(), cfg.Server, otpID, strings.TrimSpace(code), password)
	})
}

// codeAccountSource is the source of a code sign-in: it signs in and takes a code.
type codeAccountSource interface {
	syncclient.AccountSource
	syncclient.CodeSignIn
}

// finishWithCode is SignInWithCode and ResetPassword: check how old the code is, run check, and
// link the phone.
func (c *Core) finishWithCode(code, deviceName string, check func(codeAccountSource, syncclient.Config, string) error) (string, error) {
	if c.secrets == nil {
		return "", errors.New("this phone has no secure store for the sign-in")
	}
	c.mu.Lock()
	step, email, otpID, sentAt := c.code.step, c.code.email, c.code.otpID, c.code.sentAt
	c.mu.Unlock()
	if otpID == "" {
		return "", errors.New("ask for a code first")
	}
	if c.nowTime().Sub(sentAt) > data.CodeValidFor {
		if err := c.sendCode(email, true, step); err != nil {
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
	source, ok := c.accountSource(cfg).(codeAccountSource)
	if !ok {
		return "", c.codeFailed(syncclient.ErrSignInUnsupported)
	}
	if err := check(source, cfg, otpID); err != nil {
		if errors.Is(err, syncclient.ErrInvalidCode) {
			c.noteCodeFailure(err)
			return "", errors.New("that code isn't right")
		}
		return "", c.codeFailed(err)
	}
	return c.finishSignIn(cfg, source, deviceName)
}

// codeFailed records why an attempt failed and returns the sentence for it.
func (c *Core) codeFailed(err error) error {
	c.noteCodeFailure(err)
	switch {
	case errors.Is(err, syncclient.ErrInvalidEmail):
		return errors.New("that email address isn't valid")
	case errors.Is(err, syncclient.ErrEmailTaken):
		return errors.New("this email already has an account")
	case errors.Is(err, syncclient.ErrWeakPassword):
		return errors.New("the password needs at least 8 characters")
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
	case errors.Is(err, syncclient.ErrEmailTaken):
		return failureEmailTaken
	case errors.Is(err, syncclient.ErrInvalidEmail):
		return failureInvalidEmail
	case errors.Is(err, syncclient.ErrWeakPassword):
		return failureWeakPassword
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
