package syncclient

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrGoogleCancelled says the Google sign-in was cancelled by the caller.
	ErrGoogleCancelled = errors.New("syncclient: Google sign-in was cancelled")
	// ErrGoogleFailed says the Google sign-in did not finish: it timed out, Google or the server
	// refused it, or the server has no Google provider.
	ErrGoogleFailed = errors.New("syncclient: Google sign-in failed")
)

// GoogleTimeout is how long a Google sign-in waits for the person to finish in the browser.
const GoogleTimeout = 5 * time.Minute

const (
	googleProvider    = "google"
	oauth2Topic       = "@oauth2"
	realtimeConnected = "PB_CONNECT"
)

// GoogleSignIn is "Continue with Google" of a PocketBase server (M13 P3). PocketBaseSource
// implements it; a caller holding an AccountSource type-asserts for it.
type GoogleSignIn interface {
	// GoogleAvailable reports whether the server lists a Google provider in its auth methods.
	GoogleAvailable(ctx context.Context, server string) (bool, error)
	// SignInWithGoogle runs the whole flow: open is called with the URL for the system browser,
	// then it waits up to GoogleTimeout for the result and stores the token.
	SignInWithGoogle(ctx context.Context, server string, open func(url string) error) error
	// StartGoogle starts the flow and returns it with the URL to open; Wait finishes it. It is
	// the two-step form for callers that open the browser themselves (the phone).
	StartGoogle(ctx context.Context, server string) (*GoogleFlow, string, error)
}

var _ GoogleSignIn = (*PocketBaseSource)(nil)

// authMethods is the part of GET /api/collections/users/auth-methods the client reads.
type authMethods struct {
	OAuth2 struct {
		Enabled   bool `json:"enabled"`
		Providers []struct {
			Name         string `json:"name"`
			State        string `json:"state"`
			AuthURL      string `json:"authURL"`
			CodeVerifier string `json:"codeVerifier"`
		} `json:"providers"`
	} `json:"oauth2"`
}

// GoogleAvailable reads the auth methods of the server's users collection.
func (s *PocketBaseSource) GoogleAvailable(ctx context.Context, server string) (bool, error) {
	methods, err := s.authMethods(ctx, strings.TrimRight(strings.TrimSpace(server), "/"))
	if err != nil {
		return false, err
	}
	return methods.OAuth2.Enabled && methods.google() >= 0, nil
}

// google is the index of the Google provider, or -1.
func (m authMethods) google() int {
	for i, p := range m.OAuth2.Providers {
		if p.Name == googleProvider && p.AuthURL != "" {
			return i
		}
	}
	return -1
}

func (s *PocketBaseSource) httpClient() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return http.DefaultClient
}

func (s *PocketBaseSource) authMethods(ctx context.Context, base string) (authMethods, error) {
	var methods authMethods
	if base == "" {
		return methods, errors.New("syncclient: sync server URL is required")
	}
	err := retryTransient(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/collections/users/auth-methods", nil)
		if err != nil {
			return err
		}
		resp, err := s.httpClient().Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		switch {
		case resp.StatusCode == http.StatusOK:
			return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&methods)
		case resp.StatusCode == http.StatusNotFound:
			return ErrSignInUnsupported
		case resp.StatusCode/100 == 5:
			return &retryableStatus{status: resp.StatusCode}
		}
		return fmt.Errorf("syncclient: PocketBase answered %d", resp.StatusCode)
	})
	return methods, err
}

// SignInWithGoogle is StartGoogle, open and Wait with GoogleTimeout over all of it. A cancelled
// ctx is ErrGoogleCancelled; a timeout, a refusal or a server without Google is ErrGoogleFailed.
func (s *PocketBaseSource) SignInWithGoogle(ctx context.Context, server string, open func(url string) error) error {
	ctx, cancel := context.WithTimeout(ctx, GoogleTimeout)
	defer cancel()
	flow, authURL, err := s.StartGoogle(ctx, server)
	if err != nil {
		return err
	}
	defer flow.Close()
	if err := open(authURL); err != nil {
		return fmt.Errorf("syncclient: open the browser: %w", err)
	}
	return flow.Wait(ctx)
}

// GoogleFlow is a Google sign-in that was started: the realtime connection on which the server
// will deliver Google's answer. Close it when done; Wait closes it too.
type GoogleFlow struct {
	s           *PocketBaseSource
	base        string
	redirectURL string
	verifier    string
	clientID    string
	events      chan oauth2Event
	cancel      context.CancelFunc
}

type oauth2Event struct {
	State string `json:"state"`
	Code  string `json:"code"`
	Error string `json:"error"`
	err   error
}

// StartGoogle follows PocketBase's own JS SDK (authWithOAuth2): connect to /api/realtime and read
// the client id from PB_CONNECT, subscribe that client to "@oauth2", take Google's auth URL from
// auth-methods, and build the URL to open from it: redirect_uri is the server's
// /api/oauth2-redirect and state is the realtime client id, which is how the server's redirect page
// finds this connection. ctx bounds the whole flow, the wait in Wait included.
func (s *PocketBaseSource) StartGoogle(ctx context.Context, server string) (*GoogleFlow, string, error) {
	if s == nil || s.Secrets == nil {
		return nil, "", errors.New("syncclient: PocketBase sign-in needs a secrets store")
	}
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return nil, "", errors.New("syncclient: sync server URL is required")
	}
	methods, err := s.authMethods(ctx, base)
	if err != nil {
		return nil, "", googleError(ctx, err)
	}
	i := methods.google()
	if !methods.OAuth2.Enabled || i < 0 {
		return nil, "", fmt.Errorf("%w: the server has no Google provider", ErrGoogleFailed)
	}
	provider := methods.OAuth2.Providers[i]

	flowCtx, cancel := context.WithCancel(ctx)
	flow := &GoogleFlow{
		s: s, base: base, redirectURL: base + "/api/oauth2-redirect", verifier: provider.CodeVerifier,
		events: make(chan oauth2Event, 4), cancel: cancel,
	}
	if err := flow.connect(flowCtx); err != nil {
		cancel()
		return nil, "", googleError(ctx, err)
	}
	authURL, err := url.Parse(provider.AuthURL)
	if err != nil {
		cancel()
		return nil, "", fmt.Errorf("%w: the provider URL is malformed", ErrGoogleFailed)
	}
	q := authURL.Query()
	q.Set("state", flow.clientID)
	q.Set("redirect_uri", flow.redirectURL)
	authURL.RawQuery = q.Encode()
	return flow, authURL.String(), nil
}

// connect opens the event stream, reads the client id and subscribes to "@oauth2". The stream is
// then read in the background until ctx ends.
func (f *GoogleFlow) connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.base+"/api/realtime", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := f.s.httpClient().Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return ErrSignInUnsupported
		}
		return fmt.Errorf("syncclient: PocketBase answered %d", resp.StatusCode)
	}
	reader := bufio.NewReader(resp.Body)
	name, data, err := readEvent(reader)
	if err != nil {
		_ = resp.Body.Close()
		return fmt.Errorf("syncclient: read the realtime connection: %w", err)
	}
	var connect struct {
		ClientID string `json:"clientId"`
	}
	if name != realtimeConnected || json.Unmarshal([]byte(data), &connect) != nil || connect.ClientID == "" {
		_ = resp.Body.Close()
		return fmt.Errorf("%w: the realtime connection gave no client id", ErrGoogleFailed)
	}
	f.clientID = connect.ClientID
	go func() {
		defer func() { _ = resp.Body.Close() }()
		for {
			name, data, err := readEvent(reader)
			if err != nil {
				f.events <- oauth2Event{err: err}
				return
			}
			if name != oauth2Topic {
				continue
			}
			var ev oauth2Event
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				ev = oauth2Event{err: err}
			}
			f.events <- ev
			return
		}
	}()
	body, _ := json.Marshal(map[string]any{"clientId": f.clientID, "subscriptions": []string{oauth2Topic}})
	_, err = f.s.send(ctx, f.base+"/api/realtime", "", body, ErrGoogleFailed)
	return err
}

// Wait blocks until the browser's answer arrives, exchanges its code for the account's token with
// auth-with-oauth2 and stores it as SignIn does. It closes the flow.
func (f *GoogleFlow) Wait(ctx context.Context) error {
	defer f.Close()
	var ev oauth2Event
	select {
	case <-ctx.Done():
		return googleError(ctx, ctx.Err())
	case ev = <-f.events:
	}
	switch {
	case errors.Is(ev.err, context.Canceled):
		return ErrGoogleCancelled
	case ev.err != nil:
		return googleError(ctx, ev.err)
	case ev.State != f.clientID, ev.Error != "", ev.Code == "":
		return fmt.Errorf("%w: Google or the server refused it", ErrGoogleFailed)
	}
	body, err := json.Marshal(map[string]string{
		"provider": googleProvider, "code": ev.Code, "codeVerifier": f.verifier, "redirectURL": f.redirectURL,
	})
	if err != nil {
		return err
	}
	answer, err := f.s.postAuth(ctx, f.base+"/api/collections/users/auth-with-oauth2", "", body, ErrGoogleFailed)
	if err != nil {
		return googleError(ctx, err)
	}
	if answer.Token == "" {
		return fmt.Errorf("%w: the sign-in answer is missing a token", ErrGoogleFailed)
	}
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	f.s.Server = f.base
	return f.s.storeLocked(answer)
}

// Close drops the realtime connection. It is safe to call twice.
func (f *GoogleFlow) Close() {
	if f != nil {
		f.cancel()
	}
}

// googleError maps what ended a step: a cancelled ctx is ErrGoogleCancelled, an expired one
// ErrGoogleFailed, anything else is returned as it is.
func googleError(ctx context.Context, err error) error {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return ErrGoogleCancelled
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("%w: it timed out", ErrGoogleFailed)
	}
	return err
}

// readEvent reads one server-sent event: its name and data. Comment and id lines are skipped.
func readEvent(r *bufio.Reader) (name, data string, err error) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return "", "", err
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if name != "" {
				return name, data, nil
			}
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}
