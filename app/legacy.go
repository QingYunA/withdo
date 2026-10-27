package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// The app used to be called ADHD Todo. mygo's name is also the executable, bundle and installer
// name, and an old version updates itself into the old place under the old name (the new bundle is
// renamed to /Applications/ADHD Todo.app; the files of the new archive are swapped into
// %LOCALAPPDATA%\Programs\ADHD Todo), then starts the old executable name again. The fix-ups here
// run in the new version and make such an install look and behave like a fresh Withdo one.
const (
	legacyName = "ADHD Todo"
	appName    = "Withdo"
	// uninstallKey is the Windows uninstall entry the installer wrote for the app's identifier.
	uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\dev.adhdtodo.desktop`
)

// bundleMove is the rename of an old install's bundle to the new name.
type bundleMove struct{ from, to string }

// planBundleMove decides whether the bundle that holds exe (…/ADHD Todo.app/Contents/MacOS/Withdo)
// moves to Withdo.app in the same folder: only an updated old install in a folder the user can
// write, where Withdo.app does not exist. A bundle on a disk image or read-only volume, or one that
// Gatekeeper translocated, stays as it is (as for the adhd-core link, linkBundledCore). writable
// and exists look at the file system.
func planBundleMove(exe string, writable func(dir string) bool, exists func(path string) bool) (bundleMove, bool) {
	macos := filepath.Dir(exe)
	bundle := filepath.Dir(filepath.Dir(macos))
	if filepath.Base(macos) != "MacOS" || filepath.Base(bundle) != legacyName+".app" {
		return bundleMove{}, false
	}
	if strings.HasPrefix(exe, "/Volumes/") || strings.Contains(exe, "/AppTranslocation/") {
		return bundleMove{}, false
	}
	dir := filepath.Dir(bundle)
	to := filepath.Join(dir, appName+".app")
	if !writable(dir) || exists(to) {
		return bundleMove{}, false
	}
	return bundleMove{from: bundle, to: to}, true
}

// bundleOps are what moveBundle does to the system.
type bundleOps struct {
	rename func(from, to string) error
	// open starts a new instance of the app at path with the arguments of this launch.
	open func(path string) error
}

// moveBundle renames the bundle and starts the app from its new path, and reports whether it did:
// this process then has to end. If it cannot start the new bundle it puts the old name back and
// carries on in place, to try again at the next launch.
func moveBundle(m bundleMove, ops bundleOps) bool {
	if err := ops.rename(m.from, m.to); err != nil {
		log.Printf("rename %s: %v", m.from, err)
		return false
	}
	if err := ops.open(m.to); err != nil {
		log.Printf("start %s: %v", m.to, err)
		if err := ops.rename(m.to, m.from); err != nil {
			log.Printf("rename back %s: %v", m.to, err)
		}
		return false
	}
	return true
}

// restartRenamedBundle gives an updated old install its new name, ADHD Todo.app to Withdo.app, and
// starts the app from there. It reports whether it did, and then main ends without further ado.
//
// mygo remembers the path it started from (startExe) and uses it to find the bundle an in-session
// update replaces and to relaunch after one, so after the rename the process must be a new one.
// It runs at the very top of main, before this process takes the single-instance lock, so the new
// instance gets the lock; open -n makes a new instance although the same app (by bundle identifier)
// is still running as this process. Renaming a running bundle is allowed on macOS.
func restartRenamedBundle() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return false
	}
	move, ok := planBundleMove(exe, writableDir, pathExists)
	if !ok {
		return false
	}
	return moveBundle(move, bundleOps{rename: os.Rename, open: func(path string) error {
		args := []string{"-n", "-a", path}
		if len(os.Args) > 1 {
			args = append(args, "--args")
			args = append(args, os.Args[1:]...)
		}
		return exec.Command("/usr/bin/open", args...).Run()
	}})
}

// writableDir reports whether a file can be made in dir, as mygo's updater probes before it
// replaces the app; a read-only volume fails too.
func writableDir(dir string) bool {
	probe, err := os.MkdirTemp(dir, ".withdo-rename-")
	if err != nil {
		return false
	}
	_ = os.Remove(probe)
	return true
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// windowsInstall is what the Windows fix-up reads and changes.
type windowsInstall struct {
	exe                string // the running executable
	startMenu, desktop string // the folders of the shortcuts the installer made
	exists             func(path string) bool
	rename             func(from, to string) error
	// getString and setString read and write a value of the app's uninstall entry; getString fails
	// when there is none.
	getString func(name string) (string, error)
	setString func(name, value string) error
}

// fixWindowsInstall makes an install that an old version updated look like one of Withdo: the
// shortcuts ADHD Todo.lnk of the Start menu and the Desktop become Withdo.lnk (they point at the
// launcher that stands in for the old executable, so they keep working), and the uninstall entry
// names Withdo and its icon. It only touches an install in a folder named ADHD Todo, which no
// installer of Withdo makes, and does nothing the second time. Failures are logged.
func fixWindowsInstall(w windowsInstall) {
	if !strings.EqualFold(filepath.Base(filepath.Dir(w.exe)), legacyName) {
		return
	}
	for _, folder := range []string{w.startMenu, w.desktop} {
		if folder == "" {
			continue
		}
		from, to := filepath.Join(folder, legacyName+".lnk"), filepath.Join(folder, appName+".lnk")
		if !w.exists(from) || w.exists(to) {
			continue
		}
		if err := w.rename(from, to); err != nil {
			log.Printf("rename %s: %v", from, err)
		}
	}
	if name, err := w.getString("DisplayName"); err == nil && name == legacyName {
		if err := w.setString("DisplayName", appName); err != nil {
			log.Printf("uninstall entry DisplayName: %v", err)
		}
	}
	if icon, err := w.getString("DisplayIcon"); err == nil {
		if next, ok := withNewExe(icon); ok {
			if err := w.setString("DisplayIcon", next); err != nil {
				log.Printf("uninstall entry DisplayIcon: %v", err)
			}
		}
	}
}

// withNewExe turns the path of the old executable ("…\ADHD Todo.exe", with either separator and
// maybe in quotes) into the one of Withdo.exe in the same folder; ok is false for any other path.
func withNewExe(path string) (string, bool) {
	bare := strings.Trim(path, `"`)
	i := strings.LastIndexAny(bare, `\/`)
	if !strings.EqualFold(bare[i+1:], legacyName+".exe") {
		return "", false
	}
	return bare[:i+1] + appName + ".exe", true
}

// migrateWindowsInstall runs fixWindowsInstall for the running app on Windows.
func migrateWindowsInstall() {
	if runtime.GOOS != "windows" {
		return
	}
	if w, ok := newWindowsInstall(); ok {
		fixWindowsInstall(w)
	}
}
