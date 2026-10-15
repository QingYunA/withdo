package anthropic_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/providers/anthropic"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// api is a fake Messages endpoint: it records the last request and answers with reply.
type api struct {
	t      *testing.T
	srv    *httptest.Server
	path   string
	header http.Header
	body   map[string]any
}

func newAPI(t *testing.T, status int, reply string) *api {
	t.Helper()
	a := &api{t: t}
	a.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		a.body = nil
		_ = json.Unmarshal(b, &a.body)
		a.path, a.header = r.URL.Path, r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-should-retry", "false") // the SDK would retry 429 and 5xx with a delay
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(a.srv.Close)
	return a
}

func (a *api) model(t *testing.T) *anthropic.ChatModel {
	t.Helper()
	m, err := anthropic.New(anthropic.Config{APIKey: "sk-test", BaseURL: a.srv.URL, Model: "claude-test", MaxTokens: 512})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

const textReply = `{"id":"msg_1","type":"message","role":"assistant","model":"claude-test",
"content":[{"type":"text","text":"Hello, "},{"type":"text","text":"world."}],
"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":4}}`

const toolReply = `{"id":"msg_2","type":"message","role":"assistant","model":"claude-test",
"content":[{"type":"text","text":"Adding it."},
{"type":"tool_use","id":"toolu_1","name":"add_task","input":{"title":"Buy milk"}},
{"type":"tool_use","id":"toolu_2","name":"add_task","input":{"title":"Call mum"}}],
"stop_reason":"tool_use","usage":{"input_tokens":20,"output_tokens":9,"cache_read_input_tokens":5,"cache_creation_input_tokens":3}}`

func addTask() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "add_task", Desc: "Add a task",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"title": {Type: schema.String, Desc: "What to do", Required: true},
			"note":  {Type: schema.String, Desc: "More detail"},
		}),
	}
}

func list(v any) []any         { l, _ := v.([]any); return l }
func obj(v any) map[string]any { m, _ := v.(map[string]any); return m }

func TestGenerateSendsTheRequest(t *testing.T) {
	a := newAPI(t, 200, textReply)
	t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:1") // the environment must not redirect us
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "env-token")
	m := a.model(t)

	_, err := m.Generate(context.Background(), []*schema.Message{
		schema.SystemMessage("Be brief."),
		schema.UserMessage("Hi"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.path != "/v1/messages" {
		t.Errorf("path = %q", a.path)
	}
	if a.header.Get("x-api-key") != "sk-test" || a.header.Get("Authorization") != "" {
		t.Errorf("auth headers: x-api-key %q, Authorization %q", a.header.Get("x-api-key"), a.header.Get("Authorization"))
	}
	if a.body["model"] != "claude-test" || a.body["max_tokens"] != float64(512) {
		t.Errorf("model %v, max_tokens %v", a.body["model"], a.body["max_tokens"])
	}
	sys := list(a.body["system"])
	if len(sys) != 1 || obj(sys[0])["text"] != "Be brief." {
		t.Errorf("system = %v", a.body["system"])
	}
	msgs := list(a.body["messages"])
	if len(msgs) != 1 || obj(msgs[0])["role"] != "user" || obj(list(obj(msgs[0])["content"])[0])["text"] != "Hi" {
		t.Errorf("messages = %v", a.body["messages"])
	}
	if _, ok := a.body["tools"]; ok {
		t.Errorf("tools sent without any: %v", a.body["tools"])
	}
}

func TestGenerateMapsTheTextReply(t *testing.T) {
	a := newAPI(t, 200, textReply)
	msg, err := a.model(t).Generate(context.Background(), []*schema.Message{schema.UserMessage("Hi")})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Role != schema.Assistant || msg.Content != "Hello, world." || len(msg.ToolCalls) != 0 {
		t.Errorf("message = %+v", msg)
	}
	u := msg.ResponseMeta.Usage
	if msg.ResponseMeta.FinishReason != "end_turn" || u.PromptTokens != 10 || u.CompletionTokens != 4 || u.TotalTokens != 14 {
		t.Errorf("meta = %+v usage = %+v", msg.ResponseMeta, u)
	}
}

func TestToolsConversationAndToolCalls(t *testing.T) {
	a := newAPI(t, 200, toolReply)
	m, err := a.model(t).WithTools([]*schema.ToolInfo{addTask()})
	if err != nil {
		t.Fatal(err)
	}
	history := []*schema.Message{
		schema.UserMessage("Add two things"),
		{Role: schema.Assistant, Content: "On it.", ToolCalls: []schema.ToolCall{
			{ID: "toolu_a", Type: "function", Function: schema.FunctionCall{Name: "add_task", Arguments: `{"title":"A"}`}},
			{ID: "toolu_b", Type: "function", Function: schema.FunctionCall{Name: "add_task"}},
		}},
		schema.ToolMessage("added A", "toolu_a"),
		schema.ToolMessage("added B", "toolu_b"),
	}
	msg, err := m.Generate(context.Background(), history)
	if err != nil {
		t.Fatal(err)
	}

	tools := list(a.body["tools"])
	if len(tools) != 1 {
		t.Fatalf("tools = %v", a.body["tools"])
	}
	tool := obj(tools[0])
	schemaJSON := obj(tool["input_schema"])
	if tool["name"] != "add_task" || tool["description"] != "Add a task" || schemaJSON["type"] != "object" {
		t.Errorf("tool = %v", tool)
	}
	if props := obj(schemaJSON["properties"]); obj(props["title"])["type"] != "string" || obj(props["note"])["description"] != "More detail" {
		t.Errorf("properties = %v", schemaJSON["properties"])
	}
	if req := list(schemaJSON["required"]); len(req) != 1 || req[0] != "title" {
		t.Errorf("required = %v", schemaJSON["required"])
	}

	msgs := list(a.body["messages"])
	if len(msgs) != 3 {
		t.Fatalf("want user, assistant and one user message with both results, got %d: %v", len(msgs), msgs)
	}
	asst := list(obj(msgs[1])["content"])
	if obj(msgs[1])["role"] != "assistant" || len(asst) != 3 {
		t.Fatalf("assistant = %v", msgs[1])
	}
	if b := obj(asst[0]); b["type"] != "text" || b["text"] != "On it." {
		t.Errorf("text block = %v", b)
	}
	if b := obj(asst[1]); b["type"] != "tool_use" || b["id"] != "toolu_a" || obj(b["input"])["title"] != "A" {
		t.Errorf("tool_use block = %v", b)
	}
	if b := obj(asst[2]); b["id"] != "toolu_b" || len(obj(b["input"])) != 0 {
		t.Errorf("a call without arguments should send {}: %v", b)
	}
	results := list(obj(msgs[2])["content"])
	if obj(msgs[2])["role"] != "user" || len(results) != 2 {
		t.Fatalf("results = %v", msgs[2])
	}
	for i, want := range []string{"toolu_a", "toolu_b"} {
		if b := obj(results[i]); b["type"] != "tool_result" || b["tool_use_id"] != want {
			t.Errorf("tool_result %d = %v", i, b)
		}
	}

	if msg.Content != "Adding it." || len(msg.ToolCalls) != 2 {
		t.Fatalf("message = %+v", msg)
	}
	c := msg.ToolCalls[1]
	if c.ID != "toolu_2" || c.Function.Name != "add_task" || c.Function.Arguments != `{"title":"Call mum"}` {
		t.Errorf("tool call = %+v", c)
	}
	if msg.ResponseMeta.FinishReason != "tool_use" {
		t.Errorf("finish reason = %q", msg.ResponseMeta.FinishReason)
	}
	if u := msg.ResponseMeta.Usage; u.PromptTokens != 28 || u.PromptTokenDetails.CachedTokens != 5 || u.PromptTokenDetails.CacheWriteTokens != 3 || u.TotalTokens != 37 {
		t.Errorf("usage = %+v", u)
	}
}

func TestCallOptionsOverrideTheConfig(t *testing.T) {
	a := newAPI(t, 200, textReply)
	m, _ := a.model(t).WithTools([]*schema.ToolInfo{addTask()})
	other := &schema.ToolInfo{Name: "other", Desc: "Other"}
	_, err := m.Generate(context.Background(), []*schema.Message{schema.UserMessage("Hi")},
		model.WithModel("claude-other"), model.WithMaxTokens(1), model.WithTemperature(0.5),
		model.WithTools([]*schema.ToolInfo{other}))
	if err != nil {
		t.Fatal(err)
	}
	if a.body["model"] != "claude-other" || a.body["max_tokens"] != float64(1) || a.body["temperature"] != 0.5 {
		t.Errorf("body = %v", a.body)
	}
	tools := list(a.body["tools"])
	if len(tools) != 1 || obj(tools[0])["name"] != "other" || obj(obj(tools[0])["input_schema"])["type"] != "object" {
		t.Errorf("tools = %v", a.body["tools"])
	}
}

func TestWithToolsReturnsACopy(t *testing.T) {
	a := newAPI(t, 200, textReply)
	base := a.model(t)
	bound, err := base.WithTools([]*schema.ToolInfo{addTask()})
	if err != nil {
		t.Fatal(err)
	}
	if bound == model.ToolCallingChatModel(base) {
		t.Fatal("WithTools returned the receiver")
	}
	if _, err := base.Generate(context.Background(), []*schema.Message{schema.UserMessage("Hi")}); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.body["tools"]; ok {
		t.Error("the original model got the tools")
	}
	if _, err := base.WithTools(nil); err == nil {
		t.Error("binding no tools should fail")
	}
}

func TestStreamIsOneChunk(t *testing.T) {
	a := newAPI(t, 200, toolReply)
	sr, err := a.model(t).Stream(context.Background(), []*schema.Message{schema.UserMessage("Hi")})
	if err != nil {
		t.Fatal(err)
	}
	defer sr.Close()
	msg, err := sr.Recv()
	if err != nil || msg.Content != "Adding it." || len(msg.ToolCalls) != 2 {
		t.Fatalf("Recv = %+v, %v", msg, err)
	}
	if _, err := sr.Recv(); err != io.EOF {
		t.Errorf("second Recv = %v, want EOF", err)
	}
}

func TestBadInput(t *testing.T) {
	a := newAPI(t, 200, textReply)
	m := a.model(t)
	badCall := &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{
		{ID: "t", Function: schema.FunctionCall{Name: "x", Arguments: "{nope"}},
	}}
	if _, err := m.Generate(context.Background(), []*schema.Message{schema.UserMessage("Hi"), badCall}); err == nil {
		t.Error("tool arguments that are not JSON should fail")
	}
	if _, err := anthropic.New(anthropic.Config{MaxTokens: 1}); err == nil {
		t.Error("no model should fail")
	}
	if _, err := anthropic.New(anthropic.Config{Model: "m"}); err == nil {
		t.Error("no max tokens should fail")
	}
}
