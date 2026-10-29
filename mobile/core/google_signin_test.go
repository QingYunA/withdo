package core

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

// addGoogle mounts PocketBase's Google routes on the fake: auth-methods with a Google provider,
// the realtime stream (one client, "c1"), its subscription and auth-with-oauth2. The returned
// func plays the server's /api/oauth2-redirect.
func addGoogle(f *codeFake) func(code, errText string) {
	events := make(chan string, 2)
	f.mux.HandleFunc("GET /api/collections/users/auth-methods", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"oauth2": map[string]any{"enabled": true, "providers": []map[string]string{{
			"name": "google", "codeVerifier": "v", "authURL": "https://accounts.google.com/auth?state=x&redirect_uri=",
		}}}})
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
		payload, _ := json.Marshal(map[string]any{"exp": time.Now().Add(7 * 24 * time.Hour).Unix(), "id": "rec1"})
		id := "rec1"
		if f.foreignID != "" {
			id = f.foreignID
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig",
			"record": map[string]any{"id": id, "email": "ada@gmail.com", "name": "Ada"},
		})
	})
	f.mux.HandleFunc("GET /api/collections/_externalAuths/records", func(w http.ResponseWriter, _ *http.Request) {
		items := []map[string]string{}
		if f.linked {
			items = append(items, map[string]string{"provider": "google"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	})
	return func(code, errText string) {
		data, _ := json.Marshal(map[string]string{"state": "c1", "code": code, "error": errText})
		events <- string(data)
	}
}

func TestGoogleStartReturnsTheUrlAndFinishSignsIn(t *testing.T) {
	f := newCodeFake(t)
	redirect := addGoogle(f)
	phone, _ := codePhone(t, f)

	authURL, err := phone.GoogleStart()
	if err != nil {
		t.Fatalf("GoogleStart: %v", err)
	}
	u, err := url.Parse(authURL)
	if err != nil || u.Host != "accounts.google.com" || u.Query().Get("state") != "c1" ||
		u.Query().Get("redirect_uri") != f.URL+"/api/oauth2-redirect" {
		t.Fatalf("auth URL = %q, want Google's with the realtime client id as state", authURL)
	}
	if status := phoneStatus(t, phone); status.SignInStage != "google" {
		t.Fatalf("stage while waiting = %q, want google", status.SignInStage)
	}
	redirect("authcode", "")
	// The fake server's /v1/devices fails, so the sign-in ends right after Google's answer; what
	// matters is that the code was accepted and the link was tried.
	_, err = phone.GoogleFinish("phone")
	if err == nil {
		t.Fatal("GoogleFinish linked against a server that fails /v1/devices")
	}
	if state := codeState(t, phone); state.Failure == "google_failed" || state.Failure == "google_cancelled" {
		t.Fatalf("failure = %q; Google's answer was fine", state.Failure)
	}
	if status := phoneStatus(t, phone); status.SignInStage != "" {
		t.Fatalf("stage after = %q, want none", status.SignInStage)
	}
}

func TestGoogleCancelEndsTheWait(t *testing.T) {
	f := newCodeFake(t)
	addGoogle(f)
	phone, _ := codePhone(t, f)
	if _, err := phone.GoogleStart(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := phone.GoogleFinish("phone"); done <- err }()
	time.AfterFunc(50*time.Millisecond, phone.GoogleCancel)
	select {
	case err := <-done:
		if err == nil || err.Error() != "Google sign-in was cancelled" {
			t.Fatalf("err = %v, want the cancelled sentence", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GoogleFinish did not return after GoogleCancel")
	}
	if state := codeState(t, phone); state.Failure != "google_cancelled" {
		t.Fatalf("failure = %q, want google_cancelled", state.Failure)
	}
}

func TestGoogleRefusalIsAFailure(t *testing.T) {
	f := newCodeFake(t)
	redirect := addGoogle(f)
	phone, _ := codePhone(t, f)
	if _, err := phone.GoogleStart(); err != nil {
		t.Fatal(err)
	}
	redirect("", "access_denied")
	_, err := phone.GoogleFinish("phone")
	if err == nil || err.Error() != "Google sign-in didn't finish; try again" {
		t.Fatalf("err = %v, want the didn't-finish sentence", err)
	}
	if state := codeState(t, phone); state.Failure != "google_failed" {
		t.Fatalf("failure = %q, want google_failed", state.Failure)
	}
}

func TestGoogleFinishWithoutStartFails(t *testing.T) {
	f := newCodeFake(t)
	phone, _ := codePhone(t, f)
	if _, err := phone.GoogleFinish("phone"); err == nil {
		t.Fatal("GoogleFinish without GoogleStart should fail")
	}
}

func TestGoogleStartWithoutAProviderFails(t *testing.T) {
	f := newCodeFake(t)
	f.mux.HandleFunc("GET /api/collections/users/auth-methods", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"oauth2":{"enabled":false,"providers":[]}}`)
	})
	phone, _ := codePhone(t, f)
	if _, err := phone.GoogleStart(); err == nil {
		t.Fatal("GoogleStart should fail when the server lists no Google provider")
	}
	if state := codeState(t, phone); state.Failure != "google_failed" {
		t.Fatalf("failure = %q, want google_failed", state.Failure)
	}
}

func phoneStatus(t *testing.T, phone *Core) statusJSON {
	t.Helper()
	var status statusJSON
	if err := json.Unmarshal([]byte(phone.SyncStatus()), &status); err != nil {
		t.Fatal(err)
	}
	return status
}

// googleLinkPhone is a signed-in phone (the token is PocketBase's, the link to a device failed
// against the fake, which does not matter here).
func googleLinkPhone(t *testing.T, f *codeFake) *Core {
	t.Helper()
	phone, _ := codePhone(t, f)
	if err := phone.SignUp("ada@example.com", "a long password", "en"); err != nil {
		t.Fatal(err)
	}
	_, _ = phone.SignInWithCode("123456", "phone")
	return phone
}

func TestGoogleLinkStartFinishLinks(t *testing.T) {
	f := newCodeFake(t)
	redirect := addGoogle(f)
	phone := googleLinkPhone(t, f)
	if linked, err := phone.GoogleLinked(); err != nil || linked {
		t.Fatalf("GoogleLinked = %v, %v; want false", linked, err)
	}
	authURL, err := phone.GoogleLinkStart()
	if err != nil || authURL == "" {
		t.Fatalf("GoogleLinkStart = %q, %v", authURL, err)
	}
	redirect("authcode", "")
	if err := phone.GoogleLinkFinish(); err != nil {
		t.Fatalf("GoogleLinkFinish: %v", err)
	}
	f.linked = true
	if linked, err := phone.GoogleLinked(); err != nil || !linked {
		t.Fatalf("GoogleLinked after = %v, %v; want true", linked, err)
	}
}

func TestGoogleLinkKinds(t *testing.T) {
	t.Run("another account's Google", func(t *testing.T) {
		f := newCodeFake(t)
		redirect := addGoogle(f)
		phone := googleLinkPhone(t, f)
		f.foreignID = "someone-else"
		if _, err := phone.GoogleLinkStart(); err != nil {
			t.Fatal(err)
		}
		redirect("authcode", "")
		if err := phone.GoogleLinkFinish(); err == nil || err.Error() != "already_linked" {
			t.Fatalf("err = %v, want already_linked", err)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		f := newCodeFake(t)
		addGoogle(f)
		phone := googleLinkPhone(t, f)
		if _, err := phone.GoogleLinkStart(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- phone.GoogleLinkFinish() }()
		time.AfterFunc(50*time.Millisecond, phone.GoogleLinkCancel)
		select {
		case err := <-done:
			if err == nil || err.Error() != "cancelled" {
				t.Fatalf("err = %v, want cancelled", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("GoogleLinkFinish did not return after GoogleLinkCancel")
		}
	})
	t.Run("signed out", func(t *testing.T) {
		f := newCodeFake(t)
		addGoogle(f)
		phone, _ := codePhone(t, f)
		if _, err := phone.GoogleLinkStart(); err == nil || err.Error() != "signed_out" {
			t.Fatalf("err = %v, want signed_out", err)
		}
	})
}
