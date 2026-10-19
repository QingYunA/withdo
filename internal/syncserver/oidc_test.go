package syncserver_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

// fakeIssuer is an OIDC provider for tests: discovery, a JWKS and RS256 token signing. It serves
// enough of the provider metadata for go-oidc's discovery.
type fakeIssuer struct {
	*httptest.Server
	key     *rsa.PrivateKey
	keyID   string
	subject string
}

func newFakeIssuer(t *testing.T) *fakeIssuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIssuer{key: key, keyID: "test-key", subject: "subject-1"}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON := func(value any) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(value)
		}
		writeJSON(map[string]any{
			"issuer":                                f.Server.URL,
			"jwks_uri":                              f.Server.URL + "/jwks",
			"authorization_endpoint":                f.Server.URL + "/authorize",
			"token_endpoint":                        f.Server.URL + "/token",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: &f.key.PublicKey, KeyID: f.keyID, Algorithm: string(jose.RS256), Use: "sig",
		}}})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// token signs an RS256 JWT with the issuer's key, or with key when a test wants another signer.
func (f *fakeIssuer) token(t *testing.T, claims map[string]any, key *rsa.PrivateKey) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", f.keyID))
	if err != nil {
		t.Fatal(err)
	}
	object, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := object.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (f *fakeIssuer) claims(subject, audience string, expires time.Time) map[string]any {
	now := time.Now()
	return map[string]any{
		"iss": f.Server.URL, "sub": subject, "aud": audience,
		"iat": now.Unix(), "exp": expires.Unix(),
	}
}

func bearerRequest(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/pull", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

// TestS4a1ServerOIDC is acceptance criterion 1: a token signed by the issuer for the audience is
// accepted as its subject; a wrong audience, issuer, expiry or key and a missing token are 401;
// two subjects get separate workspaces.
func TestS4a1ServerOIDC(t *testing.T) {
	const audience = "sync-client"
	issuer := newFakeIssuer(t)
	ctx := context.Background()
	auth, err := syncserver.NewOIDCAuthenticator(ctx, issuer.URL, audience)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator: %v", err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("accepts a valid token as its subject", func(t *testing.T) {
		token := issuer.token(t, issuer.claims("subject-1", audience, time.Now().Add(time.Hour)), issuer.key)
		subject, err := auth.Subject(bearerRequest(token))
		if err != nil {
			t.Fatalf("Subject: %v", err)
		}
		if subject != "subject-1" {
			t.Fatalf("subject = %q, want subject-1", subject)
		}
	})

	t.Run("accepts a token that expired within the clock skew", func(t *testing.T) {
		token := issuer.token(t, issuer.claims("subject-1", audience, time.Now().Add(-30*time.Second)), issuer.key)
		if _, err := auth.Subject(bearerRequest(token)); err != nil {
			t.Fatalf("Subject: %v", err)
		}
	})

	bad := []struct {
		name  string
		token string
	}{
		{"wrong audience", issuer.token(t, issuer.claims("subject-1", "another-client", time.Now().Add(time.Hour)), issuer.key)},
		{"wrong issuer", issuer.token(t, map[string]any{
			"iss": "https://another-issuer.example", "sub": "subject-1", "aud": audience,
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		}, issuer.key)},
		{"expired", issuer.token(t, issuer.claims("subject-1", audience, time.Now().Add(-5*time.Minute)), issuer.key)},
		{"another key", issuer.token(t, issuer.claims("subject-1", audience, time.Now().Add(time.Hour)), otherKey)},
		{"missing token", ""},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.Subject(bearerRequest(tt.token))
			if !errors.Is(err, syncserver.ErrUnauthenticated) {
				t.Fatalf("Subject error = %v, want ErrUnauthenticated", err)
			}
		})
	}

	t.Run("two subjects have separate workspaces", func(t *testing.T) {
		h := newHarnessWithAuth(t, auth)
		subjectA := "alice@example.com"
		subjectB := "bob@example.com"
		tokenA := issuer.token(t, issuer.claims(subjectA, audience, time.Now().Add(time.Hour)), issuer.key)
		tokenB := issuer.token(t, issuer.claims(subjectB, audience, time.Now().Add(time.Hour)), issuer.key)

		a1 := h.device(t, tokenA, "alice-one")
		if code := h.seeded(t, tokenA, a1.DeviceID); code != http.StatusOK {
			t.Fatalf("end seeding = %d", code)
		}
		a2 := h.device(t, tokenA, "alice-two")
		b := h.device(t, tokenB, "bob-one")
		if a1.WorkspaceID != a2.WorkspaceID {
			t.Fatalf("one subject got workspaces %d and %d", a1.WorkspaceID, a2.WorkspaceID)
		}
		if a1.WorkspaceID == b.WorkspaceID {
			t.Fatalf("two subjects share workspace %d", a1.WorkspaceID)
		}
		if a1.WorkspaceID == 0 || b.WorkspaceID == 0 {
			t.Fatal("a workspace id is zero")
		}
		if a1.DeviceID == a2.DeviceID {
			t.Fatalf("two registrations got device id %q", a1.DeviceID)
		}
	})
}
