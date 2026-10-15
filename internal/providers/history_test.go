package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// The conversation after ask_user ends with a tool result, and the user's answer follows it. Both
// wire formats must carry that without a gap: this is what the follow-up turn sends.
func followUp() []*schema.Message {
	return []*schema.Message{
		schema.SystemMessage("sys"),
		schema.UserMessage("PR 12 is merged"),
		schema.AssistantMessage("", []schema.ToolCall{{ID: "call_1", Type: "function",
			Function: schema.FunctionCall{Name: "ask_user", Arguments: `{"question":"Which?","options":["a","b"]}`}}}),
		schema.ToolMessage(`{"ok":true}`, "call_1", schema.WithToolName("ask_user")),
		schema.UserMessage("a"),
	}
}

func TestAnthropicCarriesToolResultThenAnswer(t *testing.T) {
	srv, got := serve(t, Anthropic, http.StatusOK)
	r, _ := Config{}.With("anthropic", Entry{BaseURL: srv.URL}).Resolve("anthropic")
	m, err := NewModel(context.Background(), r, "k", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Generate(context.Background(), followUp()); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got.Body["messages"])
	t.Logf("anthropic messages: %s", b)
	msgs, _ := got.Body["messages"].([]any)
	if len(msgs) < 3 {
		t.Fatalf("messages = %s", b)
	}
	// user, assistant with tool_use, then a user turn that opens with the tool_result.
	if role(msgs[1]) != "assistant" || firstBlock(msgs[1], "tool_use") == nil {
		t.Errorf("no tool_use in %s", b)
	}
	if role(msgs[2]) != "user" || firstBlock(msgs[2], "tool_result") == nil {
		t.Errorf("tool_result does not follow tool_use: %s", b)
	}
	if got.Body["system"] == nil {
		t.Error("system prompt missing")
	}
}

func TestOpenAICarriesToolResultThenAnswer(t *testing.T) {
	srv, got := serve(t, OpenAI, http.StatusOK)
	r, _ := Config{}.With("openai", Entry{BaseURL: srv.URL}).Resolve("openai")
	m, err := NewModel(context.Background(), r, "k", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Generate(context.Background(), followUp()); err != nil {
		t.Fatal(err)
	}
	msgs, _ := got.Body["messages"].([]any)
	var roles []string
	for _, m := range msgs {
		roles = append(roles, role(m))
	}
	want := []string{"system", "user", "assistant", "tool", "user"}
	if len(roles) != len(want) {
		t.Fatalf("roles = %v", roles)
	}
	for i := range want {
		if roles[i] != want[i] {
			t.Fatalf("roles = %v, want %v", roles, want)
		}
	}
}

func role(m any) string {
	mm, _ := m.(map[string]any)
	s, _ := mm["role"].(string)
	return s
}

func firstBlock(m any, typ string) map[string]any {
	mm, _ := m.(map[string]any)
	blocks, _ := mm["content"].([]any)
	for _, b := range blocks {
		if bm, ok := b.(map[string]any); ok && bm["type"] == typ {
			return bm
		}
	}
	return nil
}
