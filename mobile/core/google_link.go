package core

import (
	"context"
	"errors"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// The errors GoogleLinkFinish returns. The text is the kind, so Kotlin words it in the user's
// language without matching sentences.
const (
	linkAlreadyLinked = "already_linked" // the Google account belongs to another account
	linkCancelled     = "cancelled"
	linkFailed        = "failed"
	linkNetwork       = "network"
	linkSignedOut     = "signed_out"
)

// googleLinker returns the signed-in account's source and server for the Google link calls.
func (c *Core) googleLinker() (syncclient.GoogleLinker, syncclient.Config, error) {
	var cfg syncclient.Config
	if c.secrets == nil {
		return nil, cfg, errors.New(linkSignedOut)
	}
	cfg, found, err := c.loadConfig()
	if err != nil || !found || !cfg.SignsInWithAccount() || !c.hasCredentials(cfg) {
		return nil, cfg, errors.New(linkSignedOut)
	}
	linker, ok := c.accountSource(cfg).(syncclient.GoogleLinker)
	if !ok {
		return nil, cfg, errors.New(linkFailed)
	}
	return linker, cfg, nil
}

// GoogleLinkAvailable reports whether the Sync screen shows the Google row: the phone is signed
// in and the server lists a Google provider. Any failure to ask is false. It reads the network:
// call it off the main thread.
func (c *Core) GoogleLinkAvailable() bool {
	if _, _, err := c.googleLinker(); err != nil {
		return false
	}
	return c.GoogleAvailable()
}

// GoogleLinked reports whether the signed-in account has a Google identity. It reads the network:
// call it off the main thread. The error is a GoogleLinkFinish kind.
func (c *Core) GoogleLinked() (bool, error) {
	linker, cfg, err := c.googleLinker()
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	linked, err := linker.GoogleLinked(ctx, cfg.Server)
	if err != nil {
		return false, linkKind(err)
	}
	return linked, nil
}

// GoogleLinkStart begins "Link Google" for the signed-in account and returns the URL for the
// browser (a Custom Tab). Call GoogleLinkFinish next: it waits for the answer. Starting again
// drops an earlier unfinished link.
func (c *Core) GoogleLinkStart() (string, error) {
	c.dropGoogleLink()
	linker, cfg, err := c.googleLinker()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), syncclient.GoogleTimeout)
	flow, authURL, err := linker.StartGoogleLink(ctx, cfg.Server)
	if err != nil {
		cancel()
		return "", linkKind(err)
	}
	c.mu.Lock()
	c.googleLink = &googleSignIn{flow: flow, ctx: ctx, cancel: cancel, cfg: cfg}
	c.mu.Unlock()
	return authURL, nil
}

// GoogleLinkFinish waits for the browser's answer to GoogleLinkStart and links. It blocks for up
// to five minutes: call it off the main thread. It returns nil once Google is linked. An error's
// text is one of "already_linked" (the Google account is another account's), "cancelled",
// "failed", "network" or "signed_out".
func (c *Core) GoogleLinkFinish() error {
	c.mu.Lock()
	link := c.googleLink
	c.mu.Unlock()
	if link == nil {
		return errors.New(linkFailed)
	}
	defer func() {
		link.cancel()
		c.mu.Lock()
		if c.googleLink == link {
			c.googleLink = nil
		}
		c.mu.Unlock()
	}()
	if err := link.flow.Wait(link.ctx); err != nil {
		return linkKind(err)
	}
	return nil
}

// GoogleLinkCancel ends a GoogleLinkFinish that waits: it returns "cancelled".
func (c *Core) GoogleLinkCancel() {
	c.mu.Lock()
	link := c.googleLink
	c.mu.Unlock()
	if link != nil {
		link.cancel()
	}
}

// dropGoogleLink forgets an unfinished link and closes its connection.
func (c *Core) dropGoogleLink() {
	c.mu.Lock()
	link := c.googleLink
	c.googleLink = nil
	c.mu.Unlock()
	if link != nil {
		link.cancel()
		link.flow.Close()
	}
}

// linkKind maps a link failure to the error GoogleLinkFinish returns.
func linkKind(err error) error {
	switch {
	case errors.Is(err, syncclient.ErrGoogleAlreadyLinked):
		return errors.New(linkAlreadyLinked)
	case errors.Is(err, syncclient.ErrGoogleCancelled):
		return errors.New(linkCancelled)
	case errors.Is(err, syncclient.ErrSignedOut), syncclient.IsUnauthorized(err):
		return errors.New(linkSignedOut)
	case errors.Is(err, syncclient.ErrGoogleFailed):
		return errors.New(linkFailed)
	case isNetworkError(err):
		return errors.New(linkNetwork)
	}
	return errors.New(linkFailed)
}
