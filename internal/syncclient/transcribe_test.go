package syncclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// voiceServer answers POST /v1/transcribe with a fixed status and body and records the request.
func voiceServer(t *testing.T, status int, body string) (*httptest.Server, *http.Request, *[]byte) {
	t.Helper()
	var seen http.Request
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = *r.Clone(context.Background())
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &seen, &got
}

func voiceClient(base string, tokens TokenSource) *Client {
	return &Client{base: base, tokens: tokens, http: http.DefaultClient}
}

type failingToken struct{ err error }

func (f failingToken) Token(context.Context) (string, error) { return "", f.err }

func TestTranscribe(t *testing.T) {
	t.Run("sends the WAV with the token and returns the text", func(t *testing.T) {
		srv, seen, got := voiceServer(t, 200, `{"text":" 明天九点给房东打电话 "}`)
		text, err := voiceClient(srv.URL, StaticToken("tok")).Transcribe(context.Background(), []byte("RIFFwav"))
		if err != nil || text != "明天九点给房东打电话" {
			t.Fatalf("got %q, %v", text, err)
		}
		if seen.Header.Get("Authorization") != "Bearer tok" || seen.Header.Get("Content-Type") != "audio/wav" ||
			seen.URL.Path != "/v1/transcribe" || string(*got) != "RIFFwav" {
			t.Fatalf("request: %v %v %q", seen.Header, seen.URL, *got)
		}
	})

	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"401 is not signed in", 401, `{"error":"unauthenticated"}`, ErrVoiceNotSignedIn},
		{"503 is voice off", 503, `{"error":"voice_disabled"}`, ErrVoiceOff},
		{"429 is quota", 429, `{"error":"voice_quota"}`, ErrVoiceQuota},
		{"502 is failed", 502, `{"error":"transcribe_failed"}`, ErrVoiceFailed},
		{"400 is failed", 400, `{"error":"bad_audio"}`, ErrVoiceFailed},
		{"empty text is no speech", 200, `{"text":"  "}`, ErrVoiceNoSpeech},
		{"junk is failed", 200, `nope`, ErrVoiceFailed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv, _, _ := voiceServer(t, c.status, c.body)
			_, err := voiceClient(srv.URL, StaticToken("tok")).Transcribe(context.Background(), []byte("x"))
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}

	t.Run("signed out never reaches the server", func(t *testing.T) {
		srv, seen, _ := voiceServer(t, 200, `{"text":"hi"}`)
		for _, ts := range []TokenSource{nil, StaticToken(""), failingToken{ErrSignedOut}} {
			if _, err := voiceClient(srv.URL, ts).Transcribe(context.Background(), []byte("x")); !errors.Is(err, ErrVoiceNotSignedIn) {
				t.Fatalf("err = %v", err)
			}
		}
		if seen.Method != "" {
			t.Fatal("a signed-out client sent a request")
		}
	})

	t.Run("an unreachable server is a network error", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 200, `{}`)
		base := srv.URL
		srv.Close()
		_, err := voiceClient(base, StaticToken("tok")).Transcribe(context.Background(), []byte("x"))
		if !errors.Is(err, ErrVoiceNetwork) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("a cancelled context is its own error", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 200, `{"text":"hi"}`)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := voiceClient(srv.URL, StaticToken("tok")).Transcribe(ctx, []byte("x"))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	})
}
