package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const (
	// requestTimeout bounds one model request. A turn has several, so this is generous.
	requestTimeout = 60 * time.Second
	maxTokens      = 1024
	// testTimeout bounds the Test button.
	testTimeout = 20 * time.Second
)

// NewModel builds the Eino chat model for a resolved preset and the key to use with it. client may
// be nil for the default HTTP client.
func NewModel(ctx context.Context, r Resolved, key string, client *http.Client) (model.ToolCallingChatModel, error) {
	if strings.TrimSpace(r.Model) == "" {
		return nil, fmt.Errorf("%s: no model chosen", r.Preset.Name)
	}
	if r.Preset.FixedKey != "" {
		key = r.Preset.FixedKey
	}
	switch r.Protocol {
	case Anthropic:
		return newClaude(ctx, r, key, client)
	case OpenAI:
		return newOpenAI(ctx, r, key, client)
	}
	return nil, fmt.Errorf("%s: unknown protocol %q", r.Preset.Name, r.Protocol)
}

func newClaude(ctx context.Context, r Resolved, key string, client *http.Client) (model.ToolCallingChatModel, error) {
	cfg := &claude.Config{APIKey: key, Model: r.Model, MaxTokens: maxTokens, HTTPClient: client, RequestTimeout: requestTimeout}
	if u := strings.TrimRight(r.BaseURL, "/"); u != "" {
		cfg.BaseURL = &u
	}
	m, err := claude.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", r.Preset.Name, err)
	}
	return m, nil
}

func newOpenAI(ctx context.Context, r Resolved, key string, client *http.Client) (model.ToolCallingChatModel, error) {
	if strings.TrimSpace(r.BaseURL) == "" {
		return nil, fmt.Errorf("%s: no base URL", r.Preset.Name)
	}
	m, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: key, BaseURL: strings.TrimRight(r.BaseURL, "/"), Model: r.Model,
		HTTPClient: clientOrDefault(client),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", r.Preset.Name, err)
	}
	return m, nil
}

func clientOrDefault(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return &http.Client{Timeout: requestTimeout}
}

// Probe sends the smallest request the model accepts (one token) and reports how long the round
// trip took. It is what the Test button runs.
func Probe(ctx context.Context, m model.ToolCallingChatModel) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()
	start := time.Now()
	if _, err := m.Generate(ctx, []*schema.Message{schema.UserMessage("Reply with: ok")}, model.WithMaxTokens(1)); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}
