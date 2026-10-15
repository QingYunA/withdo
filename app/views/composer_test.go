package views

import (
	"errors"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The composer: the one input at the top of Now, of every project and of the Inbox, and at the
// bottom of the inspector. Enter sends, Shift+Enter breaks the line, and it tells the assistant
// where it is.

func (r *assistRig) pageBuf() string      { return r.m.cmp[surfacePage].buf }
func (r *assistRig) inspectorBuf() string { return r.m.cmp[surfaceInspector].buf }

// inInspector opens a task with its composer focused, as "Ask about this task…" does, and types.
func (r *assistRig) inInspector(t *testing.T, task int, text string) {
	t.Helper()
	r.m.OpenTask(task)
	r.m.focusInspectorComposer = true
	for i := 0; i < 5; i++ {
		r.tt.Frame()
	}
	r.tt.Type(text)
}

func TestEnterSendsToTheAssistantWithThePlaceOfThePage(t *testing.T) {
	tests := []struct {
		name string
		nav  nav
		want data.Place
	}{
		{"now", nav{}, data.Place{Kind: data.PlaceNow}},
		{"inbox", nav{screen: screenInbox}, data.Place{Kind: data.PlaceInbox}},
		{"project", nav{screen: screenProject, projectID: 1}, data.Place{Kind: data.PlaceProject, ProjectID: 1}},
		{"another project", nav{screen: screenProject, projectID: 2}, data.Place{Kind: data.PlaceProject, ProjectID: 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fa := &data.FakeAssistant{Ready: true}
			r := newAssistRig(t, fa)
			r.m.goTo(tc.nav)
			r.tt.Frame()
			r.ask("buy milk")
			r.rec.drain(t)
			if got := fa.Places(); len(got) != 1 || got[0] != tc.want {
				t.Errorf("sent from %+v, want %+v", got, tc.want)
			}
			if got := fa.Asked(); len(got) != 1 || got[0] != "buy milk" {
				t.Errorf("asked %v", got)
			}
			if r.pageBuf() != "" {
				t.Errorf("the field still holds %q", r.pageBuf())
			}
		})
	}
}

func TestTheInspectorComposerSendsWithTheTaskAsItsPlace(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	r := newAssistRig(t, fa)
	r.inInspector(t, 5, "move it to friday afternoon")
	if r.inspectorBuf() != "move it to friday afternoon" || r.pageBuf() != "" {
		t.Fatalf("typed into the wrong field: inspector %q, page %q", r.inspectorBuf(), r.pageBuf())
	}
	r.tt.Key(0, ui.KeyEnter)
	r.rec.drain(t)
	if got := fa.Places(); len(got) != 1 || got[0] != (data.Place{Kind: data.PlaceTask, TaskID: 5}) {
		t.Errorf("places %+v", got)
	}
	if r.inspectorBuf() != "" {
		t.Errorf("the field still holds %q", r.inspectorBuf())
	}
}

func TestAskAboutThisTaskFlagFocusesTheInspectorComposerOnce(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.inInspector(t, 5, "x")
	if r.inspectorBuf() != "x" {
		t.Fatalf("the inspector's composer did not take the focus: %q", r.inspectorBuf())
	}
	if r.m.focusInspectorComposer {
		t.Error("the flag stays set after the focus was taken")
	}
}

func TestShiftEnterBreaksTheLineAndSendsNothing(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	r := newAssistRig(t, fa)
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Type("one")
	r.tt.Key(ui.Shift, ui.KeyEnter)
	r.tt.Type("two")
	if r.pageBuf() != "one\ntwo" {
		t.Errorf("buffer %q", r.pageBuf())
	}
	if len(fa.Asked()) != 0 {
		t.Errorf("Shift+Enter sent %v", fa.Asked())
	}
	r.tt.Key(0, ui.KeyEnter)
	r.rec.drain(t)
	if got := fa.Asked(); len(got) != 1 || got[0] != "one\ntwo" {
		t.Errorf("asked %q: the whole text goes to the assistant", got)
	}
}

func TestTheComposerGrowsFromTwoToSixLinesThenScrolls(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.tt.Frame()
	chipY := func() float32 {
		rect, ok := r.tt.Find("# adhd-todo")
		if !ok {
			t.Fatalf("no chip in %q", r.tt.Texts())
		}
		return rect.Y
	}
	r.tt.Key(ui.Cmd, ui.KeyK)
	empty := chipY()
	lines := func(n int) {
		r.tt.Key(ui.Cmd, ui.KeyA)
		r.tt.Key(0, ui.KeyBackspace)
		for i := 1; i <= n; i++ {
			r.tt.Type("line")
			if i < n {
				r.tt.Key(ui.Shift, ui.KeyEnter)
			}
		}
	}
	lines(2)
	two := chipY()
	lines(4)
	four := chipY()
	lines(6)
	six := chipY()
	lines(9)
	nine := chipY()
	if two != empty {
		t.Errorf("two lines moved the bar: %v -> %v; the empty composer is two lines tall", empty, two)
	}
	if !(four > two && six > four) {
		t.Errorf("the composer does not grow with its text: 2 lines %v, 4 lines %v, 6 lines %v", two, four, six)
	}
	if nine != six {
		t.Errorf("it kept growing past six lines: %v -> %v", six, nine)
	}
}

func TestWithoutAModelTheTextBecomesAPlainTaskInThePlace(t *testing.T) {
	tests := []struct {
		name      string
		nav       nav
		project   *int
		wantNotes string
	}{
		{"now goes to the inbox", nav{}, nil, "Added to Inbox"},
		{"inbox", nav{screen: screenInbox}, nil, "Added to Inbox"},
		{"project", nav{screen: screenProject, projectID: 1}, data.IntPtr(1), "Added to adhd-todo"},
	}
	for _, tc := range tests {
		for _, assistant := range []data.Assistant{&data.FakeAssistant{Ready: false}, nil} {
			t.Run(tc.name, func(t *testing.T) {
				app, _, _ := newTestApp(t)
				app.Assistant = assistant
				tt, m, _ := newTesterFor(app, 1100)
				m.goTo(tc.nav)
				tt.Frame()
				tt.Key(ui.Cmd, ui.KeyK)
				tt.Type("Water the plants")
				tt.Key(0, ui.KeyEnter)
				var got *data.Task
				for _, task := range app.snap.tasks {
					if task.Title == "Water the plants" {
						got = &task
					}
				}
				if got == nil {
					t.Fatal("no task was created")
				}
				if (got.ProjectID == nil) != (tc.project == nil) || (got.ProjectID != nil && *got.ProjectID != *tc.project) {
					t.Errorf("task is in project %v, want %v", got.ProjectID, tc.project)
				}
				if got.Source != data.SourceUser {
					t.Errorf("source %v", got.Source)
				}
				has(t, tt, tc.wantNotes)
				if m.cmp[surfacePage].buf != "" {
					t.Errorf("the field still holds %q", m.cmp[surfacePage].buf)
				}
			})
		}
	}
}

func TestWithoutAModelAMultiLineTextIsOneTitle(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Type("one")
	r.tt.Key(ui.Shift, ui.KeyEnter)
	r.tt.Type("two")
	r.tt.Key(0, ui.KeyEnter)
	for _, task := range r.app.snap.tasks {
		if task.Title == "one two" {
			return
		}
	}
	t.Errorf("no task titled %q", "one two")
}

func TestTheInspectorComposerIsHiddenWithoutAModel(t *testing.T) {
	for _, assistant := range []data.Assistant{&data.FakeAssistant{Ready: false}, nil} {
		app, _, _ := newTestApp(t)
		app.Assistant = assistant
		tt, m, _ := newTesterFor(app, 1100)
		m.OpenTask(5)
		tt.Frame()
		hasNot(t, tt, "Change anything — “move it to friday afternoon”")
	}
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.m.OpenTask(5)
	r.tt.Frame()
	has(t, r.tt, "Change anything — “move it to friday afternoon”")
}

func TestThePlaceholderSaysWhereTheComposerIs(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	has(t, r.tt, "What should I remember? Write it the way you'd say it — “review PR 12 before standup, it's urgent”")
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.tt.Frame()
	has(t, r.tt, "What's next in adhd-todo? “fix the login redirect loop, p1, friday”", "# adhd-todo")
	r.m.goTo(nav{screen: screenInbox})
	r.tt.Frame()
	has(t, r.tt, "What should I remember? Write it the way you'd say it — “review PR 12 before standup, it's urgent”", "Inbox")
}

func TestTheChipNamesTheModelOrOffersSetup(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	mustNil(t, r.app.Models.Save("anthropic", data.ProviderEdit{Model: "claude-haiku-5-5", Key: "sk-x"}))
	mustNil(t, r.app.Models.SetActive("anthropic"))
	r.tt.Frame()
	has(t, r.tt, "Assistant · Haiku 5.5")
	hasNot(t, r.tt, "Set up the assistant")

	off := newAssistRig(t, &data.FakeAssistant{Ready: false})
	has(t, off.tt, "Set up the assistant")
	if err := off.tt.Click("Set up the assistant"); err != nil {
		t.Fatal(err)
	}
	if len(off.rec.panes) != 1 || off.rec.panes[0] != PaneAssistant {
		t.Errorf("asked to show %v, want the Assistant pane", off.rec.panes)
	}
}

func TestTheSendButtonSendsAndIsNeutralWhileEmpty(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	r := newAssistRig(t, fa)
	if err := r.tt.Click("Send"); err == nil && len(fa.Asked()) != 0 {
		t.Error("an empty composer sent something")
	}
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Type("remind me to stretch")
	if err := r.tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	r.rec.drain(t)
	if got := fa.Asked(); len(got) != 1 || got[0] != "remind me to stretch" {
		t.Errorf("asked %v", got)
	}
}

func TestCmdKFocusesThePageComposerWhereItIsAndGoesToNowElsewhere(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("remind me")
	if m.nav != (nav{screen: screenProject, projectID: 1}) {
		t.Errorf("Cmd+K left the project for %+v", m.nav)
	}
	if got := m.cmp[surfacePage].buf; got != "remind me" {
		t.Errorf("typing went elsewhere: %q", got)
	}

	m.goTo(nav{screen: screenLogbook})
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyK)
	if m.nav != (nav{}) {
		t.Errorf("Cmd+K on the Logbook went to %+v, want Now", m.nav)
	}
	tt.Type("again")
	if got := m.cmp[surfacePage].buf; got != "again" {
		t.Errorf("typing went elsewhere: %q", got)
	}
}

func TestCompletedAndSearchHaveNoComposer(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenLogbook})
	tt.Frame()
	hasNot(t, tt, "What should I remember? Write it the way you'd say it — “review PR 12 before standup, it's urgent”")
}

func TestEscClosesTheReplyThenLeavesTheComposer(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.ask("明天九点提醒我开会")
	r.answer(t)
	has(t, r.tt, "Team meeting")
	r.tt.Type("draft")
	r.tt.Key(0, ui.KeyEscape)
	if r.m.convo.open {
		t.Error("the first Esc left the reply open")
	}
	if r.pageBuf() != "draft" {
		t.Errorf("Esc touched the draft: %q", r.pageBuf())
	}
	r.tt.Key(0, ui.KeyEscape) // nothing left to close: the field lets go of the keys
	r.tt.Frame()
	r.tt.Type("zz")
	if r.pageBuf() != "draft" {
		t.Errorf("the field kept the keyboard after Esc: %q", r.pageBuf())
	}
	r.tt.Key(0, ui.KeyDown)
	if r.m.sel == 0 {
		t.Error("the arrows did not reach the rows after Esc")
	}
}

func TestTheReplyShowsUnderTheComposerThatSentItAndGoesWhenThePageDoes(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)})
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.tt.Frame()
	r.ask("remind me at 9")
	r.answer(t)
	has(t, r.tt, "Okay, I'll remind you tomorrow at 9.", "Team meeting")
	if !above(t, r.tt, "# adhd-todo", "Okay, I'll remind you tomorrow at 9.") {
		t.Error("the reply is not under the composer")
	}
	r.m.goTo(nav{screen: screenInbox})
	r.tt.Frame()
	hasNot(t, r.tt, "Okay, I'll remind you tomorrow at 9.")
}

func TestTheInspectorReplyShowsInTheInspector(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Text: "Moved it to Friday 2 PM."}, nil)})
	r.inInspector(t, 5, "friday afternoon")
	r.tt.Key(0, ui.KeyEnter)
	r.answer(t)
	rect, ok := r.tt.Find("Moved it to Friday 2 PM.")
	if !ok {
		t.Fatalf("no reply in %q", r.tt.Texts())
	}
	if rect.X < 1100-360 {
		t.Errorf("the reply is at x=%v, outside the inspector", rect.X)
	}
}

func TestRetryAndOptionsKeepThePlaceOfTheTurn(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	calls := 0
	fa.Script = func(string) (data.Reply, error) {
		calls++
		if calls == 1 {
			return data.Reply{}, errors.New("boom")
		}
		return data.Reply{Question: &data.Question{Text: "Which?", Options: []string{"first", "second"}}}, nil
	}
	r := newAssistRig(t, fa)
	r.m.goTo(nav{screen: screenProject, projectID: 2})
	r.tt.Frame()
	r.ask("do it")
	r.answer(t)
	if err := r.tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	r.answer(t)
	if err := r.tt.Click("second"); err != nil {
		t.Fatal(err)
	}
	r.answer(t)
	want := data.Place{Kind: data.PlaceProject, ProjectID: 2}
	for i, p := range fa.Places() {
		if p != want {
			t.Errorf("turn %d went out with %+v, want %+v", i, p, want)
		}
	}
	if len(fa.Places()) != 3 {
		t.Errorf("places %+v", fa.Places())
	}
}

func TestACancelledRequestPutsTheTextBackInTheComposerItCameFrom(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Block: make(chan struct{})})
	r.inInspector(t, 5, "a slow one")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Key(0, ui.KeyEscape)
	r.rec.drain(t)
	r.tt.Frame()
	if r.inspectorBuf() != "a slow one" {
		t.Errorf("inspector composer holds %q, want the text back", r.inspectorBuf())
	}
	if r.pageBuf() != "" {
		t.Errorf("page composer holds %q", r.pageBuf())
	}
}
