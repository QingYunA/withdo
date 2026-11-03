package providers

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// The official model (M15): the sync server's POST /v1/llm/chat/completions, an OpenAI-compatible
// route that holds the upstream key and counts turns per account per day. This file is everything a
// client needs to use it; it knows nothing of the sync client, so the desktop and the phone's core
// both import it.

// HeaderTurn names the turn a request belongs to. The server counts distinct ids per account per
// day, so every request of one turn carries the same one.
const HeaderTurn = "X-Withdo-Turn"

// OfficialModelName is the model name the client sends. The server replaces it with its own.
const OfficialModelName = "withdo"

// officialTimeout bounds one request to the server, which gives its upstream 90 s.
const officialTimeout = 95 * time.Second

// Failures of the official model that the UI words for the person. Test them with errors.Is, or
// ask Classify.
var (
	// ErrOfficialQuota says today's turns are used up (429 llm_quota).
	ErrOfficialQuota = errors.New("the official model's limit for today is reached (llm_quota)")
	// ErrOfficialOff says the server has no official model (503 llm_disabled).
	ErrOfficialOff = errors.New("the official model is off on this server (llm_disabled)")
	// ErrOfficialSignedOut says there is no session to ask with: signed out, or the server
	// refused the token.
	ErrOfficialSignedOut = errors.New("sign in to use the official model (llm_unauthenticated)")
)

// Official is where the official model is and how to sign a request to it: the server's /v1/llm
// root and the session's bearer token, asked for on every request so a refreshed one counts.
type Official struct {
	// BaseURL is <server>/v1/llm.
	BaseURL string
	// Token returns the session's bearer token. It may refresh it; an error that says the session
	// is signed out becomes ErrOfficialSignedOut.
	Token func(ctx context.Context) (string, error)
	// SignedOut tells whether a Token error means "no session"; nil treats every error as a network
	// failure.
	SignedOut func(error) bool
	// HTTP is the transport to the server; nil uses a default client.
	HTTP *http.Client
}

// NewModel builds the chat model of one turn: the OpenAI protocol at BaseURL, with the session as
// the bearer and a fresh turn id on every request it makes. The agent asks for its model once per
// Ask, so one Ask is one turn however many model calls it takes; build a new model for the next one.
func (o Official) NewModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	if strings.TrimSpace(o.BaseURL) == "" || o.Token == nil {
		return nil, ErrNotConfigured
	}
	base := http.DefaultTransport
	if o.HTTP != nil && o.HTTP.Transport != nil {
		base = o.HTTP.Transport
	}
	client := &http.Client{Timeout: officialTimeout, Transport: &officialTransport{base: base, official: o, turn: newTurnID()}}
	m, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  "withdo", // replaced by the session's token in the transport
		BaseURL: strings.TrimRight(o.BaseURL, "/"), Model: OfficialModelName, HTTPClient: client,
	})
	if err != nil {
		return nil, fmt.Errorf("Withdo: %w", err)
	}
	return m, nil
}

// officialTransport signs every request of one turn: the session's bearer and the turn id. It also
// reads the server's refusals into the Err* sentinels, since the chat client would only make text
// of them.
type officialTransport struct {
	base     http.RoundTripper
	official Official
	turn     string
}

func (t *officialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.official.Token(req.Context())
	if err != nil {
		if t.official.SignedOut != nil && t.official.SignedOut(err) {
			return nil, ErrOfficialSignedOut
		}
		return nil, fmt.Errorf("the session could not be refreshed: %w", err)
	}
	if token == "" {
		return nil, ErrOfficialSignedOut
	}
	signed := req.Clone(req.Context())
	signed.Header.Set("Authorization", "Bearer "+token)
	signed.Header.Set(HeaderTurn, t.turn)
	resp, err := t.base.RoundTrip(signed)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		_ = resp.Body.Close()
		return nil, ErrOfficialSignedOut
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		switch {
		case bytes.Contains(raw, []byte(`"llm_quota"`)):
			return nil, ErrOfficialQuota
		case bytes.Contains(raw, []byte(`"llm_disabled"`)):
			return nil, ErrOfficialOff
		}
		resp.Body = io.NopCloser(bytes.NewReader(raw))
	}
	return resp, nil
}

// newTurnID is a random UUID (version 4).
func newTurnID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("providers: no randomness: " + err.Error())
	}
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Source says which model answers.
type Source int

// The sources, in the order the rule tries them.
const (
	// SourceNone: no model; the input reads dates and tags with local rules (internal/quickadd).
	SourceNone Source = iota
	// SourceOwn: the provider the user chose in Settings > Model.
	SourceOwn
	// SourceOfficial: the server's model, for anyone signed in to a server that offers it.
	SourceOfficial
)

// Choose is the rule: the user's own provider when one is chosen and ready; otherwise the official
// model when it is available (signed in, the server offers it and today's turns are not used up);
// otherwise none.
func Choose(ownReady, officialAvailable bool) Source {
	switch {
	case ownReady:
		return SourceOwn
	case officialAvailable:
		return SourceOfficial
	}
	return SourceNone
}

// Selector picks the model of each turn by Choose. The desktop gives it the provider manager and
// the sync session; the phone gives it only the session. Both ask Configured for "is there a
// model" and pass Model to the agent as its ModelFunc.
type Selector struct {
	// Own is the user's own providers; nil where there is no such setting (the phone).
	Own *Manager
	// Official returns the official model and whether it can be used now: the user is signed in, the
	// server says it offers the model, and today's turns are not used up. It is called for every
	// frame the UI draws, so it must be cheap and must not touch the network.
	Official func() (Official, bool)
}

// Source is the model that would answer now.
func (s Selector) Source() Source {
	own := s.Own != nil && s.Own.Configured()
	official := false
	if !own && s.Official != nil {
		_, official = s.Official()
	}
	return Choose(own, official)
}

// Configured reports whether some model would answer.
func (s Selector) Configured() bool { return s.Source() != SourceNone }

// Model builds the chat model of one turn: it is the agent's ModelFunc.
func (s Selector) Model(ctx context.Context) (model.ToolCallingChatModel, error) {
	switch s.Source() {
	case SourceOwn:
		return s.Own.Model(ctx)
	case SourceOfficial:
		o, _ := s.Official()
		return o.NewModel(ctx)
	}
	return nil, ErrNotConfigured
}
