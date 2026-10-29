package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// recentScreen is the Recent changes page: the header, then a section per project.
func (m *Main) recentScreen(c *ui.Context, t *theme.Tokens) {
	now := m.app.Now()
	m.recentTick(c, now)
	secs := m.recentSections()
	m.page(c, t, func() {
		m.recentHeader(c, t, secs, now)
		m.recentBody(c, t, secs, now)
		ui.Box(c).Height(48).Shrink(0)
	})
}

// recentRepo is the repository shown: the one picked, or "" when it is none of the projects'.
func (m *Main) recentRepo() string {
	for _, p := range m.recentProjects() {
		if p.RepoSlug == m.recent.repo {
			return p.RepoSlug
		}
	}
	return ""
}

// recentHeader is the title, the repository pop-up, the 7 | 30 days segment and, at the right, when
// GitHub was read.
func (m *Main) recentHeader(c *ui.Context, t *theme.Tokens, secs []recentSec, now time.Time) {
	st := &m.recent
	ui.Row(c.Key("recent-header")).FillWidth().Height(36).Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Title, t.Text, m.say.NavRecent()).Shrink(0)
		if repo, ok := m.repoPopUp(c, t, m.recentRepo(), m.repoOptions(secs)); ok {
			m.recentPick(repo)
		}
		sel := 0
		if st.days == 30 {
			sel = 1
		}
		if i := segmented(c, t, "recent-days", m.say.NavRecent(), []string{m.say.RecentDays7(), m.say.RecentDays30()}, sel); i >= 0 {
			m.recentPickDays([]int{7, 30}[i])
		}
		ui.Spacer(c)
		m.recentUpdated(c, t, now)
	})
}

// recentUpdated is "Updated 2 min ago" at the right of the header; a click reads GitHub again.
func (m *Main) recentUpdated(c *ui.Context, t *theme.Tokens, now time.Time) {
	st := &m.recent
	if m.app.Recent == nil {
		return
	}
	btn := ui.ButtonBase(c.Key("recent-updated")).Height(theme.HitTarget).Padding(0, 6).Gap(6).Radius(theme.RadiusControl).Shrink(1).MinWidth(0).
		AlignItems(ui.Center).Label(m.say.RecentRefresh()).Tooltip(m.say.RecentRefresh()).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.TextSecondary
	}
	btn.Children(func() {
		if st.fetching {
			theme.PixelSpinner(c.Key("recent-spin"), theme.IconSmall, t.Accent, theme.SpinnerFrame(now, c.Preferences().ReduceMotion)).Shrink(0)
			c.After(theme.SpinnerStep)
			theme.Text(c, theme.Meta, ink, m.say.RecentUpdating()).Ellipsis("…")
			return
		}
		ui.Icon(c, theme.IconClock).Size(theme.IconSmall, theme.IconSmall).TextColor(ink).Shrink(0)
		if !st.fetchedAt.IsZero() {
			theme.Text(c, theme.Meta, ink, m.recentAge(now.Sub(st.fetchedAt))).Ellipsis("…")
		}
	})
	if btn.Clicked() && !st.fetching {
		st.again = true
	}
}

// recentAge says how long ago GitHub was read: "Updated just now", "Updated 2 min ago".
func (m *Main) recentAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return m.say.RecentUpdatedNow()
	case d < time.Hour:
		return m.say.RecentUpdatedMinutes(int(d / time.Minute))
	}
	return m.say.RecentUpdatedHours(int(d / time.Hour))
}

// repoOption is a line of the repository pop-up: a repository, by the project's name, with how many
// items it has in the period (-1 when that is not known).
type repoOption struct {
	value, label string
	count        int
}

// repoOptions are All repositories and then each project's repository with its count.
func (m *Main) repoOptions(secs []recentSec) []repoOption {
	opts := []repoOption{{value: "", label: m.say.RecentAllRepos(), count: -1}}
	for _, p := range m.recentProjects() {
		o := repoOption{value: p.RepoSlug, label: p.Name, count: -1}
		if s, ok := recentOf(secs, p.RepoSlug); ok && s.err == nil {
			o.count = s.Count()
		}
		opts = append(opts, o)
	}
	return opts
}

// repoPopUp is the pop-up that picks the repository. Each repository shows its count; one with
// nothing in the period is dimmed but can be picked. It returns the value picked this frame.
func (m *Main) repoPopUp(c *ui.Context, t *theme.Tokens, current string, opts []repoOption) (string, bool) {
	sel := current
	parts := ui.SelectBase(c.Key("pop-recent-repo"), &sel)
	label := m.say.RecentAllRepos()
	for _, o := range opts {
		if o.value == current {
			label = o.label
		}
	}
	parts.Trigger.Padding(3, 8, 3, 10).Gap(6).Radius(7).AlignItems(ui.Center).Shrink(0).Background(t.BgSubtle).Cursor(ui.CursorPointer).
		Label(m.say.RecentRepoPopUp() + ": " + label)
	parts.Trigger.Children(func() {
		if current != "" {
			ui.Icon(c, theme.IconFolder).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextSecondary).Shrink(0)
		}
		theme.Text(c, theme.Compact, t.Text, label).Ellipsis("…")
		ui.Icon(c, theme.IconChevron).Size(12, 12).TextColor(t.TextTertiary).Shrink(0)
	})
	parts.Popup(func(panel ui.Element) {
		panel.Margin(4, 0, 0).Padding(4).MinWidth(260).Radius(theme.RadiusCard).Background(t.Bg).Border(theme.Hairline, t.Border).Shadow(0, 8, 24, 0, t.Shadow)
		for _, o := range opts {
			item := parts.Item(o.value).Padding(5, 10).Radius(theme.RadiusControl).Gap(8).AlignItems(ui.Center)
			if o.count >= 0 {
				item.Label(o.label + ", " + itoa(o.count))
			}
			ink, quiet := t.Text, t.TextTertiary
			if item.Highlighted() {
				item.Background(t.Accent)
				ink, quiet = t.AccentText, t.AccentText
			}
			if o.count == 0 && !item.Highlighted() {
				ink = t.TextTertiary
			}
			item.Children(func() {
				if o.value != "" {
					ui.Icon(c, theme.IconFolder).Size(theme.IconSmall, theme.IconSmall).TextColor(ink).Shrink(0)
				}
				theme.Text(c, theme.Compact, ink, o.label).Grow(1).MinWidth(0).Ellipsis("…")
				if o.count >= 0 {
					theme.Text(c, theme.Meta, quiet, itoa(o.count)).Shrink(0)
				}
			})
		}
	})
	return sel, sel != current
}

// recentBody is what is under the header: the sections, or the quiet state that stands in for them.
func (m *Main) recentBody(c *ui.Context, t *theme.Tokens, secs []recentSec, now time.Time) {
	st := &m.recent
	switch {
	case len(m.recentProjects()) == 0:
		m.quietState(c, t, m.say.RecentNoReposTitle(), m.say.RecentNoReposBody(), nil)
		return
	case st.results == nil && m.app.Recent != nil:
		m.recentLoading(c, t, now)
		return
	}
	if problem := recentGHProblem(secs); problem != data.RecentOtherProblem {
		m.ghMissingState(c, t, problem)
		return
	}
	pick := m.recentRepo()
	var shown []recentSec
	for _, s := range secs {
		if pick == "" && (s.Count() > 0 || s.err != nil) || pick != "" && s.project.RepoSlug == pick {
			shown = append(shown, s)
		}
	}
	if len(shown) == 0 || len(shown) == 1 && shown[0].err == nil && shown[0].Count() == 0 {
		m.quietState(c, t, m.say.RecentEmptyTitle(st.days), "", nil)
		return
	}
	for _, s := range shown {
		m.recentSection(c, t, s, now)
	}
}

// recentGHProblem is why nothing could be read, when every repository failed because gh is missing
// or signed out: then the page says what to run, and not once for each project.
func recentGHProblem(secs []recentSec) data.RecentProblem {
	if len(secs) == 0 {
		return data.RecentOtherProblem
	}
	problem := data.RecentOtherProblem
	for _, s := range secs {
		if s.err == nil || s.err.Problem != data.RecentNoGH && s.err.Problem != data.RecentSignedOut {
			return data.RecentOtherProblem
		}
		if problem == data.RecentOtherProblem || s.err.Problem == data.RecentNoGH {
			problem = s.err.Problem
		}
	}
	return problem
}

// recentLoading is the quiet line before GitHub has answered the first time.
func (m *Main) recentLoading(c *ui.Context, t *theme.Tokens, now time.Time) {
	ui.Row(c.Key("recent-loading")).Margin(40, 0, 0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.PixelSpinner(c.Key("recent-loading-spin"), theme.IconRow, t.Accent, theme.SpinnerFrame(now, c.Preferences().ReduceMotion))
		c.After(theme.SpinnerStep)
		theme.Text(c, theme.Body, t.TextSecondary, m.say.RecentLoading())
	})
}

// quietState is the assistant's avatar with a title, a line under it and, when extra is given,
// something more.
func (m *Main) quietState(c *ui.Context, t *theme.Tokens, title, body string, extra func()) {
	ui.Column(c.Key("recent-quiet")).FillWidth().Padding(72, 0, 0).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
		theme.AvatarIcon(c, theme.Assistant, theme.AvatarLarge)
		theme.Paragraph(c, theme.BodyStrong, t.Text, title, 0).TextAlign(ui.Center).MaxWidth(emptyBodyMax).Margin(14, 0, 0)
		if body != "" {
			theme.Paragraph(c, theme.Body, t.TextSecondary, body, 0).TextAlign(ui.Center).MaxWidth(emptyBodyMax)
		}
		if extra != nil {
			extra()
		}
	})
}

// ghMissingState says what to run when gh is missing or signed out, with Copy.
func (m *Main) ghMissingState(c *ui.Context, t *theme.Tokens, problem data.RecentProblem) {
	title, command := m.say.RecentSignedOutTitle(), ghLoginCommand
	if problem == data.RecentNoGH {
		title, command = m.say.RecentNoGHTitle(), ghInstallCommand()
	}
	m.quietState(c, t, title, m.say.RecentGHBody(), func() {
		ui.Column(c).Margin(12, 0, 0).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Padding(6, 12).Radius(theme.RadiusControl).Background(t.BgSubtle).Children(func() {
				theme.Text(c, theme.Mono, t.Text, command).Selectable()
			})
			m.copyCommand(c, t, "recent-copy-gh", command)
		})
	})
}

// copyCommand is a Copy button for a command; for a moment after the click it says Copied.
func (m *Main) copyCommand(c *ui.Context, t *theme.Tokens, key, command string) {
	st := &m.recent
	copied := st.copiedKey == key && !st.copied.IsZero() && m.app.Now().Sub(st.copied) < copiedFor
	btn := ui.ButtonBase(c.Key(key)).Height(theme.HitTarget).Padding(0, 6).Gap(4).Radius(theme.RadiusControl).Shrink(0).
		AlignItems(ui.Center).Label(m.say.A11yCopyCommand()).Cursor(ui.CursorPointer)
	ink := t.Accent
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		if copied {
			ui.Icon(c, theme.IconCheck).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextSecondary)
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.SettingsCopied())
			return
		}
		ui.Icon(c, theme.IconCopy).Size(theme.IconSmall, theme.IconSmall).TextColor(ink)
		theme.Text(c, theme.Meta, ink, m.say.SettingsCopy())
	})
	if btn.Clicked() {
		c.WriteClipboard(command)
		st.copied, st.copiedKey = m.app.Now(), key
		c.After(copiedFor + 100*time.Millisecond)
	}
}

// recentSection is a project's part of the page: its head, the assistant's lines, and its groups.
func (m *Main) recentSection(c *ui.Context, t *theme.Tokens, s recentSec, now time.Time) {
	repo := s.project.RepoSlug
	if s.err == nil {
		m.describeIfDue(repo, m.recent.results[repo].Changes)
		m.linesIfDue(s)
	}
	ui.Column(c.Key("recent-"+repo)).FillWidth().Shrink(0).Margin(18, 0, 0).Children(func() {
		m.sectionHead(c, t, s)
		if s.err != nil {
			m.sectionError(c, t, s)
			return
		}
		m.linesBubble(c, t, s, now)
		m.recentGroup(c, t, repo, m.say.RecentGroupNew(), s.New, now)
		m.recentGroup(c, t, repo, m.say.RecentGroupFixed(), s.Fixed, now)
		m.otherGroup(c, t, s, now)
		m.recentGroup(c, t, repo, m.say.RecentGroupOpen(), s.Open, now)
		m.recentGroup(c, t, repo, m.say.RecentGroupIssues(), s.Issues, now)
	})
}

// sectionHead is the folder, the project's name (a click picks its repository), the repository, and
// at the right how many merged, are open and were issues.
func (m *Main) sectionHead(c *ui.Context, t *theme.Tokens, s recentSec) {
	repo := s.project.RepoSlug
	ui.Row(c.Key("recent-head-"+repo)).FillWidth().Shrink(0).Margin(6, 0, 8).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, theme.IconFolder).Size(theme.IconRow, theme.IconRow).TextColor(theme.ProjectColor(s.project.Name)).Shrink(0)
		name := ui.ButtonBase(c.Key("recent-name-" + repo)).Shrink(0).Radius(theme.RadiusControl).Label(m.say.RecentPickRepo(repo)).
			Tooltip(m.say.RecentPickRepo(repo)).Cursor(ui.CursorPointer)
		ink := t.Text
		if name.Hovered() {
			ink = t.TextSecondary
		}
		name.Children(func() { theme.Text(c, theme.BodyBold, ink, s.project.Name) })
		if name.Clicked() {
			m.recentPick(repo)
		}
		theme.Text(c, theme.Meta, t.TextSecondary, repo).Shrink(1).MinWidth(0).Ellipsis("…")
		ui.Spacer(c)
		if counts := m.sectionCounts(s); counts != "" {
			theme.Text(c, theme.Meta, t.TextSecondary, counts).Shrink(0)
		}
	})
}

// sectionCounts is "9 merged · 2 open · 2 issues": the parts there is something of.
func (m *Main) sectionCounts(s recentSec) string {
	var parts []string
	if n := s.Merged(); n > 0 {
		parts = append(parts, m.say.RecentCountMerged(n))
	}
	if n := len(s.Open); n > 0 {
		parts = append(parts, m.say.RecentCountOpen(n))
	}
	if n := len(s.Issues); n > 0 {
		parts = append(parts, m.say.RecentCountIssues(n))
	}
	return strings.Join(parts, " · ")
}

// sectionError is the one line of a repository gh could not read, with a command to copy when there
// is one that helps.
func (m *Main) sectionError(c *ui.Context, t *theme.Tokens, s recentSec) {
	repo := s.project.RepoSlug
	line, command := m.say.RecentSectionError(s.err.Text), ""
	switch s.err.Problem {
	case data.RecentOffline:
		line = m.say.RecentSectionOffline()
	case data.RecentNoAccess, data.RecentSignedOut, data.RecentNoGH:
		line, command = m.say.RecentSectionFailed(), "gh repo view "+repo
	}
	ui.Row(c.Key("recent-err-"+repo)).FillWidth().Shrink(0).Padding(0, 8).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.Paragraph(c, theme.Meta, t.TextSecondary, line, 0).Shrink(1).MinWidth(0)
		if command != "" {
			ui.Row(c).Padding(1, 6).Radius(theme.RadiusControl).Background(t.BgSubtle).Shrink(0).Children(func() {
				theme.Text(c, theme.Mono, t.Text, command).Selectable()
			})
			m.copyCommand(c, t, "recent-copy-"+repo, command)
		}
	})
}

// recentLink is the Recent changes link in the title bar of a project that has a repository, once
// the page is switched on: it opens the page on that repository.
func (m *Main) recentLink(c *ui.Context, t *theme.Tokens) {
	if m.nav.screen != screenProject || !m.app.RecentChangesOn() {
		return
	}
	p, ok := m.app.Project(m.nav.projectID)
	if !ok || p.RepoSlug == "" {
		return
	}
	btn := ui.ButtonBase(c.Key("recent-link")).Height(28).Padding(0, 8, 0, 6).Gap(4).Radius(theme.RadiusControl).Shrink(0).Margin(0, 8, 0, 0).
		AlignItems(ui.Center).Label(m.say.RecentLinkTip()).Tooltip(m.say.RecentLinkTip()).Cursor(ui.CursorPointer)
	ink := t.TextSecondary
	if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.Text
	}
	btn.Children(func() {
		ui.Icon(c, theme.IconBranch).Size(theme.IconSmall, theme.IconSmall).TextColor(ink)
		theme.Text(c, theme.Meta, ink, m.say.RecentLink())
	})
	if btn.Clicked() {
		m.ShowRecent(p.RepoSlug)
	}
}
