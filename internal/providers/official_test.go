package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// officialServer is a fake sync server's /v1/llm: it records each request's headers and body and
// answers with status and body.
type officialServer struct {
	*httptest.Server
	mu    sync.Mutex
	turns []string
	auths []string
	paths []string
	model []string
}

func newOfficialServer(t *testing.T, status int, body string) *officialServer {
	t.Helper()
	o := &officialServer{}
	o.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(raw, &req)
		o.mu.Lock()
		o.turns = append(o.turns, r.Header.Get(HeaderTurn))
		o.auths = append(o.auths, r.Header.Get("Authorization"))
		o.paths = append(o.paths, r.URL.Path)
		o.model = append(o.model, req.Model)
		o.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(o.Close)
	return o
}

const officialAnswer = `{"id":"c1","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`

func officialAt(o *officialServer) Official {
	return Official{
		BaseURL: o.URL + "/v1/llm",
		Token:   func(context.Context) (string, error) { return "session-token", nil },
	}
}

func TestOfficialSignsEveryRequestOfATurnWithOneTurnID(t *testing.T) {
	srv := newOfficialServer(t, 200, officialAnswer)
	o := officialAt(srv)
	first, err := o.NewModel(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	msgs := []*schema.Message{schema.UserMessage("hi")}
	for i := 0; i < 3; i++ { // the calls of a tool loop
		if _, err := first.Generate(context.Background(), msgs); err != nil {
			t.Fatal(err)
		}
	}
	withTools, err := first.WithTools(nil) // the agent builds its tool-calling copy this way
	if err == nil {
		if _, err := withTools.Generate(context.Background(), msgs); err != nil {
			t.Fatal(err)
		}
	}
	second, _ := o.NewModel(context.Background())
	if _, err := second.Generate(context.Background(), msgs); err != nil {
		t.Fatal(err)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.turns) < 4 {
		t.Fatalf("requests = %d", len(srv.turns))
	}
	uuid := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	last := len(srv.turns) - 1
	for i, turn := range srv.turns[:last] {
		if turn != srv.turns[0] || !uuid.MatchString(turn) {
			t.Fatalf("request %d turn = %q, want one UUID %q", i, turn, srv.turns[0])
		}
	}
	if srv.turns[last] == srv.turns[0] {
		t.Fatal("the next turn reused the turn id")
	}
	for i := range srv.auths {
		if srv.auths[i] != "Bearer session-token" || srv.paths[i] != "/v1/llm/chat/completions" || srv.model[i] != OfficialModelName {
			t.Fatalf("request %d: auth %q path %q model %q", i, srv.auths[i], srv.paths[i], srv.model[i])
		}
	}
}

func TestOfficialAsksForTheTokenOnEveryRequest(t *testing.T) {
	srv := newOfficialServer(t, 200, officialAnswer)
	n := 0
	o := officialAt(srv)
	o.Token = func(context.Context) (string, error) { n++; return "t" + string(rune('0'+n)), nil }
	m, _ := o.NewModel(context.Background())
	for i := 0; i < 2; i++ {
		if _, err := m.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")}); err != nil {
			t.Fatal(err)
		}
	}
	if srv.auths[0] != "Bearer t1" || srv.auths[1] != "Bearer t2" {
		t.Fatalf("auths = %v", srv.auths)
	}
}

func TestOfficialRefusalsAreTheirOwnErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
		kind   Kind
	}{
		{"quota", 429, `{"error":{"message":"llm_quota","type":"withdo","code":"llm_quota"},"code":"llm_quota"}`, ErrOfficialQuota, KindOfficialQuota},
		{"off", 503, `{"error":{"message":"llm_disabled","type":"withdo","code":"llm_disabled"},"code":"llm_disabled"}`, ErrOfficialOff, KindOfficialOff},
		{"session refused", 401, `{"error":"unauthenticated"}`, ErrOfficialSignedOut, KindOfficialSignedOut},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newOfficialServer(t, tc.status, tc.body)
			m, _ := officialAt(srv).NewModel(context.Background())
			_, err := m.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
			if err == nil {
				t.Fatal("no error")
			}
			if Classify(err) != tc.kind {
				t.Fatalf("Classify(%v) = %v, want %v", err, Classify(err), tc.kind)
			}
			if Friendly(err) == "" {
				t.Fatal("no words")
			}
		})
	}
	t.Run("a busy upstream is rate limited, not the quota", func(t *testing.T) {
		srv := newOfficialServer(t, 429, `{"error":{"message":"llm_busy","type":"withdo","code":"llm_busy"},"code":"llm_busy"}`)
		m, _ := officialAt(srv).NewModel(context.Background())
		_, err := m.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
		if err == nil || Classify(err) != KindRateLimited {
			t.Fatalf("err = %v kind %v", err, Classify(err))
		}
	})
}

func TestOfficialSignedOutNeverReachesTheServer(t *testing.T) {
	srv := newOfficialServer(t, 200, officialAnswer)
	o := officialAt(srv)
	signedOut := errors.New("signed out")
	o.Token = func(context.Context) (string, error) { return "", signedOut }
	o.SignedOut = func(err error) bool { return errors.Is(err, signedOut) }
	m, _ := o.NewModel(context.Background())
	_, err := m.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
	if Classify(err) != KindOfficialSignedOut {
		t.Fatalf("err = %v", err)
	}
	if len(srv.turns) != 0 {
		t.Fatal("a signed-out request reached the server")
	}
	if _, err := (Official{}).NewModel(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("empty Official = %v", err)
	}
}

func TestChooseAndSelector(t *testing.T) {
	for _, tc := range []struct {
		own, official bool
		want          Source
	}{
		{false, false, SourceNone},
		{true, false, SourceOwn},
		{false, true, SourceOfficial},
		{true, true, SourceOwn}, // a chosen provider wins over the official model
	} {
		if got := Choose(tc.own, tc.official); got != tc.want {
			t.Errorf("Choose(%v, %v) = %v, want %v", tc.own, tc.official, got, tc.want)
		}
	}

	srv := newOfficialServer(t, 200, officialAnswer)
	available := true
	official := func() (Official, bool) { return officialAt(srv), available }
	keys := NewMemoryKeyring()
	mgr := newManager(t, filepath.Join(t.TempDir(), ConfigFile), keys, nil)

	sel := Selector{Own: mgr, Official: official}
	if sel.Source() != SourceOfficial || !sel.Configured() {
		t.Fatalf("no own provider: source %v", sel.Source())
	}
	if _, err := sel.Model(context.Background()); err != nil {
		t.Fatalf("official Model = %v", err)
	}
	_ = mgr.Save(Config{Active: "ollama"})
	if sel.Source() != SourceOwn {
		t.Fatalf("own chosen: source %v", sel.Source())
	}
	_ = mgr.Save(mgr.Config().WithoutActive())
	if sel.Source() != SourceOfficial {
		t.Fatalf("own cleared: source %v", sel.Source())
	}
	if _, ok := mgr.Config().Resolve("ollama"); !ok || mgr.Config().Active != "" {
		t.Fatal("WithoutActive dropped what it should keep")
	}
	available = false
	if sel.Source() != SourceNone || sel.Configured() {
		t.Fatalf("official unavailable: source %v", sel.Source())
	}
	if _, err := sel.Model(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Model with none = %v", err)
	}
	phone := Selector{Official: official} // no own providers
	available = true
	if phone.Source() != SourceOfficial {
		t.Fatalf("phone source %v", phone.Source())
	}
}
