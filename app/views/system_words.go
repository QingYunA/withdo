package views

import (
	"runtime"

	"github.com/QingYunA/adhd-todo/app/locales"
)

// systemGOOS is the system the words are written for: its key hints, and its own apps and commands.
// The tests switch it.
var systemGOOS = runtime.GOOS

// showInFolderLabel is the link that shows a folder in the system's file manager.
func showInFolderLabel(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsShowInExplorer()
	}
	return say.SettingsShowInFinder()
}

// viewerOpenLabel is the viewer's button that opens the picture in another app: Preview on a Mac,
// the default app on Windows.
func viewerOpenLabel(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.ViewerOpen()
	}
	return say.ViewerOpenInPreview()
}

// ghInstallCommand is what installs the GitHub CLI: Homebrew on a Mac, winget on Windows. It is a
// command, so it is not translated.
func ghInstallCommand() string {
	if systemGOOS == "windows" {
		return "winget install --id GitHub.cli"
	}
	return "brew install gh"
}
