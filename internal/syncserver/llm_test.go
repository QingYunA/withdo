package syncserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const llmFakeKey = "llm-key-do-not-leak"

// fakeUpstream answers like the chat completions API and records what it was asked.
type fakeUpstream struct {
	*httptest.Server
	calls  atomic.Int32
	auth   atomic.Value
	body   atomic.Value
	status int
	reply  string
}

func newFakeUpstream(t *testing.T, status int, reply string) *fakeUpstream {
	t.Helper()
	f := &fakeUpstream{status: status, reply: reply}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.auth.Store(r.Header.Get("Authorization"))
		raw, _ := io.ReadAll(r.Body)
		f.body.Store(string(raw))
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		if strings.Contains(string(raw), `"stream":true`) {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, chunk := range []string{"data: {\"a\":1}\n\n", "data: [DONE]\n\n"} {
				_, _ = io.WriteString(w, chunk)
				w.(http.Flusher).Flush()
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(f.status)
		_, _ = io.WriteString(w, f.reply)
	}))
	t.Cleanup(f.Close)
	return f
}

func newTestLLM(up *fakeUpstream, turns int) *llm {
	l := newLLM(StaticTokens{"tok": "alice", "tok2": "bob"})
	opts := LLMOptions{DailyTurns: turns, Model: "official-model"}
	if up != nil {
		opts.APIKey, opts.BaseURL = llmFakeKey, up.URL
	}
	l.configure(opts)
	return l
}

func llmChat(l *llm, token, turn, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/llm/chat/completions", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if turn != "" {
		req.Header.Set(HeaderLLMTurn, turn)
	}
	w := httptest.NewRecorder()
	l.chat(w, req)
	return w
}

func llmCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", w.Body.String(), err)
	}
	return out["code"]
}

const chatBody = `{"model":"gpt-whatever","messages":[{"role":"user","content":"明天九点给房东打电话"}],"temperature":0.2}`
const chatAnswer = `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`

func TestLLMForwardsWithTheServersKeyAndModel(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 5)
	w := llmChat(l, "tok", "t1", chatBody)
	if w.Code != 200 || w.Body.String() != chatAnswer {
		t.Fatalf("answer = %d %s", w.Code, w.Body)
	}
	if up.auth.Load() != "Bearer "+llmFakeKey {
		t.Fatalf("upstream auth = %v", up.auth.Load())
	}
	var sent map[string]any
	if err := json.Unmarshal([]byte(up.body.Load().(string)), &sent); err != nil {
		t.Fatal(err)
	}
	if sent["model"] != "official-model" || sent["temperature"] != 0.2 || sent["messages"] == nil {
		t.Fatalf("upstream got %v", sent)
	}
	if got := w.Header().Get(HeaderLLMTurnsLeft); got != "4" {
		t.Fatalf("turns left = %q", got)
	}
}

func TestLLMStreamPassesThrough(t *testing.T) {
	up := newFakeUpstream(t, 200, "")
	w := llmChat(newTestLLM(up, 5), "tok", "t1", `{"stream":true,"messages":[]}`)
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/event-stream" ||
		w.Body.String() != "data: {\"a\":1}\n\ndata: [DONE]\n\n" {
		t.Fatalf("stream = %d %q %q", w.Code, w.Header().Get("Content-Type"), w.Body)
	}
}

func TestLLMAuthAndOff(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 5)
	if w := llmChat(l, "", "", chatBody); w.Code != 401 {
		t.Fatalf("no token = %d", w.Code)
	}
	if w := llmChat(l, "nope", "", chatBody); w.Code != 401 {
		t.Fatalf("bad token = %d", w.Code)
	}
	off := newTestLLM(nil, 5)
	w := llmChat(off, "tok", "", chatBody)
	if w.Code != 503 || llmCode(t, w) != CodeLLMDisabled {
		t.Fatalf("off = %d %s", w.Code, w.Body)
	}
	if up.calls.Load() != 0 {
		t.Fatal("the upstream was called without a session")
	}
}

func TestLLMBodyLimitAndShape(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 5)
	big := `{"messages":"` + strings.Repeat("x", llmMaxBody) + `"}`
	if w := llmChat(l, "tok", "t1", big); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("big = %d", w.Code)
	}
	if w := llmChat(l, "tok", "t1", `[1,2]`); w.Code != 400 {
		t.Fatalf("array = %d", w.Code)
	}
	if w := llmChat(l, "tok", "t1", `nope`); w.Code != 400 {
		t.Fatalf("junk = %d", w.Code)
	}
	if up.calls.Load() != 0 {
		t.Fatal("a refused body reached the upstream")
	}
	// None of those was counted.
	if _, _, left := l.take("alice", "check"); left != 4 {
		t.Fatalf("left = %d", left)
	}
}

func TestLLMCountsTurnsNotRequests(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 2)
	for i := 0; i < 5; i++ { // one turn, several model calls
		if w := llmChat(l, "tok", "turn-a", chatBody); w.Code != 200 {
			t.Fatalf("request %d = %d", i, w.Code)
		}
	}
	if w := llmChat(l, "tok", "turn-b", chatBody); w.Code != 200 || w.Header().Get(HeaderLLMTurnsLeft) != "0" {
		t.Fatalf("second turn = %d left %q", w.Code, w.Header().Get(HeaderLLMTurnsLeft))
	}
	w := llmChat(l, "tok", "turn-c", chatBody)
	if w.Code != 429 || llmCode(t, w) != CodeLLMQuota || w.Header().Get(HeaderLLMTurnsLeft) != "0" || w.Header().Get("Retry-After") == "" {
		t.Fatalf("third turn = %d %s %v", w.Code, w.Body, w.Header())
	}
	// A turn already begun may go on after the quota is used up: it was paid for.
	if w := llmChat(l, "tok", "turn-b", chatBody); w.Code != 200 {
		t.Fatalf("a begun turn = %d", w.Code)
	}
	// Another account has its own count.
	if w := llmChat(l, "tok2", "turn-c", chatBody); w.Code != 200 {
		t.Fatalf("bob = %d", w.Code)
	}
}

func TestLLMNoHeaderIsATurnOfItsOwn(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 2)
	for i := 0; i < 2; i++ {
		if w := llmChat(l, "tok", "", chatBody); w.Code != 200 {
			t.Fatalf("request %d = %d", i, w.Code)
		}
	}
	if w := llmChat(l, "tok", "", chatBody); w.Code != 429 {
		t.Fatalf("third = %d", w.Code)
	}
	if w := llmChat(l, "tok2", strings.Repeat("x", llmTurnIDMax+1), chatBody); w.Code != 200 {
		t.Fatalf("a long id is a turn of its own: %d", w.Code)
	}
}

func TestLLMHardCaps(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 100)
	for i := 0; i < LLMRequestsPerTurn; i++ {
		if w := llmChat(l, "tok", "one", chatBody); w.Code != 200 {
			t.Fatalf("request %d = %d", i, w.Code)
		}
	}
	if w := llmChat(l, "tok", "one", chatBody); w.Code != 429 || llmCode(t, w) != CodeLLMQuota {
		t.Fatalf("past the per-turn cap = %d %s", w.Code, w.Body)
	}
}

func TestLLMNewUTCDayResets(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 1)
	day := time.Date(2026, 11, 3, 23, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return day }
	if w := llmChat(l, "tok", "a", chatBody); w.Code != 200 {
		t.Fatalf("first = %d", w.Code)
	}
	if w := llmChat(l, "tok", "b", chatBody); w.Code != 429 {
		t.Fatalf("second = %d", w.Code)
	}
	day = day.Add(2 * time.Hour)
	if w := llmChat(l, "tok", "b", chatBody); w.Code != 200 {
		t.Fatalf("next day = %d", w.Code)
	}
}

func TestLLMUpstreamFailures(t *testing.T) {
	cases := []struct {
		name         string
		status, want int
		wantCode     string
		refunded     bool
	}{
		{"the server's key is refused", 401, 502, "llm_upstream", true},
		{"payment required", 402, 502, "llm_upstream", true},
		{"busy", 429, 429, "llm_busy", true},
		{"upstream down", 500, 502, "llm_upstream", true},
		{"bad request goes through", 400, 400, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := newFakeUpstream(t, tc.status, `{"error":{"message":"nope"}}`)
			l := newTestLLM(up, 3)
			w := llmChat(l, "tok", "t1", chatBody)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
			if tc.wantCode != "" && llmCode(t, w) != tc.wantCode {
				t.Fatalf("code = %s", w.Body)
			}
			if strings.Contains(w.Body.String(), llmFakeKey) {
				t.Fatal("the key leaked")
			}
			l.mu.Lock()
			left := l.left("alice")
			l.mu.Unlock()
			if want := map[bool]int{true: 3, false: 2}[tc.refunded]; left != want {
				t.Fatalf("turns left = %d, want %d", left, want)
			}
		})
	}
	t.Run("unreachable", func(t *testing.T) {
		up := newFakeUpstream(t, 200, chatAnswer)
		l := newTestLLM(up, 3)
		up.Close()
		if w := llmChat(l, "tok", "t1", chatBody); w.Code != 502 {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestLLMQuotaRoute(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 100)
	l.now = func() time.Time { return time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC) }
	get := func(token string, l *llm) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/llm/quota", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		l.quota(w, req)
		return w
	}
	if w := get("", l); w.Code != 401 {
		t.Fatalf("no token = %d", w.Code)
	}
	for i := 0; i < 14; i++ {
		llmChat(l, "tok", "turn-"+strconv.Itoa(i), chatBody)
	}
	w := get("tok", l)
	var out struct {
		Limit    int       `json:"limit"`
		Left     int       `json:"left"`
		ResetsAt time.Time `json:"resets_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 {
		t.Fatalf("quota = %d %s", w.Code, w.Body)
	}
	if out.Limit != 100 || out.Left != 86 || !out.ResetsAt.Equal(time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("quota = %+v", out)
	}
	if w := get("tok", newTestLLM(nil, 5)); w.Code != 503 {
		t.Fatalf("off = %d", w.Code)
	}
}

func TestLLMNeverLogsBodiesOrKeys(t *testing.T) {
	up := newFakeUpstream(t, 200, chatAnswer)
	l := newTestLLM(up, 5)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(io.Discard) })
	llmChat(l, "tok", "t1", chatBody)
	llmChat(l, "tok", "t2", `{"messages":[{"content":"secret words"}`)
	if out := logs.String(); strings.Contains(out, llmFakeKey) || strings.Contains(out, "房东") || strings.Contains(out, "secret") ||
		strings.Contains(out, "tok") || !strings.Contains(out, "account=alice") || !strings.Contains(out, "status=200") {
		t.Fatalf("log = %q", out)
	}
}
