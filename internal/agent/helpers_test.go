package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/testutil"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// step is one scripted model answer. It sees everything the model was sent.
type step func(ctx context.Context, in []*schema.Message) (*schema.Message, error)

// fakeModel is a scripted model.ToolCallingChatModel: no network, answers come from steps in order.
type fakeModel struct {
	mu    sync.Mutex
	steps []step
	calls [][]*schema.Message // what each call was sent
}

func (f *fakeModel) Generate(ctx context.Context, in []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	f.mu.Lock()
	f.calls = append(f.calls, in)
	if len(f.steps) == 0 {
		f.mu.Unlock()
		return nil, fmt.Errorf("fake model: script ran out after %d calls", len(f.calls))
	}
	s := f.steps[0]
	f.steps = f.steps[1:]
	f.mu.Unlock()
	return s(ctx, in)
}

func (f *fakeModel) Stream(ctx context.Context, in []*schema.Message, o ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m, err := f.Generate(ctx, in, o...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{m}), nil
}

func (f *fakeModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) { return f, nil }

func (f *fakeModel) sent(i int) []*schema.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[i]
}

// say answers with plain text.
func say(text string) step {
	return func(context.Context, []*schema.Message) (*schema.Message, error) {
		return schema.AssistantMessage(text, nil), nil
	}
}

// call answers with a tool call whose arguments are the JSON of args.
func call(name string, args any) step {
	return func(context.Context, []*schema.Message) (*schema.Message, error) {
		b, _ := json.Marshal(args)
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID: "call-" + name, Type: "function",
			Function: schema.FunctionCall{Name: name, Arguments: string(b)},
		}}), nil
	}
}

// fail answers with an error, like a provider that is down.
func fail401(context.Context, []*schema.Message) (*schema.Message, error) {
	return nil, fmt.Errorf("401 invalid x-api-key")
}

// lastToolText is the text of the newest tool result the model was sent.
func lastToolText(in []*schema.Message) string {
	for i := len(in) - 1; i >= 0; i-- {
		if in[i].Role == schema.Tool {
			return in[i].Content
		}
	}
	return ""
}

// m is a shorthand for tool arguments.
type m map[string]any

type rig struct {
	t     *testing.T
	core  *domain.Core
	agent *Agent
	model *fakeModel
	clock *testClock
}

type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func loadZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Skipf("no tz database: %v", err)
	}
	return loc
}

// newRig is an Agent over a throwaway database with a fixed clock in the given zone.
func newRig(t *testing.T, zone string, at time.Time, steps ...step) *rig {
	t.Helper()
	testutil.IsolateGit(t)
	db, _ := testutil.OpenDB(t)
	clk := &testClock{t: at.In(loadZone(t, zone))}
	core := domain.New(db, domain.Options{Now: clk.now})
	fm := &fakeModel{steps: steps}
	ag := New(Config{
		Core: core, Now: clk.now,
		Model: func(context.Context) (model.ToolCallingChatModel, error) { return fm, nil },
	})
	return &rig{t: t, core: core, agent: ag, model: fm, clock: clk}
}

func (r *rig) task(id int64) *domain.Task {
	r.t.Helper()
	tk, err := r.core.GetTask(context.Background(), id)
	if err != nil {
		r.t.Fatal(err)
	}
	return tk
}

func (r *rig) ask(text string) Result {
	r.t.Helper()
	res, err := r.agent.Ask(context.Background(), text)
	if err != nil {
		r.t.Fatalf("Ask(%q): %v", text, err)
	}
	return res
}

func (r *rig) mustTask(in domain.CreateTaskInput) *domain.Task {
	r.t.Helper()
	if in.Title == "" {
		in.Title = "t"
	}
	tk, err := r.core.CreateTask(context.Background(), in)
	if err != nil {
		r.t.Fatal(err)
	}
	return tk
}

func contains(t *testing.T, what, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s = %q, want it to contain %q", what, got, want)
	}
}

func str(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// wantMoment checks that a detail is of kind and names the instant written as RFC 3339.
func wantMoment(t *testing.T, d Detail, kind DetailKind, rfc3339 string) {
	t.Helper()
	want, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != kind || !d.At.Equal(want) {
		t.Errorf("detail = %+v, want kind %v at %s", d, kind, rfc3339)
	}
}
