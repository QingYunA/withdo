package pbauth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

type memSecrets struct {
	mu sync.Mutex
	m  map[string]string
}

func (s *memSecrets) Get(k string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[k]
	if !ok {
		return "", syncclient.ErrSecretNotFound
	}
	return v, nil
}

func (s *memSecrets) Set(k, v string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]string{}
	}
	s.m[k] = v
	return nil
}

func (s *memSecrets) Delete(k string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, k)
	return nil
}

// fakeGoogleEndpoints stands in for Google's token and userinfo endpoints.
func fakeGoogleEndpoints(t *testing.T, email string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "authcode" || r.Form.Get("code_verifier") == "" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "expires_in": 3600})
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub": "g-1", "name": "Grace Hopper", "email": email, "email_verified": true,
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// addGoogle does by hand what the owner does in PocketBase's settings: a google provider. Its
// token and userinfo URLs point at the fake endpoints.
func addGoogle(t *testing.T, app core.App, google *httptest.Server) {
	t.Helper()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	users.OAuth2.Providers = []core.OAuth2ProviderConfig{{
		Name: "google", ClientId: "cid", ClientSecret: "secret",
		TokenURL: google.URL + "/token", UserInfoURL: google.URL + "/userinfo",
	}}
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}
}

// browse plays the browser: it follows Google's consent URL by going straight to the redirect
// with the state of the auth URL and a code.
func browse(t *testing.T) func(string) error {
	return func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		if q.Get("client_id") != "cid" || q.Get("code_challenge") == "" {
			t.Errorf("auth URL = %s, want the provider's query with PKCE", authURL)
		}
		go func() {
			noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			resp, err := noFollow.Get(q.Get("redirect_uri") + "?state=" + url.QueryEscape(q.Get("state")) + "&code=authcode")
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

func TestSignInWithGoogleAgainstPocketBase(t *testing.T) {
	app := openApp(t)
	google := fakeGoogleEndpoints(t, "grace@example.com")
	base := serve(t, app, pbauth.NewAuthenticator(app))
	secrets := &memSecrets{}
	source := syncclient.NewPocketBaseSource(base, secrets)

	if ok, err := source.GoogleAvailable(context.Background(), base); err != nil || ok {
		t.Fatalf("before the owner adds Google: available = %v, %v, want false", ok, err)
	}
	addGoogle(t, app, google)
	if ok, err := source.GoogleAvailable(context.Background(), base); err != nil || !ok {
		t.Fatalf("available = %v, %v, want true", ok, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := source.SignInWithGoogle(ctx, base, browse(t)); err != nil {
		t.Fatalf("sign in: %v", err)
	}
	token, err := source.Token(ctx)
	if err != nil || token == "" {
		t.Fatalf("token = %q, %v", token, err)
	}
	if status, subject := get(t, base+"/v1/whoami", token); status != http.StatusOK || subject == "" {
		t.Fatalf("whoami with the token = %d %q", status, subject)
	}
	user, err := app.FindAuthRecordByEmail("users", "grace@example.com")
	if err != nil {
		t.Fatalf("the sign-in should have created the user: %v", err)
	}
	if user.GetString("name") != "Grace Hopper" || !user.Verified() {
		t.Errorf("user = %q verified %v, want the Google name and verified", user.GetString("name"), user.Verified())
	}

	// A second sign-in is the same account.
	if err := source.SignInWithGoogle(ctx, base, browse(t)); err != nil {
		t.Fatalf("second sign in: %v", err)
	}
	if total, _ := app.CountRecords("users"); total != 1 {
		t.Fatalf("users = %d, want 1", total)
	}
}

func TestGoogleSignInJoinsAnExistingEmail(t *testing.T) {
	app := openApp(t)
	google := fakeGoogleEndpoints(t, "ada@example.com")
	subject, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	addGoogle(t, app, google)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	source := syncclient.NewPocketBaseSource(base, &memSecrets{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := source.SignInWithGoogle(ctx, base, browse(t)); err != nil {
		t.Fatalf("sign in: %v", err)
	}
	token, _ := source.Token(ctx)
	if status, got := get(t, base+"/v1/whoami", token); status != http.StatusOK || got != subject {
		t.Fatalf("whoami = %d %q, want the existing account %q", status, got, subject)
	}
	if total, _ := app.CountRecords("users"); total != 1 {
		t.Fatalf("users = %d, want 1", total)
	}
}
