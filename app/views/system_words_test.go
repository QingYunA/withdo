package views

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
)

// The system's own apps and commands are named for the system the app runs on: Finder, Preview and
// Homebrew on a Mac, File Explorer, the default app and winget on Windows.
func TestSystemWordsNameTheSystemsOwnApps(t *testing.T) {
	cases := []struct {
		goos, lang                              string
		showInFolder, folderApp, viewerOpen, gh string
	}{
		{"darwin", "", "Show in Finder", "Finder", "Open in Preview", "brew install gh"},
		{"darwin", "zh-Hans", "在访达中显示", "Finder", "在预览中打开", "brew install gh"},
		{"windows", "", "Show in File Explorer", "File Explorer", "Open", "winget install --id GitHub.cli"},
		{"windows", "zh-Hans", "在文件资源管理器中显示", "文件资源管理器", "打开", "winget install --id GitHub.cli"},
	}
	for _, tc := range cases {
		t.Run(tc.goos+" "+tc.lang, func(t *testing.T) {
			old := systemGOOS
			systemGOOS = tc.goos
			t.Cleanup(func() { systemGOOS = old })
			say := locales.In(tc.lang)
			if got := showInFolderLabel(say); got != tc.showInFolder {
				t.Errorf("showInFolderLabel = %q, want %q", got, tc.showInFolder)
			}
			if got := openWithName(say, OpenFinder); got != tc.folderApp {
				t.Errorf("openWithName(OpenFinder) = %q, want %q", got, tc.folderApp)
			}
			if got := viewerOpenLabel(say); got != tc.viewerOpen {
				t.Errorf("viewerOpenLabel = %q, want %q", got, tc.viewerOpen)
			}
			if got := ghInstallCommand(); got != tc.gh {
				t.Errorf("ghInstallCommand = %q, want %q", got, tc.gh)
			}
			if got := githubFix(data.GitHubMissing); got != tc.gh {
				t.Errorf("githubFix(missing) = %q, want %q", got, tc.gh)
			}
			if got, want := say.SettingsRecentNoGH(ghInstallCommand()), tc.gh; !strings.Contains(got, want) {
				t.Errorf("SettingsRecentNoGH = %q, want it to name %q", got, want)
			}
		})
	}
}
