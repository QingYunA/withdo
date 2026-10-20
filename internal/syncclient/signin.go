package syncclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// signInRedirect is the fixed loopback redirect registered in Zitadel for in-app sign-in.
const signInRedirect = "http://127.0.0.1/callback"

// Errors SignIn returns for the answers the client UI shows as text. Every other failure is an
// ordinary error with the request that failed.
var (
	// ErrInvalidCredentials says the login name or password was refused.
	ErrInvalidCredentials = errors.New("syncclient: invalid login name or password")
	// ErrAccountLocked says the account is locked or disabled.
	ErrAccountLocked = errors.New("syncclient: the account is locked or disabled")
	// ErrSecondFactor says the account has an authentication method beyond a password.
	ErrSecondFactor = errors.New("syncclient: the account needs a second factor")
	// ErrTooManyAttempts says the server rate-limited the sign-in.
	ErrTooManyAttempts = errors.New("syncclient: too many sign-in attempts")
	// ErrSignInUnsupported says the sync server has no POST /v1/sign-in.
	ErrSignInUnsupported = errors.New("syncclient: the sync server does not support in-app sign-in")
)

// signInAuthRequest asks the issuer's authorize endpoint for a pending auth request without
// following its redirect to the login page.
func (s *OIDCSource) signInAuthRequest(ctx context.Context, authURL string) (string, error) {
	var client http.Client
	if s.HTTP != nil {
		client = *s.HTTP
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	var location *url.URL
	err := retryTransient(ctx, func() error {
		location = nil
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		if resp.StatusCode/100 != 3 {
			return fmt.Errorf("syncclient: OIDC authorize answered %d", resp.StatusCode)
		}
		raw := resp.Header.Get("Location")
		if raw == "" {
			return errors.New("syncclient: OIDC authorize returned no location")
		}
		location, err = url.Parse(raw)
		if err != nil {
			return fmt.Errorf("syncclient: parse the OIDC authorize location: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	query := location.Query()
	if msg := query.Get("error"); msg != "" {
		return "", fmt.Errorf("syncclient: OIDC sign-in was refused (%s)", msg)
	}
	authRequest := query.Get("authRequest")
	if authRequest == "" {
		return "", errors.New("syncclient: OIDC authorize returned no auth request")
	}
	return authRequest, nil
}

// postSignIn has the sync server complete the auth request. Only the five statuses the client UI
// understands become sentinel errors; the answer never contains a token.
func (s *OIDCSource) postSignIn(ctx context.Context, server, authRequest, loginName, password string) (string, error) {
	body, err := json.Marshal(syncproto.SignInRequest{
		AuthRequest: authRequest, LoginName: loginName, Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("syncclient: encode the sign-in request: %w", err)
	}
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	var callbackURL string
	err = retryTransient(ctx, func() error {
		callbackURL = ""
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			server+"/v1/sign-in", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
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
		case http.StatusOK:
			var out syncproto.SignInResponse
			if err := json.Unmarshal(raw, &out); err != nil {
				return fmt.Errorf("syncclient: decode the sign-in answer: %w", err)
			}
			if out.CallbackURL == "" {
				return errors.New("syncclient: the sign-in answer has no callback URL")
			}
			callbackURL = out.CallbackURL
			return nil
		case http.StatusUnauthorized:
			return ErrInvalidCredentials
		case http.StatusLocked:
			return ErrAccountLocked
		case http.StatusForbidden:
			return ErrSecondFactor
		case http.StatusTooManyRequests:
			return ErrTooManyAttempts
		case http.StatusNotFound:
			return ErrSignInUnsupported
		default:
			return fmt.Errorf("syncclient: sign-in answered %d", resp.StatusCode)
		}
	})
	return callbackURL, err
}

// signInCode checks the callback the server returned and pulls out the code. The redirect and its
// state must be the ones this sign-in started with.
func signInCode(callbackURL, state string) (string, error) {
	callback, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("syncclient: parse the sign-in callback: %w", err)
	}
	if callback.Scheme != "http" || callback.Host != "127.0.0.1" || callback.Path != callbackPath {
		return "", errors.New("syncclient: the sign-in callback is not the loopback redirect")
	}
	query := callback.Query()
	if query.Get("state") != state {
		return "", errors.New("syncclient: the sign-in callback state does not match")
	}
	if msg := query.Get("error"); msg != "" {
		return "", fmt.Errorf("syncclient: OIDC sign-in was refused (%s)", msg)
	}
	code := query.Get("code")
	if code == "" {
		return "", errors.New("syncclient: OIDC sign-in returned no code")
	}
	return code, nil
}
