package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// atToday is a time of the pinned day, fixedNow's: Wednesday 14 October.
func atToday(h, m int) time.Time {
	y, mo, d := fixedNow().Date()
	return time.Date(y, mo, d, h, m, 0, 0, time.Local)
}

// summaryApp is the seeded fake with a busy day: eight tasks finished, three by me, three by Claude
// Code (two of them with merged pull requests) and two by Codex. The assistant is ready and answers
// with a paragraph and the kinds of the goods.
func summaryApp(t *testing.T) (*App, *data.Real, *recorder) {
	t.Helper()
	app, store, rec := newTestApp(t)
	finish := func(id int, by string, at time.Time, said string, merged int) {
		if by != "" {
			store.SeedReportAs(id, by, said, at.Add(-time.Second))
		}
		if merged > 0 {
			store.SeedPullRequest(id, data.PRMerged, data.PRChecksPassing, data.PRReviewApproved)
		}
		store.SeedCompleteAt(id, at)
	}
	finish(21, "", atToday(9, 40), "", 0)                                                // me: the packaging answer, from the seed
	finish(7, "", atToday(11, 30), "", 0)                                                // me: reply to design feedback
	finish(8, "", atToday(12, 15), "", 0)                                                // me: book the dentist
	finish(10, "codex", atToday(10, 20), "Watcher ported around PRAGMA data_version", 1) // codex, merged
	finish(9, "codex", atToday(13, 2), "Windows build scaffolded", 0)                    // codex
	finish(2, "claude-code", atToday(11, 5), "Board, lists and inspector are in", 1)     // claude, merged
	finish(1, "claude-code", atToday(14, 0), "Core ported and the migrations pass", 0)   // claude
	// Port the MCP server (20) was finished by the seed's merge, 40 minutes ago: its pull request merged.
	store.SeedPullRequest(20, data.PRMerged, data.PRChecksPassing, data.PRReviewApproved)
	app.Assistant = &data.FakeAssistant{Ready: true, SummaryScript: func(req data.SummaryRequest) (data.SummaryText, error) {
		kinds := map[int]string{}
		for _, task := range req.Tasks {
			kinds[task.ID] = data.GuessKind(task.Title)
		}
		kinds[2], kinds[20], kinds[1] = "gem", "gear", "gem"
		return data.SummaryText{
			Paragraph: "Eight finished. You answered the packaging question, replied to the design feedback and booked the dentist; Claude Code merged the native UI and the MCP server, and Codex merged the watcher. Native UI review is still waiting.",
			Kinds:     kinds, Model: "fake-model",
		}, nil
	}}
	app.Refresh()
	return app, store, rec
}

// summaryMain is a Main on the Summary page, with the scene drawn still (Reduce Motion).
func summaryMain(t *testing.T, app *App, week bool) (*ui.Tester, *Main) {
	t.Helper()
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 900)
	m.sum.week = week
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	return tt, m
}
