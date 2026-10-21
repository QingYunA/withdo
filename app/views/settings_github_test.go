package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// githubPane opens Settings > Agents, scrolled to the GitHub group at its end, in an app whose GitHub status is st.
func githubPane(t *testing.T, st data.GitHubStatus) (*settingsRig, *data.FakeGitHub) {
	t.Helper()
	gh := data.NewFakeGitHub(st)
	r := newSettingsRig(t, func(a *App) { a.GitHub = gh })
	r.pane(t, "Agents")
	r.tt.Scroll(340, 300, 0, 9000)
	r.tt.Frame()
	return r, gh
}

func readyAt(ago time.Duration) data.GitHubStatus {
	return data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-ago)}
}

func TestGitHubPaneShowsTheAccountAndWhenItWasChecked(t *testing.T) {
	r, _ := githubPane(t, readyAt(2*time.Minute))
	has(t, r.tt, "Connected as QingYunA · checked 2m ago", "Check now")
	hasNot(t, r.tt, "brew install gh", "gh auth login")
}

func TestGitHubPaneJustChecked(t *testing.T) {
	r, _ := githubPane(t, readyAt(10*time.Second))
	has(t, r.tt, "Connected as QingYunA · checked just now")
}

func TestGitHubPaneWithoutAnAccountName(t *testing.T) {
	st := readyAt(5 * time.Minute)
	st.Account = ""
	r, _ := githubPane(t, st)
	has(t, r.tt, "Connected to GitHub · checked 5m ago")
}

func TestGitHubPaneWhileChecking(t *testing.T) {
	st := readyAt(2 * time.Minute)
	st.Checking = true
	r, gh := githubPane(t, st)
	has(t, r.tt, "Connected as QingYunA · checked 2m ago", "Checking…")
	hasNot(t, r.tt, "Check now")
	_ = r.tt.Click("Checking…")
	if gh.Checks != 0 {
		t.Error("the inert Checking… asked for another check")
	}
}

func TestGitHubPaneBeforeTheFirstCheck(t *testing.T) {
	r, _ := githubPane(t, data.GitHubStatus{Checking: true})
	has(t, r.tt, "Checking GitHub…")
	hasNot(t, r.tt, "Check now", "Connected")
}

func TestGitHubPaneFixTexts(t *testing.T) {
	tests := []struct {
		name    string
		state   data.GitHubState
		text    string
		command string
	}{
		{"missing", data.GitHubMissing, "The GitHub CLI isn't installed. Install it to follow your pull requests.", "brew install gh"},
		{"signed out", data.GitHubSignedOut, "You're not signed in to GitHub. Sign in to follow your pull requests.", "gh auth login"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := githubPane(t, data.GitHubStatus{State: tc.state, CheckedAt: fixedNow()})
			has(t, r.tt, tc.text, tc.command, "Check now")
			hasNot(t, r.tt, "Connected as")
		})
	}
}

func TestGitHubPaneOffline(t *testing.T) {
	r, _ := githubPane(t, data.GitHubStatus{State: data.GitHubOffline, CheckedAt: fixedNow()})
	has(t, r.tt, "Can't reach GitHub. Trying again soon.", "Check now")
	hasNot(t, r.tt, "brew install gh", "gh auth login")
}

func TestGitHubPaneInChinese(t *testing.T) {
	tests := []struct {
		name string
		st   data.GitHubStatus
		want []string
	}{
		{"ready", readyAt(2 * time.Minute), []string{"已连接：QingYunA · 2分钟前检查过", "立即检查"}},
		{"just now", readyAt(5 * time.Second), []string{"已连接：QingYunA · 刚刚检查过"}},
		{"checking", data.GitHubStatus{Checking: true}, []string{"正在检查 GitHub…"}},
		{"missing", data.GitHubStatus{State: data.GitHubMissing}, []string{"未安装 GitHub CLI。安装后即可跟进你的 PR。", "brew install gh"}},
		{"signed out", data.GitHubStatus{State: data.GitHubSignedOut}, []string{"尚未登录 GitHub。登录后即可跟进你的 PR。", "gh auth login"}},
		{"offline", data.GitHubStatus{State: data.GitHubOffline}, []string{"无法连接 GitHub，稍后会再试。"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newSettingsRig(t, func(a *App) { a.GitHub = data.NewFakeGitHub(tc.st) })
			inChinese(r.tt)
			r.s.Show(PaneAgents)
			r.tt.Frame()
			r.tt.Scroll(340, 300, 0, 9000)
			r.tt.Frame()
			has(t, r.tt, tc.want...)
			if r.title != "Agent" {
				t.Errorf("title %q", r.title)
			}
		})
	}
}
