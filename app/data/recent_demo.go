package data

import (
	"fmt"
	"time"
)

// DemoRecent is a FakeRecent with a week in the demo's repositories, dated back from now: the
// demo's and the screenshots' data. PRs 12 and 14 are the ones the seeded tasks hold.
func DemoRecent(now time.Time) *FakeRecent {
	f := NewFakeRecent()
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	const h, day = time.Hour, 24 * time.Hour
	mk := func(repo string, kind RecentKind, n int, title, body, author string, at time.Time, files ...string) RecentItem {
		it := RecentItem{
			Repo: repo, Kind: kind, Number: n, Title: title, Body: body, Author: author, Files: files,
			URL: fmt.Sprintf("https://github.com/%s/pull/%d", repo, n), CreatedAt: at.Add(-3 * h), UpdatedAt: at,
		}
		switch kind {
		case RecentMerged:
			it.ClosedAt = at
		case RecentOpen:
			it.CreatedAt = at
		case RecentIssue:
			it.URL = fmt.Sprintf("https://github.com/%s/issues/%d", repo, n)
			it.CreatedAt = at
		}
		return it
	}
	const me = "QingYunA"
	bot := mk("QingYunA/withdo", RecentOpen, 40, "chore(mcp): bump the MCP server to 0.5", "", "cursor-agent", ago(20*time.Minute), "go.mod")
	bot.Bot = true
	const repo = "QingYunA/withdo"
	solved := mk(repo, RecentIssue, 56, "Summary ledger shows an ellipsis for the total", "The total is cut to an ellipsis when it has two digits.", me, ago(3*day))
	solved.ClosedAt, solved.UpdatedAt, solved.FixedBy = ago(2*h), ago(2*h), 41
	f.Set(RepoChanges{
		Repo: repo,
		Merged: []RecentItem{
			mk(repo, RecentMerged, 41, "fix(summary): give a two-digit ledger count room, so 48 is not cut to an ellipsis", "One digit keeps its width; two or more digits get room.", me, ago(2*h), "app/views/summary_ledger.go", "app/views/summary_ledger_width_test.go"),
			mk(repo, RecentMerged, 42, "test(settings): fix the flaky model list test", "The test read the model list before the provider had posted it.", me, ago(5*h), "app/views/settings_assistant_test.go"),
			mk(repo, RecentMerged, 34, "feat(summary): quiet wood-box sounds while the day is shipped", "Sound effects for the lid, each landing and the final hop; a switch in Settings turns them off.", me, ago(30*h), "app/theme/sounds.go", "app/theme/sounds/land.wav", "app/views/summary_sound.go"),
			mk(repo, RecentMerged, 35, "feat(summary): figures for Gemini, Cursor, Copilot and OpenCode", "The scene seats six people; the seventh throws from off the right edge.", me, ago(31*h), "app/theme/scene_day.go", "app/theme/sprites/actors/gemini/idle.png", "third_party/kenney-tiny-town/License.txt"),
			mk(repo, RecentMerged, 29, "feat(settings): copy the setup text for any agent", "A copy button in Settings > Agents hands any agent the MCP setup text.", me, ago(3*day), "app/views/settings_agents.go"),
			mk(repo, RecentMerged, 12, "feat(ui): native UI with mygo", "Board, lists and the inspector.", me, ago(4*day), "app/views/board.go", "app/views/list.go"),
			mk(repo, RecentMerged, 14, "feat(mcp): port the MCP server", "The MCP tools on the Go core.", me, ago(40*time.Minute), "internal/mcpserver/server.go"),
			mk(repo, RecentMerged, 43, "docs(spec): write down the recent changes page", "The spec and mockup of the page.", me, ago(day), "docs/design/ui-spec.md"),
			mk(repo, RecentMerged, 44, "chore: bump the version to 0.4.0", "", me, ago(2*day), "app/main.go"),
			mk(repo, RecentMerged, 45, "docs(screens): refresh the summary screenshots", "", me, ago(2*day+h), "docs/design/screens/summary-light.png"),
		},
		Open: []RecentItem{
			mk(repo, RecentOpen, 39, "feat(categories): pixel icons for Work and Life", "Pixelarticons for the two categories in the sidebar and the inspector.", me, ago(2*h), "app/theme/icons.go"),
			bot,
		},
		Issues: []RecentItem{
			mk(repo, RecentIssue, 58, "Export Markdown fails for a task with pictures", "Exporting a day that has a task with pictures stops with an error.", me, ago(day)),
			solved,
		},
	})
	f.changes["QingYunA/picpocket"] = RepoChanges{
		Repo: "QingYunA/picpocket",
		Merged: []RecentItem{
			mk("QingYunA/picpocket", RecentMerged, 44, "feat(watch): map a new worktree back to its main repository", "A new branch's worktree is mapped back as soon as it appears.", me, ago(2*day), "internal/watch/worktree.go"),
			mk("QingYunA/picpocket", RecentMerged, 43, "fix(tray): the tray no longer misses focus at launch", "", me, ago(5*day), "app/tray.go"),
		},
	}
	f.changes["QingYunA/answer-me-with-html"] = RepoChanges{Repo: "QingYunA/answer-me-with-html"}
	return f
}
