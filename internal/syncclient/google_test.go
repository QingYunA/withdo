package syncclient_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"context"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// fakeGoogle is a PocketBase with a Google provider: auth-methods, the realtime stream, the
// subscription and auth-with-oauth2. Its redirect plays the server's /api/oauth2-redirect.
type fakeGoogle struct {
	*httptest.Server
	pb *fakePocketBase

	mu          sync.Mutex
	noProvider  bool
	streams     map[string]chan string
	subscribed  map[string]bool
	exchange    map[string]string // the body fields of the last auth-with-oauth2
	exchangeBad bool
	next        int
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	t.Helper()
	f := &fakeGoogle{pb: newFakePocketBase(t), streams: map[string]chan string{}, subscribed: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/collections/users/auth-methods", f.authMethods)
	mux.HandleFunc("GET /api/realtime", f.realtime)
	mux.HandleFunc("POST /api/realtime", f.subscribe)
	mux.HandleFunc("GET /api/oauth2-redirect", f.redirect)
	mux.HandleFunc("POST /api/collections/users/auth-with-oauth2", f.authWithOAuth2)
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakeGoogle) authMethods(w http.ResponseWriter, _ *http.Request) {
	providers := []map[string]string{{
		"name": "google", "state": "pbstate", "codeVerifier": "verifier-1",
		"authURL": "https://accounts.google.com/o/oauth2/auth?client_id=cid&state=pbstate&scope=email&redirect_uri=",
	}}
	if f.noProvider {
		providers = nil
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"oauth2": map[string]any{"enabled": true, "providers": providers}})
}

func (f *fakeGoogle) realtime(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.next++
	id := fmt.Sprintf("client%d", f.next)
	ch := make(chan string, 2)
	f.streams[id] = ch
	f.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = fmt.Fprintf(w, "id:%s\nevent:PB_CONNECT\ndata:{\"clientId\":%q}\n\n", id, id)
	w.(http.Flusher).Flush()
	for {
		select {
		case data := <-ch:
			_, _ = fmt.Fprintf(w, "id:%s\nevent:@oauth2\ndata:%s\n\n", id, data)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (f *fakeGoogle) subscribe(w http.ResponseWriter, r *http.Request) {
	var form struct {
		ClientID      string   `json:"clientId"`
		Subscriptions []string `json:"subscriptions"`
	}
	_ = json.NewDecoder(r.Body).Decode(&form)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.streams[form.ClientID] == nil || len(form.Subscriptions) != 1 || form.Subscriptions[0] != "@oauth2" {
		http.Error(w, "bad", http.StatusBadRequest)
		return
	}
	f.subscribed[form.ClientID] = true
	w.WriteHeader(http.StatusNoContent)
}

// redirect delivers state, code and error to the stream of the client named by state.
func (f *fakeGoogle) redirect(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f.mu.Lock()
	ch, ok := f.streams[q.Get("state")], f.subscribed[q.Get("state")]
	f.mu.Unlock()
	if ch == nil || !ok {
		_, _ = io.WriteString(w, "Failed to authenticate.")
		return
	}
	data, _ := json.Marshal(map[string]string{"state": q.Get("state"), "code": q.Get("code"), "error": q.Get("error")})
	ch <- string(data)
	_, _ = io.WriteString(w, "Auth completed.")
}

func (f *fakeGoogle) authWithOAuth2(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	f.mu.Lock()
	f.exchange = body
	bad := f.exchangeBad
	f.mu.Unlock()
	if bad {
		http.Error(w, "{}", http.StatusBadRequest)
		return
	}
	f.pb.mu.Lock()
	token := f.pb.mint()
	f.pb.mu.Unlock()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token": token, "record": map[string]string{"email": "ada@example.com", "name": "Ada Google"},
	})
}

// browser plays the person: it calls the server's redirect with the state of the auth URL it was
// given, as Google's redirect would, and the given code and error.
func (f *fakeGoogle) browser(t *testing.T, code, errText string) func(string) error {
	return func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		if u.Host != "accounts.google.com" || q.Get("client_id") != "cid" || q.Get("scope") != "email" {
			t.Errorf("auth URL = %s, want Google's with the provider's own query", authURL)
		}
		if q.Get("redirect_uri") != f.URL+"/api/oauth2-redirect" {
			t.Errorf("redirect_uri = %q, want the server's /api/oauth2-redirect", q.Get("redirect_uri"))
		}
		if q.Get("state") == "pbstate" || !strings.HasPrefix(q.Get("state"), "client") {
			t.Errorf("state = %q, want the realtime client id", q.Get("state"))
		}
		go func() {
			v := url.Values{"state": {q.Get("state")}, "code": {code}, "error": {errText}}
			resp, err := http.Get(f.URL + "/api/oauth2-redirect?" + v.Encode())
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

func TestSignInWithGoogleStoresTheToken(t *testing.T) {
	g := newFakeGoogle(t)
	secrets := newFakeSecrets()
	source := syncclient.NewPocketBaseSource(g.URL, secrets)
	if ok, err := source.GoogleAvailable(context.Background(), g.URL); err != nil || !ok {
		t.Fatalf("available = %v, %v", ok, err)
	}
	if err := source.SignInWithGoogle(context.Background(), g.URL, g.browser(t, "authcode", "")); err != nil {
		t.Fatalf("sign in: %v", err)
	}
	if g.exchange["provider"] != "google" || g.exchange["code"] != "authcode" ||
		g.exchange["codeVerifier"] != "verifier-1" || g.exchange["redirectURL"] != g.URL+"/api/oauth2-redirect" {
		t.Fatalf("exchange = %v", g.exchange)
	}
	token, err := source.Token(context.Background())
	if err != nil || token == "" {
		t.Fatalf("token = %q, %v", token, err)
	}
	if account, ok := source.Account(); !ok || account.Email != "ada@example.com" || account.Name != "Ada Google" {
		t.Fatalf("account = %+v, %v", account, ok)
	}
	if !syncclient.HasPocketBaseToken(secrets, "") {
		t.Fatal("the token should be in the secrets store")
	}
}

func TestSignInWithGoogleCancelled(t *testing.T) {
	g := newFakeGoogle(t)
	source := syncclient.NewPocketBaseSource(g.URL, newFakeSecrets())
	ctx, cancel := context.WithCancel(context.Background())
	err := source.SignInWithGoogle(ctx, g.URL, func(string) error {
		time.AfterFunc(50*time.Millisecond, cancel)
		return nil
	})
	if !errors.Is(err, syncclient.ErrGoogleCancelled) {
		t.Fatalf("err = %v, want ErrGoogleCancelled", err)
	}
}

func TestSignInWithGoogleFailures(t *testing.T) {
	t.Run("the browser reports an error", func(t *testing.T) {
		g := newFakeGoogle(t)
		source := syncclient.NewPocketBaseSource(g.URL, newFakeSecrets())
		err := source.SignInWithGoogle(context.Background(), g.URL, g.browser(t, "", "access_denied"))
		if !errors.Is(err, syncclient.ErrGoogleFailed) {
			t.Fatalf("err = %v, want ErrGoogleFailed", err)
		}
	})
	t.Run("no Google provider", func(t *testing.T) {
		g := newFakeGoogle(t)
		g.noProvider = true
		source := syncclient.NewPocketBaseSource(g.URL, newFakeSecrets())
		if ok, err := source.GoogleAvailable(context.Background(), g.URL); err != nil || ok {
			t.Fatalf("available = %v, %v, want false", ok, err)
		}
		err := source.SignInWithGoogle(context.Background(), g.URL, func(string) error {
			t.Error("the browser should not open")
			return nil
		})
		if !errors.Is(err, syncclient.ErrGoogleFailed) {
			t.Fatalf("err = %v, want ErrGoogleFailed", err)
		}
	})
	t.Run("the server refuses the code", func(t *testing.T) {
		g := newFakeGoogle(t)
		g.exchangeBad = true
		source := syncclient.NewPocketBaseSource(g.URL, newFakeSecrets())
		err := source.SignInWithGoogle(context.Background(), g.URL, g.browser(t, "authcode", ""))
		if !errors.Is(err, syncclient.ErrGoogleFailed) {
			t.Fatalf("err = %v, want ErrGoogleFailed", err)
		}
	})
	t.Run("a deadline is a failure, not a cancel", func(t *testing.T) {
		g := newFakeGoogle(t)
		source := syncclient.NewPocketBaseSource(g.URL, newFakeSecrets())
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := source.SignInWithGoogle(ctx, g.URL, func(string) error { return nil })
		if !errors.Is(err, syncclient.ErrGoogleFailed) {
			t.Fatalf("err = %v, want ErrGoogleFailed", err)
		}
	})
	t.Run("an unreachable server is a network error", func(t *testing.T) {
		g := newFakeGoogle(t)
		g.Close()
		source := syncclient.NewPocketBaseSource(g.URL, newFakeSecrets())
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := source.SignInWithGoogle(ctx, g.URL, func(string) error { return nil })
		if err == nil || errors.Is(err, syncclient.ErrGoogleFailed) || errors.Is(err, syncclient.ErrGoogleCancelled) {
			t.Fatalf("err = %v, want the network error", err)
		}
	})
}

func TestStartGoogleThenCloseCancels(t *testing.T) {
	g := newFakeGoogle(t)
	source := syncclient.NewPocketBaseSource(g.URL, newFakeSecrets())
	flow, authURL, err := source.StartGoogle(context.Background(), g.URL)
	if err != nil || authURL == "" {
		t.Fatalf("start: %v %q", err, authURL)
	}
	done := make(chan error, 1)
	go func() { done <- flow.Wait(context.Background()) }()
	time.AfterFunc(50*time.Millisecond, flow.Close)
	select {
	case err := <-done:
		if !errors.Is(err, syncclient.ErrGoogleCancelled) {
			t.Fatalf("err = %v, want ErrGoogleCancelled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not return after Close")
	}
}
