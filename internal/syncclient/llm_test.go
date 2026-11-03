package syncclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLLMQuota(t *testing.T) {
	t.Run("reads the quota with the token", func(t *testing.T) {
		srv, seen, _ := voiceServer(t, 200, `{"limit":100,"left":86,"resets_at":"2026-11-04T00:00:00Z"}`)
		q, err := voiceClient(srv.URL, StaticToken("tok")).LLMQuota(context.Background())
		if err != nil || q.Limit != 100 || q.Left != 86 || !q.ResetsAt.Equal(time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("got %+v, %v", q, err)
		}
		if seen.Header.Get("Authorization") != "Bearer tok" || seen.URL.Path != "/v1/llm/quota" || seen.Method != http.MethodGet {
			t.Fatalf("request: %v %v", seen.Header, seen.URL)
		}
	})
	cases := []struct {
		name   string
		status int
		want   error
	}{
		{"401 is not signed in", 401, ErrLLMNotSignedIn},
		{"503 is off", 503, ErrLLMOff},
		{"404 is an older server", 404, ErrLLMOff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _, _ := voiceServer(t, tc.status, `{"error":"x"}`)
			if _, err := voiceClient(srv.URL, StaticToken("tok")).LLMQuota(context.Background()); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
	t.Run("no token and a signed-out source", func(t *testing.T) {
		if _, err := voiceClient("http://127.0.0.1:1", nil).LLMQuota(context.Background()); !errors.Is(err, ErrLLMNotSignedIn) {
			t.Fatalf("err = %v", err)
		}
		if _, err := voiceClient("http://127.0.0.1:1", failingToken{ErrSignedOut}).LLMQuota(context.Background()); !errors.Is(err, ErrLLMNotSignedIn) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		if _, err := voiceClient(url, StaticToken("tok")).LLMQuota(context.Background()); !errors.Is(err, ErrLLMNetwork) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestLLMOfferedAndEndpoint(t *testing.T) {
	for body, want := range map[string]bool{`{"auth":"pocketbase","llm":true}`: true, `{"llm":false}`: false, `{"auth":"pocketbase"}`: false} {
		srv, _, _ := voiceServer(t, 200, body)
		if got := voiceClient(srv.URL, nil).LLMOffered(context.Background()); got != want {
			t.Fatalf("%s: LLMOffered = %v", body, got)
		}
	}
	c := voiceClient("http://127.0.0.1:9", StaticToken("tok"))
	if got := c.LLMBaseURL(); got != "http://127.0.0.1:9/v1/llm" {
		t.Fatalf("LLMBaseURL = %q", got)
	}
	if tok, err := c.BearerToken(context.Background()); err != nil || tok != "tok" {
		t.Fatalf("BearerToken = %q, %v", tok, err)
	}
}
