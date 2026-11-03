package syncclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Errors LLMQuota returns for the answers the model UI words for the person. Test them with
// errors.Is.
var (
	// ErrLLMNotSignedIn says there is no signed-in account to ask for: the token source has none,
	// or the server refused the token.
	ErrLLMNotSignedIn = errors.New("syncclient: sign in to use the official model")
	// ErrLLMOff says the server has no official model configured.
	ErrLLMOff = errors.New("syncclient: the official model is off on this server")
	// ErrLLMNetwork says the server could not be reached.
	ErrLLMNetwork = errors.New("syncclient: the server could not be reached")
)

// LLMQuota is the account's turns with the official model today: Limit a day, Left now, and when
// the count starts again.
type LLMQuota struct {
	Limit    int
	Left     int
	ResetsAt time.Time
}

// LLMBaseURL is the API root of the official model, <server>/v1/llm: an OpenAI-compatible client
// pointed at it posts to /chat/completions. Its bearer is BearerToken, asked for on every request
// so a refreshed session counts.
func (c *Client) LLMBaseURL() string { return c.base + "/v1/llm" }

// BearerToken is the signed-in session's token for this request, refreshed the way every other call
// does it. It is empty with no error for a client with no token source. A signed-out session is
// ErrSignedOut (IsSignedOut).
func (c *Client) BearerToken(ctx context.Context) (string, error) { return c.token(ctx) }

// LLMOffered asks GET /v1/config whether the server offers the official model. It needs no session.
// An older server, one that cannot be reached, and one with the model off all say false.
func (c *Client) LLMOffered(ctx context.Context) bool {
	cfg, err := FetchConfig(ctx, c.base, c.http)
	return err == nil && cfg.LLM
}

// LLMQuota reads GET /v1/llm/quota: the turns the account has left today.
func (c *Client) LLMQuota(ctx context.Context) (LLMQuota, error) {
	token, err := c.token(ctx)
	if err != nil {
		if IsSignedOut(err) {
			return LLMQuota{}, ErrLLMNotSignedIn
		}
		return LLMQuota{}, fmt.Errorf("%w: %v", ErrLLMNetwork, err)
	}
	if token == "" {
		return LLMQuota{}, ErrLLMNotSignedIn
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/v1/llm/quota", nil)
	if err != nil {
		return LLMQuota{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return LLMQuota{}, ctx.Err()
		}
		return LLMQuota{}, fmt.Errorf("%w: %v", ErrLLMNetwork, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return LLMQuota{}, ErrLLMNotSignedIn
	case resp.StatusCode == http.StatusServiceUnavailable, resp.StatusCode == http.StatusNotFound: // an older server has no route
		return LLMQuota{}, ErrLLMOff
	case resp.StatusCode/100 != 2:
		return LLMQuota{}, newStatusError(resp.StatusCode, raw)
	}
	var out struct {
		Limit    int       `json:"limit"`
		Left     int       `json:"left"`
		ResetsAt time.Time `json:"resets_at"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.Limit <= 0 {
		return LLMQuota{}, fmt.Errorf("decode /v1/llm/quota: %s", strings.TrimSpace(errorMessage(raw)))
	}
	return LLMQuota{Limit: out.Limit, Left: max(0, out.Left), ResetsAt: out.ResetsAt}, nil
}
