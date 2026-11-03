package core

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// fakeLLM is a sync server's /v1/config, /v1/llm/quota and /v1/llm/chat/completions: the chat
// route answers from a script, one reply per request.
type fakeLLM struct {
	*httptest.Server
	mu      sync.Mutex
	offered bool
	left    int
	resets  time.Time
	replies []fakeReply
	turns   []string
	auths   []string
	bodies  []string
}

// fakeReply is one answer of the chat route: a status and body, or a tool call, or text.
type fakeReply struct {
	status int
	body   string
	block  bool // hold the request until the client gives up
}

func toolCallReply(name, args string) fakeReply {
	call := map[string]any{
		"id": "call_1", "type": "function", "function": map[string]any{"name": name, "arguments": args},
	}
	return chatReply(map[string]any{"role": "assistant", "content": "", "tool_calls": []any{call}}, "tool_calls")
}

func textReply(text string) fakeReply {
	return chatReply(map[string]any{"role": "assistant", "content": text}, "stop")
}

func chatReply(message map[string]any, finish string) fakeReply {
	raw, _ := json.Marshal(map[string]any{
		"id": "c1", "object": "chat.completion", "model": "m",
		"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": finish}},
		"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
	})
	return fakeReply{status: 200, body: string(raw)}
}

func newFakeLLM(t *testing.T, replies ...fakeReply) *fakeLLM {
	t.Helper()
	f := &fakeLLM{offered: true, left: 86, resets: time.Now().Add(6 * time.Hour), replies: replies}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		offered, left, resets := f.offered, f.left, f.resets
		f.mu.Unlock()
		switch r.URL.Path {
		case "/v1/config":
			writeTestJSON(w, map[string]any{"llm": offered})
		case "/v1/llm/quota":
			writeTestJSON(w, map[string]any{"limit": 100, "left": left, "resets_at": resets.UTC().Format(time.RFC3339)})
		case "/v1/llm/chat/completions":
			raw, _ := io.ReadAll(r.Body)
			f.mu.Lock()
			f.turns = append(f.turns, r.Header.Get("X-Withdo-Turn"))
			f.auths = append(f.auths, r.Header.Get("Authorization"))
			f.bodies = append(f.bodies, string(raw))
			var reply fakeReply
			if len(f.replies) > 0 {
				reply, f.replies = f.replies[0], f.replies[1:]
			} else {
				reply = textReply("ok")
			}
			f.mu.Unlock()
			if reply.block {
				<-r.Context().Done()
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(reply.status)
			_, _ = io.WriteString(w, reply.body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeLLM) setLeft(left int) {
	f.mu.Lock()
	f.left = left
	f.mu.Unlock()
}

// assistantPhone opens a phone signed in to the fake server with a development token.
func assistantPhone(t *testing.T, f *fakeLLM) *Core {
	t.Helper()
	dir := t.TempDir()
	phone, err := open(dir, newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	if err := syncclient.SaveConfigAt(filepath.Join(dir, "sync.json"), syncclient.Config{Server: f.URL, Token: "dev-token"}); err != nil {
		t.Fatalf("write sync.json: %v", err)
	}
	return phone
}

// modelPhase reads ModelState until phase shows (the refresh runs in the background) or fails.
func modelPhase(t *testing.T, phone *Core, want string) modelStateJSON {
	t.Helper()
	var got modelStateJSON
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		raw, err := phone.ModelState()
		if err != nil {
			t.Fatalf("ModelState: %v", err)
		}
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode ModelState: %v (%s)", err, raw)
		}
		if got.Phase == want {
			return got
		}
	}
	t.Fatalf("ModelState phase = %q, want %q (%+v)", got.Phase, want, got)
	return got
}

// searchTitles are the titles Search finds for text.
func searchTitles(t *testing.T, phone *Core, text string) []string {
	t.Helper()
	raw, err := phone.Search(text)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	var tasks []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(raw), &tasks); err != nil {
		t.Fatalf("decode Search: %v (%s)", err, raw)
	}
	titles := make([]string, len(tasks))
	for i, task := range tasks {
		titles[i] = task.Title
	}
	return titles
}

func askReply(t *testing.T, phone *Core, text, place string) replyJSON {
	t.Helper()
	raw, err := phone.Ask(text, place)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	var out replyJSON
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode Ask: %v (%s)", err, raw)
	}
	return out
}

func TestModelStateSaysWhichModelWouldAnswer(t *testing.T) {
	// A phone that never chose a server nor signed in: the line says to sign in, with no request.
	fresh, err := open(t.TempDir(), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = fresh.Close() }()
	raw, err := fresh.ModelState()
	if err != nil || raw != `{"source":"none","phase":"signed_out","left":-1,"limit":0}` {
		t.Errorf("fresh ModelState = %s, %v", raw, err)
	}

	f := newFakeLLM(t)
	ready := modelPhase(t, assistantPhone(t, f), "ready")
	if ready.Source != "official" || ready.Left != 86 || ready.Limit != 100 {
		t.Errorf("ready = %+v", ready)
	}

	f.setLeft(0)
	used := modelPhase(t, assistantPhone(t, f), "used_up")
	if used.Source != "none" || used.Left != 0 {
		t.Errorf("used up = %+v", used)
	}

	f.mu.Lock()
	f.offered = false
	f.mu.Unlock()
	if hidden := modelPhase(t, assistantPhone(t, f), "hidden"); hidden.Source != "none" {
		t.Errorf("hidden = %+v", hidden)
	}

	// A server chosen but nobody signed in: it would be there after signing in.
	f.mu.Lock()
	f.offered = true
	f.mu.Unlock()
	dir := t.TempDir()
	out, err := open(dir, newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = out.Close() }()
	if err := syncclient.SaveConfigAt(filepath.Join(dir, "sync.json"), syncclient.Config{Server: f.URL}); err != nil {
		t.Fatal(err)
	}
	if got := modelPhase(t, out, "signed_out"); got.Source != "none" {
		t.Errorf("signed out = %+v", got)
	}
}

func TestAskCreatesATaskThroughTheOfficialModel(t *testing.T) {
	f := newFakeLLM(t,
		toolCallReply("create_task", `{"title":"打给房东","remind_at":"2026-10-21T09:00:00+08:00"}`),
		textReply("已添加。"),
	)
	phone := assistantPhone(t, f)
	modelPhase(t, phone, "ready")
	changed := make(chan struct{}, 8)
	phone.SetListener(listenerFunc(func() { changed <- struct{}{} }))

	reply := askReply(t, phone, "明天 9 点打给房东", `{"kind":"today","zone":"Asia/Shanghai","offset":28800}`)
	if reply.Problem != "" {
		t.Fatalf("problem %q: %s", reply.Problem, reply.Error)
	}
	if reply.Text != "已添加。" || len(reply.Receipts) != 1 {
		t.Fatalf("reply = %+v", reply)
	}
	r := reply.Receipts[0]
	if r.Title != "打给房东" || !r.Bell || r.Detail.Kind != "reminder" || r.Detail.At != "2026-10-21T09:00:00+08:00" || r.TaskID == 0 {
		t.Errorf("receipt = %+v", r)
	}
	if titles := searchTitles(t, phone, "房东"); !contains(titles, "打给房东") {
		t.Fatalf("the task was not stored: %v", titles)
	}
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Error("the listener was not told about the change")
	}

	f.mu.Lock()
	turns, auths, body := append([]string(nil), f.turns...), append([]string(nil), f.auths...), f.bodies[0]
	f.mu.Unlock()
	if len(turns) != 2 || turns[0] == "" || turns[0] != turns[1] {
		t.Errorf("turn ids = %v, want one id for the whole turn", turns)
	}
	if auths[0] != "Bearer dev-token" {
		t.Errorf("auth = %q", auths[0])
	}
	if !strings.Contains(body, "Asia/Shanghai") {
		t.Errorf("the prompt does not carry the phone's zone: %.300s", body)
	}

	if err := phone.Undo(r.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if titles := searchTitles(t, phone, "房东"); contains(titles, "打给房东") {
		t.Errorf("Undo left the task: %v", titles)
	}
}

func TestAskPutsANewTaskWhereTheRequestCameFrom(t *testing.T) {
	f := newFakeLLM(t, toolCallReply("create_task", `{"title":"buy milk"}`), textReply("ok"))
	phone := assistantPhone(t, f)
	modelPhase(t, phone, "ready")
	reply := askReply(t, phone, "buy milk", `{"kind":"calendar","day":"2026-11-05","zone":"Asia/Shanghai","offset":28800}`)
	if reply.Problem != "" || len(reply.Receipts) != 1 {
		t.Fatalf("reply = %+v", reply)
	}
	f.mu.Lock()
	body := f.bodies[0]
	f.mu.Unlock()
	if !strings.Contains(body, "Calendar page") || !strings.Contains(body, "2026-11-05") {
		t.Errorf("the prompt does not name the calendar day: %.400s", body)
	}
}

func TestAskReturnsAQuestionWithItsOptions(t *testing.T) {
	f := newFakeLLM(t, toolCallReply("ask_user", `{"question":"哪个项目?","options":["withdo","picpocket"]}`))
	phone := assistantPhone(t, f)
	modelPhase(t, phone, "ready")
	reply := askReply(t, phone, "修 bug", `{"kind":"today"}`)
	if reply.Question == nil || reply.Question.Text != "哪个项目?" || len(reply.Question.Options) != 2 || reply.Text != "" {
		t.Errorf("reply = %+v", reply)
	}
}

func TestAskNamesWhyTheModelDidNotAnswer(t *testing.T) {
	cases := map[string]struct {
		reply   fakeReply
		problem string
	}{
		"quota":      {fakeReply{status: 429, body: `{"code":"llm_quota"}`}, "quota"},
		"off":        {fakeReply{status: 503, body: `{"code":"llm_disabled"}`}, "off"},
		"signed out": {fakeReply{status: 401, body: `{"code":"unauthenticated"}`}, "signed_out"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFakeLLM(t, tc.reply)
			phone := assistantPhone(t, f)
			modelPhase(t, phone, "ready")
			reply := askReply(t, phone, "hi", `{"kind":"today"}`)
			if reply.Problem != tc.problem || reply.Error == "" || reply.Receipts == nil {
				t.Errorf("reply = %+v, want problem %q", reply, tc.problem)
			}
		})
	}
}

func TestAskAfterTheLastTurnReadsAsUsedUp(t *testing.T) {
	f := newFakeLLM(t, fakeReply{status: 429, body: `{"code":"llm_quota"}`})
	phone := assistantPhone(t, f)
	modelPhase(t, phone, "ready")
	f.setLeft(0)
	if reply := askReply(t, phone, "hi", `{}`); reply.Problem != "quota" {
		t.Fatalf("problem = %q", reply.Problem)
	}
	// The turn asks the server again, so the chip goes and the local rules take over.
	if got := modelPhase(t, phone, "used_up"); got.Source != "none" {
		t.Errorf("after the last turn: %+v", got)
	}
}

func TestStopAskCancelsTheTurnInFlight(t *testing.T) {
	f := newFakeLLM(t, fakeReply{block: true})
	phone := assistantPhone(t, f)
	modelPhase(t, phone, "ready")
	done := make(chan replyJSON, 1)
	go func() { done <- askReply(t, phone, "hi", `{"kind":"today"}`) }()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		f.mu.Lock()
		n := len(f.turns)
		f.mu.Unlock()
		if n > 0 {
			break
		}
	}
	// A second Ask while one runs is refused, not queued.
	if reply := askReply(t, phone, "again", `{}`); reply.Problem != "busy" {
		t.Errorf("second Ask problem = %q, want busy", reply.Problem)
	}
	phone.StopAsk()
	select {
	case reply := <-done:
		if reply.Problem != "cancelled" {
			t.Errorf("problem = %q, want cancelled", reply.Problem)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StopAsk did not end the turn")
	}
}

func TestAskRejectsAPlaceItCannotRead(t *testing.T) {
	phone, err := open(t.TempDir(), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = phone.Close() }()
	if _, err := phone.Ask("hi", "{not json"); err == nil {
		t.Error("Ask accepted a broken place")
	}
}

// listenerFunc is a Listener that only counts changes.
type listenerFunc func()

func (f listenerFunc) OnChange()     { f() }
func (listenerFunc) OnStatus(string) {}
