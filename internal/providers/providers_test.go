package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/providers/anthropic"
	"github.com/cloudwego/eino-ext/components/model/openai"
)

// seen is what a fake provider server received.
type seen struct {
	Path, Key, Auth string
	Body            map[string]any
}

// serve starts a fake provider that answers every request with the reply of its protocol.
func serve(t *testing.T, proto Protocol, status int) (*httptest.Server, *seen) {
	t.Helper()
	got := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got.Body)
		got.Path, got.Key, got.Auth = r.URL.Path, r.Header.Get("x-api-key"), r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = w.Write([]byte(`{"error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
			return
		}
		if proto == Anthropic {
			_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"c1","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func TestPresetDefaults(t *testing.T) {
	want := map[string]struct {
		proto Protocol
		url   string
	}{
		"anthropic":   {Anthropic, ""},
		"magpie":      {Anthropic, "http://127.0.0.1:3425"},
		"openai":      {OpenAI, "https://api.openai.com/v1"},
		"deepseek":    {OpenAI, "https://api.deepseek.com"},
		"kimi":        {OpenAI, "https://api.moonshot.cn/v1"},
		"zhipu":       {OpenAI, "https://open.bigmodel.cn/api/paas/v4"},
		"qwen":        {OpenAI, "https://dashscope.aliyuncs.com/compatible-mode/v1"},
		"gemini":      {OpenAI, "https://generativelanguage.googleapis.com/v1beta/openai"},
		"siliconflow": {OpenAI, "https://api.siliconflow.cn/v1"},
		"openrouter":  {OpenAI, "https://openrouter.ai/api/v1"},
		"ollama":      {OpenAI, "http://127.0.0.1:11434/v1"},
		"custom":      {OpenAI, ""},
	}
	if len(Presets) != len(want) {
		t.Fatalf("%d presets, want %d", len(Presets), len(want))
	}
	for _, p := range Presets {
		w, ok := want[p.ID]
		if !ok || p.Protocol != w.proto || p.BaseURL != w.url {
			t.Errorf("%s = %s %q, want %s %q", p.ID, p.Protocol, p.BaseURL, w.proto, w.url)
		}
		r, _ := Config{}.Resolve(p.ID)
		if p.ID != IDCustom && r.Model == "" {
			t.Errorf("%s has no default model", p.ID)
		}
	}
	if _, ok := PresetByID("nope"); ok {
		t.Error("found a preset that does not exist")
	}
	if got, _ := PresetByID(IDAnthropic); got.Display() != "https://api.anthropic.com" {
		t.Errorf("Display = %q", got.Display())
	}
}

func TestEveryPresetBuildsTheRightModelAgainstItsBaseURL(t *testing.T) {
	for _, p := range Presets {
		t.Run(p.ID, func(t *testing.T) {
			srv, got := serve(t, p.Protocol, http.StatusOK)
			cfg := Config{}.With(p.ID, Entry{BaseURL: srv.URL, Model: "test-model"})
			r, _ := cfg.Resolve(p.ID)
			m, err := NewModel(context.Background(), r, "sk-test", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Probe(context.Background(), m); err != nil {
				t.Fatal(err)
			}
			if got.Body["model"] != "test-model" {
				t.Errorf("model = %v", got.Body["model"])
			}
			switch p.Protocol {
			case Anthropic:
				if _, ok := m.(*anthropic.ChatModel); !ok {
					t.Errorf("built %T", m)
				}
				if got.Path != "/v1/messages" {
					t.Errorf("path = %q", got.Path)
				}
				wantKey := "sk-test"
				if p.FixedKey != "" {
					wantKey = p.FixedKey
				}
				if got.Key != wantKey {
					t.Errorf("x-api-key = %q, want %q", got.Key, wantKey)
				}
			case OpenAI:
				if _, ok := m.(*openai.ChatModel); !ok {
					t.Errorf("built %T", m)
				}
				if got.Path != "/chat/completions" {
					t.Errorf("path = %q", got.Path)
				}
				wantKey := "sk-test"
				if p.FixedKey != "" {
					wantKey = p.FixedKey
				}
				if got.Auth != "Bearer "+wantKey {
					t.Errorf("authorization = %q", got.Auth)
				}
			}
			if mt, _ := got.Body["max_tokens"].(float64); mt != 1 {
				t.Errorf("probe max_tokens = %v, want 1", got.Body["max_tokens"])
			}
		})
	}
}

func TestCustomPresetPicksItsProtocol(t *testing.T) {
	cfg := Config{}.With(IDCustom, Entry{BaseURL: "http://x", Model: "m", Protocol: Anthropic})
	if r, _ := cfg.Resolve(IDCustom); r.Protocol != Anthropic {
		t.Errorf("protocol = %s", r.Protocol)
	}
	cfg = Config{}.With("openai", Entry{Protocol: Anthropic}) // ignored: not a custom preset
	if r, _ := cfg.Resolve("openai"); r.Protocol != OpenAI {
		t.Errorf("protocol = %s", r.Protocol)
	}
}

func TestNewModelRejectsMissingPieces(t *testing.T) {
	cases := map[string]Resolved{
		"no model":    {Preset: Preset{Name: "x"}, Protocol: OpenAI, BaseURL: "http://x"},
		"no base url": {Preset: Preset{Name: "x"}, Protocol: OpenAI, Model: "m"},
		"bad proto":   {Preset: Preset{Name: "x"}, Protocol: "smoke", Model: "m", BaseURL: "http://x"},
	}
	for name, r := range cases {
		if _, err := NewModel(context.Background(), r, "k", nil); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestConfigStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", ConfigFile)
	s := NewStore(path)
	if c, err := s.Load(); err != nil || c.Active != "" {
		t.Fatalf("missing file: %+v, %v", c, err)
	}
	cfg := Config{}.With("openai", Entry{Model: "gpt-x"}).WithActive("openai")
	if err := s.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil || got.Active != "openai" || got.Providers["openai"].Model != "gpt-x" {
		t.Fatalf("loaded %+v, %v", got, err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", fi.Mode().Perm())
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Error("temp file left behind")
	}
	if err := os.WriteFile(path, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Error("a corrupt file loaded")
	}
	orig := Config{}
	_ = orig.With("a", Entry{Model: "x"})
	if len(orig.Providers) != 0 {
		t.Error("With mutated its receiver")
	}
}

func TestConfigNeverHoldsAKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFile)
	m := newManager(t, path, NewMemoryKeyring(), nil)
	if err := m.SetKey("anthropic", "sk-secret-123"); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(Config{Active: "anthropic"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "secret") {
		t.Errorf("key leaked into %s: %s", path, b)
	}
}

func newManager(t *testing.T, path string, keys Keyring, client *http.Client) *Manager {
	t.Helper()
	m, err := NewManager(NewStore(path), keys, client)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestManagerConfigured(t *testing.T) {
	keys := NewMemoryKeyring()
	m := newManager(t, filepath.Join(t.TempDir(), ConfigFile), keys, nil)
	if m.Configured() {
		t.Fatal("configured with nothing")
	}
	if _, err := m.Model(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Model = %v", err)
	}
	_ = m.Save(Config{Active: "anthropic"})
	if m.Configured() {
		t.Error("Anthropic without a key counts as configured")
	}
	if _, err := m.Model(context.Background()); !errors.Is(err, ErrNoKey) {
		t.Errorf("Model without key = %v", err)
	}
	if err := m.SetKey("anthropic", "sk-1"); err != nil {
		t.Fatal(err)
	}
	if !m.Configured() || !m.HasKey("anthropic") {
		t.Error("not configured after the key was stored")
	}
	if _, err := m.Model(context.Background()); err != nil {
		t.Errorf("Model = %v", err)
	}
	_ = m.SetKey("anthropic", "")
	if m.Configured() || m.HasKey("anthropic") {
		t.Error("still configured after the key was removed")
	}
	for _, id := range []string{"magpie", "ollama"} { // no key needed
		_ = m.Save(Config{Active: id})
		if !m.Configured() {
			t.Errorf("%s needs no key but is not configured", id)
		}
	}
	_ = m.Save(Config{Active: IDCustom})
	if m.Configured() {
		t.Error("Custom with no URL counts as configured")
	}
	_ = m.Save(Config{Active: IDCustom}.With(IDCustom, Entry{BaseURL: "http://x", Model: "m"}))
	if !m.Configured() {
		t.Error("Custom with URL and model, no key, should be configured")
	}
}

func TestManagerTestButton(t *testing.T) {
	srv, got := serve(t, Anthropic, http.StatusOK)
	m := newManager(t, filepath.Join(t.TempDir(), ConfigFile), NewMemoryKeyring(), nil)
	if _, err := m.Test(context.Background(), "anthropic", Entry{BaseURL: srv.URL}, ""); !errors.Is(err, ErrNoKey) {
		t.Fatalf("no key: %v", err)
	}
	d, err := m.Test(context.Background(), "anthropic", Entry{BaseURL: srv.URL, Model: "typed"}, "sk-typed")
	if err != nil || d <= 0 {
		t.Fatalf("Test = %v, %v", d, err)
	}
	if got.Key != "sk-typed" || got.Body["model"] != "typed" {
		t.Errorf("sent key %q model %v", got.Key, got.Body["model"])
	}
	if m.HasKey("anthropic") {
		t.Error("Test stored the key it was given")
	}
	if _, err := m.Test(context.Background(), "nope", Entry{}, "k"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("unknown preset: %v", err)
	}
}

func TestManagerTestSurfacesProviderErrors(t *testing.T) {
	srv, _ := serve(t, Anthropic, http.StatusUnauthorized)
	m := newManager(t, filepath.Join(t.TempDir(), ConfigFile), NewMemoryKeyring(), nil)
	_, err := m.Test(context.Background(), "anthropic", Entry{BaseURL: srv.URL}, "sk-bad")
	if err == nil {
		t.Fatal("want an error")
	}
	if got := Friendly(err); got != "The provider didn't accept the API key." {
		t.Errorf("Friendly = %q (raw %v)", got, err)
	}
	if strings.Contains(err.Error(), "sk-bad") {
		t.Error("the key is in the error")
	}
}

func TestFriendly(t *testing.T) {
	cases := map[string]error{
		"":                                   nil,
		"No model is connected yet.":         ErrNotConfigured,
		"This provider needs an API key.":    ErrNoKey,
		"Cancelled.":                         context.Canceled,
		"The model took too long to answer.": context.DeadlineExceeded,
		"Couldn't reach the model. Is the address right and the server running?": errors.New("dial tcp 127.0.0.1:3425: connect: connection refused"),
		"The provider didn't accept the API key.":                                errors.New("POST /v1/messages: 403 Forbidden"),
		"The provider doesn't know this model or address.":                       errors.New("404 page not found"),
		"The provider is busy or out of quota. Try again in a moment.":           errors.New("429 rate limit"),
		"This model cannot read images.":                                         errors.New(`400 {"error":{"message":"unknown variant image_url, expected text"}}`),
		"something odd":                                                          errors.New("something odd\nsecond line"),
	}
	for want, err := range cases {
		if got := Friendly(err); got != want {
			t.Errorf("Friendly(%v) = %q, want %q", err, got, want)
		}
	}
	if got := Friendly(errors.New(strings.Repeat("x", 300))); len(got) > 170 {
		t.Errorf("long error not shortened: %d", len(got))
	}
}

// An Anthropic-protocol failure, from the SDK through the model to Friendly, reads the same as
// before the SDK's bedrock and vertex packages were dropped.
func TestAnthropicFailuresAreFriendly(t *testing.T) {
	reply := func(status int, kind string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("x-should-retry", "false") // no SDK retry delay
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"` + kind + `","message":"nope"}}`))
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close() // nothing listens there any more

	cases := map[string]struct {
		url  string
		want string
	}{
		"401":         {reply(401, "authentication_error").URL, "The provider didn't accept the API key."},
		"403":         {reply(403, "permission_error").URL, "The provider didn't accept the API key."},
		"404":         {reply(404, "not_found_error").URL, "The provider doesn't know this model or address."},
		"429":         {reply(429, "rate_limit_error").URL, "The provider is busy or out of quota. Try again in a moment."},
		"529":         {reply(529, "overloaded_error").URL, "The provider is busy or out of quota. Try again in a moment."},
		"unreachable": {gone.URL, "Couldn't reach the model. Is the address right and the server running?"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := Config{}.With("magpie", Entry{BaseURL: c.url}).Resolve("magpie")
			m, err := NewModel(context.Background(), r, "sk-secret", nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Probe(context.Background(), m)
			if err == nil {
				t.Fatal("want an error")
			}
			if got := Friendly(err); got != c.want {
				t.Errorf("Friendly = %q, want %q (raw %v)", got, c.want, err)
			}
			if strings.Contains(err.Error(), "sk-secret") {
				t.Error("the key is in the error")
			}
		})
	}
}

func TestMemoryKeyring(t *testing.T) {
	m := NewMemoryKeyring()
	if _, err := m.Get("a"); !errors.Is(err, ErrNoKey) {
		t.Fatal(err)
	}
	_ = m.Set("a", "1")
	if v, _ := m.Get("a"); v != "1" {
		t.Error(v)
	}
	_ = m.Delete("a")
	if _, err := m.Get("a"); !errors.Is(err, ErrNoKey) {
		t.Error("not deleted")
	}
}
