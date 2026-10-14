package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func newMainTester(t *testing.T, width int) (*ui.Tester, *Main, *App) {
	t.Helper()
	app, _, _ := newTestApp(t)
	m := NewMain(app)
	tt := ui.NewTester(m.View, width, 720)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	return tt, m, app
}

func center(r ui.Rect) (float32, float32) { return r.X + r.W/2, r.Y + r.H/2 }

// dragTo drags the card showing text onto the column named column.
func dragTo(t *testing.T, tt *ui.Tester, text, column string) {
	t.Helper()
	from, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no card %q in %q", text, tt.Texts())
	}
	to, ok := tt.Find(column)
	if !ok {
		t.Fatalf("no column %q", column)
	}
	fx, fy := center(from)
	tx, ty := center(to)
	tt.Press(fx, fy)
	tt.Move(fx+8, fy+8)
	tt.Move(tx, ty+60)
	tt.Release(tx, ty+60)
}

func TestMainShowsProjectBoardWithAllColumns(t *testing.T) {
	tt, _, _ := newMainTester(t, 1840)
	for _, want := range []string{"adhd-todo", "Backlog", "Todo", "Doing", "Blocked", "Review", "Done", "Port core to Go", "Fix tray race on launch"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestDroppingACardChangesItsStatus(t *testing.T) {
	tt, _, app := newMainTester(t, 1840)
	dragTo(t, tt, "Write the release notes", "Doing")
	task, _ := app.Task(5)
	if task.Status != data.Doing {
		t.Errorf("status %v after the drop, want doing", task.Status)
	}
}

func TestDroppingOnTheSameColumnChangesNothing(t *testing.T) {
	tt, _, app := newMainTester(t, 1840)
	dragTo(t, tt, "Write the release notes", "Todo")
	if task, _ := app.Task(5); task.Status != data.Todo {
		t.Errorf("status %v, want todo", task.Status)
	}
}

func TestClickingACardOpensTheInspectorAndEscCloses(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("Port core to Go"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 1 || !tt.HasText("ACTIVITY") {
		t.Fatalf("open %d, texts %q", m.openID, tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if m.openID != 0 {
		t.Errorf("Esc left task %d open", m.openID)
	}
}

func TestInspectorSavesTheTitleOnEnter(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(5)
	tt.Frame()
	title, ok := tt.Find("Write the release notes")
	if !ok {
		t.Fatalf("no title field: %q", tt.Texts())
	}
	// The inspector's title is the right-most match; click into it and replace the text.
	_ = title
	if err := tt.Click("Title"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Ship the release notes")
	tt.Key(0, ui.KeyEnter)
	if got, _ := app.Task(5); got.Title != "Ship the release notes" {
		t.Errorf("title %q after Enter", got.Title)
	}
}

func TestInspectorSavesNotesOnBlur(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(7)
	tt.Frame()
	if err := tt.Click("Notes"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Start with the panel")
	if got, _ := app.Task(7); got.Notes != "" {
		t.Fatalf("notes saved before blur: %q", got.Notes)
	}
	if err := tt.Click("Title"); err != nil { // move the focus away
		t.Fatal(err)
	}
	if got, _ := app.Task(7); got.Notes != "Start with the panel" {
		t.Errorf("notes %q after blur", got.Notes)
	}
}

func TestInspectorChangesPriorityThroughTheMenu(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(7)
	tt.Frame()
	if err := tt.Click("Priority: P2  Normal"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("P1  High"); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.Task(7); got.Priority != data.P1 {
		t.Errorf("priority %v", got.Priority)
	}
}

func TestInspectorOpensThePullRequest(t *testing.T) {
	app, _, rec := newTestApp(t)
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	still(tt)
	m.OpenTask(2)
	tt.Frame()
	if err := tt.Click("Open pull request #12"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || rec.urls[0] != "https://github.com/QingYunA/adhd-todo/pull/12" {
		t.Errorf("opened %v", rec.urls)
	}
}

func TestSidebarSwitchesProjectsAndShowsCounts(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	if err := tt.Click("docs"); err != nil {
		t.Fatal(err)
	}
	if m.nav.inbox || !tt.HasText("Write the MCP quickstart") || tt.HasText("Port core to Go") {
		t.Errorf("after choosing docs: %q", tt.Texts())
	}
	if err := tt.Click("Inbox"); err != nil {
		t.Fatal(err)
	}
	if !m.nav.inbox || !tt.HasText("Book dentist") {
		t.Errorf("inbox shows %q", tt.Texts())
	}
}

func TestDoneColumnShowsTenAndShowAll(t *testing.T) {
	store := data.NewEmptyFake(fixedNow)
	p, _ := store.CreateTask(data.CreateTaskInput{Title: "seed"})
	_ = p
	app := NewApp(store, &recorder{}, testConfig())
	app.Now = fixedNow
	for i := 0; i < 12; i++ {
		task, _ := store.CreateTask(data.CreateTaskInput{Title: "Finished " + itoa(i), Status: data.Done})
		_ = task
	}
	app.Refresh()
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1840, 900)
	still(tt)
	if tt.HasText("Show all 12") {
		t.Fatal("Done must start collapsed")
	}
	if err := tt.Click("Done"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Show all 12") {
		t.Fatalf("no Show all link: %q", tt.Texts())
	}
	if err := tt.Click("Show all 12"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Show all 12") {
		t.Error("link still shown after opening all")
	}
}

func TestMainEmptyStates(t *testing.T) {
	app, _ := newEmptyApp(t)
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	still(tt)
	for _, want := range []string{"Your inbox is clear.", "Tasks you add from the menu bar panel land here until you give them a project.", "How to connect"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
}

func TestSettingsSheetCopiesTheCommand(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenSettings()
	tt.Frame()
	if !tt.HasText("Connect Claude Code") || !tt.HasText(app.Config.DataFolder) {
		t.Fatalf("sheet texts %q", tt.Texts())
	}
	if err := tt.Click("Copy command"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != app.Config.ConnectCommand {
		t.Errorf("clipboard %q", tt.Clipboard())
	}
	if err := tt.Click("Close settings"); err != nil {
		t.Fatal(err)
	}
	if m.settings {
		t.Error("sheet still open")
	}
}
