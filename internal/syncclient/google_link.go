package syncclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrGoogleAlreadyLinked says the Google account the person chose already belongs to another
// account of the server. PocketBase does not refuse that: it answers the sign-in of that other
// account. The link flow notices the foreign id and drops the token, so nothing changes here.
var ErrGoogleAlreadyLinked = errors.New("syncclient: this Google account is already linked to another account")

// GoogleLinker is "Link Google" of Settings > Account: add a Google identity to the account that
// is already signed in, so Continue with Google later lands on the same account. PocketBaseSource
// implements it.
//
// PocketBase links when auth-with-oauth2 is called with the signed-in user's Authorization header
// and Google's identity is not linked to any user yet (apis/record_auth_with_oauth2.go: the
// logged-in record is the fallback when no _externalAuths row has this provider id). The user's
// own links are listable under the _externalAuths collection's owner rule.
type GoogleLinker interface {
	// GoogleLinked reports whether the signed-in account has a Google identity.
	GoogleLinked(ctx context.Context, server string) (bool, error)
	// LinkGoogle runs the flow like SignInWithGoogle, but links instead of signing in and stores
	// no token. ErrGoogleAlreadyLinked says the Google account is another account's.
	LinkGoogle(ctx context.Context, server string, open func(url string) error) error
	// StartGoogleLink is the two-step form for callers that open the browser themselves.
	StartGoogleLink(ctx context.Context, server string) (*GoogleFlow, string, error)
}

var _ GoogleLinker = (*PocketBaseSource)(nil)

// GoogleLinked lists the account's own external auths and looks for a Google one.
func (s *PocketBaseSource) GoogleLinked(ctx context.Context, server string) (bool, error) {
	base := strings.TrimRight(strings.TrimSpace(server), "/")
	if base == "" {
		return false, errors.New("syncclient: sync server URL is required")
	}
	token, err := s.Token(ctx)
	if err != nil {
		return false, err
	}
	query := url.Values{}
	query.Set("perPage", "1")
	query.Set("filter", "provider='"+googleProvider+"'")
	query.Set("skipTotal", "1")
	var linked bool
	err = retryTransient(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			base+"/api/collections/_externalAuths/records?"+query.Encode(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", token)
		resp, err := s.httpClient().Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		switch {
		case resp.StatusCode == http.StatusOK:
			var out struct {
				Items []json.RawMessage `json:"items"`
			}
			if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
				return fmt.Errorf("syncclient: decode the linked accounts: %w", err)
			}
			linked = len(out.Items) > 0
			return nil
		case resp.StatusCode == http.StatusUnauthorized:
			return ErrSignedOut
		case resp.StatusCode/100 == 5:
			return &retryableStatus{status: resp.StatusCode}
		}
		return fmt.Errorf("syncclient: PocketBase answered %d", resp.StatusCode)
	})
	return linked, err
}

// LinkGoogle is StartGoogleLink, open and Wait with GoogleTimeout over all of it.
func (s *PocketBaseSource) LinkGoogle(ctx context.Context, server string, open func(url string) error) error {
	ctx, cancel := context.WithTimeout(ctx, GoogleTimeout)
	defer cancel()
	flow, authURL, err := s.StartGoogleLink(ctx, server)
	if err != nil {
		return err
	}
	defer flow.Close()
	if err := open(authURL); err != nil {
		return fmt.Errorf("syncclient: open the browser: %w", err)
	}
	return flow.Wait(ctx)
}

// StartGoogleLink is StartGoogle for a person who is signed in: Wait sends their token with the
// exchange, so PocketBase attaches Google's identity to their record.
func (s *PocketBaseSource) StartGoogleLink(ctx context.Context, server string) (*GoogleFlow, string, error) {
	token, err := s.Token(ctx)
	if err != nil {
		return nil, "", err
	}
	id := tokenRecordID(token)
	if id == "" {
		return nil, "", fmt.Errorf("%w: the sign-in token names no account", ErrGoogleFailed)
	}
	flow, authURL, err := s.StartGoogle(ctx, server)
	if err != nil {
		return nil, "", err
	}
	flow.linkToken, flow.linkID = token, id
	return flow, authURL, nil
}

// finishLink checks the answer of a link exchange. PocketBase answers the sign-in of the Google
// identity's owner when that identity is linked to someone else, so a token for another record
// means "already linked elsewhere". The answer is never stored.
func (f *GoogleFlow) finishLink(answer pocketBaseAnswer) error {
	got := answer.Record.ID
	if got == "" {
		got = tokenRecordID(answer.Token)
	}
	switch {
	case got == "":
		return fmt.Errorf("%w: the answer names no account", ErrGoogleFailed)
	case got != f.linkID:
		return ErrGoogleAlreadyLinked
	}
	return nil
}

// tokenRecordID reads the record id claim of a PocketBase JWT without verifying it.
func tokenRecordID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &claims) != nil {
		return ""
	}
	return claims.ID
}
