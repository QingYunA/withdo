package syncserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// TestSignInRateLimit covers both fixed windows: every request counts per source, only failed
// passwords count per login name, X-Forwarded-For is trusted only from loopback, and a new window
// starts after ten minutes.
func TestSignInRateLimit(t *testing.T) {
	t.Run("the eleventh request from one source is 429", func(t *testing.T) {
		h, _ := newSignInCase(t, nil)
		for i := 0; i < 10; i++ {
			if w := postSignIn(t, h, "198.51.100.1:1111", "", signInBody()); w.Code != http.StatusOK {
				t.Fatalf("request %d = %d %s", i+1, w.Code, w.Body.String())
			}
		}
		w := postSignIn(t, h, "198.51.100.1:1111", "", signInBody())
		requireSignInError(t, w, http.StatusTooManyRequests, "too_many_attempts")
		if w.Header().Get("Retry-After") == "" {
			t.Fatal("429 has no Retry-After")
		}
	})

	t.Run("the eleventh failed password for one login is 429", func(t *testing.T) {
		h, z := newSignInCase(t, nil)
		for i := 0; i < 10; i++ {
			body := signInBody()
			body.Password = "wrong"
			// Distinct loopback X-Forwarded-For values keep the source window out of the way.
			requireSignInError(t, postSignIn(t, h, "127.0.0.1:1111",
				fmt.Sprintf("10.0.0.%d", i), body), http.StatusUnauthorized, "invalid_credentials")
		}
		w := postSignIn(t, h, "127.0.0.1:1111", "10.0.0.99", signInBody())
		requireSignInError(t, w, http.StatusTooManyRequests, "too_many_attempts")
		if w.Header().Get("Retry-After") == "" {
			t.Fatal("429 has no Retry-After")
		}
		z.mu.Lock()
		defer z.mu.Unlock()
		if z.created != 0 {
			t.Fatalf("failed passwords created %d sessions", z.created)
		}
	})

	t.Run("a non-loopback peer's X-Forwarded-For is ignored", func(t *testing.T) {
		h, _ := newSignInCase(t, nil)
		for i := 0; i < 10; i++ {
			if w := postSignIn(t, h, "198.51.100.2:1111",
				fmt.Sprintf("10.1.0.%d", i), signInBody()); w.Code != http.StatusOK {
				t.Fatalf("request %d = %d %s", i+1, w.Code, w.Body.String())
			}
		}
		requireSignInError(t, postSignIn(t, h, "198.51.100.2:1111", "10.1.0.99", signInBody()),
			http.StatusTooManyRequests, "too_many_attempts")
	})

	t.Run("the window expires", func(t *testing.T) {
		h, _ := newSignInCase(t, nil)
		now := time.Now()
		h.server.SetSignInClock(func() time.Time { return now })
		for i := 0; i < 10; i++ {
			if w := postSignIn(t, h, "198.51.100.3:1111", "", signInBody()); w.Code != http.StatusOK {
				t.Fatalf("request %d = %d %s", i+1, w.Code, w.Body.String())
			}
		}
		requireSignInError(t, postSignIn(t, h, "198.51.100.3:1111", "", signInBody()),
			http.StatusTooManyRequests, "too_many_attempts")
		now = now.Add(10*time.Minute + time.Second)
		if w := postSignIn(t, h, "198.51.100.3:1111", "", signInBody()); w.Code != http.StatusOK {
			t.Fatalf("after the window = %d %s", w.Code, w.Body.String())
		}
	})
}

// lockedBuffer lets the log test read what the logger wrote while other parallel tests write too.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestSignInLogsOnlyTheResult checks the one log line names the outcome and never the password,
// session token, tokens, code, state, PAT or login name.
func TestSignInLogsOnlyTheResult(t *testing.T) {
	h, z := newSignInCase(t, nil)
	var logs lockedBuffer
	old := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(old)

	w := postSignIn(t, h, "203.0.113.8:1111", "", signInBody())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	z.mu.Lock()
	var sessionToken string
	for _, sess := range z.sessions {
		sessionToken = sess.Token
	}
	z.mu.Unlock()
	if sessionToken == "" {
		t.Fatal("the fake recorded no session token")
	}
	if !strings.Contains(logs.String(), "sign-in: ok") {
		t.Fatalf("log = %q, want the ok line", logs.String())
	}
	for _, secret := range []string{
		"correct horse", sessionToken, "code-for-", "state-",
		"access-token", "refresh-token", "id-token", z.pat, z.loginName, "callback",
	} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("log %q contains %q", logs.String(), secret)
		}
	}
}

// TestPasswordSignInDisabled checks a server without the PAT file: the route is absent and the
// config does not advertise it.
func TestPasswordSignInDisabled(t *testing.T) {
	h := newHarness(t)
	w := postSignIn(t, h, "203.0.113.9:1111", "", signInBody())
	if w.Code != http.StatusNotFound {
		t.Fatalf("POST /v1/sign-in = %d %s, want 404", w.Code, w.Body.String())
	}
	_, body := h.do(t, http.MethodGet, "/v1/config", "", nil)
	if strings.Contains(string(body), "password_sign_in") {
		t.Fatalf("config advertises password sign-in: %s", body)
	}
}

// TestPasswordSignInConfig checks the flag appears only after EnablePasswordSignIn.
func TestPasswordSignInConfig(t *testing.T) {
	h, _ := newSignInCase(t, nil)
	resp, body := h.do(t, http.MethodGet, "/v1/config", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/config = %d %s", resp.StatusCode, body)
	}
	var cfg syncproto.ConfigResponse
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.PasswordSignIn {
		t.Fatalf("config = %s, want password_sign_in", body)
	}
}
