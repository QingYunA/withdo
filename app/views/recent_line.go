package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const (
	recentBubbleEdge float32 = 2 // pt, the frame of the assistant's lines
	recentIconSlot   float32 = 18
)

// recentGroup is a group of a section: its name and count, then a line for each item. An empty group
// is left out.
func (m *Main) recentGroup(c *ui.Context, t *theme.Tokens, repo, title string, lines []data.RecentLine, now time.Time) {
	if len(lines) == 0 {
		return
	}
	ui.Row(c.Key("recent-group-"+repo+title)).Margin(10, 0, 2).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, title)
		theme.Text(c, theme.Meta, t.TextSecondary, itoa(len(lines)))
	})
	for _, l := range lines {
		m.recentLine(c, t, l, now)
	}
}

// otherGroup is the merged pull requests that are neither new nor fixes, folded to one line that says
// how many and what; a click unfolds them, a line each.
func (m *Main) otherGroup(c *ui.Context, t *theme.Tokens, s recentSec, now time.Time) {
	if len(s.Other) == 0 {
		return
	}
	st, repo := &m.recent, s.project.RepoSlug
	open := st.moreOpen[repo]
	text := m.say.RecentMoreNoTopics(len(s.Other))
	if topics := s.OtherTopics(); len(topics) > 0 {
		text = m.say.RecentMore(len(s.Other), strings.Join(topics, m.say.RecentTopicSep()))
	}
	row := ui.Row(c.Key("recent-more-"+repo)).Margin(8, 0, 0).Shrink(0).Padding(3, 8).Gap(4).Radius(7).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Label(m.say.RecentFoldOther())
	ink := t.TextTertiary
	if row.Hovered() {
		row.Background(t.BgHover)
		ink = t.TextSecondary
	}
	row.Children(func() {
		icon := theme.IconForward
		if open {
			icon = theme.IconChevron
		}
		ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(ink).Shrink(0)
		theme.Text(c, theme.Meta, ink, text).Ellipsis("…")
	})
	if row.Clicked() {
		st.moreOpen[repo] = !open
	}
	if open {
		for _, l := range s.Other {
			m.recentLine(c, t, l, now)
		}
	}
}

// recentLine is an item: its icon, its sentence, who and when under it; a click unfolds the
// assistant's explanation under it.
func (m *Main) recentLine(c *ui.Context, t *theme.Tokens, l data.RecentLine, now time.Time) {
	st, it := &m.recent, l.Item
	open := st.open[it.URL]
	ui.Column(c.Key("recent-item-" + it.URL)).FillWidth().Shrink(0).Children(func() {
		row := ui.Row(c.Key("recent-row-"+it.URL)).FillWidth().Shrink(0).Padding(4, 8).Gap(8).Radius(7).AlignItems(ui.Start).
			Cursor(ui.CursorPointer).Label(m.say.RecentToggleItem(l.Note.Sentence))
		if open || row.Hovered() {
			row.Background(t.BgHover)
		}
		row.Children(func() {
			m.recentIcon(c, t, l)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				theme.Text(c, theme.Compact, t.Text, l.Note.Sentence).Ellipsis("…")
				m.recentMeta(c, t, it, now)
			})
		})
		if row.Clicked() {
			st.open[it.URL] = !open
		}
		if open {
			m.recentExplanation(c, t, it)
		}
	})
}

// recentIcon is what marks an item: the good of Summary for a merged pull request (a gem for a new
// feature, a jar for a fix, whatever the title points to for the rest), a green pull request for an
// open one, and the ring with a dot for an issue, green while open and purple once closed.
func (m *Main) recentIcon(c *ui.Context, t *theme.Tokens, l data.RecentLine) {
	slot := ui.Box(c).Size(recentIconSlot, recentIconSlot).Shrink(0).Margin(1, 0, 0)
	switch l.Item.Kind {
	case data.RecentMerged:
		kind := data.GuessKind(l.Item.Title)
		switch l.Note.Category {
		case data.RecentNew:
			kind = "gem"
		case data.RecentFixed:
			kind = "jar"
		}
		slot.Children(func() { theme.GoodIcon(c, kind, recentIconSlot) })
	case data.RecentOpen:
		slot.Children(func() {
			ui.Icon(c, theme.IconPR).Size(recentIconSlot, recentIconSlot).TextColor(t.Status(data.Done))
		})
	default:
		ink := t.Status(data.Done)
		if l.Item.Solved() {
			ink = t.Merged
		}
		slot.Children(func() {
			ui.Icon(c, theme.IconNow).Size(recentIconSlot, recentIconSlot).TextColor(ink)
			ui.Box(c).Absolute().Left(7.5).Top(7.5).Size(3, 3).Background(ink).PassThrough()
		})
	}
}

// recentMeta is the quiet line under a sentence: who, when, and the number.
func (m *Main) recentMeta(c *ui.Context, t *theme.Tokens, it data.RecentItem, now time.Time) {
	ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		who, name, hasWho := m.recentWho(it)
		var parts []string
		switch {
		case it.Kind == data.RecentIssue && it.Solved():
			parts = append(parts, m.say.RecentIssueSolved())
			if it.FixedBy > 0 {
				parts = append(parts, m.say.RecentFixedBy(it.FixedBy))
			}
			parts = append(parts, m.recentWhen(it.ClosedAt, now))
		case it.Kind == data.RecentIssue:
			parts = append(parts, m.say.RecentIssueNew(), name, m.recentWhen(it.CreatedAt, now))
		case it.Kind == data.RecentOpen:
			if hasWho {
				theme.AvatarIcon(c, who, theme.AvatarSmall).Shrink(0)
			}
			parts = append(parts, name, m.recentOpenFor(it.CreatedAt, now))
		default:
			if hasWho {
				theme.AvatarIcon(c, who, theme.AvatarSmall).Shrink(0)
			}
			parts = append(parts, name, m.recentWhen(it.ClosedAt, now))
		}
		parts = append(parts, "#"+itoa(it.Number))
		theme.Text(c, theme.Meta, t.TextSecondary, strings.Join(parts, " · ")).Ellipsis("…")
	})
}

// recentWho is who did an item: the agent whose task holds the pull request's URL, else its GitHub
// author, and you when the author is the account gh is signed in to.
func (m *Main) recentWho(it data.RecentItem) (who theme.Who, name string, hasWho bool) {
	if task, ok := m.taskOfPR(it.URL); ok {
		return m.app.taskWho(task), m.taskWhoName(task), true
	}
	switch {
	case m.app.githubAccount() != "" && strings.EqualFold(it.Author, m.app.githubAccount()):
		if it.Kind == data.RecentIssue {
			return theme.You, m.say.RecentWhoYouLower(), true
		}
		return theme.You, m.say.DoneByYou(), true
	case it.Bot:
		return theme.AgentAvatar(it.Author), it.Author, true
	}
	return theme.Who{}, it.Author, false
}

// githubAccount is the account gh is signed in to, when it says.
func (a *App) githubAccount() string {
	if a.GitHub == nil {
		return ""
	}
	return a.GitHub.Status().Account
}

// taskOfPR is the task that holds a pull request's URL.
func (m *Main) taskOfPR(url string) (data.Task, bool) {
	for _, t := range m.app.snap.tasks {
		if t.PRURL == url {
			return t, true
		}
	}
	return data.Task{}, false
}

// recentWhen is when something happened: today, yesterday, 3 days ago, then the date.
func (m *Main) recentWhen(at, now time.Time) string {
	switch n := -data.DaysFrom(at, now); {
	case n <= 0:
		return m.say.RecentToday()
	case n == 1:
		return m.say.RecentYesterday()
	case n < 7:
		return m.say.RecentDaysAgo(n)
	}
	return m.clock.Date(at, now)
}

// recentOpenFor is how long a pull request has been open.
func (m *Main) recentOpenFor(since, now time.Time) string {
	d := max(now.Sub(since), 0)
	switch {
	case d < time.Hour:
		return m.say.RecentOpenMinutes(max(1, int(d/time.Minute)))
	case d < 24*time.Hour:
		return m.say.RecentOpenHours(int(d / time.Hour))
	}
	return m.say.RecentOpenDays(int(d / (24 * time.Hour)))
}

// recentExplanation is what unfolds under an item: the assistant's three points, written the first
// time it is opened and kept, then Open on GitHub and the task that holds the pull request.
func (m *Main) recentExplanation(c *ui.Context, t *theme.Tokens, it data.RecentItem) {
	m.explainIfDue(it)
	task, hasTask := m.taskOfPR(it.URL)
	ui.Column(c.Key("recent-explain-"+it.URL)).Shrink(0).Margin(2, 0, 6, 34).Padding(8, 12).Gap(6).Radius(8).Background(t.BgSubtle).Children(func() {
		if m.hasModel() {
			m.explanationBody(c, t, it)
		}
		ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
			if settingsOutLink(c, t, "recent-gh-"+it.URL, m.say.RecentOpenOnGitHub(), m.say.RecentOpenOnGitHub()).Clicked() {
				m.app.Act.OpenURL(it.URL)
			}
			if hasTask && link(c.Key("recent-task-"+it.URL), t, m.say.RecentTask(task.ID)).Clicked() {
				m.openRow(task.ID)
			}
		})
	})
}

// explanationBody is the source line and the points, or that it is being written, or failed.
func (m *Main) explanationBody(c *ui.Context, t *theme.Tokens, it data.RecentItem) {
	source := m.say.RecentExplainSource()
	if it.Kind == data.RecentIssue {
		source = m.say.RecentExplainSourceIssue()
	}
	theme.Text(c, theme.Meta, t.TextSecondary, source)
	x, ok := m.recentCache().Explanation(m.recent.lang, it)
	w := m.explanationOf(it.URL)
	switch {
	case ok:
		for _, p := range []struct{ label, text string }{
			{m.say.RecentWhat(), x.What}, {m.say.RecentWhy(), x.Why}, {m.say.RecentWhere(), x.Where},
		} {
			if p.text == "" {
				continue
			}
			ui.Row(c.Key("recent-point-" + it.URL + p.label)).FillWidth().Gap(10).AlignItems(ui.Start).Children(func() {
				theme.Text(c, theme.MetaStrong, t.TextSecondary, p.label).Width(68).Shrink(0)
				theme.Paragraph(c, theme.Compact, t.Text, p.text, 0).Grow(1).MinWidth(0)
			})
		}
	case w.failed:
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.RecentExplainFailed())
			if link(c.Key("recent-retry-"+it.URL), t, m.say.ConvoRetry()).Clicked() {
				w.failed = false
			}
		})
	default:
		now := m.app.Now()
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.PixelSpinner(c.Key("recent-explain-spin-"+it.URL), theme.IconSmall, t.Accent, theme.SpinnerFrame(now, c.Preferences().ReduceMotion)).Shrink(0)
			c.After(theme.SpinnerStep)
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.RecentExplainReading())
		})
	}
}

// linesBubble is the assistant's lines at the head of a section: its avatar and a small pixel speech
// bubble, with Rewrite and, once more merged since they were written, a dot beside it.
func (m *Main) linesBubble(c *ui.Context, t *theme.Tokens, s recentSec, now time.Time) {
	if !m.hasModel() || s.Count() == 0 {
		return
	}
	st, repo := &m.recent, s.project.RepoSlug
	stored, has := m.recentCache().Lines(repo, st.days, st.lang)
	w := m.linesOf(repo)
	edge := recentBubbleEdge
	ui.Row(c.Key("recent-ai-"+repo)).FillWidth().Shrink(0).Margin(0, 0, 8).AlignItems(ui.Start).Children(func() {
		theme.AvatarIcon(c.Key("recent-ai-face-"+repo), theme.Assistant, 24).Shrink(0)
		ui.Box(c.Key("recent-ai-tail-"+repo)).Size(4*edge, 5*edge).Shrink(0).Margin(9, -edge, 0, 8).
			Draw(func(p *ui.Painter, r ui.Rect) { drawTailCells(p, r, t.Text, t.Bg, edge) })
		box := ui.Column(c.Key("recent-bubble-"+repo)).Grow(1).MinWidth(0).Padding(5+edge, 10+edge, 7+edge).Background(t.Bg).Border(edge, t.Text)
		box.Children(func() {
			for i, at := range [4][2]bool{{true, true}, {false, true}, {true, false}, {false, false}} {
				corner := ui.Box(c.Key("recent-cut-"+repo+itoa(i))).Absolute().Size(edge, edge).Background(t.Bg).PassThrough()
				if at[0] {
					corner.Left(-edge)
				} else {
					corner.Right(-edge)
				}
				if at[1] {
					corner.Top(-edge)
				} else {
					corner.Bottom(-edge)
				}
			}
			ui.Row(c.Key("recent-ai-head-" + repo)).FillWidth().Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Meta, t.TextSecondary, m.say.RecentWritten(st.days)).Shrink(1).MinWidth(0).Ellipsis("…")
				ui.Spacer(c)
				if has && s.newSince(stored.WrittenAt) && !w.writing {
					theme.Dot(c, t.Accent, 7).Shrink(0).Tooltip(m.say.RecentRewriteNew())
				}
				if w.writing {
					theme.PixelSpinner(c.Key("recent-ai-spin-"+repo), 12, t.Accent, theme.SpinnerFrame(now, c.Preferences().ReduceMotion)).Shrink(0)
					c.After(theme.SpinnerStep)
				} else if has || w.failed {
					if link(c.Key("recent-rewrite-"+repo), t, m.say.SummaryRewrite()).Clicked() {
						m.writeLines(s)
					}
				}
			})
			switch {
			case has:
				theme.Paragraph(c, theme.Compact, t.Text, stored.Text, 0).FillWidth()
			case w.failed:
				theme.Paragraph(c, theme.Compact, t.TextSecondary, m.say.SummaryFailed(), 0).FillWidth()
			default:
				ui.Box(c).Height(theme.Compact.Line).Shrink(0)
			}
		})
	})
}
