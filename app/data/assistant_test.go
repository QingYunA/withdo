package data

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// scriptModel answers from a list of functions, like the agent's own tests.
type scriptModel struct {
	steps []func(ctx context.Context) (*schema.Message, error)
}

func (s *scriptModel) Generate(ctx context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	f := s.steps[0]
	s.steps = s.steps[1:]
	return f(ctx)
}

func (s *scriptModel) Stream(ctx context.Context, in []*schema.Message, o ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m, err := s.Generate(ctx, in, o...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{m}), nil
}

func (s *scriptModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return s, nil
}

func toolCall(name, args string) func(context.Context) (*schema.Message, error) {
	return func(context.Context) (*schema.Message, error) {
		return schema.AssistantMessage("", []schema.ToolCall{{ID: "c", Type: "function", Function: schema.FunctionCall{Name: name, Arguments: args}}}), nil
	}
}

func newAssistant(t *testing.T, steps ...func(context.Context) (*schema.Message, error)) (*AgentAssistant, *Real) {
	t.Helper()
	r, _ := realPair(t, nil)
	sm := &scriptModel{steps: steps}
	ag := agent.New(agent.Config{
		Core: r.Core(), Model: func(context.Context) (model.ToolCallingChatModel, error) { return sm, nil },
	})
	mgr := newManager(t, providers.NewMemoryKeyring(), nil)
	return &AgentAssistant{Agent: ag, Provider: mgr}, r
}

func newManager(t *testing.T, keys providers.Keyring, client *http.Client) *providers.Manager {
	t.Helper()
	m, err := providers.NewManager(providers.NewStore(filepath.Join(t.TempDir(), providers.ConfigFile)), keys, client)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAgentAssistantMapsReceiptsAndWritesThroughTheStore(t *testing.T) {
	a, r := newAssistant(t,
		toolCall("create_task", `{"title":"开会","remind_at":"2099-01-02T09:00:00+08:00"}`),
		func(context.Context) (*schema.Message, error) { return schema.AssistantMessage("好的。", nil), nil })
	if a.Configured() {
		t.Error("configured with no provider")
	}
	reply, err := a.Ask(bg, "明天九点提醒我开会")
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != "好的。" || len(reply.Receipts) != 1 {
		t.Fatalf("reply %+v", reply)
	}
	rc := reply.Receipts[0]
	if rc.Title != "开会" || !rc.Bell || rc.Status != Todo || rc.TaskID == 0 || rc.Detail.Kind != agent.DetailReminder {
		t.Errorf("receipt %+v", rc)
	}
	tasks, _ := r.ListTasks(Filter{})
	if len(tasks) != 1 || tasks[0].RemindAt == nil || tasks[0].Source != SourceUser {
		t.Fatalf("tasks %+v", tasks)
	}
	if err := a.Undo(rc.ID); err != nil {
		t.Fatal(err)
	}
	if tasks, _ := r.ListTasks(Filter{}); len(tasks) != 0 {
		t.Errorf("Undo left %+v", tasks)
	}
	a.Reset()
}

func TestAgentAssistantMapsQuestions(t *testing.T) {
	a, _ := newAssistant(t, toolCall("ask_user", `{"question":"Which?","options":["a","b"]}`))
	reply, err := a.Ask(bg, "x")
	if err != nil || reply.Question == nil || reply.Question.Text != "Which?" || len(reply.Question.Options) != 2 {
		t.Errorf("reply %+v, %v", reply, err)
	}
}

func TestAgentAssistantErrorsAreInPlainWords(t *testing.T) {
	a, _ := newAssistant(t, func(context.Context) (*schema.Message, error) {
		return nil, errors.New("401 invalid x-api-key sk-secret")
	})
	_, err := a.Ask(bg, "x")
	if p := (*ProblemError)(nil); !errors.As(err, &p) || p.Problem != ProblemKeyRejected || p.Text != "The provider didn't accept the API key." {
		t.Errorf("err = %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Error("the key is in the message")
	}
}

func TestAgentAssistantKeepsCancelAndBusyRecognizable(t *testing.T) {
	started := make(chan struct{})
	a, _ := newAssistant(t, func(ctx context.Context) (*schema.Message, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithCancel(bg)
	done := make(chan error, 1)
	go func() { _, err := a.Ask(ctx, "x"); done <- err }()
	<-started
	if _, err := a.Ask(bg, "y"); !errors.Is(err, agent.ErrBusy) {
		t.Errorf("second Ask = %v", err)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled Ask = %v", err)
	}
}

func TestProviderSettings(t *testing.T) {
	keys := providers.NewMemoryKeyring()
	ps := &ProviderSettings{Provider: newManager(t, keys, nil)}
	list := ps.Providers()
	if len(list) != len(providers.Presets) || list[0].ID != "anthropic" || list[0].DefaultBaseURL != "https://api.anthropic.com" {
		t.Fatalf("providers %+v", list)
	}
	for _, p := range list {
		if p.Active {
			t.Errorf("%s active on a fresh install", p.ID)
		}
	}
	if ps.KeySaved("openai") {
		t.Error("key saved on a fresh install")
	}
	if err := ps.Save("openai", ProviderEdit{Model: "gpt-x", Key: "sk-1"}); err != nil {
		t.Fatal(err)
	}
	if !ps.KeySaved("openai") {
		t.Error("key not saved")
	}
	if v, _ := keys.Get("openai"); v != "sk-1" {
		t.Errorf("keychain holds %q", v)
	}
	if err := ps.SetActive("openai"); err != nil {
		t.Fatal(err)
	}
	for _, p := range ps.Providers() {
		if p.ID == "openai" && (!p.Active || p.Model != "gpt-x" || p.BaseURL != "https://api.openai.com/v1") {
			t.Errorf("openai = %+v", p)
		}
	}
	if err := ps.Save("custom", ProviderEdit{BaseURL: "http://x", Model: "m", Anthropic: true}); err != nil {
		t.Fatal(err)
	}
	for _, p := range ps.Providers() {
		if p.ID == "custom" && !p.Anthropic {
			t.Error("protocol not kept")
		}
	}
}

func TestProviderSettingsTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("x-api-key") != "sk-good" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"m","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer srv.Close()
	ps := &ProviderSettings{Provider: newManager(t, providers.NewMemoryKeyring(), nil)}
	d, err := ps.Test(bg, "anthropic", ProviderEdit{BaseURL: srv.URL, Key: "sk-good"})
	if err != nil || d <= 0 {
		t.Fatalf("Test = %v, %v", d, err)
	}
	_, err = ps.Test(bg, "anthropic", ProviderEdit{BaseURL: srv.URL, Key: "sk-bad"})
	if p := (*ProblemError)(nil); !errors.As(err, &p) || p.Problem != ProblemKeyRejected || p.Text != "The provider didn't accept the API key." {
		t.Errorf("err = %v", err)
	}
	if _, err := ps.Test(bg, "anthropic", ProviderEdit{BaseURL: srv.URL}); ProblemOf(err) != ProblemNoKey {
		t.Errorf("no key: %v", err)
	}
}

func TestEveryKindOfFailureIsRecognized(t *testing.T) {
	for want, err := range map[Problem]error{
		ProblemNotConfigured: providers.ErrNotConfigured,
		ProblemNoKey:         providers.ErrNoKey,
		ProblemCancelled:     context.Canceled,
		ProblemTimeout:       context.DeadlineExceeded,
		ProblemUnreachable:   errors.New("Post http://x: dial tcp 127.0.0.1:9: connect: connection refused"),
		ProblemKeyRejected:   errors.New("403 forbidden"),
		ProblemUnknownModel:  errors.New("404 model_not_found"),
		ProblemRateLimited:   errors.New("429 rate limit exceeded"),
		ProblemBusy:          agent.ErrBusy,
	} {
		if got := ProblemOf(problemFor(err)); got != want {
			t.Errorf("%v: problem %v, want %v", err, got, want)
		}
	}
	if got := problemFor(errors.New("something odd\nmore")); ProblemOf(got) != ProblemOther || got.Error() != "something odd" {
		t.Errorf("an unknown failure: %v", got)
	}
}

func TestFakeAssistantAndModels(t *testing.T) {
	block := make(chan struct{})
	f := &FakeAssistant{Ready: true, Block: block, Script: func(s string) (Reply, error) { return Reply{Text: "re: " + s}, nil }}
	ctx, cancel := context.WithCancel(bg)
	cancel()
	if _, err := f.Ask(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Errorf("blocked Ask = %v", err)
	}
	close(block)
	if r, _ := f.Ask(bg, "hi"); r.Text != "re: hi" {
		t.Errorf("reply %+v", r)
	}
	_ = f.Undo(3)
	f.Reset()
	if !f.Configured() || len(f.Asked()) != 2 || f.Undone()[0] != 3 || f.Resets() != 1 {
		t.Error("fake bookkeeping")
	}
	if r, err := (&FakeAssistant{}).Ask(bg, "x"); err != nil || r.Text != "" {
		t.Errorf("no script: %+v %v", r, err)
	}
	fm := &FakeModels{}
	if d, err := fm.Test(bg, "a", ProviderEdit{}); err != nil || d != 640*time.Millisecond {
		t.Errorf("default Test = %v, %v", d, err)
	}
	fm.TestFn = func(string, ProviderEdit) (time.Duration, error) { return 0, errors.New("no") }
	if _, err := fm.Test(bg, "a", ProviderEdit{}); err == nil {
		t.Error("TestFn ignored")
	}
}
