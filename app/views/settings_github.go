package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// What to run when gh is missing or signed out. They are commands, so they are not translated.
const (
	ghInstallCommand = "brew install gh"
	ghLoginCommand   = "gh auth login"
)

// ghRefreshEvery redraws the open sheet so that "checked 2m ago" keeps up with the clock.
const ghRefreshEvery = 30 * time.Second

// githubBlock is Settings > GitHub: whether pull requests are followed, and when they were last
// looked at, or how to make it work. It is left out when the app has no poller.
func (m *Main) githubBlock(c *ui.Context, t *theme.Tokens) {
	gh := m.app.GitHub
	if gh == nil {
		return
	}
	st := gh.Status()
	c.After(ghRefreshEvery)
	ui.Column(c).Gap(8).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, m.say.SettingsGitHub())
		ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
			theme.Paragraph(c, theme.Meta, t.TextSecondary, m.githubLine(st), 0).Grow(1).MinWidth(0)
			m.checkNowLink(c, t, st)
		})
		if command := githubFix(st.State); command != "" {
			ui.Row(c).Padding(10, 12).Radius(theme.RadiusCard).Background(t.BgSubtle).Children(func() {
				theme.Text(c, theme.Mono, t.Text, command).Selectable()
			})
		}
	})
}

// githubLine is the sentence under the title for a status.
func (m *Main) githubLine(st data.GitHubStatus) string {
	switch st.State {
	case data.GitHubReady:
		checked := m.checkedAgo(st.CheckedAt)
		if st.Account == "" {
			return m.say.SettingsGitHubConnectedAnon(checked)
		}
		return m.say.SettingsGitHubConnected(st.Account, checked)
	case data.GitHubMissing:
		return m.say.SettingsGitHubMissing()
	case data.GitHubSignedOut:
		return m.say.SettingsGitHubSignedOut()
	case data.GitHubOffline:
		return m.say.SettingsGitHubOffline()
	}
	return m.say.SettingsGitHubCheckingFirst()
}

// githubFix is the command that fixes a state, or empty when there is nothing to run.
func githubFix(s data.GitHubState) string {
	switch s {
	case data.GitHubMissing:
		return ghInstallCommand
	case data.GitHubSignedOut:
		return ghLoginCommand
	}
	return ""
}

func (m *Main) checkedAgo(at time.Time) string {
	now := m.app.Now()
	if now.Sub(at) < time.Minute {
		return m.say.SettingsGitHubCheckedNow()
	}
	return m.say.SettingsGitHubChecked(m.clock.Age(at, now))
}

// checkNowLink asks for a check. While one runs it reads Checking… and does nothing; before the
// first check has ended there is nothing to ask for.
func (m *Main) checkNowLink(c *ui.Context, t *theme.Tokens, st data.GitHubStatus) {
	switch {
	case st.State == data.GitHubUnknown:
	case st.Checking:
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.SettingsGitHubChecking()).Shrink(0)
	default:
		if link(c.Key("github-check"), t, m.say.SettingsGitHubCheckNow()).Shrink(0).Clicked() {
			m.app.GitHub.CheckNow()
		}
	}
}
