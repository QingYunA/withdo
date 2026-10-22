package syncclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// pocketBaseRefreshWindow is how close to its expiry a PocketBase token is replaced. Its tokens
// last seven days by default, so a device that syncs at least once a day never signs in again.
const pocketBaseRefreshWindow = 24 * time.Hour

// pocketBaseTokenKey is the Secrets key of the PocketBase auth token.
const pocketBaseTokenKey = "sync/pocketbase-token"

// PocketBaseSource signs in to the sync server's PocketBase with an email and password
// (POST /api/collections/users/auth-with-password) and supplies its auth token to the sync client.
// The token lives in Secrets, per database as the OIDC refresh token does, and is renewed with
// auth-refresh when less than a day is left. PocketBase keeps old tokens valid after a refresh, so
// the app and adhd-core can share a database's token without racing.
type PocketBaseSource struct {
	// Server is the sync server's base URL, which also serves PocketBase's API.
	Server  string
	Secrets Secrets
	HTTP    *http.Client
	// DBPath keys the token to one database; empty keeps one fixed key.
	DBPath string

	mu         sync.Mutex
	token      string
	account    Account
	hasAccount bool
}

// NewPocketBaseSource returns a source for the sync server at server. It does not touch the
// network.
func NewPocketBaseSource(server string, secrets Secrets) *PocketBaseSource {
	return &PocketBaseSource{Server: server, Secrets: secrets}
}

// pocketBaseAnswer is the part of an auth answer the client uses.
type pocketBaseAnswer struct {
	Token  string `json:"token"`
	Record struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"record"`
}

// SignIn signs in with an email and password and stores the token. The password is not stored or
// put into an error.
func (s *PocketBaseSource) SignIn(ctx context.Context, server, loginName, password string) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: PocketBase sign-in needs a secrets store")
	}
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return errors.New("syncclient: sync server URL is required")
	}
	body, err := json.Marshal(map[string]string{"identity": loginName, "password": password})
	if err != nil {
		return fmt.Errorf("syncclient: encode the sign-in request: %w", err)
	}
	answer, err := s.post(ctx, base+"/api/collections/users/auth-with-password", "", body)
	if err != nil {
		return err
	}
	if answer.Token == "" {
		return errors.New("syncclient: the sign-in answer is missing a token")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Server = base
	return s.storeLocked(answer)
}

// RequestCode asks the server to mail a 6-digit sign-in code to email; an email with no account
// gets one. It returns the id of the code, which SignInWithCode needs. A malformed email is
// ErrInvalidCode too, as PocketBase answers 400 for it.
func (s *PocketBaseSource) RequestCode(ctx context.Context, server, email string) (string, error) {
	if s == nil {
		return "", errors.New("syncclient: PocketBase sign-in needs a source")
	}
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return "", errors.New("syncclient: sync server URL is required")
	}
	body, err := json.Marshal(map[string]string{"email": strings.TrimSpace(email)})
	if err != nil {
		return "", fmt.Errorf("syncclient: encode the code request: %w", err)
	}
	raw, err := s.send(ctx, base+"/api/collections/users/request-otp", "", body, ErrInvalidCode)
	if err != nil {
		return "", err
	}
	var out struct {
		OTPID string `json:"otpId"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.OTPID == "" {
		return "", errors.New("syncclient: the code request answer is missing an id")
	}
	return out.OTPID, nil
}

// SignInWithCode signs in with the code mailed after RequestCode and stores the token as SignIn
// does. A wrong, used or expired code is ErrInvalidCode.
func (s *PocketBaseSource) SignInWithCode(ctx context.Context, server, otpID, code string) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: PocketBase sign-in needs a secrets store")
	}
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return errors.New("syncclient: sync server URL is required")
	}
	body, err := json.Marshal(map[string]string{"otpId": otpID, "password": strings.TrimSpace(code)})
	if err != nil {
		return fmt.Errorf("syncclient: encode the sign-in request: %w", err)
	}
	answer, err := s.postAuth(ctx, base+"/api/collections/users/auth-with-otp", "", body, ErrInvalidCode)
	if err != nil {
		return err
	}
	if answer.Token == "" {
		return errors.New("syncclient: the sign-in answer is missing a token")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Server = base
	return s.storeLocked(answer)
}

// RequestPasswordReset asks the server to mail a reset link to email. PocketBase answers the same
// whether or not the email has an account.
func (s *PocketBaseSource) RequestPasswordReset(ctx context.Context, server, email string) error {
	if s == nil {
		return errors.New("syncclient: PocketBase sign-in needs a source")
	}
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return errors.New("syncclient: sync server URL is required")
	}
	body, err := json.Marshal(map[string]string{"email": strings.TrimSpace(email)})
	if err != nil {
		return fmt.Errorf("syncclient: encode the reset request: %w", err)
	}
	_, err = s.send(ctx, base+"/api/collections/users/request-password-reset", "", body, ErrInvalidCode)
	return err
}

// Token implements TokenSource: the stored token, renewed when it is close to expiring. A refused
// renewal deletes the token and returns ErrSignedOut; a renewal that fails for another reason
// keeps using the token while it is still valid.
func (s *PocketBaseSource) Token(ctx context.Context) (string, error) {
	if s == nil || s.Secrets == nil {
		return "", ErrSignedOut
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" {
		token, err := s.Secrets.Get(s.key())
		if errors.Is(err, ErrSecretNotFound) || (err == nil && token == "") {
			return "", ErrSignedOut
		}
		if err != nil {
			return "", fmt.Errorf("syncclient: read the PocketBase token: %w", err)
		}
		s.token = token
	}
	expiry, ok := tokenExpiry(s.token)
	if ok && time.Until(expiry) > pocketBaseRefreshWindow {
		return s.token, nil
	}
	answer, err := s.post(ctx, strings.TrimRight(s.Server, "/")+"/api/collections/users/auth-refresh", s.token, nil)
	switch {
	case err == nil && answer.Token != "":
		if err := s.storeLocked(answer); err != nil {
			return "", err
		}
		return s.token, nil
	case errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrAccountLocked):
		s.token = ""
		_ = s.Secrets.Delete(s.key())
		return "", ErrSignedOut
	}
	if ok && time.Until(expiry) > time.Minute {
		return s.token, nil
	}
	if err == nil {
		err = errors.New("the refresh answer is missing a token")
	}
	return "", fmt.Errorf("syncclient: renew the PocketBase token: %w", err)
}

// HasPocketBaseToken reports whether dbPath has a stored PocketBase token.
func HasPocketBaseToken(secrets Secrets, dbPath string) bool {
	if secrets == nil {
		return false
	}
	token, err := secrets.Get(pocketBaseKeyFor(dbPath))
	return err == nil && token != ""
}

// DeletePocketBaseToken removes dbPath's stored PocketBase token, if any.
func DeletePocketBaseToken(secrets Secrets, dbPath string) error {
	if secrets == nil {
		return nil
	}
	return deleteSecret(secrets, pocketBaseKeyFor(dbPath))
}

// pocketBaseKeyFor is the Secrets key of dbPath's PocketBase token, keyed like the refresh token.
func pocketBaseKeyFor(dbPath string) string {
	return pocketBaseTokenKey + strings.TrimPrefix(RefreshTokenKeyFor(dbPath), RefreshTokenKey)
}

// Logout forgets the token and the account.
func (s *PocketBaseSource) Logout() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.token = ""
	s.account, s.hasAccount = Account{}, false
	s.mu.Unlock()
	if s.Secrets == nil {
		return nil
	}
	return deleteSecret(s.Secrets, s.key())
}

// Account returns who this source signed in as, from the record of the last sign-in or renewal.
func (s *PocketBaseSource) Account() (Account, bool) {
	if s == nil {
		return Account{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.account, s.hasAccount
}

// storeLocked keeps a new token and its account. The caller holds s.mu.
func (s *PocketBaseSource) storeLocked(answer pocketBaseAnswer) error {
	if err := s.Secrets.Set(s.key(), answer.Token); err != nil {
		s.token = ""
		return fmt.Errorf("syncclient: store the PocketBase token: %w", err)
	}
	s.token = answer.Token
	account := Account{Name: strings.TrimSpace(answer.Record.Name), Email: strings.TrimSpace(answer.Record.Email)}
	if account.Name != "" || account.Email != "" {
		s.account, s.hasAccount = account, true
	}
	return nil
}

// key is this source's Secrets key.
func (s *PocketBaseSource) key() string {
	return pocketBaseKeyFor(s.DBPath)
}

// post is one PocketBase auth request that answers a token. A 400 is a refused sign-in or token,
// whatever the reason: PocketBase answers a wrong password and an unknown email alike.
func (s *PocketBaseSource) post(ctx context.Context, url, token string, body []byte) (pocketBaseAnswer, error) {
	return s.postAuth(ctx, url, token, body, ErrInvalidCredentials)
}

// postAuth is post with the error a 400 maps to.
func (s *PocketBaseSource) postAuth(ctx context.Context, url, token string, body []byte, bad error) (pocketBaseAnswer, error) {
	raw, err := s.send(ctx, url, token, body, bad)
	if err != nil {
		return pocketBaseAnswer{}, err
	}
	var answer pocketBaseAnswer
	if err := json.Unmarshal(raw, &answer); err != nil {
		return pocketBaseAnswer{}, fmt.Errorf("syncclient: decode the sign-in answer: %w", err)
	}
	return answer, nil
}

// send is one POST to PocketBase; it returns the body of a 200 or 204. A 5xx is retried.
func (s *PocketBaseSource) send(ctx context.Context, url, token string, body []byte, bad error) ([]byte, error) {
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	var out []byte
	err := retryTransient(ctx, func() error {
		out = nil
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return fmt.Errorf("syncclient: read the sign-in answer: %w", err)
		}
		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
			out = raw
			return nil
		case http.StatusBadRequest:
			return bad
		case http.StatusUnauthorized:
			return ErrInvalidCredentials
		case http.StatusForbidden:
			return ErrAccountLocked
		case http.StatusTooManyRequests:
			return ErrTooManyAttempts
		case http.StatusNotFound:
			return ErrSignInUnsupported
		default:
			if resp.StatusCode/100 == 5 {
				return &retryableStatus{status: resp.StatusCode}
			}
			return fmt.Errorf("syncclient: PocketBase answered %d", resp.StatusCode)
		}
	})
	return out, err
}

// tokenExpiry reads the exp claim of a JWT without verifying it; the server checks every token.
func tokenExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp float64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp <= 0 {
		return time.Time{}, false
	}
	return time.Unix(int64(claims.Exp), 0), true
}
