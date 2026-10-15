package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The M4a screens: the pull request rows of Needs you and Settings > GitHub, in light and dark,
// English and Chinese (the -zh suffix). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensGitHub

func githubStates() []struct {
	name string
	st   data.GitHubStatus
} {
	return []struct {
		name string
		st   data.GitHubStatus
	}{
		{"", data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-2 * time.Minute)}},
		{"-checking", data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-6 * time.Minute), Checking: true}},
		{"-missing", data.GitHubStatus{State: data.GitHubMissing, CheckedAt: fixedNow()}},
		{"-signed-out", data.GitHubStatus{State: data.GitHubSignedOut, CheckedAt: fixedNow()}},
		{"-offline", data.GitHubStatus{State: data.GitHubOffline, CheckedAt: fixedNow()}},
	}
}

func TestScreensGitHub(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := prApp(t)
		mainShot(t, "now-pull-requests-"+a.suffix+".png", app, a.dark, nil)
		mainShot(t, "now-pull-requests-narrow-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			tt.SetSize(860, 600)
			m.OpenTask(1)
		})
	}
}

func TestScreensGitHubChinese(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := prApp(t)
		zhShot(t, "now-pull-requests-zh-"+a.suffix+".png", app, a.dark, nil)
		zhShot(t, "now-pull-requests-narrow-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			tt.SetSize(860, 600)
			m.OpenTask(1)
		})
	}
}
