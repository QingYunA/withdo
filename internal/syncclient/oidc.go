package syncclient

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	// LoginTimeout is how long Login waits for the browser when the source does not set one.
	LoginTimeout = 5 * time.Minute
	// tokenRefreshSkew refreshes an access token that expires within this much time.
	tokenRefreshSkew = 60 * time.Second
	// callbackPath is the loopback redirect's path.
	callbackPath = "/callback"
)

// OIDCSource signs in through the system browser and supplies OIDC access tokens to the sync
// client. The refresh token lives in Secrets; the access token only in memory. A zero LoginTimeout
// means five minutes. Scopes are asked for on top of "openid offline_access profile email".
type OIDCSource struct {
	Issuer       string
	ClientID     string
	Scopes       []string
	Secrets      Secrets
	HTTP         *http.Client
	LoginTimeout time.Duration
	// DBPath is the database whose key holds the refresh token. Empty keeps the old fixed key,
	// which is what the package's tests and a caller with no database use.
	DBPath string
	// DoneURL, when set, is where the loopback sends the browser as soon as the code arrives,
	// before the exchange: the phone's app link, which brings the app back to the front. Android
	// cuts a background app's network, so the exchange must not run while the browser is in front.
	DoneURL string

	mu           sync.Mutex
	oauth        *oauth2.Config
	accessToken  string
	accessExpiry time.Time
	account      Account
	hasAccount   bool
}

// NewOIDCSource returns a source for one issuer and client. It does not touch the network; Login
// or Token discovers the issuer's endpoints.
func NewOIDCSource(issuer, clientID string, scopes []string, secrets Secrets) *OIDCSource {
	return &OIDCSource{Issuer: issuer, ClientID: clientID, Scopes: scopes, Secrets: secrets}
}

// Login runs the authorization code flow with PKCE (S256) and a random state. It opens a listener
// on 127.0.0.1 with a random port (RFC 8252), passes the authorization URL to open, and waits for
// the browser's redirect. The listener answers with a short page and stops. Nothing is stored
// unless the exchange succeeds.
func (s *OIDCSource) Login(ctx context.Context, open func(url string) error) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: OIDC sign-in needs a secrets store")
	}
	if open == nil {
		return errors.New("syncclient: OIDC sign-in needs a way to open the browser")
	}
	if strings.TrimSpace(s.Issuer) == "" || strings.TrimSpace(s.ClientID) == "" {
		return errors.New("syncclient: OIDC issuer and client id are required")
	}
	state, err := randomURLToken()
	if err != nil {
		return err
	}
	verifier, err := randomURLToken()
	if err != nil {
		return err
	}
	challenge := pkceChallenge(verifier)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("syncclient: open the sign-in listener: %w", err)
	}
	redirect := "http://" + listener.Addr().String() + callbackPath

	s.mu.Lock()
	base, err := s.oauthConfigLocked(ctx)
	s.mu.Unlock()
	if err != nil {
		_ = listener.Close()
		return err
	}
	oauthConfig := *base
	oauthConfig.RedirectURL = redirect
	authURL := oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	type loginResult struct {
		token *oauth2.Token
		err   error
	}
	results := make(chan loginResult, 1)
	var finishOnce sync.Once
	finish := func(result loginResult) {
		finishOnce.Do(func() { results <- result })
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("state") != state {
			// Not our redirect: any page in the browser can reach a loopback port, so a request
			// with the wrong state is refused without ending the sign-in it would otherwise abort.
			http.Error(w, "sign-in failed: state mismatch", http.StatusBadRequest)
			return
		}
		if msg := query.Get("error"); msg != "" {
			http.Error(w, "sign-in failed", http.StatusBadRequest)
			finish(loginResult{err: fmt.Errorf("syncclient: OIDC sign-in was refused (%s)", msg)})
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(w, "sign-in failed: no code", http.StatusBadRequest)
			finish(loginResult{err: errors.New("syncclient: OIDC sign-in returned no code")})
			return
		}
		// The phone goes back to the app first; the exchange's retries cover the moment until
		// Android gives the app its network back. Errors then reach the app, not the browser.
		answered := false
		if s.DoneURL != "" {
			http.Redirect(w, r, s.DoneURL, http.StatusFound)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			answered = true
		}
		var token *oauth2.Token
		err := retryTransient(ctx, func() error {
			var exchangeErr error
			token, exchangeErr = oauthConfig.Exchange(httpContext(ctx, s.HTTP), code,
				oauth2.SetAuthURLParam("code_verifier", verifier))
			return exchangeErr
		})
		if err != nil {
			if !answered {
				http.Error(w, "sign-in failed", http.StatusBadGateway)
			}
			finish(loginResult{err: fmt.Errorf("syncclient: exchange the OIDC code: %w", err)})
			return
		}
		if err := s.storeToken(token); err != nil {
			if !answered {
				http.Error(w, "sign-in failed", http.StatusInternalServerError)
			}
			finish(loginResult{err: err})
			return
		}
		if !answered {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<!doctype html><title>Signed in</title><p>You can close this tab.</p>")
		}
		finish(loginResult{token: token})
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()

	if err := open(authURL); err != nil {
		_ = server.Close()
		return fmt.Errorf("syncclient: open the sign-in page: %w", err)
	}

	timeout := s.LoginTimeout
	if timeout <= 0 {
		timeout = LoginTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-results:
		_ = server.Close()
		return result.err
	case <-timer.C:
		_ = server.Close()
		return errors.New("syncclient: OIDC sign-in timed out")
	case <-ctx.Done():
		_ = server.Close()
		return ctx.Err()
	}
}

// Token implements TokenSource. It returns the access token while it has more than 60 seconds
// left, and otherwise refreshes it from the stored refresh token. A refused refresh deletes the
// stored token and returns ErrSignedOut.
func (s *OIDCSource) Token(ctx context.Context) (string, error) {
	if s == nil || s.Secrets == nil {
		return "", ErrSignedOut
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.accessToken != "" && (s.accessExpiry.IsZero() || time.Now().Before(s.accessExpiry.Add(-tokenRefreshSkew))) {
		return s.accessToken, nil
	}
	refreshToken, err := s.loadRefreshToken()
	if errors.Is(err, ErrSecretNotFound) || (err == nil && refreshToken == "") {
		return "", ErrSignedOut
	}
	if err != nil {
		return "", fmt.Errorf("syncclient: read the OIDC refresh token: %w", err)
	}
	oauthConfig, err := s.oauthConfigLocked(ctx)
	if err != nil {
		return "", err
	}
	token, err := oauthConfig.TokenSource(httpContext(ctx, s.HTTP),
		&oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil && refreshRefused(err) {
		// The issuer rotates refresh tokens: another source or process (the app and adhd-core share
		// a database's key) may have refreshed first and stored the next one. Use that one before
		// concluding the sign-in is gone.
		if stored, loadErr := s.loadRefreshToken(); loadErr == nil && stored != "" && stored != refreshToken {
			refreshToken = stored
			token, err = oauthConfig.TokenSource(httpContext(ctx, s.HTTP),
				&oauth2.Token{RefreshToken: refreshToken}).Token()
		}
	}
	if err != nil {
		if refreshRefused(err) {
			s.accessToken, s.accessExpiry = "", time.Time{}
			_ = s.deleteRefreshToken()
			return "", ErrSignedOut
		}
		return "", fmt.Errorf("syncclient: refresh the OIDC token: %w", err)
	}
	if err := s.storeTokenLocked(token, refreshToken); err != nil {
		return "", err
	}
	return s.accessToken, nil
}

// Logout forgets the access token, the account and deletes the stored refresh token.
func (s *OIDCSource) Logout() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.accessToken, s.accessExpiry = "", time.Time{}
	s.account, s.hasAccount = Account{}, false
	s.mu.Unlock()
	return s.deleteRefreshToken()
}

// Account returns who this source signed in as, from the ID token of the last sign-in or refresh.
// ok is false before the first token and for a static token, which has no account.
func (s *OIDCSource) Account() (Account, bool) {
	if s == nil {
		return Account{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasAccount {
		return Account{}, false
	}
	return s.account, true
}

// oauthConfigLocked discovers the issuer once. The caller holds s.mu.
func (s *OIDCSource) oauthConfigLocked(ctx context.Context) (*oauth2.Config, error) {
	if s.oauth != nil {
		return s.oauth, nil
	}
	issuer := strings.TrimRight(strings.TrimSpace(s.Issuer), "/")
	if issuer == "" {
		return nil, errors.New("syncclient: OIDC issuer is required")
	}
	discoveryCtx := ctx
	if s.HTTP != nil {
		discoveryCtx = oidc.ClientContext(ctx, s.HTTP)
	}
	provider, err := oidc.NewProvider(discoveryCtx, issuer)
	if err != nil {
		return nil, fmt.Errorf("syncclient: discover OIDC issuer %s: %w", issuer, err)
	}
	// profile and email put the name and email in the ID token, which Settings shows as the account.
	scopes := []string{oidc.ScopeOpenID, oidc.ScopeOfflineAccess, "profile", "email"}
	for _, scope := range s.Scopes {
		if scope = strings.TrimSpace(scope); scope != "" && !slices.Contains(scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	s.oauth = &oauth2.Config{ClientID: s.ClientID, Endpoint: provider.Endpoint(), Scopes: scopes}
	return s.oauth, nil
}

// storeToken remembers a token from Login. The caller does not hold s.mu.
func (s *OIDCSource) storeToken(token *oauth2.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storeTokenLocked(token, "")
}

// storeTokenLocked remembers token. A new refresh token replaces the stored one; without one the
// old refresh token, refreshToken, stays. The caller holds s.mu.
func (s *OIDCSource) storeTokenLocked(token *oauth2.Token, refreshToken string) error {
	nextRefresh := token.RefreshToken
	if nextRefresh == "" {
		nextRefresh = refreshToken
	}
	if nextRefresh != "" && nextRefresh != refreshToken {
		if err := s.storeRefreshToken(nextRefresh); err != nil {
			s.accessToken, s.accessExpiry = "", time.Time{}
			return fmt.Errorf("syncclient: store the OIDC refresh token: %w", err)
		}
	}
	s.accessToken, s.accessExpiry = token.AccessToken, token.Expiry
	// A refresh may omit the ID token; keeping the previous account is what the pane needs.
	if raw, ok := token.Extra("id_token").(string); ok {
		if account, ok := accountFromIDToken(raw); ok {
			s.account, s.hasAccount = account, true
		}
	}
	return nil
}

// loadRefreshToken reads this source's refresh token. With a database it uses that database's key
// and moves an old fixed-key token over; without one it keeps the old key.
func (s *OIDCSource) loadRefreshToken() (string, error) {
	if strings.TrimSpace(s.DBPath) == "" {
		return getSecret(s.Secrets, RefreshTokenKey)
	}
	return LoadRefreshToken(s.Secrets, s.DBPath)
}

// storeRefreshToken writes this source's refresh token under its key.
func (s *OIDCSource) storeRefreshToken(token string) error {
	if strings.TrimSpace(s.DBPath) == "" {
		return s.Secrets.Set(RefreshTokenKey, token)
	}
	return SaveRefreshToken(s.Secrets, s.DBPath, token)
}

// deleteRefreshToken removes this source's refresh token.
func (s *OIDCSource) deleteRefreshToken() error {
	if strings.TrimSpace(s.DBPath) == "" {
		return DeleteRefreshToken(s.Secrets)
	}
	return DeleteRefreshTokenAt(s.Secrets, s.DBPath)
}

// refreshRefused reports whether the token endpoint answered a refresh with a client error, which
// means the refresh token is no longer good.
func refreshRefused(err error) bool {
	var retrieve *oauth2.RetrieveError
	return errors.As(err, &retrieve) && retrieve.Response != nil &&
		retrieve.Response.StatusCode >= 400 && retrieve.Response.StatusCode < 500
}

// httpContext installs client for oauth2's token request, when one is set.
func httpContext(ctx context.Context, client *http.Client) context.Context {
	if client == nil {
		return ctx
	}
	return context.WithValue(ctx, oauth2.HTTPClient, client)
}

// randomURLToken returns 32 random bytes as URL-safe base64 without padding.
func randomURLToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("syncclient: read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// pkceChallenge is BASE64URL(SHA256(verifier)), the S256 code challenge.
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
