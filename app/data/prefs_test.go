package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilePrefsRoundTripsAndToleratesAMissingFile(t *testing.T) {
	store := FilePrefs{Path: filepath.Join(t.TempDir(), "sub", "prefs.json")}
	if got, err := store.LoadPrefs(); err != nil || got != (Prefs{}) {
		t.Fatalf("a missing file: %v, %v", got, err)
	}
	if err := store.SavePrefs(Prefs{OpenWith: "terminal"}); err != nil {
		t.Fatal(err)
	}
	if got, err := store.LoadPrefs(); err != nil || got.OpenWith != "terminal" {
		t.Errorf("round trip: %v, %v", got, err)
	}
}

func TestFilePrefsReportsACorruptFileAndKeepsGoing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := (FilePrefs{Path: path}).LoadPrefs(); err == nil || got != (Prefs{}) {
		t.Errorf("corrupt file: %v, %v: want the defaults and an error", got, err)
	}
}

func TestMemoryPrefsKeepsWhatWasSaved(t *testing.T) {
	var m MemoryPrefs
	_ = m.SavePrefs(Prefs{OpenWith: "finder"})
	if got, _ := m.LoadPrefs(); got.OpenWith != "finder" {
		t.Errorf("stored %v", got)
	}
}

func TestWorkDirIsTheNewestReportedDirectoryThenTheProjectFolder(t *testing.T) {
	newestFirst := []Report{{Summary: "no cwd"}, {Cwd: "/new"}, {Cwd: "/old"}}
	if got := WorkDir(newestFirst, "/repo"); got != "/new" {
		t.Errorf("WorkDir = %q, want the newest report that names one", got)
	}
	if got := WorkDir([]Report{{Summary: "x"}}, "/repo"); got != "/repo" {
		t.Errorf("WorkDir = %q, want the project folder", got)
	}
	if got := WorkDir(nil, ""); got != "" {
		t.Errorf("WorkDir = %q, want none", got)
	}
}
