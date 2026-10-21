package syncserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// authRequestIDPattern is the shape of an auth request id Zitadel hands out.
var authRequestIDPattern = regexp.MustCompile(`^V2_[0-9]+$`)

// zitadelLogin talks to Zitadel's Session API with the PAT of a machine user that only has the
// IAM_LOGIN_CLIENT role, and to the ordinary OIDC endpoints on the same base. It is the part of
// Zitadel the sync server drives so a user can sign in inside the app: start the authorization,
// create a session for the password, then complete the auth request and exchange its code.
type zitadelLogin struct {
	base   string
	host   string
	pat    string
	client *http.Client
}

// newZitadelLogin builds the client. internalURL, when set, is where every Zitadel request goes
// (for example http://proxy on the Zitadel Docker network); the issuer still names the host the
// request asks for, so Zitadel sees the address its public URLs are built from.
func newZitadelLogin(issuer, pat, internalURL string) *zitadelLogin {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	base, host := issuer, ""
	if internal := strings.TrimRight(strings.TrimSpace(internalURL), "/"); internal != "" {
		base = internal
		if parsed, err := url.Parse(issuer); err == nil {
			host = parsed.Host
		}
	}
	return &zitadelLogin{
		base:   base,
		host:   host,
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

// authorize starts an OIDC authorization request and returns the auth request id Zitadel put in
// the login redirect. The redirect is not followed; only its query is read.
func (z *zitadelLogin) authorize(ctx context.Context, clientID, redirectURI string, scopes []string, challenge, state string) (string, error) {
	endpoint, err := url.Parse(z.base + "/oauth/v2/authorize")
	if err != nil {
		return "", fmt.Errorf("build the authorize URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(scopes, " "))
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	query.Set("state", state)
	endpoint.RawQuery = query.Encode()

	client := *z.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	z.setHost(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET authorize: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 3 {
		return "", fmt.Errorf("authorize answered %d", resp.StatusCode)
	}
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return "", fmt.Errorf("parse the authorize location: %w", err)
	}
	id := location.Query().Get("authRequest")
	if !authRequestIDPattern.MatchString(id) {
		return "", errors.New("authorize returned no auth request")
	}
	return id, nil
}

// exchangeCode redeems the authorization code with the PKCE verifier and returns the tokens. It
// carries no PAT: the code and verifier are the whole proof.
func (z *zitadelLogin) exchangeCode(ctx context.Context, clientID, code, verifier string) (syncproto.SignInTokenResponse, error) {
	var out syncproto.SignInTokenResponse
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {loopbackRedirectURI},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		z.base+"/oauth/v2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	z.setHost(req)
	resp, err := z.client.Do(req)
	if err != nil {
		return out, fmt.Errorf("POST token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out, fmt.Errorf("read the token answer: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return out, fmt.Errorf("token answered %d", resp.StatusCode)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode the token answer: %w", err)
	}
	if out.AccessToken == "" {
		return out, errors.New("the token answer has no access token")
	}
	return out, nil
}

// setHost makes an internal-base request ask for the issuer's host, so Zitadel sees the public name
// its URLs are built from.
func (z *zitadelLogin) setHost(req *http.Request) {
	if z.host != "" {
		req.Host = z.host
	}
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
	z.setHost(req)
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
