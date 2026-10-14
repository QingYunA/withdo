package views

import (
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func newPanelTester(t *testing.T) (*ui.Tester, *Panel, *App, *recorder) {
	t.Helper()
	app, _, rec := newTestApp(t)
	p := NewPanel(app)
	tt := ui.NewTester(p.View, 440, 560)
	still(tt)
	return tt, p, app, rec
}

func TestPanelListsAgentsAndMine(t *testing.T) {
	tt, _, _, _ := newPanelTester(t)
	for _, want := range []string{"Fix tray race on launch", "Port core to Go", "Reply to design feedback", "4 active", "! 3d stale", "in 1h"} {
		if !tt.HasText(want) {
			t.Errorf("panel is missing %q; texts %q", want, tt.Texts())
		}
	}
	if tt.HasText("Renew passport") {
		t.Error("a backlog task is in Mine")
	}
}

func TestPanelFiltersWhileTyping(t *testing.T) {
	tt, _, _, _ := newPanelTester(t)
	tt.Type("tray")
	if !tt.HasText("Fix tray race on launch") || tt.HasText("Port core to Go") || tt.HasText("Reply to design feedback") {
		t.Errorf("filter 'tray' left %q", tt.Texts())
	}
	if tt.HasText("MINE") {
		t.Error("Mine is shown with no match in it")
	}
}

func TestPanelArrowKeysSelectAndEnterOpens(t *testing.T) {
	tt, p, _, rec := newPanelTester(t)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	if p.selected != 1 {
		t.Fatalf("selected %d after two Down, want 1", p.selected)
	}
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyUp)
	if p.selected != -1 {
		t.Errorf("selected %d after Up past the first row, want -1", p.selected)
	}
	tt.Key(0, ui.KeyDown) // the blocked task comes first
	tt.Key(0, ui.KeyEnter)
	if len(rec.opened) != 1 || rec.opened[0] != 3 || rec.panelClosed != 1 {
		t.Errorf("Enter on the first row opened %v, closed %d", rec.opened, rec.panelClosed)
	}
}

func TestPanelCmdEnterOpensSelection(t *testing.T) {
	tt, _, _, rec := newPanelTester(t)
	tt.Key(0, ui.KeyDown)
	tt.Key(ui.Cmd, ui.KeyEnter)
	if len(rec.opened) != 1 || rec.opened[0] != 3 {
		t.Errorf("Cmd+Enter opened %v, want task 3", rec.opened)
	}
}

func TestPanelEnterAddsToInbox(t *testing.T) {
	tt, p, app, _ := newPanelTester(t)
	before := len(app.snap.tasks)
	tt.Type("Water the plants")
	tt.Key(0, ui.KeyEnter)
	if len(app.snap.tasks) != before+1 {
		t.Fatalf("tasks %d, want %d", len(app.snap.tasks), before+1)
	}
	var added data.Task
	for _, task := range app.snap.tasks {
		if task.Title == "Water the plants" {
			added = task
		}
	}
	if added.Title != "Water the plants" || added.ProjectID != nil || added.Source != data.SourceUser {
		t.Errorf("added %+v", added)
	}
	if p.query != "" || !tt.HasText("Added to Inbox") {
		t.Errorf("query %q, texts %q", p.query, tt.Texts())
	}
}

func TestPanelEnterWithNothingTypedAddsNothing(t *testing.T) {
	tt, _, app, _ := newPanelTester(t)
	before := len(app.snap.tasks)
	tt.Key(0, ui.KeyEnter)
	if len(app.snap.tasks) != before {
		t.Error("an empty Enter created a task")
	}
}

func TestPanelEscapeCloses(t *testing.T) {
	tt, _, _, rec := newPanelTester(t)
	tt.Key(0, ui.KeyEscape)
	if rec.panelClosed != 1 {
		t.Errorf("closed %d times, want 1", rec.panelClosed)
	}
}

func TestPanelInputIsFocusedOnOpen(t *testing.T) {
	tt, _, _, _ := newPanelTester(t)
	tt.Type("x")
	if !tt.HasText("x") {
		t.Errorf("typed text did not reach the input: %q", tt.Texts())
	}
}

func TestPanelEmptyStates(t *testing.T) {
	app, rec := newEmptyApp(t)
	p := NewPanel(app)
	tt := ui.NewTester(p.View, 440, 560)
	still(tt)
	for _, want := range []string{"No agent is reporting yet.", "Coding agents report here through the adhd-todo MCP server.", "How to connect", "Nothing on your plate. Type above to add a task."} {
		if !tt.HasText(want) {
			t.Errorf("empty panel is missing %q; texts %q", want, tt.Texts())
		}
	}
	tt.Type("zzz")
	if !tt.HasText("Nothing matches") {
		t.Errorf("no-match state missing: %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("How to connect"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || !strings.Contains(rec.urls[0], "github.com/QingYunA/adhd-todo") {
		t.Errorf("opened %v", rec.urls)
	}
}

func TestPanelStatusMenuMarksDoneAndFadesTheRow(t *testing.T) {
	tt, _, app, _ := newPanelTester(t)
	tt.Type("dentist")
	if err := tt.Click("Status: Todo"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Done"); err != nil {
		t.Fatal(err)
	}
	task, _ := app.Task(8)
	if task.Status != data.Done || task.CompletedAt == nil {
		t.Fatalf("task is %v after choosing Done", task.Status)
	}
	if !tt.HasText("Book dentist") {
		t.Error("the row vanished before its completion animation ran")
	}
	app.Now = func() time.Time { return fixedNow().Add(completeTotal + time.Millisecond) }
	tt.Frame()
	if tt.HasText("Book dentist") {
		t.Error("the row is still listed after the animation")
	}
}
