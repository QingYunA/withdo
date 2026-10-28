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

// The Sync pane names the computer the app runs on: "this Mac" on a Mac, "this PC" on Windows.
// Each helper picks the catalog message written for the system.

func syncThisDevice(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncThisMacWindows()
	}
	return say.SettingsSyncThisMac()
}

func syncThisDeviceBadge(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncThisMacBadgeWindows()
	}
	return say.SettingsSyncThisMacBadge()
}

func syncUploading(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncUploadingWindows()
	}
	return say.SettingsSyncUploading()
}

func syncWaiting(say locales.Messages, count string) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncWaitingWindows(count)
	}
	return say.SettingsSyncWaiting(count)
}

func syncBothTitle(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncBothTitleWindows()
	}
	return say.SettingsSyncBothTitle()
}

func syncBothBody(say locales.Messages, devices string, tasks string, projects string) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncBothBodyWindows(devices, tasks, projects)
	}
	return say.SettingsSyncBothBody(devices, tasks, projects)
}

func syncMergeBody(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncMergeBodyWindows()
	}
	return say.SettingsSyncMergeBody()
}

func syncUseAccountBody(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncUseAccountBodyWindows()
	}
	return say.SettingsSyncUseAccountBody()
}

func syncLocalNote(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncLocalNoteWindows()
	}
	return say.SettingsSyncLocalNote()
}

func syncWrongAccount(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncWrongAccountWindows()
	}
	return say.SettingsSyncWrongAccount()
}

func syncOtherTitle(say locales.Messages) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncOtherTitleWindows()
	}
	return say.SettingsSyncOtherTitle()
}

func syncOtherOldBody(say locales.Messages, account string) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncOtherOldBodyWindows(account)
	}
	return say.SettingsSyncOtherOldBody(account)
}

func syncOtherMoveLabel(say locales.Messages, account string) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncOtherMoveWindows(account)
	}
	return say.SettingsSyncOtherMove(account)
}

func syncOtherMoveBody(say locales.Messages, account string, previous string) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncOtherMoveBodyWindows(account, previous)
	}
	return say.SettingsSyncOtherMoveBody(account, previous)
}

func syncOtherMoveBodyUnknown(say locales.Messages, account string) string {
	if systemGOOS == "windows" {
		return say.SettingsSyncOtherMoveBodyUnknownWindows(account)
	}
	return say.SettingsSyncOtherMoveBodyUnknown(account)
}
