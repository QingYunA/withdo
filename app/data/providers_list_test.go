package data

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/providers"
)

func TestProviderSettingsCarriesLogoKeyPageAndWhetherItIsSetUp(t *testing.T) {
	ps := &ProviderSettings{Provider: newManager(t, providers.NewMemoryKeyring(), nil)}
	byID := func() map[string]Provider {
		m := map[string]Provider{}
		for _, p := range ps.Providers() {
			m[p.ID] = p
		}
		return m
	}
	got := byID()
	if got["kimi"].Monogram != "K" || got["kimi"].KeyURL == "" || !got["kimi"].KeyRequired || got["ollama"].KeyRequired {
		t.Errorf("kimi %+v, ollama %+v", got["kimi"], got["ollama"])
	}
	for id, p := range got {
		if p.Configured {
			t.Errorf("%s is set up on a fresh install", id)
		}
	}
	if err := ps.Save("deepseek", ProviderEdit{Key: "sk"}); err != nil {
		t.Fatal(err)
	}
	if err := ps.Save("ollama", ProviderEdit{Model: "qwen3:8b"}); err != nil {
		t.Fatal(err)
	}
	if err := ps.Save("openai", ProviderEdit{Model: "gpt-4.1"}); err != nil { // a model without a key is not set up
		t.Fatal(err)
	}
	got = byID()
	if !got["deepseek"].Configured || !got["ollama"].Configured || got["openai"].Configured {
		t.Errorf("deepseek %v, ollama %v, openai %v", got["deepseek"].Configured, got["ollama"].Configured, got["openai"].Configured)
	}
	if err := ps.SetActive("openai"); err != nil {
		t.Fatal(err)
	}
	if !byID()["openai"].Configured {
		t.Error("the provider in use is not set up")
	}
}

func TestProviderSettingsListModelsAsksTheProviderWithTheStoredKey(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[{"id":"b"},{"id":"a"}]}`))
	}))
	defer srv.Close()
	ps := &ProviderSettings{Provider: newManager(t, providers.NewMemoryKeyring(), nil)}
	if err := ps.Save("kimi", ProviderEdit{Key: "sk-stored"}); err != nil {
		t.Fatal(err)
	}
	got, err := ps.ListModels(bg, "kimi", ProviderEdit{BaseURL: srv.URL})
	if err != nil || !slices.Equal(got, []string{"a", "b"}) || auth != "Bearer sk-stored" {
		t.Errorf("%v, %v, %q", got, err, auth)
	}
	got, err = ps.ListModels(bg, "kimi", ProviderEdit{BaseURL: srv.URL, Key: "sk-typed"})
	if err != nil || auth != "Bearer sk-typed" {
		t.Errorf("%v, %v, %q", got, err, auth)
	}
}
