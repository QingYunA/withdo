package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var oct14 = time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC) // 22:30 in Shanghai

func TestReminderInShanghai(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14,
		call("create_task", m{"title": "开会", "remind_at": "2026-10-15T09:00:00+08:00"}),
		say("好的，明天 9:00 提醒你开会。"))
	res := r.ask("明天九点提醒我开会")
	if res.Reply != "好的，明天 9:00 提醒你开会。" {
		t.Errorf("reply = %q", res.Reply)
	}
	if len(res.Receipts) != 1 {
		t.Fatalf("receipts = %+v", res.Receipts)
	}
	rc := res.Receipts[0]
	if rc.Title != "开会" || !rc.Bell {
		t.Errorf("receipt = %+v", rc)
	}
	wantMoment(t, rc.Detail, DetailReminder, "2026-10-15T09:00:00+08:00")
	tk := r.task(rc.TaskID)
	eq(t, "remind_at", str(tk.RemindAt), "2026-10-15T01:00:00.000Z")
	eq(t, "source", string(tk.Source), "user")
	eq(t, "status", string(tk.Status), "todo")
}

func TestReminderInNewYork(t *testing.T) {
	// The same wall-clock request in another zone stores another instant.
	r := newRig(t, "America/New_York", oct14,
		call("create_task", m{"title": "standup", "remind_at": "2026-10-15T09:00:00-04:00"}),
		say("Done."))
	res := r.ask("remind me about standup tomorrow 9am")
	eq(t, "remind_at", str(r.task(res.Receipts[0].TaskID).RemindAt), "2026-10-15T13:00:00.000Z")
	wantMoment(t, res.Receipts[0].Detail, DetailReminder, "2026-10-15T09:00:00-04:00")
}

func TestReminderAcrossDST(t *testing.T) {
	// 2026-03-08 is the day New York springs forward: 9:00 "tomorrow" is -04:00, not today's -05:00.
	at := time.Date(2026, 3, 8, 1, 0, 0, 0, time.UTC) // Sat 20:00 EST
	r := newRig(t, "America/New_York", at,
		call("create_task", m{"title": "call mom", "remind_at": "2026-03-08T09:00:00-04:00"}),
		say("ok"))
	res := r.ask("remind me to call mom tomorrow at 9")
	eq(t, "remind_at", str(r.task(res.Receipts[0].TaskID).RemindAt), "2026-03-08T13:00:00.000Z")
	wantMoment(t, res.Receipts[0].Detail, DetailReminder, "2026-03-08T09:00:00-04:00")
	// What a model would get wrong by adding a day to today's offset is a different instant.
	if naive := time.Date(2026, 3, 8, 14, 0, 0, 0, time.UTC); str(r.task(res.Receipts[0].TaskID).RemindAt) == domain.FormatTime(naive) {
		t.Error("stored the naive -05:00 instant")
	}
}

func TestTimeWithoutZoneIsRejectedAndRetried(t *testing.T) {
	var toolSaw string
	r := newRig(t, "Asia/Shanghai", oct14,
		call("create_task", m{"title": "开会", "remind_at": "2026-10-15T09:00:00"}),
		func(ctx context.Context, in []*schema.Message) (*schema.Message, error) {
			toolSaw = lastToolText(in)
			return call("create_task", m{"title": "开会", "remind_at": "2026-10-15T09:00:00+08:00"})(ctx, in)
		},
		say("ok"))
	res := r.ask("明天九点提醒我开会")
	contains(t, "tool result after the bad time", toolSaw, "RFC 3339 with a UTC offset")
	if len(res.Receipts) != 1 {
		t.Fatalf("want exactly the retried task, got %+v", res.Receipts)
	}
	tasks, _ := r.core.ListTasks(context.Background(), domain.TaskFilter{})
	if len(tasks) != 1 {
		t.Errorf("the rejected call must not create a task: %d tasks", len(tasks))
	}
}

func TestPastReminderIsRejected(t *testing.T) {
	var toolSaw string
	r := newRig(t, "Asia/Shanghai", oct14,
		call("create_task", m{"title": "x", "remind_at": "2026-10-14T09:00:00+08:00"}),
		func(_ context.Context, in []*schema.Message) (*schema.Message, error) {
			toolSaw = lastToolText(in)
			return schema.AssistantMessage("That time has passed.", nil), nil
		})
	res := r.ask("remind me at 9")
	contains(t, "tool result", toolSaw, "in the past")
	if len(res.Receipts) != 0 {
		t.Errorf("receipts = %+v", res.Receipts)
	}
}

func TestAmbiguousTaskAsksThenUpdates(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	a := r.mustTask(domain.CreateTaskInput{Title: "Review PR 12 docs"})
	b := r.mustTask(domain.CreateTaskInput{Title: "Review PR 12 code"})
	r.model.steps = []step{
		call("find_tasks", m{"query": "PR 12"}),
		call("ask_user", m{"question": "Which one is done?", "options": []string{"Review PR 12 docs", "Review PR 12 code"}}),
		// second turn: the user picked the second option
		call("complete_task", m{"task_id": b.ID}),
		say("Marked done."),
	}
	res := r.ask("PR 12 is merged, close that task")
	if res.Question == nil || res.Question.Text != "Which one is done?" || len(res.Question.Options) != 2 {
		t.Fatalf("question = %+v", res.Question)
	}
	if res.Reply != "" || len(res.Receipts) != 0 {
		t.Errorf("a question ends the turn: %+v", res)
	}
	contains(t, "find_tasks result", lastToolTextOfCall(r.model, 1), "Review PR 12 code")

	res = r.ask("Review PR 12 code")
	eq(t, "reply", res.Reply, "Marked done.")
	eq(t, "b status", string(r.task(b.ID).Status), "done")
	eq(t, "a status", string(r.task(a.ID).Status), "todo")
	// The follow-up was sent with the whole conversation, not as a fresh request.
	in := r.model.sent(2)
	var roles []schema.RoleType
	for _, msg := range in {
		roles = append(roles, msg.Role)
	}
	want := []schema.RoleType{schema.System, schema.User, schema.Assistant, schema.Tool, schema.Assistant, schema.Tool, schema.User}
	if len(roles) != len(want) {
		t.Fatalf("roles sent on the second turn = %v, want %v", roles, want)
	}
	for i := range want {
		if roles[i] != want[i] {
			t.Fatalf("roles sent on the second turn = %v, want %v", roles, want)
		}
	}
	eq(t, "first user line", in[1].Content, "PR 12 is merged, close that task")
	eq(t, "answer", in[6].Content, "Review PR 12 code")
}

func lastToolTextOfCall(f *fakeModel, i int) string { return lastToolText(f.sent(i)) }

func TestIdleResetForgetsTheConversation(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, say("one"), say("two"), say("three"))
	r.ask("first")
	r.clock.advance(5 * time.Minute)
	r.ask("second") // still remembered
	if n := len(r.model.sent(1)); n != 4 {
		t.Errorf("messages after 5 minutes = %d, want system+user+assistant+user", n)
	}
	r.clock.advance(11 * time.Minute)
	r.ask("third")
	if n := len(r.model.sent(2)); n != 2 {
		t.Errorf("messages after 11 idle minutes = %d, want system+user", n)
	}
}

func TestResetForgetsTheConversation(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, say("one"), say("two"))
	r.ask("first")
	r.agent.Reset()
	r.ask("second")
	if n := len(r.model.sent(1)); n != 2 {
		t.Errorf("messages after Reset = %d, want system+user", n)
	}
}

func TestCancelMidRequestKeepsReceipts(t *testing.T) {
	started := make(chan struct{})
	r := newRig(t, "Asia/Shanghai", oct14,
		call("create_task", m{"title": "kept"}),
		func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var res Result
	var err error
	go func() { res, err = r.agent.Ask(ctx, "add kept and then think", Scope{}); close(done) }()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Ask did not return after cancel")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if len(res.Receipts) != 1 {
		t.Fatalf("receipts after cancel = %+v", res.Receipts)
	}
	if err := r.agent.Undo(context.Background(), res.Receipts[0].ID); err != nil {
		t.Fatalf("undo after cancel: %v", err)
	}
	if tk := r.task(res.Receipts[0].TaskID); tk != nil {
		t.Errorf("task still there: %+v", tk)
	}
}

func TestProviderErrorSurfaces(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14, fail401, say("fine"))
	_, err := r.agent.Ask(context.Background(), "hi", Scope{})
	if err == nil {
		t.Fatal("want an error")
	}
	contains(t, "error", err.Error(), "401 invalid x-api-key")
	// A failed turn leaves the agent usable and the conversation empty.
	res := r.ask("again")
	eq(t, "reply", res.Reply, "fine")
	if n := len(r.model.sent(1)); n != 2 {
		t.Errorf("failed turn leaked into history: %d messages", n)
	}
}

func TestModelUnavailable(t *testing.T) {
	r := newRig(t, "Asia/Shanghai", oct14)
	boom := errors.New("no provider configured")
	r.agent.cfg.Model = func(context.Context) (model.ToolCallingChatModel, error) { return nil, boom }
	if _, err := r.agent.Ask(context.Background(), "hi", Scope{}); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

func TestSecondRequestWhileBusy(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	r := newRig(t, "Asia/Shanghai", oct14, func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
		close(started)
		<-release
		return schema.AssistantMessage("ok", nil), nil
	})
	done := make(chan struct{})
	go func() { _, _ = r.agent.Ask(context.Background(), "one", Scope{}); close(done) }()
	<-started
	if _, err := r.agent.Ask(context.Background(), "two", Scope{}); !errors.Is(err, ErrBusy) {
		t.Errorf("err = %v, want ErrBusy", err)
	}
	close(release)
	<-done
}

func eq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
