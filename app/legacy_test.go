//go:build !windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanBundleMove(t *testing.T) {
	yes := func(string) bool { return true }
	no := func(string) bool { return false }
	exe := func(dir, bundle string) string {
		return filepath.Join(dir, bundle, "Contents", "MacOS", "Withdo")
	}
	cases := []struct {
		name     string
		exe      string
		writable func(string) bool
		exists   func(string) bool
		want     bool
	}{
		{"an updated old install", exe("/Applications", "ADHD Todo.app"), yes, no, true},
		{"already named Withdo", exe("/Applications", "Withdo.app"), yes, no, false},
		{"a run outside a bundle", "/tmp/build/app", yes, no, false},
		{"a folder the user cannot write", exe("/Applications", "ADHD Todo.app"), no, no, false},
		{"Withdo.app already there", exe("/Applications", "ADHD Todo.app"), yes, yes, false},
		{"a disk image", exe("/Volumes/ADHD Todo", "ADHD Todo.app"), yes, no, false},
		{"a translocated app", "/private/var/folders/x/T/AppTranslocation/0A1B/d/ADHD Todo.app/Contents/MacOS/Withdo", yes, no, false},
	}
	for _, c := range cases {
		got, ok := planBundleMove(c.exe, c.writable, c.exists)
		if ok != c.want {
			t.Errorf("%s: moves = %v, want %v", c.name, ok, c.want)
		}
		if c.want && (got.from != "/Applications/ADHD Todo.app" || got.to != "/Applications/Withdo.app") {
			t.Errorf("%s: %+v", c.name, got)
		}
	}
}

func TestWritableDirAndPathExists(t *testing.T) {
	dir := t.TempDir()
	if !writableDir(dir) {
		t.Error("a temp folder is writable")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("the probe left %d entries", len(entries))
	}
	if writableDir(filepath.Join(dir, "missing")) {
		t.Error("a missing folder is not writable")
	}
	if !pathExists(dir) || pathExists(filepath.Join(dir, "missing")) {
		t.Error("pathExists is wrong")
	}
}

func TestMoveBundleRenamesAndStartsTheNewPath(t *testing.T) {
	m := bundleMove{from: "/A/ADHD Todo.app", to: "/A/Withdo.app"}
	var renames [][2]string
	var opened string
	ops := bundleOps{
		rename: func(from, to string) error { renames = append(renames, [2]string{from, to}); return nil },
		open:   func(path string) error { opened = path; return nil },
	}
	if !moveBundle(m, ops) {
		t.Fatal("moveBundle reported no restart")
	}
	if len(renames) != 1 || renames[0] != [2]string{m.from, m.to} || opened != m.to {
		t.Errorf("renames %v, opened %q", renames, opened)
	}
}

func TestMoveBundleCarriesOnInPlaceOnFailure(t *testing.T) {
	m := bundleMove{from: "/A/ADHD Todo.app", to: "/A/Withdo.app"}
	boom := errors.New("boom")

	var renames [][2]string
	ops := bundleOps{
		rename: func(from, to string) error { renames = append(renames, [2]string{from, to}); return boom },
		open:   func(string) error { t.Error("started the app after a failed rename"); return nil },
	}
	if moveBundle(m, ops) {
		t.Error("restarted after a failed rename")
	}

	renames = nil
	ops = bundleOps{
		rename: func(from, to string) error { renames = append(renames, [2]string{from, to}); return nil },
		open:   func(string) error { return boom },
	}
	if moveBundle(m, ops) {
		t.Error("restarted although the new bundle did not start")
	}
	if len(renames) != 2 || renames[1] != [2]string{m.to, m.from} {
		t.Errorf("the old name was not put back: %v", renames)
	}
}

// fakeWindows is a file system of paths and an uninstall entry, with a record of the renames.
type fakeWindows struct {
	files   map[string]bool
	reg     map[string]string
	renames [][2]string
	failSet bool
}

func (f *fakeWindows) install(exe string) windowsInstall {
	return windowsInstall{
		exe:       exe,
		startMenu: "/menu",
		desktop:   "/desk",
		exists:    func(p string) bool { return f.files[p] },
		rename: func(from, to string) error {
			f.renames = append(f.renames, [2]string{from, to})
			delete(f.files, from)
			f.files[to] = true
			return nil
		},
		getString: func(name string) (string, error) {
			if v, ok := f.reg[name]; ok {
				return v, nil
			}
			return "", errors.New("not found")
		},
		setString: func(name, value string) error {
			if f.failSet {
				return errors.New("denied")
			}
			f.reg[name] = value
			return nil
		},
	}
}

func legacyWindows() *fakeWindows {
	return &fakeWindows{
		files: map[string]bool{"/menu/ADHD Todo.lnk": true, "/desk/ADHD Todo.lnk": true},
		reg: map[string]string{
			"DisplayName": "ADHD Todo",
			"DisplayIcon": `C:\Users\u\AppData\Local\Programs\ADHD Todo\ADHD Todo.exe`,
		},
	}
}

const legacyExe = "/Programs/ADHD Todo/Withdo.exe"

func TestFixWindowsInstallRenamesShortcutsAndUninstallEntry(t *testing.T) {
	f := legacyWindows()
	fixWindowsInstall(f.install(legacyExe))
	if !f.files["/menu/Withdo.lnk"] || !f.files["/desk/Withdo.lnk"] || f.files["/menu/ADHD Todo.lnk"] || f.files["/desk/ADHD Todo.lnk"] {
		t.Errorf("shortcuts: %v", f.files)
	}
	if f.reg["DisplayName"] != "Withdo" {
		t.Errorf("DisplayName = %q", f.reg["DisplayName"])
	}
	if want := `C:\Users\u\AppData\Local\Programs\ADHD Todo\Withdo.exe`; f.reg["DisplayIcon"] != want {
		t.Errorf("DisplayIcon = %q, want %q", f.reg["DisplayIcon"], want)
	}

	// The second launch has nothing left to do.
	before := len(f.renames)
	fixWindowsInstall(f.install(legacyExe))
	if len(f.renames) != before {
		t.Errorf("renamed again: %v", f.renames)
	}
}

func TestFixWindowsInstallLeavesOtherInstallsAlone(t *testing.T) {
	f := legacyWindows()
	fixWindowsInstall(f.install("/Programs/Withdo/Withdo.exe"))
	if len(f.renames) != 0 || f.reg["DisplayName"] != "ADHD Todo" {
		t.Errorf("touched a fresh install: %v %v", f.renames, f.reg)
	}
}

func TestFixWindowsInstallKeepsShortcutsThatExist(t *testing.T) {
	f := legacyWindows()
	f.files["/menu/Withdo.lnk"] = true
	fixWindowsInstall(f.install(legacyExe))
	if !f.files["/menu/ADHD Todo.lnk"] || !f.files["/desk/Withdo.lnk"] {
		t.Errorf("shortcuts: %v", f.files)
	}
	if len(f.renames) != 1 {
		t.Errorf("renames: %v", f.renames)
	}
}

func TestFixWindowsInstallSurvivesAMissingEntryAndAFailedWrite(t *testing.T) {
	f := legacyWindows()
	f.reg = map[string]string{}
	f.files = map[string]bool{}
	fixWindowsInstall(f.install(legacyExe)) // no entry, no shortcuts: nothing to do

	f = legacyWindows()
	f.failSet = true
	fixWindowsInstall(f.install(legacyExe))
	if f.reg["DisplayName"] != "ADHD Todo" || !f.files["/menu/Withdo.lnk"] {
		t.Errorf("a failed registry write must not stop the rest: %v %v", f.reg, f.files)
	}

	f = legacyWindows()
	f.reg["DisplayName"] = "Something else"
	fixWindowsInstall(f.install(legacyExe))
	if f.reg["DisplayName"] != "Something else" {
		t.Errorf("DisplayName = %q", f.reg["DisplayName"])
	}
}

func TestWithNewExe(t *testing.T) {
	cases := []struct{ in, want string }{
		{`C:\P\ADHD Todo\ADHD Todo.exe`, `C:\P\ADHD Todo\Withdo.exe`},
		{`"C:\P\ADHD Todo\ADHD Todo.exe"`, `C:\P\ADHD Todo\Withdo.exe`},
		{"/p/ADHD Todo.exe", "/p/Withdo.exe"},
		{"ADHD Todo.exe", "Withdo.exe"},
		{`C:\P\Withdo\Withdo.exe`, ""},
		{"", ""},
	}
	for _, c := range cases {
		got, ok := withNewExe(c.in)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("withNewExe(%q) = %q, %v", c.in, got, ok)
		}
	}
}
