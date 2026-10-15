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

// ghRefreshEvery redraws the open pane so that "checked 2m ago" keeps up with the clock.
const ghRefreshEvery = 30 * time.Second

// githubPane is whether pull requests are followed, and when they were last looked at, or how to
// make it work. With no poller it says so.
func (s *Settings) githubPane(c *ui.Context, t *theme.Tokens) {
	gh := s.app.GitHub
	if gh == nil {
		group(c, t, "github", func() { settingsRow(c, t, "none", s.say.SettingsGitHubNone(), "", true, nil) })
		return
	}
	st := gh.Status()
	c.After(ghRefreshEvery)
	group(c, t, "github", func() {
		ui.Row(c).Shrink(0).MinHeight(theme.SettingsRowMin).Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(func() {
			theme.Paragraph(c, theme.Body, t.Text, s.githubLine(st), 0).Grow(1).MinWidth(0)
			s.checkNowLink(c, t, st)
		})
		if command := githubFix(st.State); command != "" {
			hairline(c, t).Margin(0, 14)
			ui.Column(c).Shrink(0).Padding(10, 14, 12).Children(func() {
				commandWell(c, t, command, nil)
			})
		}
	})
}

// githubLine is the sentence for a status.
func (s *Settings) githubLine(st data.GitHubStatus) string {
	switch st.State {
	case data.GitHubReady:
		checked := s.checkedAgo(st.CheckedAt)
		if st.Account == "" {
			return s.say.SettingsGitHubConnectedAnon(checked)
		}
		return s.say.SettingsGitHubConnected(st.Account, checked)
	case data.GitHubMissing:
		return s.say.SettingsGitHubMissing()
	case data.GitHubSignedOut:
		return s.say.SettingsGitHubSignedOut()
	case data.GitHubOffline:
		return s.say.SettingsGitHubOffline()
	}
	return s.say.SettingsGitHubCheckingFirst()
}

// githubFix is the command that fixes a state, or empty when there is nothing to run.
func githubFix(st data.GitHubState) string {
	switch st {
	case data.GitHubMissing:
		return ghInstallCommand
	case data.GitHubSignedOut:
		return ghLoginCommand
	}
	return ""
}

func (s *Settings) checkedAgo(at time.Time) string {
	now := s.app.Now()
	if now.Sub(at) < time.Minute {
		return s.say.SettingsGitHubCheckedNow()
	}
	return s.say.SettingsGitHubChecked(s.clock.Age(at, now))
}

// checkNowLink asks for a check. While one runs it reads Checking… and does nothing; before the
// first check has ended there is nothing to ask for.
func (s *Settings) checkNowLink(c *ui.Context, t *theme.Tokens, st data.GitHubStatus) {
	switch {
	case st.State == data.GitHubUnknown:
	case st.Checking:
		theme.Text(c, theme.Body, t.TextTertiary, s.say.SettingsGitHubChecking()).Shrink(0)
	default:
		if settingsLink(c, t, "github-check", s.say.SettingsGitHubCheckNow(), s.say.SettingsGitHubCheckNow()).Clicked() {
			s.app.GitHub.CheckNow()
		}
	}
}
