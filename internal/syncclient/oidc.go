package syncclient

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
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

// OIDCSource supplies OIDC access tokens to the sync client. Login signs in through the system
// browser; SignIn signs in inside the app with an account and password. The refresh token lives in
// Secrets; the access token only in memory. A zero LoginTimeout means five minutes. Scopes are
// asked for on top of "openid offline_access profile email".
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

// NewOIDCSource returns a source for one issuer and client. It does not touch the network; Login,
// SignIn or Token discovers the issuer's endpoints.
func NewOIDCSource(issuer, clientID string, scopes []string, secrets Secrets) *OIDCSource {
	return &OIDCSource{Issuer: issuer, ClientID: clientID, Scopes: scopes, Secrets: secrets}
}

// SignIn runs the same authorization code flow with PKCE as Login, but without a browser. It asks
// the issuer for a pending auth request, has server check the account and password through
// Zitadel, and finishes the exchange with its own verifier. The password is only sent to server;
// it is not stored or put into an error.
func (s *OIDCSource) SignIn(ctx context.Context, server, loginName, password string) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: OIDC sign-in needs a secrets store")
	}
	if strings.TrimSpace(s.Issuer) == "" || strings.TrimSpace(s.ClientID) == "" {
		return errors.New("syncclient: OIDC issuer and client id are required")
	}
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return errors.New("syncclient: sync server URL is required")
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

	s.mu.Lock()
	discovered, err := s.oauthConfigLocked(ctx)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	oauthConfig := *discovered
	oauthConfig.RedirectURL = signInRedirect
	authURL := oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	authRequest, err := s.signInAuthRequest(ctx, authURL)
	if err != nil {
		return err
	}
	callbackURL, err := s.postSignIn(ctx, base, authRequest, loginName, password)
	if err != nil {
		return err
	}
	code, err := signInCode(callbackURL, state)
	if err != nil {
		return err
	}

	var token *oauth2.Token
	err = retryTransient(ctx, func() error {
		var exchangeErr error
		token, exchangeErr = oauthConfig.Exchange(httpContext(ctx, s.HTTP), code,
			oauth2.SetAuthURLParam("code_verifier", verifier))
		return exchangeErr
	})
	if err != nil {
		return fmt.Errorf("syncclient: exchange the OIDC code: %w", err)
	}
	return s.storeToken(token)
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
