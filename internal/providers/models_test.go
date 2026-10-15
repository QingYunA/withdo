package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestPresetsAreInTheGridOrderAndHaveALogoAndAKeyPage(t *testing.T) {
	var ids []string
	for _, p := range Presets {
		ids = append(ids, p.ID)
		if p.Monogram == "" {
			t.Errorf("%s has no monogram", p.ID)
		}
		if p.HasKeyField() && p.ID != IDCustom && !strings.HasPrefix(p.KeyURL, "https://") {
			t.Errorf("%s: key page %q", p.ID, p.KeyURL)
		}
		if !p.HasKeyField() && p.KeyURL != "" {
			t.Errorf("%s has no key field but a key page", p.ID)
		}
	}
	want := []string{"anthropic", "openai", "deepseek", "kimi", "zhipu", "qwen", "gemini", "openrouter", "siliconflow", "ollama", "magpie", "custom"}
	if !slices.Equal(ids, want) {
		t.Errorf("order %v, want %v", ids, want)
	}
}

// modelServer answers GET /models (or /v1/models) with body and records the request.
func modelServer(t *testing.T, status int, body string) (*httptest.Server, *http.Request) {
	t.Helper()
	got := new(http.Request)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = *r.Clone(context.Background())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func resolved(id, base string) Resolved {
	r, _ := Config{}.With(id, Entry{BaseURL: base}).Resolve(id)
	return r
}

func TestListModelsAsksAnOpenAICompatibleProviderForItsModelsWithTheKey(t *testing.T) {
	srv, req := modelServer(t, 200, `{"data":[{"id":"gpt-b"},{"id":"text-embedding-3-small"},{"id":"gpt-a"},{"id":"whisper-1"}]}`)
	got, err := ListModels(context.Background(), resolved("openai", srv.URL), "sk-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL.Path != "/models" || req.Header.Get("Authorization") != "Bearer sk-test" {
		t.Errorf("asked %s with %q", req.URL.Path, req.Header.Get("Authorization"))
	}
	if want := []string{"gpt-a", "gpt-b"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v (sorted, chat models only)", got, want)
	}
}

func TestListModelsAsksAnthropicWithItsHeadersAndKeepsItsOrder(t *testing.T) {
	srv, req := modelServer(t, 200, `{"data":[{"id":"claude-opus-5-5","type":"model"},{"id":"claude-haiku-5-5","type":"model"}],"has_more":false}`)
	got, err := ListModels(context.Background(), resolved("anthropic", srv.URL), "sk-ant", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL.Path != "/v1/models" || req.Header.Get("x-api-key") != "sk-ant" || req.Header.Get("anthropic-version") == "" {
		t.Errorf("asked %s with key %q, version %q", req.URL.Path, req.Header.Get("x-api-key"), req.Header.Get("anthropic-version"))
	}
	if want := []string{"claude-opus-5-5", "claude-haiku-5-5"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want the newest first as the API gives them", got)
	}
}

func TestListModelsUsesTheFixedKeyOfAKeylessPreset(t *testing.T) {
	srv, req := modelServer(t, 200, `{"data":[{"id":"qwen3:8b"}]}`)
	if _, err := ListModels(context.Background(), resolved("ollama", srv.URL), "", nil); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "Bearer ollama" {
		t.Errorf("authorization %q", req.Header.Get("Authorization"))
	}
}

func TestListModelsFailsInWordsClassifyKnows(t *testing.T) {
	srv, _ := modelServer(t, 401, `{"error":{"message":"bad key sk-secret"}}`)
	_, err := ListModels(context.Background(), resolved("deepseek", srv.URL), "sk-secret", nil)
	if err == nil || Classify(err) != KindKeyRejected {
		t.Fatalf("err %v, kind %v", err, Classify(err))
	}
	if strings.Contains(err.Error(), "sk-secret") {
		t.Errorf("the error shows the key: %v", err)
	}
	if _, err := ListModels(context.Background(), resolved("deepseek", srv.URL), "", nil); err == nil || Classify(err) != KindNoKey {
		t.Errorf("no key: %v", err)
	}
	bad, _ := modelServer(t, 200, `not json`)
	if _, err := ListModels(context.Background(), resolved("kimi", bad.URL), "k", nil); err == nil {
		t.Error("a reply that is not a model list is accepted")
	}
	empty, _ := modelServer(t, 200, `{"data":[]}`)
	if got, err := ListModels(context.Background(), resolved("kimi", empty.URL), "k", nil); err != nil || len(got) != 0 {
		t.Errorf("empty list: %v, %v", got, err)
	}
}

func TestManagerModelsResolvesWhatIsTypedAndFallsBackOnTheKeyStored(t *testing.T) {
	srv, req := modelServer(t, 200, `{"data":[{"id":"m1"}]}`)
	m, err := NewManager(NewStore(t.TempDir()+"/"+ConfigFile), NewMemoryKeyring(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetKey("kimi", "stored-key"); err != nil {
		t.Fatal(err)
	}
	got, err := m.Models(context.Background(), "kimi", Entry{BaseURL: srv.URL}, "")
	if err != nil || !slices.Equal(got, []string{"m1"}) {
		t.Fatalf("%v, %v", got, err)
	}
	if req.Header.Get("Authorization") != "Bearer stored-key" {
		t.Errorf("authorization %q", req.Header.Get("Authorization"))
	}
	if _, err := m.Models(context.Background(), "deepseek", Entry{BaseURL: srv.URL}, ""); err == nil {
		t.Error("a provider that needs a key listed models without one")
	}
}
