package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The Summary page of a crowded day: you and six agents (Claude Code, Codex, Gemini CLI, Cursor,
// GitHub Copilot and OpenCode) finished something, so six people stand and the seventh throws from
// past the right edge. One frame in the middle of that last throw, and the last frame. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSummaryCrowd

// summaryCrowdApp is summaryShotApp with five more tasks finished today, by Gemini CLI (two),
// Cursor, GitHub Copilot and OpenCode.
func summaryCrowdApp(t *testing.T) *App {
	t.Helper()
	app := summaryShotApp(t, false)
	store := app.Store.(*data.Fake)
	kinds := map[int]string{2: "gem", 20: "gear", 1: "gem", 21: "mail", 7: "mail", 8: "crate", 10: "gear", 9: "jar", 13: "book", 11: "gem", 12: "jar"}
	for _, f := range []struct {
		id    int
		by    string
		kind  string
		title string
		at    time.Time
	}{
		{201, "gemini-cli", "jar", "Fix the empty-state crash", atToday(9, 55)},
		{202, "gemini-cli", "flask", "Cover the export sheet with tests", atToday(13, 20)},
		{203, "cursor", "gem", "Add the board's keyboard shortcuts", atToday(10, 50)},
		{204, "GitHub Copilot", "book", "Write the Windows setup notes", atToday(12, 40)},
		{205, "opencode", "scroll", "Plan the sync engine rework", atToday(14, 10)},
	} {
		task := store.AddTask(data.Task{ID: f.id, ProjectID: data.IntPtr(1), Title: f.title, Status: data.Doing, Source: data.SourceAgent})
		store.ReportAs(task.ID, f.by, f.title, f.at.Add(-time.Second))
		store.CompleteAt(task.ID, f.at)
		kinds[f.id] = f.kind
	}
	_, _ = store.PutSummary(data.DayPeriod(app.Now(), 0), data.StoredSummary{
		Paragraph: "Thirteen finished. You answered the packaging question, replied to the design feedback and booked the dentist; Claude Code merged the native UI and the MCP server, Codex the watcher, and Gemini CLI, Cursor, GitHub Copilot and OpenCode each shipped a piece of their own.",
		Model:     "fake-model", Kinds: kinds,
	})
	app.Refresh()
	return app
}

func TestScreensSummaryCrowd(t *testing.T) {
	for _, a := range appearances() {
		app := summaryCrowdApp(t)
		mainShot(t, "summary-crowd-"+a.suffix+".png", app, a.dark, toSummary(false, nil))
		mainShot(t, "summary-crowd-throw-"+a.suffix+".png", app, a.dark, toSummary(false, func(tt *ui.Tester, m *Main) {
			at := 4000 * time.Millisecond
			m.sum.freeze = &at
		}))
	}
}
