package syncclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

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
	// ErrInvalidCode says the emailed sign-in code was wrong, used, expired, or the email malformed.
	ErrInvalidCode = errors.New("syncclient: invalid or expired sign-in code")
	// ErrSignInUnsupported says the sync server has no POST /v1/sign-in.
	ErrSignInUnsupported = errors.New("syncclient: the sync server does not support in-app sign-in")
)

// SignIn asks the sync server to sign in with an account and password. The server does the whole
// OIDC flow and answers with tokens; only the refresh token is stored, as after a browser sign-in.
// The password is not stored or put into an error.
func (s *OIDCSource) SignIn(ctx context.Context, server, loginName, password string) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: OIDC sign-in needs a secrets store")
	}
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return errors.New("syncclient: sync server URL is required")
	}
	answer, err := s.postSignIn(ctx, base, loginName, password)
	if err != nil {
		return err
	}
	if answer.AccessToken == "" || answer.RefreshToken == "" {
		return errors.New("syncclient: the sign-in answer is missing a token")
	}
	token := (&oauth2.Token{
		AccessToken:  answer.AccessToken,
		RefreshToken: answer.RefreshToken,
		TokenType:    answer.TokenType,
		Expiry:       time.Now().Add(time.Duration(answer.ExpiresIn) * time.Second),
	}).WithExtra(map[string]any{"id_token": answer.IDToken})
	return s.storeToken(token)
}

// postSignIn is one POST /v1/sign-in. Only the statuses the client UI understands become sentinel
// errors; a 5xx is retried, a refusal is not.
func (s *OIDCSource) postSignIn(ctx context.Context, server, loginName, password string) (syncproto.SignInTokenResponse, error) {
	body, err := json.Marshal(syncproto.SignInRequest{LoginName: loginName, Password: password})
	if err != nil {
		return syncproto.SignInTokenResponse{}, fmt.Errorf("syncclient: encode the sign-in request: %w", err)
	}
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	var answer syncproto.SignInTokenResponse
	err = retryTransient(ctx, func() error {
		answer = syncproto.SignInTokenResponse{}
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
			if err := json.Unmarshal(raw, &answer); err != nil {
				return fmt.Errorf("syncclient: decode the sign-in answer: %w", err)
			}
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
			if resp.StatusCode/100 == 5 {
				return &retryableStatus{status: resp.StatusCode}
			}
			return fmt.Errorf("syncclient: sign-in answered %d", resp.StatusCode)
		}
	})
	return answer, err
}
