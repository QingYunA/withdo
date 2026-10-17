package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// hover moves the pointer over the text and draws the frame the hover shows in.
func hover(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q in %q", text, tt.Texts())
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	tt.Frame()
}

var rowButtons = []string{"Open PR", "Accept", "Answer", "Review"}

func TestAnAgentsRowShowsItsButtonsOnlyWhileHoveredAndOnlyWhenItWaitsForYou(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	hasNot(t, tt, rowButtons...)
	for _, c := range []struct {
		title string
		want  []string
	}{
		{"Native UI with mygo", []string{"Open PR", "Accept"}}, // in review, with a pull request
		{"Fix tray race on launch", []string{"Answer"}},        // blocked
		{"Port core to Go", nil},                               // doing: a click opens it
	} {
		hover(t, tt, c.title)
		for _, b := range rowButtons {
			if got, want := tt.HasText(b), contains(c.want, b); got != want {
				t.Errorf("hovering %q: %q shown is %v, want %v", c.title, b, got, want)
			}
		}
	}
	tt.Move(2, 700)
	tt.Frame()
	tt.Frame()
	hasNot(t, tt, rowButtons...)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestAReviewRowWithoutAPullRequestOffersReviewAndAcceptInTheColumn(t *testing.T) {
	app, store, _ := newTestApp(t)
	_, err := store.CreateTask(data.CreateTaskInput{Title: "Tidy the docs", Source: data.SourceAgent, Status: data.Review, ProjectID: data.IntPtr(1)})
	if err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	hover(t, tt, "Tidy the docs")
	has(t, tt, "Review", "Accept")
	hasNot(t, tt, "Open PR")
}

func TestAcceptOnAProjectRowStampsFoldsItsGroupAndOpensItAtTheTopOfDone(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	moving(tt)
	hover(t, tt, "Native UI with mygo")
	yDoing := yOf(t, tt, "Port core to Go")
	if err := tt.Click("Accept: Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	if task, _ := app.Task(2); task.Status != data.Done {
		t.Fatalf("status %v, want done", task.Status)
	}
	if m.openID != 0 {
		t.Error("Accept opened the inspector")
	}
	tt.Move(2, 700) // the buttons stay while the task is being accepted, hovered or not
	at(tt, app, ms(100))
	has(t, tt, "In review", "Accept")
	if y := yOf(t, tt, "Port core to Go"); y != yDoing {
		t.Errorf("the group moved to %v before the fold, was %v", y, yDoing)
	}
	at(tt, app, ms(309))
	if y := yOf(t, tt, "Port core to Go"); y >= yDoing {
		t.Errorf("mid-fold the groups under it have not slid up: %v, was %v", y, yDoing)
	}
	at(tt, app, ms(419))
	almost := yOf(t, tt, "Port core to Go")
	at(tt, app, ms(450))
	hasNot(t, tt, "In review")
	if y := yOf(t, tt, "Port core to Go"); !near(y, almost) {
		t.Errorf("the page jumps %v pt when the emptied group is dropped: its header did not fold", y-almost)
	}
	if !above(t, tt, "Native UI with mygo", "Port the MCP server") {
		t.Error("the accepted task is not at the top of the column's Done")
	}
}

func TestTheInboxListHasTheSameButtons(t *testing.T) {
	app, store, _ := newTestApp(t)
	if _, err := store.CreateTask(data.CreateTaskInput{Title: "Tidy the docs", Source: data.SourceAgent, Status: data.Review}); err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenInbox})
	tt.Frame()
	hasNot(t, tt, "Accept")
	hover(t, tt, "Tidy the docs")
	has(t, tt, "Review", "Accept")
	if err := tt.Click("Accept: Tidy the docs"); err != nil {
		t.Fatal(err)
	}
	for _, task := range app.snap.tasks {
		if task.Title == "Tidy the docs" && task.Status != data.Done {
			t.Errorf("status %v, want done", task.Status)
		}
	}
}

func TestABoardCardShowsTheButtonsInPlaceOfTheAgeAndAcceptMovesItToDone(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m := boardTester(t, app, 1, 1100)
	moving(tt)
	hasNot(t, tt, rowButtons...)
	hover(t, tt, "Fix tray race on launch")
	has(t, tt, "Answer")
	hasNot(t, tt, "Accept")
	hover(t, tt, "Native UI with mygo")
	has(t, tt, "Open PR", "Accept")
	if err := tt.Click("Accept: Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	at(tt, app, ms(100))
	if !contains(laneOfTitles(m, data.Review), "Native UI with mygo") {
		t.Errorf("the card left In review during its stamp: %v", laneOfTitles(m, data.Review))
	}
	at(tt, app, ms(250))
	if contains(laneOfTitles(m, data.Review), "Native UI with mygo") {
		t.Error("the card is still in In review after its stamp")
	}
	if done := laneOfTitles(m, data.Done); len(done) == 0 || done[0] != "Native UI with mygo" {
		t.Errorf("Done lane %v: the card is not at the top", done)
	}
}
