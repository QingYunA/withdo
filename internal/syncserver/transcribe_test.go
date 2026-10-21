package syncserver

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const fakeKey = "test-key-do-not-leak"

// testWAV is a mono PCM16 WAV of the given length at 16 kHz.
func testWAV(seconds float64) []byte {
	n := int(seconds * 16000)
	var b bytes.Buffer
	b.WriteString("RIFF")
	_ = binary.Write(&b, binary.LittleEndian, uint32(36+n*2))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(16000), uint32(32000), uint16(2), uint16(16)} {
		_ = binary.Write(&b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	_ = binary.Write(&b, binary.LittleEndian, uint32(n*2))
	b.Write(make([]byte, n*2))
	return b.Bytes()
}

// fakeMiMo answers like the recognizer and records what it was asked.
type fakeMiMo struct {
	*httptest.Server
	calls  atomic.Int32
	auth   atomic.Value
	audio  atomic.Value
	status int
	body   string
}

func newFakeMiMo(t *testing.T, status int, body string) *fakeMiMo {
	t.Helper()
	f := &fakeMiMo{status: status, body: body}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.auth.Store(r.Header.Get("Authorization"))
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content []struct {
					InputAudio struct {
						Data string `json:"data"`
					} `json:"input_audio"`
				} `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "mimo-v2.5-asr" || r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		f.audio.Store(req.Messages[0].Content[0].InputAudio.Data)
		w.WriteHeader(f.status)
		_, _ = io.WriteString(w, f.body)
	}))
	t.Cleanup(f.Close)
	return f
}

func newTestVoice(upstream *fakeMiMo, limit int) *voice {
	v := newVoice(StaticTokens{"tok": "alice", "tok2": "bob"})
	key := fakeKey
	base := ""
	if upstream != nil {
		base = upstream.URL + "/v1"
	} else {
		key = ""
	}
	v.configure(VoiceOptions{APIKey: key, BaseURL: base, DailyLimit: limit})
	return v
}

func post(v *voice, token, contentType string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/transcribe", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	v.handler(w, req)
	return w
}

func errorOf(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", w.Body.String(), err)
	}
	return out["error"]
}

const okAnswer = `{"choices":[{"message":{"content":" 明天九点给房东打电话 "}}],"usage":{"seconds":6}}`

func TestTranscribe(t *testing.T) {
	t.Run("returns the recognizer's text and sends the key and the audio", func(t *testing.T) {
		up := newFakeMiMo(t, 200, okAnswer)
		v := newTestVoice(up, 5)
		wav := testWAV(1)
		w := post(v, "tok", "audio/wav", wav)
		if w.Code != 200 {
			t.Fatalf("status %d %s", w.Code, w.Body)
		}
		var out map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		if out["text"] != "明天九点给房东打电话" {
			t.Fatalf("text = %q", out["text"])
		}
		if up.auth.Load() != "Bearer "+fakeKey {
			t.Fatalf("upstream auth = %v", up.auth.Load())
		}
		if got, _ := up.audio.Load().(string); got != "data:audio/wav;base64,"+base64.StdEncoding.EncodeToString(wav) {
			t.Fatal("upstream did not get the WAV as base64")
		}
	})

	t.Run("401 without a token", func(t *testing.T) {
		v := newTestVoice(newFakeMiMo(t, 200, okAnswer), 5)
		if w := post(v, "", "audio/wav", testWAV(1)); w.Code != 401 {
			t.Fatalf("status %d", w.Code)
		}
		if w := post(v, "nope", "audio/wav", testWAV(1)); w.Code != 401 {
			t.Fatalf("status %d", w.Code)
		}
	})

	t.Run("503 voice_disabled without a key", func(t *testing.T) {
		v := newTestVoice(nil, 5)
		w := post(v, "tok", "audio/wav", testWAV(1))
		if w.Code != 503 || errorOf(t, w) != "voice_disabled" {
			t.Fatalf("status %d %s", w.Code, w.Body)
		}
	})

	t.Run("rejects a body that is not a short WAV", func(t *testing.T) {
		up := newFakeMiMo(t, 200, okAnswer)
		v := newTestVoice(up, 50)
		cases := []struct {
			name, ct string
			body     []byte
			status   int
			code     string
		}{
			{"json type", "application/json", testWAV(1), 415, "unsupported_media"},
			{"no type", "", testWAV(1), 415, "unsupported_media"},
			{"garbage", "audio/wav", []byte("hello hello hello"), 400, "bad_audio"},
			{"empty", "audio/wav", nil, 400, "bad_audio"},
			{"too long", "audio/wav", testWAV(62), 413, "audio_too_long"},
			{"too big", "audio/wav", make([]byte, voiceMaxBytes+1), 413, "audio_too_large"},
		}
		for _, c := range cases {
			w := post(v, "tok", c.ct, c.body)
			if w.Code != c.status || errorOf(t, w) != c.code {
				t.Errorf("%s: status %d %s, want %d %s", c.name, w.Code, w.Body, c.status, c.code)
			}
		}
		if up.calls.Load() != 0 {
			t.Fatalf("a rejected body reached the recognizer %d times", up.calls.Load())
		}
	})

	t.Run("429 past the daily limit, per account, and a new UTC day resets it", func(t *testing.T) {
		v := newTestVoice(newFakeMiMo(t, 200, okAnswer), 2)
		day := time.Date(2026, 10, 21, 23, 0, 0, 0, time.UTC)
		v.now = func() time.Time { return day }
		for i := 0; i < 2; i++ {
			if w := post(v, "tok", "audio/wav", testWAV(1)); w.Code != 200 {
				t.Fatalf("request %d = %d", i+1, w.Code)
			}
		}
		w := post(v, "tok", "audio/wav", testWAV(1))
		if w.Code != 429 || errorOf(t, w) != "voice_quota" || w.Header().Get("Retry-After") == "" {
			t.Fatalf("third = %d %s retry %q", w.Code, w.Body, w.Header().Get("Retry-After"))
		}
		if w := post(v, "tok2", "audio/wav", testWAV(1)); w.Code != 200 {
			t.Fatalf("another account = %d", w.Code)
		}
		day = day.Add(2 * time.Hour)
		if w := post(v, "tok", "audio/wav", testWAV(1)); w.Code != 200 {
			t.Fatalf("next day = %d", w.Code)
		}
	})

	t.Run("502 when the recognizer fails, without the key, and the quota is refunded", func(t *testing.T) {
		var logs bytes.Buffer
		log.SetOutput(&logs)
		defer log.SetOutput(io.Discard)
		for name, up := range map[string]*fakeMiMo{
			"status": newFakeMiMo(t, 500, `{"error":"boom `+fakeKey+`"}`),
			"shape":  newFakeMiMo(t, 200, `{"nothing":true}`),
			"junk":   newFakeMiMo(t, 200, `not json`),
		} {
			v := newTestVoice(up, 1)
			for i := 0; i < 3; i++ { // each failure gives its request back
				w := post(v, "tok", "audio/wav", testWAV(1))
				if w.Code != 502 || errorOf(t, w) != "transcribe_failed" {
					t.Fatalf("%s: status %d %s", name, w.Code, w.Body)
				}
				if strings.Contains(w.Body.String(), fakeKey) {
					t.Fatalf("%s: the body leaks the key", name)
				}
			}
		}
		if strings.Contains(logs.String(), fakeKey) {
			t.Fatal("the log leaks the key")
		}
	})

	t.Run("502 when the recognizer is down", func(t *testing.T) {
		up := newFakeMiMo(t, 200, okAnswer)
		v := newTestVoice(up, 5)
		up.Close()
		w := post(v, "tok", "audio/wav", testWAV(1))
		if w.Code != 502 {
			t.Fatalf("status %d", w.Code)
		}
	})
}

func TestWavSeconds(t *testing.T) {
	if s, err := wavSeconds(testWAV(2.5)); err != nil || s != 2.5 {
		t.Fatalf("2.5 s WAV = %v, %v", s, err)
	}
	if _, err := wavSeconds(testWAV(1)[:30]); err == nil {
		t.Fatal("a truncated header is accepted")
	}
}
