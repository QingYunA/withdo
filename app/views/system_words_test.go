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

// The Sync pane calls this computer a Mac on a Mac and a PC on Windows, in both languages, and the
// two variants of a message take the same placeholders.
func TestSyncWordsNameThisComputer(t *testing.T) {
	cases := []struct {
		goos, lang, thisDevice, body, other string
	}{
		{"darwin", "", "This Mac", "The account already has data from 2 devices. This Mac has 3 tasks and 4 projects.", "This Mac's tasks go to a@b.c. They stay on this Mac; x@y.z keeps its own copy."},
		{"darwin", "zh-Hans", "这台 Mac", "账号里已经有 2 台设备的数据。这台 Mac 上有 3 个任务、4 个项目。", "这台 Mac 上的任务会进入 a@b.c，本机仍然保留；x@y.z 那边保留它自己的一份。"},
		{"windows", "", "This PC", "The account already has data from 2 devices. This PC has 3 tasks and 4 projects.", "This PC's tasks go to a@b.c. They stay on this PC; x@y.z keeps its own copy."},
		{"windows", "zh-Hans", "这台电脑", "账号里已经有 2 台设备的数据。这台电脑上有 3 个任务、4 个项目。", "这台电脑上的任务会进入 a@b.c，本机仍然保留；x@y.z 那边保留它自己的一份。"},
	}
	for _, tc := range cases {
		t.Run(tc.goos+" "+tc.lang, func(t *testing.T) {
			old := systemGOOS
			systemGOOS = tc.goos
			t.Cleanup(func() { systemGOOS = old })
			say := locales.In(tc.lang)
			if got := syncThisDevice(say); got != tc.thisDevice {
				t.Errorf("syncThisDevice = %q, want %q", got, tc.thisDevice)
			}
			if got := syncBothBody(say, "2", "3", "4"); got != tc.body {
				t.Errorf("syncBothBody = %q, want %q", got, tc.body)
			}
			if got := syncOtherMoveBody(say, "a@b.c", "x@y.z"); got != tc.other {
				t.Errorf("syncOtherMoveBody = %q, want %q", got, tc.other)
			}
			// Every helper fills its placeholders, and none names a Mac on Windows.
			for name, got := range map[string]string{
				"waiting":     syncWaiting(say, "5"),
				"unknown":     syncOtherMoveBodyUnknown(say, "a@b.c"),
				"oldBody":     syncOtherOldBody(say, "a@b.c"),
				"moveLabel":   syncOtherMoveLabel(say, "a@b.c"),
				"uploading":   syncUploading(say),
				"wrongAcct":   syncWrongAccount(say),
				"localNote":   syncLocalNote(say),
				"otherTitle":  syncOtherTitle(say),
				"bothTitle":   syncBothTitle(say),
				"mergeBody":   syncMergeBody(say),
				"useAcctBody": syncUseAccountBody(say),
				"badge":       syncThisDeviceBadge(say),
			} {
				if strings.ContainsAny(got, "{}") {
					t.Errorf("%s = %q, has an unfilled placeholder", name, got)
				}
				if tc.goos == "windows" && strings.Contains(got, "Mac") {
					t.Errorf("%s = %q, names a Mac on Windows", name, got)
				}
			}
		})
	}
}
