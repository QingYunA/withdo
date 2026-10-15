package views

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// assistRig is the Now screen with a scripted assistant behind the ask field.
type assistRig struct {
	tt  *ui.Tester
	m   *Main
	app *App
	rec *recorder
	fa  *data.FakeAssistant
	now time.Time
}

func newAssistRig(t *testing.T, fa *data.FakeAssistant) *assistRig {
	t.Helper()
	app, _, rec := newTestApp(t)
	app.Assistant, app.Models = fa, newFakeModels(t)
	r := &assistRig{app: app, rec: rec, fa: fa, now: fixedNow()}
	app.Now = func() time.Time { return r.now }
	r.tt, r.m, _ = newTesterFor(app, 1100)
	return r
}

func newFakeModels(t *testing.T) *data.FakeModels {
	t.Helper()
	return fakeModelsIn(t.TempDir())
}

// ask focuses the field, types and sends.
func (r *assistRig) ask(text string) {
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Type(text)
	r.tt.Key(0, ui.KeyEnter)
}

// answer lets the assistant's goroutine finish and shows what it said.
func (r *assistRig) answer(t *testing.T) {
	t.Helper()
	r.rec.drain(t)
	r.tt.Frame()
}

// tomorrowReminder is the detail of a reminder tomorrow at 9:00, which is 18.5 hours after the
// fixed clock.
func tomorrowReminder() data.Detail {
	return data.Detail{Kind: data.DetailReminder, At: fixedNow().Add(18*time.Hour + 30*time.Minute)}
}

func reminderReply() data.Reply {
	return data.Reply{
		Text: "Okay, I'll remind you tomorrow at 9.",
		Receipts: []data.Receipt{
			{ID: 11, TaskID: 5, Title: "Team meeting", Detail: tomorrowReminder(), Status: data.Todo, Bell: true},
		},
	}
}

func scripted(reply data.Reply, err error) func(string) (data.Reply, error) {
	return func(string) (data.Reply, error) { return reply, err }
}

func has(t *testing.T, tt *ui.Tester, texts ...string) {
	t.Helper()
	for _, s := range texts {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func hasNot(t *testing.T, tt *ui.Tester, texts ...string) {
	t.Helper()
	for _, s := range texts {
		if tt.HasText(s) {
			t.Errorf("unexpected %q in %q", s, tt.Texts())
		}
	}
}

func TestTurnShowsReplyAndReceiptWithUndo(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("明天九点提醒我开会")
	if r.m.ask.buf != "" {
		t.Errorf("the field still holds %q", r.m.ask.buf)
	}
	has(t, r.tt, "明天九点提醒我开会", "Thinking…")
	r.answer(t)
	has(t, r.tt, "明天九点提醒我开会", "Okay, I'll remind you tomorrow at 9.", "Team meeting", "Tomorrow 9:00 AM · reminder", "Undo")
	hasNot(t, r.tt, "Thinking…")
	if got := r.fa.Asked(); len(got) != 1 || got[0] != "明天九点提醒我开会" {
		t.Errorf("asked %v", got)
	}
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	if got := r.fa.Undone(); len(got) != 1 || got[0] != 11 {
		t.Errorf("undone %v", got)
	}
	has(t, r.tt, "Undone")
	if err := r.tt.Click("Undo: Team meeting"); err == nil {
		t.Error("the Undo button is still there")
	}
}

func TestReceiptRowOpensTheTask(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{
		Text:     "Done.",
		Receipts: []data.Receipt{{ID: 1, TaskID: 3, Title: "Fix tray race on launch", Detail: data.Detail{Kind: data.DetailDone}, Status: data.Done}},
	}, nil)})
	r.ask("close the tray task")
	r.answer(t)
	// The same title is also a row of Needs you, so click the one inside the sheet, which is higher.
	rect, _ := r.tt.Find("Marked done")
	x, y := center(rect)
	r.tt.ClickAt(x, y)
	if r.m.openID != 3 {
		t.Errorf("openID = %d, want 3", r.m.openID)
	}
}

func TestSheetClosesFourSecondsAfterATurnThatAskedNothing(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("remind me")
	r.answer(t)
	r.now = r.now.Add(3 * time.Second)
	r.tt.Frame()
	has(t, r.tt, "Okay, I'll remind you tomorrow at 9.")
	r.now = r.now.Add(2 * time.Second)
	r.tt.Frame()
	hasNot(t, r.tt, "Okay, I'll remind you tomorrow at 9.", "Undo")
}

func TestUndoKeepsTheSheetOpenLonger(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("remind me")
	r.answer(t)
	r.now = r.now.Add(3 * time.Second)
	r.tt.Frame()
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	r.now = r.now.Add(3 * time.Second)
	r.tt.Frame()
	has(t, r.tt, "Undone")
}

func TestQuestionOffersOptionsAndAnswersContinueTheConversation(t *testing.T) {
	replies := []data.Reply{
		{Question: &data.Question{Text: "Which task?", Options: []string{"Review PR docs", "Review PR code"}}},
		{Text: "Marked done."},
	}
	call := 0
	fa := &data.FakeAssistant{Ready: true, Script: func(string) (data.Reply, error) { call++; return replies[call-1], nil }}
	r := newAssistRig(t, fa)
	r.ask("PR 12 merged")
	r.answer(t)
	has(t, r.tt, "Which task?", "Review PR docs", "Review PR code", "Type an answer, or pick one below")
	// A question never times out.
	r.now = r.now.Add(time.Minute)
	r.tt.Frame()
	has(t, r.tt, "Which task?")
	if err := r.tt.Click("Review PR code"); err != nil {
		t.Fatal(err)
	}
	r.answer(t)
	if got := fa.Asked(); len(got) != 2 || got[1] != "Review PR code" {
		t.Errorf("asked %v", got)
	}
	has(t, r.tt, "PR 12 merged", "Review PR code", "Marked done.")
	hasNot(t, r.tt, "Review PR docs") // the answered question's buttons are gone
}

func TestTypedAnswerToAQuestion(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Question: &data.Question{Text: "What time?", Options: []string{"9:00", "13:30"}}}, nil)}
	r := newAssistRig(t, fa)
	r.ask("remind me about the dentist")
	r.answer(t)
	r.tt.Type("at 7pm")
	r.tt.Key(0, ui.KeyEnter)
	if got := fa.Asked(); len(got) != 2 || got[1] != "at 7pm" {
		t.Errorf("asked %v", got)
	}
}

func TestErrorShowsNeutralWordsAndRetry(t *testing.T) {
	calls := 0
	fa := &data.FakeAssistant{Ready: true, Script: func(string) (data.Reply, error) {
		calls++
		if calls == 1 {
			return data.Reply{}, errors.New("Couldn't reach the model. Is the address right and the server running?")
		}
		return data.Reply{Text: "Done."}, nil
	}}
	r := newAssistRig(t, fa)
	r.ask("do it")
	r.answer(t)
	has(t, r.tt, "do it", "Couldn't reach the model. Is the address right and the server running?", "Retry")
	if err := r.tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	r.answer(t)
	if got := fa.Asked(); len(got) != 2 || got[1] != "do it" {
		t.Errorf("asked %v", got)
	}
	has(t, r.tt, "Done.")
	hasNot(t, r.tt, "Retry", "Couldn't reach the model. Is the address right and the server running?")
}

func TestEscCancelsARequestAndGivesTheTextBack(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true, Block: make(chan struct{})}
	r := newAssistRig(t, fa)
	r.ask("a slow one")
	has(t, r.tt, "Thinking…")
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Key(0, ui.KeyEscape)
	r.answer(t) // the cancelled request returns context.Canceled
	hasNot(t, r.tt, "Thinking…")
	if r.m.convo.open {
		t.Error("the sheet stayed open after a cancelled request that changed nothing")
	}
	if r.m.ask.buf != "a slow one" {
		t.Errorf("the field holds %q, want the text back", r.m.ask.buf)
	}
}

func TestEnterWhileThinkingKeepsWhatIsTyped(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true, Block: make(chan struct{})}
	r := newAssistRig(t, fa)
	r.ask("first")
	r.tt.Type("second")
	r.tt.Key(0, ui.KeyEnter)
	if got := fa.Asked(); len(got) != 1 {
		t.Errorf("a second request went out: %v", got)
	}
	if r.m.ask.buf != "second" {
		t.Errorf("buffer %q", r.m.ask.buf)
	}
	close(fa.Block)
	r.answer(t)
}

func TestCancelAfterChangesKeepsTheReceipts(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.m.convo.turns = []convoTurn{{user: "add two things"}}
	r.m.convo.open, r.m.convo.thinking = true, true
	r.m.answered("add two things", data.Reply{Receipts: reminderReply().Receipts}, context.Canceled)
	r.tt.Frame()
	has(t, r.tt, "Team meeting", "Stopped.", "Undo")
}

func TestEscClosesTheSheetThenAnotherEscForgets(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)}
	r := newAssistRig(t, fa)
	r.ask("remind me")
	r.answer(t)
	r.tt.Key(0, ui.KeyEscape)
	hasNot(t, r.tt, "Okay, I'll remind you tomorrow at 9.")
	if fa.Resets() != 0 {
		t.Fatal("the first Esc reset the conversation")
	}
	r.tt.Key(0, ui.KeyEscape)
	if fa.Resets() != 1 || r.m.convo.live {
		t.Errorf("resets %d live %v after the second Esc", fa.Resets(), r.m.convo.live)
	}
	r.tt.Key(0, ui.KeyEscape)
	if fa.Resets() != 1 {
		t.Error("a third Esc reset again")
	}
}

func TestSheetPushesTheSectionsDown(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	before, _ := r.tt.Find("Needs you")
	r.ask("remind me")
	r.answer(t)
	after, _ := r.tt.Find("Needs you")
	if after.Y <= before.Y+40 {
		t.Errorf("Needs you moved from %v to %v; the sheet should push it down", before.Y, after.Y)
	}
}

func TestNoModelHintAppearsOnFocusAndOpensSettingsOnModels(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	hasNot(t, r.tt, "Set up")
	r.tt.Key(ui.Cmd, ui.KeyK)
	has(t, r.tt, "Connect a model to let me understand you —", "Set up")
	if err := r.tt.Click("Set up"); err != nil {
		t.Fatal(err)
	}
	if !r.m.settings || r.m.models.tab != tabModels {
		t.Fatalf("settings %v tab %v", r.m.settings, r.m.models.tab)
	}
	has(t, r.tt, "Base URL", "Anthropic", "Magpie")
}

func TestWithoutAModelEnterStillAddsToTheInbox(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	before := len(r.app.snap.tasks)
	r.ask("Water the plants")
	if len(r.app.snap.tasks) != before+1 || !r.tt.HasText("Added to Inbox") {
		t.Errorf("tasks %d, texts %q", len(r.app.snap.tasks), r.tt.Texts())
	}
	if got := r.fa.Asked(); len(got) != 0 {
		t.Errorf("the assistant was asked %v", got)
	}
}

func TestHintYieldsToTheAddedToInboxNotice(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.ask("Water the plants")
	has(t, r.tt, "Added to Inbox")
	hasNot(t, r.tt, "Set up")
	r.now = r.now.Add(2 * time.Second)
	r.tt.Frame()
	has(t, r.tt, "Set up")
	hasNot(t, r.tt, "Added to Inbox")
}

func TestNoHintWithoutAnAssistantAtAll(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.Key(ui.Cmd, ui.KeyK)
	hasNot(t, tt, "Set up")
}

func TestNoHintOnceAModelIsConnected(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Key(ui.Cmd, ui.KeyK)
	hasNot(t, r.tt, "Set up", "Connect a model to let me understand you —")
}

func TestAnswerRefreshesTheLists(t *testing.T) {
	store := data.NewFake(fixedNow)
	rec := &recorder{}
	app := NewApp(store, rec, testConfig())
	app.Now = fixedNow
	app.Assistant = &data.FakeAssistant{Ready: true, Script: func(string) (data.Reply, error) {
		_, err := store.CreateTask(data.CreateTaskInput{Title: "Made by the agent", Source: data.SourceUser})
		return data.Reply{Text: "Added."}, err
	}}
	r := &assistRig{app: app, rec: rec, now: fixedNow()}
	r.tt, r.m, _ = newTesterFor(app, 1100)
	r.ask("add it")
	r.answer(t)
	has(t, r.tt, "Made by the agent")
}
