package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

// listTimeout bounds asking a provider for its models: the list is a nicety, so it must not hang.
const listTimeout = 10 * time.Second

// anthropicVersion is the API version header the Anthropic Models endpoint wants.
const anthropicVersion = "2023-06-01"

// notChat marks the entries of an OpenAI-style list that are not chat models.
var notChat = []string{"embedding", "whisper", "tts", "dall-e", "moderation", "rerank", "image", "audio"}

// ListModels asks the provider which models it offers: GET {base}/models for an OpenAI-compatible
// one, GET {base}/v1/models for the Anthropic API. An OpenAI-style list is sorted and kept to chat
// models; Anthropic's comes newest first and stays so. client may be nil. The error never holds
// the key.
func ListModels(ctx context.Context, r Resolved, key string, client *http.Client) ([]string, error) {
	if r.Preset.FixedKey != "" {
		key = r.Preset.FixedKey
	}
	if r.Preset.NeedsKey() && key == "" {
		return nil, ErrNoKey
	}
	endpoint, header := modelsRequest(r, key)
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", r.Preset.Name, err)
	}
	req.Header = header
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, redact(err, key)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: models: HTTP %d", r.Preset.Name, resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%s: the model list is not what was expected", r.Preset.Name)
	}
	var out []string
	for _, m := range body.Data {
		if m.ID != "" && (r.Protocol == Anthropic || isChat(m.ID)) {
			out = append(out, m.ID)
		}
	}
	if r.Protocol == OpenAI {
		slices.Sort(out)
	}
	return out, nil
}

func modelsRequest(r Resolved, key string) (string, http.Header) {
	h := http.Header{"Accept": {"application/json"}}
	base := strings.TrimRight(r.BaseURL, "/")
	if r.Protocol == Anthropic {
		if base == "" {
			base = "https://api.anthropic.com"
		}
		h.Set("x-api-key", key)
		h.Set("anthropic-version", anthropicVersion)
		return base + "/v1/models?" + url.Values{"limit": {"1000"}}.Encode(), h
	}
	if key != "" {
		h.Set("Authorization", "Bearer "+key)
	}
	return base + "/models", h
}

func isChat(id string) bool {
	low := strings.ToLower(id)
	for _, n := range notChat {
		if strings.Contains(low, n) {
			return false
		}
	}
	return true
}

// redact drops the key from a transport error, which can quote the request.
func redact(err error, key string) error {
	if key == "" || !strings.Contains(err.Error(), key) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return errors.New(strings.ReplaceAll(err.Error(), key, "…"))
}

// Models lists the models of one provider with what is typed in Settings, before it is saved. A
// non-empty key replaces the stored one for this call only.
func (m *Manager) Models(ctx context.Context, id string, e Entry, key string) ([]string, error) {
	r, ok := m.Config().With(id, e).Resolve(id)
	if !ok {
		return nil, ErrNotConfigured
	}
	if key == "" {
		stored, err := m.keys.Get(id)
		if err != nil && !errors.Is(err, ErrNoKey) {
			return nil, fmt.Errorf("keychain: %w", err)
		}
		key = stored
	}
	return ListModels(ctx, r, key, m.client)
}
