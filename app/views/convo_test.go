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

func TestAReceiptIsOneLineWithUndoAndTheUsersWordsAreNotEchoed(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("明天九点提醒我开会")
	if r.pageBuf() != "" {
		t.Errorf("the field still holds %q", r.pageBuf())
	}
	has(t, r.tt, "Thinking")
	hasNot(t, r.tt, "明天九点提醒我开会")
	r.answer(t)
	has(t, r.tt, "Added", "Team meeting", "· Tomorrow 9:00 AM · reminder", "Undo")
	hasNot(t, r.tt, "Thinking", "明天九点提醒我开会", "Okay, I'll remind you tomorrow at 9.") // the line says it
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

func receiptsOf(n int) data.Reply {
	var out data.Reply
	for i := 1; i <= n; i++ {
		out.Receipts = append(out.Receipts, data.Receipt{
			ID: 20 + i, TaskID: 5, Title: "Task " + itoa(i), Status: data.Todo,
			Detail: data.Detail{Kind: data.DetailDue, At: fixedNow().Add(2*24*time.Hour + 3*time.Hour + 30*time.Minute), Priority: "P1"},
		})
	}
	return out
}

func TestTheLineIsOnePerTaskAtMostThreeThenAndNMore(t *testing.T) {
	for _, tc := range []struct {
		n     int
		lines int
		more  string
	}{{1, 1, ""}, {2, 2, ""}, {3, 3, ""}, {4, 3, "and 1 more"}, {5, 3, "and 2 more"}} {
		r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(receiptsOf(tc.n), nil)})
		r.ask("add things")
		r.answer(t)
		shown := 0
		for i := 1; i <= tc.n; i++ {
			if r.tt.HasText("Task " + itoa(i)) {
				shown++
			}
		}
		undos := 0
		for _, s := range r.tt.Texts() {
			if s == "Undo" {
				undos++
			}
		}
		if shown != tc.lines || undos != tc.lines {
			t.Errorf("%d receipts: %d lines and %d Undo buttons, want %d each in %q", tc.n, shown, undos, tc.lines, r.tt.Texts())
		}
		if tc.more == "" {
			hasNot(t, r.tt, "more")
		} else {
			has(t, r.tt, tc.more)
		}
		has(t, r.tt, "Added", "· Fri · P1") // a task with a due time and a priority: what the words made of it
	}
}

func TestEachKindOfReceiptReadsAsOneLine(t *testing.T) {
	now := fixedNow()
	tests := []struct {
		name string
		r    data.Receipt
		verb string
		tail string
	}{
		{"added with nothing", data.Receipt{ID: 1, TaskID: 5, Title: "A", Detail: data.Detail{Kind: data.DetailAdded}}, "Added", ""},
		{"added in a project", data.Receipt{ID: 2, TaskID: 5, Title: "B", Detail: data.Detail{Kind: data.DetailAdded, Project: "adhd-todo"}}, "Added", "· adhd-todo"},
		{"done", data.Receipt{ID: 3, TaskID: 5, Title: "C", Detail: data.Detail{Kind: data.DetailDone}}, "Marked done", ""},
		{"moved", data.Receipt{ID: 4, TaskID: 5, Title: "D", Detail: data.Detail{Kind: data.DetailUpdated, Changes: []data.Change{{Kind: data.ChangeProject, Value: "adhd-todo"}}}}, "", "· Moved to adhd-todo"},
		{"reminder cleared", data.Receipt{ID: 5, TaskID: 5, Title: "E", Detail: data.Detail{Kind: data.DetailUpdated, Changes: []data.Change{{Kind: data.ChangeReminder}}}}, "", "· Reminder cleared"},
		{"rescheduled", data.Receipt{ID: 6, TaskID: 5, Title: "F", Detail: data.Detail{Kind: data.DetailUpdated, Changes: []data.Change{{Kind: data.ChangeDue, At: now.Add(26 * time.Hour)}}}}, "", "· Tomorrow 4:30 PM · due"},
		{"remembered", data.Receipt{ID: 7, Title: "Fitness is Life", Detail: data.Detail{Kind: data.DetailRemembered}}, "Remembered", ""},
		{"forgot", data.Receipt{ID: 8, Title: "Fitness", Detail: data.Detail{Kind: data.DetailForgot}}, "Forgot", ""},
	}
	for _, tc := range tests {
		r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Receipts: []data.Receipt{tc.r}}, nil)})
		r.ask("x")
		r.answer(t)
		has(t, r.tt, tc.r.Title)
		if tc.verb != "" {
			has(t, r.tt, tc.verb)
		}
		if tc.tail != "" {
			has(t, r.tt, tc.tail)
		}
		hasNot(t, r.tt, "Updated")
		if tc.name == "added with nothing" {
			hasNot(t, r.tt, "· Inbox") // on Now a new task goes to the Inbox without saying so
		}
	}
}

func TestAReceiptNamesTheProjectOnlyWhenItIsNotThePageItWasTypedOn(t *testing.T) {
	reply := data.Reply{Receipts: []data.Receipt{{ID: 1, TaskID: 5, Title: "Elsewhere", Detail: data.Detail{Kind: data.DetailAdded, Project: "picpocket"}}}}
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reply, nil)})
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.tt.Frame()
	r.ask("x")
	r.answer(t)
	has(t, r.tt, "· picpocket")
	here := data.Reply{Receipts: []data.Receipt{{ID: 2, TaskID: 5, Title: "Here", Detail: data.Detail{Kind: data.DetailAdded, Project: "adhd-todo"}}}}
	r2 := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(here, nil)})
	r2.m.goTo(nav{screen: screenProject, projectID: 1})
	r2.tt.Frame()
	r2.ask("x")
	r2.answer(t)
	has(t, r2.tt, "Here")
	hasNot(t, r2.tt, "· adhd-todo")
}

func TestReceiptRowOpensTheTask(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{
		Text:     "Done.",
		Receipts: []data.Receipt{{ID: 1, TaskID: 3, Title: "Fix tray race on launch", Detail: data.Detail{Kind: data.DetailDone}, Status: data.Done}},
	}, nil)})
	r.ask("close the tray task")
	r.answer(t)
	// The same title is also a row of Needs you, so click the one inside the line, which is higher.
	rect, _ := r.tt.Find("Marked done")
	x, y := center(rect)
	r.tt.ClickAt(x, y)
	if r.m.openID != 3 {
		t.Errorf("openID = %d, want 3", r.m.openID)
	}
}

func TestAChangeFreeTurnIsOneGreyLineWithTheSentence(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Text: "That one is already in adhd-todo, so I didn't add it again."}, nil)})
	r.ask("write the release notes")
	r.answer(t)
	has(t, r.tt, "That one is already in adhd-todo, so I didn't add it again.")
	hasNot(t, r.tt, "Undo", "write the release notes")
	r.now = r.now.Add(7 * time.Second)
	r.tt.Frame()
	hasNot(t, r.tt, "That one is already in adhd-todo, so I didn't add it again.")
}

func TestTheLineStaysSixSecondsFadesThenGoes(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("remind me")
	r.answer(t)
	r.now = r.now.Add(5 * time.Second)
	r.tt.Frame()
	has(t, r.tt, "Team meeting", "Undo")
	r.now = r.now.Add(900 * time.Millisecond) // in the last 400 ms of the 6 s it fades
	r.tt.Frame()
	has(t, r.tt, "Team meeting")
	r.now = r.now.Add(200 * time.Millisecond)
	r.tt.Frame()
	hasNot(t, r.tt, "Team meeting", "Undo")
	if r.m.convo.open {
		t.Error("the answer is still open after its line went")
	}
}

func TestTypingAgainEndsTheLineAtOnce(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("remind me")
	r.answer(t)
	has(t, r.tt, "Team meeting")
	r.tt.Type("n")
	hasNot(t, r.tt, "Team meeting", "Undo")
	r.tt.Key(0, ui.KeyBackspace)
	hasNot(t, r.tt, "Team meeting") // emptying the field does not bring it back
}

func TestAFieldTypedDuringTheWaitKeepsTheLineUntilItChanges(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Block: make(chan struct{}), Script: scripted(reminderReply(), nil)})
	r.ask("first")
	r.tt.Type("second thought")
	close(r.fa.Block)
	r.answer(t)
	has(t, r.tt, "Team meeting")
	r.tt.Type("!")
	hasNot(t, r.tt, "Team meeting")
}

func TestUndoKeepsTheLineLonger(t *testing.T) {
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

func TestCmdZUndoesTheLastChangeOfTheAssistantNewestFirst(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(receiptsOf(2), nil)})
	r.ask("add two")
	r.answer(t)
	r.tt.Key(ui.Cmd, ui.KeyZ) // the field is focused and empty
	if got := r.fa.Undone(); len(got) != 1 || got[0] != 22 {
		t.Fatalf("undone %v, want the last receipt (22)", got)
	}
	has(t, r.tt, "Undone")
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := r.fa.Undone(); len(got) != 2 || got[1] != 21 {
		t.Errorf("undone %v, want 22 then 21", got)
	}
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := r.fa.Undone(); len(got) != 2 {
		t.Errorf("a third Cmd+Z undid %v: there was nothing left", got)
	}
}

func TestCmdZWorksWhileTheConversationIsLiveEvenAfterTheLineWent(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("remind me")
	r.answer(t)
	r.now = r.now.Add(10 * time.Second)
	r.tt.Frame()
	hasNot(t, r.tt, "Team meeting")
	r.tt.Key(0, ui.KeyDown) // into the rows: the keys are the page's
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := r.fa.Undone(); len(got) != 1 || got[0] != 11 {
		t.Fatalf("undone %v", got)
	}
	has(t, r.tt, "Undone") // the line comes back to say so
}

func TestCmdZIsTheFieldsOwnWhileTextIsBeingTyped(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("remind me")
	r.answer(t)
	r.tt.Type("typing something new")
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := r.fa.Undone(); len(got) != 0 {
		t.Errorf("Cmd+Z undid the assistant's change %v while text was being typed", got)
	}
}

func TestCmdZWithNothingToUndoDoesNothing(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Key(ui.Cmd, ui.KeyZ)
	if got := r.fa.Undone(); len(got) != 0 {
		t.Errorf("undone %v", got)
	}
}

func TestAQuestionStaysUntilAnsweredAndOffersOptions(t *testing.T) {
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
	hasNot(t, r.tt, "PR 12 merged")
	// A question never times out.
	r.now = r.now.Add(time.Minute)
	r.tt.Frame()
	has(t, r.tt, "Which task?", "Review PR docs")
	if err := r.tt.Click("Review PR code"); err != nil {
		t.Fatal(err)
	}
	r.answer(t)
	if got := fa.Asked(); len(got) != 2 || got[1] != "Review PR code" {
		t.Errorf("asked %v", got)
	}
	has(t, r.tt, "Marked done.")
	hasNot(t, r.tt, "Which task?", "Review PR docs", "Review PR code") // the answered question's buttons are gone
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

func TestAnErrorStaysWithNeutralWordsAndRetry(t *testing.T) {
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
	has(t, r.tt, "Couldn't reach the model. Is the address right and the server running?", "Retry")
	hasNot(t, r.tt, "do it")
	r.now = r.now.Add(time.Minute)
	r.tt.Frame()
	has(t, r.tt, "Retry") // like a question, it does not fade
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
	has(t, r.tt, "Thinking")
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Key(0, ui.KeyEscape)
	r.answer(t) // the cancelled request returns context.Canceled
	hasNot(t, r.tt, "Thinking")
	if r.m.convo.open {
		t.Error("the sheet stayed open after a cancelled request that changed nothing")
	}
	if r.pageBuf() != "a slow one" {
		t.Errorf("the field holds %q, want the text back", r.pageBuf())
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
	if r.pageBuf() != "second" {
		t.Errorf("buffer %q", r.pageBuf())
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
	hasNot(t, r.tt, "Team meeting")
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

func TestTheLineGrowsTheDockUpAndTheSectionsStay(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	before, _ := r.tt.Find("From your agents")
	dock := belowBox(t, r)
	r.ask("remind me")
	r.answer(t)
	after, _ := r.tt.Find("From your agents")
	if after.Y != before.Y {
		t.Errorf("From your agents moved from %v to %v; the line grows up from the composer", before.Y, after.Y)
	}
	if got := belowBox(t, r); got <= dock+20 {
		t.Errorf("the dock went from %v to %v; the line should make room above the box", dock, got)
	}
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

func TestTheAddedToInboxNoticeFadesAfterOneAndAHalfSeconds(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.ask("Water the plants")
	has(t, r.tt, "Added to Inbox")
	r.now = r.now.Add(2 * time.Second)
	r.tt.Frame()
	hasNot(t, r.tt, "Added to Inbox")
}

func TestAnswerRefreshesTheLists(t *testing.T) {
	store := demoStore(t, fixedNow, true)
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
