package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// fakeOfficial is an OfficialSource with a card and a switch for whether the model can answer.
type fakeOfficial struct {
	info      OfficialInfo
	available bool
	refreshed atomic.Int32
}

func (f *fakeOfficial) OfficialInfo() OfficialInfo { return f.info }
func (f *fakeOfficial) OfficialModel() (providers.Official, bool) {
	return providers.Official{BaseURL: "http://x/v1/llm", Token: func(context.Context) (string, error) { return "t", nil }}, f.available
}
func (f *fakeOfficial) RefreshOfficial() { f.refreshed.Add(1) }
func (f *fakeOfficial) TestOfficial(context.Context) (time.Duration, error) {
	return 0, providers.ErrOfficialQuota
}

func TestTheOfficialModelAnswersWhenNoProviderOfOneIsInUse(t *testing.T) {
	off := &fakeOfficial{info: OfficialInfo{Phase: OfficialReady, Left: 3, Limit: 100}, available: true}
	mgr := newManager(t, providers.NewMemoryKeyring(), nil)
	a := &AgentAssistant{Provider: mgr, Official: off}
	settings := &ProviderSettings{Provider: mgr, OfficialFrom: off}

	if !a.Configured() || a.Selector().Source() != providers.SourceOfficial || !settings.Official().InUse {
		t.Fatal("the official model does not answer with no provider chosen")
	}
	if a.modelName() != OfficialName {
		t.Fatalf("modelName = %q", a.modelName())
	}
	if err := settings.SetActive("ollama"); err != nil {
		t.Fatal(err)
	}
	if a.Selector().Source() != providers.SourceOwn || settings.Official().InUse {
		t.Fatal("a provider chosen in Settings does not win")
	}
	if err := settings.UseOfficial(); err != nil {
		t.Fatal(err)
	}
	if a.Selector().Source() != providers.SourceOfficial || !settings.Official().InUse {
		t.Fatal("Use this does not give the answers back to the official model")
	}
	off.available = false // signed out, or the day's turns used up
	if a.Configured() || settings.Official().InUse {
		t.Fatal("a model answers with the official one unavailable")
	}
	_, err := settings.TestOfficial(context.Background())
	if ProblemOf(err) != ProblemOfficialQuota {
		t.Fatalf("TestOfficial error = %v (%v)", err, ProblemOf(err))
	}
}

// llmServer is a sync server that offers (or not) the official model and counts turns.
type llmServer struct {
	*httptest.Server
	offered atomic.Bool
	left    atomic.Int32
	quotas  atomic.Int32
}

func newLLMServer(t *testing.T, offered bool, left int) *llmServer {
	t.Helper()
	f := &llmServer{}
	f.offered.Store(offered)
	f.left.Store(int32(left))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"auth":"pocketbase","llm":` + strconv.FormatBool(f.offered.Load()) + `}`))
	})
	mux.HandleFunc("GET /v1/llm/quota", func(w http.ResponseWriter, r *http.Request) {
		f.quotas.Add(1)
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		reset := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"limit":100,"left":` + strconv.Itoa(int(f.left.Load())) + `,"resets_at":"` + reset + `"}`))
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// officialReal is a Real whose sync config names server, signed in with a token or not.
func officialReal(t *testing.T, server string, signedIn bool) *Real {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := syncclient.Config{Server: server}
	if signedIn {
		cfg.Token = "tok"
	}
	if err := syncclient.SaveConfigAt(syncclient.ConfigPath(path), cfg); err != nil {
		t.Fatal(err)
	}
	r := NewReal(db, RealOptions{SyncConfigPath: syncclient.ConfigPath(path), Secrets: appSecrets{}})
	t.Cleanup(func() { _ = r.Close() })
	return r
}

// officialIs polls until the card reads phase, since the answer comes from a background request.
func officialIs(t *testing.T, r *Real, phase OfficialPhase) OfficialInfo {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		info := r.OfficialInfo()
		if info.Phase == phase {
			return info
		}
		if time.Now().After(deadline) {
			t.Fatalf("the official card stayed %+v, want phase %v", info, phase)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestOfficialSignedInReadsTheQuotaAndOffersTheModel(t *testing.T) {
	f := newLLMServer(t, true, 86)
	r := officialReal(t, f.URL, true)
	info := officialIs(t, r, OfficialReady)
	if info.Left != 86 || info.Limit != 100 || info.ResetsAt.IsZero() {
		t.Fatalf("info = %+v", info)
	}
	o, ok := r.OfficialModel()
	if !ok || o.BaseURL != f.URL+"/v1/llm" {
		t.Fatalf("OfficialModel = %+v, %v", o, ok)
	}
	if tok, err := o.Token(context.Background()); err != nil || tok != "tok" {
		t.Fatalf("Token = %q, %v", tok, err)
	}
	if d, err := r.TestOfficial(context.Background()); err != nil || d <= 0 {
		t.Fatalf("TestOfficial = %v, %v", d, err)
	}
}

func TestOfficialUsedUpIsNotAvailable(t *testing.T) {
	f := newLLMServer(t, true, 0)
	r := officialReal(t, f.URL, true)
	info := officialIs(t, r, OfficialUsedUp)
	if info.Left != 0 {
		t.Fatalf("info = %+v", info)
	}
	if _, ok := r.OfficialModel(); ok {
		t.Fatal("the model answers with no turns left")
	}
	// A turn count that comes back after midnight lifts it on the next refresh.
	f.left.Store(100)
	r.RefreshOfficial()
	officialIs(t, r, OfficialReady)
}

func TestOfficialSignedOutAsksToSignIn(t *testing.T) {
	f := newLLMServer(t, true, 86)
	r := officialReal(t, f.URL, false)
	if got := r.OfficialInfo().Phase; got != OfficialSignedOut {
		t.Fatalf("phase = %v", got)
	}
	if _, ok := r.OfficialModel(); ok {
		t.Fatal("the model answers signed out")
	}
	time.Sleep(100 * time.Millisecond)
	if f.quotas.Load() != 0 {
		t.Fatal("the quota was asked for with no session")
	}
}

func TestOfficialIsHiddenOnAServerThatDoesNotOfferIt(t *testing.T) {
	f := newLLMServer(t, false, 0)
	for _, signedIn := range []bool{true, false} {
		r := officialReal(t, f.URL, signedIn)
		r.OfficialInfo() // starts the question
		deadline := time.Now().Add(5 * time.Second)
		for {
			r.llm.mu.Lock()
			known := r.llm.known
			r.llm.mu.Unlock()
			if known {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("signedIn=%v: the server never answered", signedIn)
			}
			time.Sleep(10 * time.Millisecond)
		}
		if info := r.OfficialInfo(); info.Offered() {
			t.Fatalf("signedIn=%v: info = %+v", signedIn, info)
		}
	}
}
