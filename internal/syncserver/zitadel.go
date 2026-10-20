package syncserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// zitadelLogin talks to Zitadel's Session API with the PAT of a machine user that only has the
// IAM_LOGIN_CLIENT role. It is the part of Zitadel the sync server drives so a user can sign in
// inside the app: create a session for the password, then complete the waiting auth request.
type zitadelLogin struct {
	base   string
	pat    string
	client *http.Client
}

func newZitadelLogin(issuer, pat string) *zitadelLogin {
	return &zitadelLogin{
		base:   strings.TrimRight(strings.TrimSpace(issuer), "/"),
		pat:    strings.TrimSpace(pat),
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// zitadelError is a non-2xx answer. code is Zitadel's gRPC status, which callers map to the
// sign-in result; message is kept for debugging and never logged or sent to a client.
type zitadelError struct {
	status  int
	code    int
	message string
}

func (e *zitadelError) Error() string {
	return fmt.Sprintf("zitadel: %s (HTTP %d, code %d)", e.message, e.status, e.code)
}

// authRequest is the pending authorization request the client's authorize call started.
type authRequest struct {
	ID          string   `json:"id"`
	ClientID    string   `json:"clientId"`
	RedirectURI string   `json:"redirectUri"`
	Scope       []string `json:"scope"`
}

// zitadelSession is what POST /v2/sessions returns. The token is needed once more to complete the
// auth request and is deleted with the session afterwards.
type zitadelSession struct {
	SessionID    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`
}

type sessionInfo struct {
	Session struct {
		ID      string `json:"id"`
		Factors struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"factors"`
	} `json:"session"`
}

// getAuthRequest reads a pending authorization request. A missing request is an HTTP 404.
func (z *zitadelLogin) getAuthRequest(ctx context.Context, id string) (authRequest, error) {
	var out struct {
		AuthRequest authRequest `json:"authRequest"`
	}
	err := z.do(ctx, http.MethodGet, "/v2/oidc/auth_requests/"+url.PathEscape(id), nil, &out)
	return out.AuthRequest, err
}

// createSession checks loginName and password. A wrong password, an unknown user and a disabled
// account are all errors; their gRPC codes tell them apart.
func (z *zitadelLogin) createSession(ctx context.Context, loginName, password string) (zitadelSession, error) {
	var out zitadelSession
	err := z.do(ctx, http.MethodPost, "/v2/sessions", map[string]any{
		"checks": map[string]any{
			"user":     map[string]string{"loginName": loginName},
			"password": map[string]string{"password": password},
		},
	}, &out)
	return out, err
}

// sessionUser returns the user id of a session whose password check passed.
func (z *zitadelLogin) sessionUser(ctx context.Context, sessionID string) (string, error) {
	var out sessionInfo
	err := z.do(ctx, http.MethodGet, "/v2/sessions/"+url.PathEscape(sessionID), nil, &out)
	return out.Session.Factors.User.ID, err
}

// authMethods lists every authentication method the user has. The sign-in handler refuses an
// account with anything beyond a password.
func (z *zitadelLogin) authMethods(ctx context.Context, userID string) ([]string, error) {
	var out struct {
		AuthMethodTypes []string `json:"authMethodTypes"`
	}
	err := z.do(ctx, http.MethodGet,
		"/v2/users/"+url.PathEscape(userID)+"/authentication_methods", nil, &out)
	return out.AuthMethodTypes, err
}

// deleteSession removes a session that must not be used, such as one for an account with a second
// factor.
func (z *zitadelLogin) deleteSession(ctx context.Context, sessionID, sessionToken string) error {
	return z.do(ctx, http.MethodDelete, "/v2/sessions/"+url.PathEscape(sessionID),
		map[string]string{"sessionToken": sessionToken}, nil)
}

// completeAuthRequest hands the session to Zitadel and returns the loopback callback URL with the
// authorization code.
func (z *zitadelLogin) completeAuthRequest(ctx context.Context, authRequestID string, sess zitadelSession) (string, error) {
	var out struct {
		CallbackURL string `json:"callbackUrl"`
	}
	err := z.do(ctx, http.MethodPost, "/v2/oidc/auth_requests/"+url.PathEscape(authRequestID),
		map[string]any{
			"session": map[string]string{
				"sessionId":    sess.SessionID,
				"sessionToken": sess.SessionToken,
			},
		}, &out)
	return out.CallbackURL, err
}

// do sends one JSON request with the PAT. Every 2xx is success: creating a session answers 201.
func (z *zitadelLogin) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, z.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+z.pat)
	req.Header.Set("Content-Type", "application/json")
	resp, err := z.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read %s %s: %w", method, path, err)
	}
	if resp.StatusCode/100 != 2 {
		ze := &zitadelError{status: resp.StatusCode}
		var parsed struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &parsed) == nil {
			ze.code, ze.message = parsed.Code, parsed.Message
		}
		return ze
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return nil
}
