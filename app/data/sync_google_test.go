package data

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// addGoogle mounts PocketBase's Google routes on the code server: auth-methods with a Google
// provider, the realtime stream (one client, "c1"), its subscription and auth-with-oauth2. The
// returned func plays the server's /api/oauth2-redirect.
func addGoogle(t *testing.T, f *codeServer, withProvider bool) func(code, errText string) {
	t.Helper()
	events := make(chan string, 2)
	f.mux.HandleFunc("GET /api/collections/users/auth-methods", func(w http.ResponseWriter, _ *http.Request) {
		providers := []map[string]string{}
		if withProvider {
			providers = append(providers, map[string]string{
				"name": "google", "codeVerifier": "v", "authURL": "https://accounts.google.com/auth?state=x&redirect_uri=",
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"oauth2": map[string]any{"enabled": true, "providers": providers}})
	})
	f.mux.HandleFunc("GET /api/realtime", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "id:c1\nevent:PB_CONNECT\ndata:{\"clientId\":\"c1\"}\n\n")
		w.(http.Flusher).Flush()
		for {
			select {
			case data := <-events:
				_, _ = fmt.Fprintf(w, "id:c1\nevent:@oauth2\ndata:%s\n\n", data)
				w.(http.Flusher).Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
	f.mux.HandleFunc("POST /api/realtime", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	f.mux.HandleFunc("POST /api/collections/users/auth-with-oauth2", func(w http.ResponseWriter, _ *http.Request) {
		payload, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix()})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig",
			"record": map[string]any{"email": "ada@gmail.com", "name": "Ada"},
		})
	})
	return func(code, errText string) {
		data, _ := json.Marshal(map[string]string{"state": "c1", "code": code, "error": errText})
		events <- string(data)
	}
}

func TestCheckServerLearnsThatGoogleIsOffered(t *testing.T) {
	f := newCodeServer(t)
	addGoogle(t, f, true)
	r, _ := codeApp(t, f)
	r.SyncCheckServer()
	waitSnap(t, r, "the probe", func(s SyncSnapshot) bool { return s.GoogleAvailable })
}

func TestCheckServerWithoutAGoogleProviderOffersNoButton(t *testing.T) {
	f := newCodeServer(t)
	addGoogle(t, f, false)
	r, _ := codeApp(t, f)
	r.SyncCheckServer()
	waitSnap(t, r, "the probe", func(s SyncSnapshot) bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.sync.code.probed == s.Server
	})
	if r.SyncSnapshot().GoogleAvailable {
		t.Fatal("a server with no Google provider offered Google")
	}
}

func TestGoogleSignInWaitsForTheBrowserThenLinks(t *testing.T) {
	f := newCodeServer(t)
	redirect := addGoogle(t, f, true)
	r, _ := codeApp(t, f)
	opened := make(chan string, 1)
	r.SyncSignInWithGoogle(func(u string) { opened <- u })
	snap := r.SyncSnapshot()
	if snap.Phase != SyncSigningIn || snap.SignInStage != SignInStageGoogle {
		t.Fatalf("phase %v stage %q, want signing in and waiting for Google", snap.Phase, snap.SignInStage)
	}
	u, err := url.Parse(waitString(t, opened, "the browser to open"))
	if err != nil || u.Host != "accounts.google.com" || u.Query().Get("state") != "c1" ||
		u.Query().Get("redirect_uri") != f.URL+"/api/oauth2-redirect" {
		t.Fatalf("opened %v: want Google's URL with the realtime client id as state", u)
	}
	redirect("authcode", "")
	waitSignal(t, f.devicesHit, "the device registration")
	if snap := r.SyncSnapshot(); snap.Phase != SyncSigningIn || snap.SignInStage != SignInStageDownloading || snap.AccountEmail != "ada@gmail.com" {
		t.Fatalf("after Google: phase %v stage %q account %q", snap.Phase, snap.SignInStage, snap.AccountEmail)
	}
	close(f.release)
	waitSnap(t, r, "the sign-in to end", func(s SyncSnapshot) bool { return s.Phase != SyncSigningIn })
}

func TestCancelDropsTheGoogleWaitAndKeepsTheButton(t *testing.T) {
	f := newCodeServer(t)
	addGoogle(t, f, true)
	r, _ := codeApp(t, f)
	r.SyncCheckServer()
	waitSnap(t, r, "the probe", func(s SyncSnapshot) bool { return s.GoogleAvailable })
	opened := make(chan string, 1)
	r.SyncSignInWithGoogle(func(u string) { opened <- u })
	waitString(t, opened, "the browser to open")

	r.SyncCancel()
	snap := r.SyncSnapshot()
	if snap.Phase != SyncSignedOut || snap.SignInFailure != SignInFailureGoogleCancelled || !snap.GoogleAvailable {
		t.Fatalf("phase %v failure %q google %v, want signed out, cancelled, button kept", snap.Phase, snap.SignInFailure, snap.GoogleAvailable)
	}
	time.Sleep(100 * time.Millisecond) // the cancelled wait must not report anything later
	if snap := r.SyncSnapshot(); snap.Phase != SyncSignedOut || snap.SignInFailure != SignInFailureGoogleCancelled {
		t.Fatalf("after the wait ended: phase %v failure %q", snap.Phase, snap.SignInFailure)
	}
}

func TestGoogleRefusalIsAFailure(t *testing.T) {
	f := newCodeServer(t)
	redirect := addGoogle(t, f, true)
	r, _ := codeApp(t, f)
	opened := make(chan string, 1)
	r.SyncSignInWithGoogle(func(u string) { opened <- u })
	waitString(t, opened, "the browser to open")
	redirect("", "access_denied")
	snap := waitSnap(t, r, "the failure", func(s SyncSnapshot) bool { return s.SignInFailure != "" })
	if snap.SignInFailure != SignInFailureGoogleFailed || snap.Phase != SyncSignedOut {
		t.Fatalf("failure %q phase %v", snap.SignInFailure, snap.Phase)
	}
}

func waitString(t *testing.T, ch <-chan string, what string) string {
	t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return ""
	}
}
