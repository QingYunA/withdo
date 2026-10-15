package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// githubSettings opens Settings in an app whose GitHub status is st.
func githubSettings(t *testing.T, st data.GitHubStatus) (*ui.Tester, *Main, *data.FakeGitHub) {
	t.Helper()
	app, _, _ := newTestApp(t)
	gh := data.NewFakeGitHub(st)
	app.GitHub = gh
	tt, m, _ := newTesterFor(app, 1100)
	m.OpenSettings()
	tt.Frame()
	return tt, m, gh
}

func readyAt(ago time.Duration) data.GitHubStatus {
	return data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-ago)}
}

func TestSettingsGitHubShowsTheAccountAndWhenItWasChecked(t *testing.T) {
	tt, _, _ := githubSettings(t, readyAt(2*time.Minute))
	has(t, tt, "GitHub", "Connected as QingYunA · checked 2m ago", "Check now")
	hasNot(t, tt, "brew install gh", "gh auth login")
}

func TestSettingsGitHubJustChecked(t *testing.T) {
	tt, _, _ := githubSettings(t, readyAt(10*time.Second))
	has(t, tt, "Connected as QingYunA · checked just now")
}

func TestSettingsGitHubWithoutAnAccountName(t *testing.T) {
	st := readyAt(5 * time.Minute)
	st.Account = ""
	tt, _, _ := githubSettings(t, st)
	has(t, tt, "Connected to GitHub · checked 5m ago")
}

func TestSettingsGitHubCheckNowAsksForACheck(t *testing.T) {
	tt, _, gh := githubSettings(t, readyAt(2*time.Minute))
	if err := tt.Click("Check now"); err != nil {
		t.Fatal(err)
	}
	if gh.Checks != 1 {
		t.Errorf("CheckNow called %d times, want 1", gh.Checks)
	}
}

func TestSettingsGitHubWhileChecking(t *testing.T) {
	st := readyAt(2 * time.Minute)
	st.Checking = true
	tt, _, gh := githubSettings(t, st)
	has(t, tt, "Connected as QingYunA · checked 2m ago", "Checking…")
	hasNot(t, tt, "Check now")
	_ = tt.Click("Checking…")
	if gh.Checks != 0 {
		t.Error("the inert Checking… asked for another check")
	}
}

func TestSettingsGitHubBeforeTheFirstCheck(t *testing.T) {
	tt, _, _ := githubSettings(t, data.GitHubStatus{Checking: true})
	has(t, tt, "Checking GitHub…")
	hasNot(t, tt, "Check now", "Connected")
}

func TestSettingsGitHubFixTexts(t *testing.T) {
	tests := []struct {
		name    string
		state   data.GitHubState
		text    string
		command string
	}{
		{"missing", data.GitHubMissing, "The GitHub CLI isn't installed. Install it to follow your pull requests.", "brew install gh"},
		{"signed out", data.GitHubSignedOut, "You're not signed in to GitHub. Sign in to follow your pull requests.", "gh auth login"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester, _, _ := githubSettings(t, data.GitHubStatus{State: tt.state, CheckedAt: fixedNow()})
			has(t, tester, "GitHub", tt.text, tt.command, "Check now")
			hasNot(t, tester, "Connected as")
		})
	}
}

func TestSettingsGitHubOffline(t *testing.T) {
	tt, _, _ := githubSettings(t, data.GitHubStatus{State: data.GitHubOffline, CheckedAt: fixedNow()})
	has(t, tt, "Can't reach GitHub. Trying again soon.", "Check now")
	hasNot(t, tt, "brew install gh", "gh auth login")
}

func TestSettingsWithoutGitHubHasNoSection(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.OpenSettings()
	tt.Frame()
	hasNot(t, tt, "GitHub", "Check now")
}

func TestSettingsGitHubInChinese(t *testing.T) {
	tests := []struct {
		name string
		st   data.GitHubStatus
		want []string
	}{
		{"ready", readyAt(2 * time.Minute), []string{"GitHub", "已连接：QingYunA · 2分钟前检查过", "立即检查"}},
		{"just now", readyAt(5 * time.Second), []string{"已连接：QingYunA · 刚刚检查过"}},
		{"checking", data.GitHubStatus{Checking: true}, []string{"正在检查 GitHub…"}},
		{"missing", data.GitHubStatus{State: data.GitHubMissing}, []string{"未安装 GitHub CLI。安装后即可跟进你的 PR。", "brew install gh"}},
		{"signed out", data.GitHubStatus{State: data.GitHubSignedOut}, []string{"尚未登录 GitHub。登录后即可跟进你的 PR。", "gh auth login"}},
		{"offline", data.GitHubStatus{State: data.GitHubOffline}, []string{"无法连接 GitHub，稍后会再试。"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, _, _ := newTestApp(t)
			app.GitHub = data.NewFakeGitHub(tc.st)
			tt, m, _ := newTesterFor(app, 1100)
			inChinese(tt)
			m.OpenSettings()
			tt.Frame()
			has(t, tt, tc.want...)
		})
	}
}
