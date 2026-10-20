package core

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// demoNow is the fixed moment the demo data and the page tests are built around.
var demoNow = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	// data.parseTime reads the database's UTC timestamps into time.Local; UTC makes the golden
	// files the same on every machine.
	time.Local = time.UTC
	os.Exit(m.Run())
}

// demoDatabase seeds app/data's demo data on a fresh phone folder and returns that folder. The
// demo store keeps its database path to itself, so the test gives it a private TMPDIR, finds the
// one folder it made there and copies the SQLite file out before closing it.
func demoDatabase(t *testing.T, now time.Time) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	demo, err := data.OpenDemo(func() time.Time { return now }, true)
	if err != nil {
		t.Fatalf("open the demo store: %v", err)
	}
	defer func() {
		if err := demo.Close(); err != nil {
			t.Errorf("close the demo store: %v", err)
		}
	}()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("read the demo folder: %v", err)
	}
	var source string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "adhd-todo-demo-") {
			if source != "" {
				t.Fatalf("more than one demo folder appeared: %s and %s", source, entry.Name())
			}
			source = filepath.Join(tmp, entry.Name())
		}
	}
	if source == "" {
		t.Fatal("the demo store did not make a temp folder")
	}
	dest := t.TempDir()
	copySQLite(t, filepath.Join(source, "demo.db"), filepath.Join(dest, "adhd-todo.db"))
	return dest
}

// copySQLite copies a database with VACUUM INTO, so a WAL with uncheckpointed rows comes along.
func copySQLite(t *testing.T, source, dest string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+source+"?mode=ro")
	if err != nil {
		t.Fatalf("open the demo database: %v", err)
	}
	defer func() { _ = db.Close() }()
	quoted := "'" + strings.ReplaceAll(filepath.ToSlash(dest), "'", "''") + "'"
	if _, err := db.Exec("VACUUM INTO " + quoted); err != nil {
		t.Fatalf("copy the demo database: %v", err)
	}
}

// checkGolden compares got, indented, with testdata/name. UPDATE_GOLDEN=1 writes the file.
func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(got), "", "  "); err != nil {
		t.Fatalf("indent %s: %v", name, err)
	}
	pretty.WriteByte('\n')
	full := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, pretty.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read %s: %v (run UPDATE_GOLDEN=1 go test ./mobile/core to write it)", full, err)
	}
	if string(want) != pretty.String() {
		t.Errorf("%s does not match the golden file; run UPDATE_GOLDEN=1 go test ./mobile/core and review the diff\n%s", full, pretty.String())
	}
}
