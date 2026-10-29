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
// The token lives in Secrets, per database, and is renewed with
// auth-refresh when less than a day is left. PocketBase keeps old tokens valid after a refresh, so
// the app and withdo can share a database's token without racing.
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
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"record"`
}

// SignIn signs in with an email and password and stores the token. The password is not stored or
// put into an error. A wrong password or an unknown email is ErrInvalidCredentials; the right
// password of an account whose email is not verified is ErrEmailNotVerified, and the caller asks
// for a code with RequestCode and verifies with SignInWithCode.
func (s *PocketBaseSource) SignIn(ctx context.Context, server, loginName, password string) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: PocketBase sign-in needs a secrets store")
	}
	base, err := serverBase(server)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"identity": strings.TrimSpace(loginName), "password": password})
	if err != nil {
		return fmt.Errorf("syncclient: encode the sign-in request: %w", err)
	}
	answer, err := s.post(ctx, base+"/api/collections/users/auth-with-password", "", body)
	if err != nil {
		return err
	}
	return s.keep(base, answer)
}

// SignUp makes an account with an email and a password and mails its first code: the account is not
// verified, so it cannot sign in until SignInWithCode is given that code. lang is the app's language
// ("en" or "zh-Hans"), the language of the code mails. It returns the id of the mailed code. A taken
// email is ErrEmailTaken, a malformed one ErrInvalidEmail, a password under 8 characters
// ErrWeakPassword.
func (s *PocketBaseSource) SignUp(ctx context.Context, server, email, password, lang string) (string, error) {
	if s == nil {
		return "", errors.New("syncclient: PocketBase sign-in needs a source")
	}
	base, err := serverBase(server)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]string{
		"email": strings.TrimSpace(email), "password": password, "passwordConfirm": password, "lang": lang,
	})
	if err != nil {
		return "", fmt.Errorf("syncclient: encode the sign-up request: %w", err)
	}
	if _, err := s.sendWith(ctx, base+"/api/collections/users/records", "", body, signUpError); err != nil {
		return "", err
	}
	return s.RequestCode(ctx, base, email)
}

// RequestCode asks the server to mail a 6-digit code to email, for the account's verification or a
// password reset. An email with no account gets the same answer and no mail (the server never says
// whether an account exists). It returns the id of the code, which SignInWithCode and ResetPassword
// need. A malformed email is ErrInvalidEmail.
func (s *PocketBaseSource) RequestCode(ctx context.Context, server, email string) (string, error) {
	if s == nil {
		return "", errors.New("syncclient: PocketBase sign-in needs a source")
	}
	base, err := serverBase(server)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]string{"email": strings.TrimSpace(email)})
	if err != nil {
		return "", fmt.Errorf("syncclient: encode the code request: %w", err)
	}
	raw, err := s.send(ctx, base+"/api/collections/users/request-otp", "", body, ErrInvalidEmail)
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

// SignInWithCode checks the code mailed after SignUp or RequestCode: it verifies the account's email
// and signs in, storing the token as SignIn does. A wrong, used or expired code is ErrInvalidCode.
func (s *PocketBaseSource) SignInWithCode(ctx context.Context, server, otpID, code string) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: PocketBase sign-in needs a secrets store")
	}
	base, err := serverBase(server)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"otpId": otpID, "password": strings.TrimSpace(code)})
	if err != nil {
		return fmt.Errorf("syncclient: encode the sign-in request: %w", err)
	}
	answer, err := s.postAuth(ctx, base+"/api/collections/users/auth-with-otp", "", body, ErrInvalidCode)
	if err != nil {
		return err
	}
	return s.keep(base, answer)
}

// ResetPassword sets a new password with the code mailed after RequestCode (POST
// /api/withdo/reset-password), verifies the email and signs in. A wrong, used or expired code, and
// the code of an email with no account, are all ErrInvalidCode; a password under 8 characters is
// ErrWeakPassword.
func (s *PocketBaseSource) ResetPassword(ctx context.Context, server, otpID, code, password string) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: PocketBase sign-in needs a secrets store")
	}
	base, err := serverBase(server)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{
		"otpId": otpID, "code": strings.TrimSpace(code), "password": password, "passwordConfirm": password,
	})
	if err != nil {
		return fmt.Errorf("syncclient: encode the reset request: %w", err)
	}
	raw, err := s.sendWith(ctx, base+"/api/withdo/reset-password", "", body, resetError)
	if err != nil {
		return err
	}
	var answer pocketBaseAnswer
	if err := json.Unmarshal(raw, &answer); err != nil {
		return fmt.Errorf("syncclient: decode the reset answer: %w", err)
	}
	return s.keep(base, answer)
}

// serverBase is the server URL without spaces and a trailing slash.
func serverBase(server string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return "", errors.New("syncclient: sync server URL is required")
	}
	return base, nil
}

// keep stores the token of a sign-in answer for the server.
func (s *PocketBaseSource) keep(base string, answer pocketBaseAnswer) error {
	if answer.Token == "" {
		return errors.New("syncclient: the sign-in answer is missing a token")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Server = base
	return s.storeLocked(answer)
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

// send is one POST to PocketBase; it returns the body of a 200 or 204. A 5xx is retried. A 400 is
// bad.
func (s *PocketBaseSource) send(ctx context.Context, url, token string, body []byte, bad error) ([]byte, error) {
	return s.sendWith(ctx, url, token, body, func(status int, _ pbErrors) error {
		if status == http.StatusBadRequest {
			return bad
		}
		return nil
	})
}

// pbErrors is the `data` of a PocketBase error answer: for each field the code of what is wrong
// with it, e.g. {"email": {"code": "validation_not_unique"}}.
type pbErrors map[string]struct {
	Code string `json:"code"`
}

// code is the error code of one field, "" when it has none.
func (e pbErrors) code(field string) string { return e[field].Code }

// errorCodeEmailNotVerified is what pbauth answers (403, data.email.code) for the right password of
// an account whose email is not verified.
const errorCodeEmailNotVerified = "email_not_verified"

// statusMapper maps an error status and the answer's field errors to the error to return. nil leaves
// the status to the default mapping.
type statusMapper func(status int, errs pbErrors) error

// signUpError maps the answers of POST /api/collections/users/records.
func signUpError(status int, errs pbErrors) error {
	if status != http.StatusBadRequest {
		return nil
	}
	switch {
	case errs.code("email") == "validation_not_unique":
		return ErrEmailTaken
	case errs.code("email") != "":
		return ErrInvalidEmail
	case errs.code("password") != "":
		return ErrWeakPassword
	}
	return fmt.Errorf("syncclient: the server refused the sign-up (status %d)", status)
}

// resetError maps the answers of POST /api/withdo/reset-password.
func resetError(status int, errs pbErrors) error {
	if status != http.StatusBadRequest {
		return nil
	}
	if errs.code("password") != "" || errs.code("passwordConfirm") != "" {
		return ErrWeakPassword
	}
	return ErrInvalidCode
}

// sendWith is send with a mapper for the error statuses.
func (s *PocketBaseSource) sendWith(ctx context.Context, url, token string, body []byte, mapStatus statusMapper) ([]byte, error) {
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
		}
		var errs pbErrors
		var answer struct {
			Data pbErrors `json:"data"`
		}
		if json.Unmarshal(raw, &answer) == nil {
			errs = answer.Data
		}
		if err := mapStatus(resp.StatusCode, errs); err != nil {
			return err
		}
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized:
			return ErrInvalidCredentials
		case http.StatusForbidden:
			if errs.code("email") == errorCodeEmailNotVerified {
				return ErrEmailNotVerified
			}
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
