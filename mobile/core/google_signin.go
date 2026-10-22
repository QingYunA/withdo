package core

import (
	"context"
	"errors"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// The failure kinds of a Google sign-in, as CodeState reports them.
const (
	failureGoogleCancelled = "google_cancelled"
	failureGoogleFailed    = "google_failed"
)

// googleSignIn is a Google sign-in between GoogleStart and GoogleFinish: the flow holding the
// realtime connection, its context (cancelled by GoogleCancel or after five minutes) and the
// settings it started with. Core keeps it under mu.
type googleSignIn struct {
	flow   *syncclient.GoogleFlow
	ctx    context.Context
	cancel context.CancelFunc
	cfg    syncclient.Config
}

// GoogleStart begins "Continue with Google" (M13 P3) and returns the URL for the browser (a Custom
// Tab). It connects to the server's realtime stream and keeps it, so call GoogleFinish next:
// that waits for the answer. Starting again drops an earlier unfinished sign-in.
func (c *Core) GoogleStart() (string, error) {
	if c.secrets == nil {
		return "", errors.New("this phone has no secure store for the sign-in")
	}
	c.dropGoogle()
	c.noteCodeFailure(nil)
	cfg, err := c.signInConfig()
	if err != nil {
		return "", c.codeFailed(err)
	}
	source, ok := c.accountSource(cfg).(syncclient.GoogleSignIn)
	if !ok {
		return "", c.codeFailed(syncclient.ErrSignInUnsupported)
	}
	ctx, cancel := context.WithTimeout(context.Background(), syncclient.GoogleTimeout)
	flow, authURL, err := source.StartGoogle(ctx, cfg.Server)
	if err != nil {
		cancel()
		return "", c.codeFailed(err)
	}
	c.mu.Lock()
	c.google = &googleSignIn{flow: flow, ctx: ctx, cancel: cancel, cfg: cfg}
	c.mu.Unlock()
	c.setSignInStage(signInStageGoogle)
	return authURL, nil
}

// GoogleFinish waits for the browser's answer to GoogleStart, then continues as SignIn does,
// answering {"state":"linked"} or {"state":"choose"}. It blocks for up to five minutes, so call it
// off the main thread. GoogleCancel ends the wait; CodeState's failure is then "google_cancelled",
// or "google_failed" when it timed out or Google or the server refused.
func (c *Core) GoogleFinish(deviceName string) (string, error) {
	c.mu.Lock()
	google := c.google
	c.mu.Unlock()
	if google == nil {
		return "", errors.New("start the Google sign-in first")
	}
	// It stays in c.google while it waits, so GoogleCancel can reach it.
	defer func() {
		google.cancel()
		c.mu.Lock()
		if c.google == google {
			c.google = nil
		}
		c.mu.Unlock()
	}()
	c.startLink()
	defer c.finishLink()
	defer c.setSignInStage(signInStageNone)
	if err := google.flow.Wait(google.ctx); err != nil {
		return "", c.codeFailed(err)
	}
	source, ok := c.accountSource(google.cfg).(syncclient.AccountSource)
	if !ok {
		return "", c.codeFailed(syncclient.ErrSignInUnsupported)
	}
	return c.finishSignIn(google.cfg, source, deviceName)
}

// GoogleCancel drops a Google sign-in: a GoogleFinish that waits returns the cancelled failure.
// It does nothing when there is none.
func (c *Core) GoogleCancel() {
	c.mu.Lock()
	google := c.google
	c.mu.Unlock()
	if google != nil {
		google.cancel()
	}
}

// dropGoogle forgets an unfinished Google sign-in and closes its connection.
func (c *Core) dropGoogle() {
	c.mu.Lock()
	google := c.google
	c.google = nil
	c.mu.Unlock()
	if google != nil {
		google.cancel()
		google.flow.Close()
	}
}

// GoogleAvailable reports whether the server lists a Google provider, so the Sync screen shows
// "Continue with Google" only where it works. Any failure to ask is false. It reads the network:
// call it off the main thread.
func (c *Core) GoogleAvailable() bool {
	if c.secrets == nil {
		return false
	}
	cfg, err := c.signInConfig()
	if err != nil {
		return false
	}
	source, ok := c.accountSource(cfg).(syncclient.GoogleSignIn)
	if !ok {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	offered, err := source.GoogleAvailable(ctx, cfg.Server)
	return err == nil && offered
}
