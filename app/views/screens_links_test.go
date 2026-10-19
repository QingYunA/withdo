package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Links from agents: the inspector's Activity with chips (hovered, and a file that is gone), and Mine
// with Open page on the rows in review. Light and dark, English and Chinese (-zh). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensLinks

// linksScreenApp is linksApp with an older report of task 2 that carries pages too, and a review task
// without a pull request whose only page is a file that is gone.
func linksScreenApp(t *testing.T) (*App, *recorder) {
	t.Helper()
	app, store, rec := linksApp(t)
	store.SeedReport(data.Report{
		TaskID: 2, Agent: "claude-code", Status: data.Doing, Summary: "Wrote the design notes for the board.",
		CreatedAt: fixedNow().Add(-90 * time.Minute),
		Links: []data.Link{
			{URL: "https://github.com/QingYunA/adhd-todo/pull/12"},
			{URL: "file:///Users/demo/notes/board-design-notes-for-the-whole-native-ui-rewrite-20261014-123456.html"},
		},
	})
	crash, err := store.CreateTask(data.CreateTaskInput{Title: "Import crash report", Source: data.SourceAgent, Status: data.Review, ProjectID: data.IntPtr(1)})
	if err != nil {
		t.Fatal(err)
	}
	store.SeedReport(data.Report{
		TaskID: crash.ID, Agent: "codex", Status: data.Review, Summary: "The crash report is written.",
		CreatedAt: fixedNow().Add(-60 * time.Minute), Links: []data.Link{{URL: goneFile, Title: "Import report"}},
	})
	app.Refresh()
	return app, rec
}

func TestScreensLinks(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix + ".png"
			if zh {
				sfx = "-zh" + sfx
			}
			shotOf := func(name string, setup func(tt *ui.Tester, m *Main)) {
				app, _ := linksScreenApp(t)
				if zh {
					zhShot(t, name+sfx, app, a.dark, setup)
					return
				}
				mainShot(t, name+sfx, app, a.dark, setup)
			}
			shotOf("inspector-links", func(tt *ui.Tester, m *Main) { tt.SetSize(1100, 1000); m.OpenTask(2) })
			shotOf("inspector-links-hover", func(tt *ui.Tester, m *Main) {
				tt.SetSize(1100, 1000)
				m.OpenTask(2)
				tt.Frame()
				r, ok := tt.Find("crash-report.html")
				if !ok {
					t.Fatal("no chip")
				}
				tt.Move(r.X+r.W/2, r.Y+r.H/2)
				tt.Frame()
				time.Sleep(700 * time.Millisecond)
			})
			shotOf("inspector-links-missing", func(tt *ui.Tester, m *Main) {
				tt.SetSize(1100, 1000)
				m.OpenTask(2)
				tt.Frame()
				r, ok := tt.Find("Import report")
				if !ok {
					t.Fatal("no chip")
				}
				if err := tt.Click("Import report"); err != nil {
					t.Fatal(err)
				}
				tt.Move(r.X+r.W/2, r.Y+r.H/2)
			})
			shotOf("mine-open-page", func(tt *ui.Tester, m *Main) {})
			shotOf("project-board-open-page", func(tt *ui.Tester, m *Main) {
				toProject(1)(tt, m)
				m.board[1] = true
				tt.Frame()
				hover(t, tt, "Native UI with mygo")
			})
		}
	}
}
