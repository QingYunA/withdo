package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// ghLoginCommand is what to run when gh is signed out. It is a command, so it is not translated.
const ghLoginCommand = "gh auth login"

// ghRefreshEvery redraws the open pane so that "checked 2m ago" keeps up with the clock.
const ghRefreshEvery = 30 * time.Second

// githubPane is whether pull requests are followed, when they were last looked at, or how to make
// it work; and the Recent changes page's switch. With no poller it says so.
func (s *Settings) githubPane(c *ui.Context, t *theme.Tokens) {
	gh := s.app.GitHub
	group(c, t, "github", func() {
		if gh == nil {
			settingsRow(c, t, "none", s.say.SettingsGitHubNone(), "", true, nil)
		} else {
			st := gh.Status()
			c.After(ghRefreshEvery)
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
		}
		s.recentChangesRow(c, t, false)
	})
}

// recentChangesRow is the switch of the Recent changes page, which is off until the user turns it
// on. Switched on with gh missing or signed out, its second line says what to run; the switch still
// turns.
func (s *Settings) recentChangesRow(c *ui.Context, t *theme.Tokens, first bool) {
	on := s.app.RecentChangesOn()
	meta, ink := s.say.SettingsRecentChangesMeta(), t.TextSecondary
	if on && s.app.GitHub != nil {
		switch s.app.GitHub.Status().State {
		case data.GitHubMissing:
			meta, ink = s.say.SettingsRecentNoGH(ghInstallCommand()), t.Stale
		case data.GitHubSignedOut:
			meta, ink = s.say.SettingsRecentSignedOut(), t.Stale
		}
	}
	settingsRowInk(c, t, "recent-changes", s.say.SettingsRecentChanges(), meta, ink, first, func() {
		ui.Switch(c.Key("recent-changes-switch"), &on).Label(s.say.SettingsRecentChangesSwitch())
	}, nil)
	if on != s.app.RecentChangesOn() {
		s.app.SetRecentChangesOn(on)
	}
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
		return ghInstallCommand()
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
		theme.Text(c, theme.Body, t.TextSecondary, s.say.SettingsGitHubChecking()).Shrink(0)
	default:
		if settingsLink(c, t, "github-check", s.say.SettingsGitHubCheckNow(), s.say.SettingsGitHubCheckNow()).Clicked() {
			s.app.GitHub.CheckNow()
		}
	}
}
