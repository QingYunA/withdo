//go:build !windows

package corelink

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// script writes an executable shell script that runs body, and returns its path.
func script(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// versionScript is a fake adhd-core that prints version for --version.
func versionScript(t *testing.T, path, version string) string {
	return script(t, path, `[ "$1" = --version ] && echo "adhd-core `+version+`" || exit 2`)
}

// bundle is the adhd-core of an app bundle under root, at version 0.5.0.
func bundle(t *testing.T, root string) string {
	return versionScript(t, filepath.Join(root, "ADHD Todo.app", "Contents", "Resources", "adhd-core"), "0.5.0")
}

func target(t *testing.T, link string) string {
	t.Helper()
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("%s is not a symlink: %v", link, err)
	}
	return got
}

func TestMissingLinkIsCreatedWithItsFolder(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	link := filepath.Join(root, "home", ".local", "bin", "adhd-core")
	r, err := Ensure(link, b)
	if err != nil || r.Action != Linked {
		t.Fatalf("Ensure = %+v, %v", r, err)
	}
	if got := target(t, link); got != b {
		t.Errorf("points at %s, want %s", got, b)
	}
}

func TestLinkIntoTheSameBundleIsLeftAlone(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	link := filepath.Join(root, "adhd-core")
	if err := os.Symlink(b, link); err != nil {
		t.Fatal(err)
	}
	if r, err := Ensure(link, b); err != nil || r.Action != Unchanged {
		t.Fatalf("Ensure = %+v, %v", r, err)
	}
}

func TestLinkIntoAnotherAppBundleIsRepointed(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	link := filepath.Join(root, "adhd-core")
	// The old app is gone, so the link is broken: it still names an app bundle.
	if err := os.Symlink("/Volumes/Old/ADHD Todo.app/Contents/Resources/adhd-core", link); err != nil {
		t.Fatal(err)
	}
	if r, err := Ensure(link, b); err != nil || r.Action != Linked {
		t.Fatalf("Ensure = %+v, %v", r, err)
	}
	if got := target(t, link); got != b {
		t.Errorf("points at %s, want %s", got, b)
	}
}

func TestLinkToSomewhereElseIsKept(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	mine := versionScript(t, filepath.Join(root, "src", "adhd-core"), "0.1.0")
	link := filepath.Join(root, "adhd-core")
	if err := os.Symlink(mine, link); err != nil {
		t.Fatal(err)
	}
	if r, err := Ensure(link, b); err != nil || r.Action != Kept {
		t.Fatalf("Ensure = %+v, %v", r, err)
	}
	if got := target(t, link); got != mine {
		t.Errorf("points at %s, want %s", got, mine)
	}
}

func TestRegularFileOlderThanTheBundledOneIsReplaced(t *testing.T) {
	for name, body := range map[string]string{
		"older":             `echo "adhd-core 0.4.0"`,
		"older patch":       `echo "adhd-core 0.4.9"`,
		"no --version flag": `echo "Unknown command: --version"; exit 2`,
		"not a program":     `exit 1`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			b := bundle(t, root)
			link := script(t, filepath.Join(root, "adhd-core"), body)
			if r, err := Ensure(link, b); err != nil || r.Action != Linked {
				t.Fatalf("Ensure = %+v, %v", r, err)
			}
			if got := target(t, link); got != b {
				t.Errorf("points at %s, want %s", got, b)
			}
			// The file it replaced is kept next to it, in case it was someone's own build.
			if kept, err := os.ReadFile(link + ".old"); err != nil || string(kept) != "#!/bin/sh\n"+body+"\n" {
				t.Errorf("the replaced file is not kept as %s.old: %q, %v", link, kept, err)
			}
		})
	}
}

func TestRegularFileAsNewAsTheBundledOneIsKept(t *testing.T) {
	for _, v := range []string{"0.5.0", "0.5.1", "0.10.0", "1.0.0"} {
		t.Run(v, func(t *testing.T) {
			root := t.TempDir()
			b := bundle(t, root)
			link := versionScript(t, filepath.Join(root, "adhd-core"), v)
			if r, err := Ensure(link, b); err != nil || r.Action != Kept {
				t.Fatalf("Ensure = %+v, %v", r, err)
			}
			if _, err := os.Readlink(link); err == nil {
				t.Error("a developer's own build was replaced by a link")
			}
		})
	}
}

func TestDirectoryIsNeverReplaced(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	link := filepath.Join(root, "adhd-core")
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	if r, err := Ensure(link, b); err != nil || r.Action != Kept {
		t.Fatalf("Ensure = %+v, %v", r, err)
	}
	if fi, err := os.Lstat(link); err != nil || !fi.IsDir() {
		t.Errorf("the directory is gone: %v", err)
	}
}

func TestMissingBundledBinaryChangesNothing(t *testing.T) {
	root := t.TempDir()
	link := versionScript(t, filepath.Join(root, "adhd-core"), "0.4.0")
	if _, err := Ensure(link, filepath.Join(root, "nope", "adhd-core")); err == nil {
		t.Fatal("no error for a missing bundled binary")
	}
	if _, err := os.Readlink(link); err == nil {
		t.Error("the file was replaced")
	}
}

func TestLinkBelowAFileIsAnError(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	file := filepath.Join(root, "not-a-folder")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(filepath.Join(file, "adhd-core"), b); err == nil {
		t.Fatal("no error for a link below a regular file")
	}
}

func TestUnreadableLinkIsAnError(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	// Readlink fails for anything that is not a symlink; Ensure only gets here after Lstat
	// saw one, so call fromSymlink on a regular file directly.
	if _, err := fromSymlink(script(t, filepath.Join(root, "adhd-core"), "exit 1"), b); err == nil {
		t.Fatal("no error for a link that cannot be read")
	}
}

func TestOldFileThatCannotBeKeptIsNotReplaced(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	link := versionScript(t, filepath.Join(root, "adhd-core"), "0.4.0")
	old := link + ".old"
	if err := os.Mkdir(old, 0o755); err != nil {
		t.Fatal(err)
	}
	// A non-empty .old cannot be removed, so the old build cannot be kept aside.
	if err := os.WriteFile(filepath.Join(old, "mine"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(link, b); err == nil {
		t.Fatal("no error when the old file cannot be kept")
	}
	if _, err := os.Readlink(link); err == nil {
		t.Error("the file was replaced without being kept")
	}
}

func TestLinkWhoseTempPathIsTakenIsAnError(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	link := filepath.Join(root, "home", ".local", "bin", "adhd-core")
	// A stale .new directory is in the way, so the new link cannot be made.
	tmp := link + ".new"
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "stale"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(link, b); err == nil {
		t.Fatal("no error when the temp path is taken")
	}
	if _, err := os.Lstat(link); err == nil {
		t.Error("the link exists after a failed replace")
	}
}

// TestReplaceFailures pins the two ways replace can fail that Ensure cannot reach: a
// folder it cannot create, and a directory it cannot rename over.
func TestReplaceFailures(t *testing.T) {
	t.Run("folder is a file", func(t *testing.T) {
		root := t.TempDir()
		b := bundle(t, root)
		file := filepath.Join(root, "not-a-folder")
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := replace(filepath.Join(file, "adhd-core"), b, "test"); err == nil {
			t.Fatal("no error for a link below a regular file")
		}
	})
	t.Run("link is a non-empty directory", func(t *testing.T) {
		root := t.TempDir()
		b := bundle(t, root)
		link := filepath.Join(root, "adhd-core")
		if err := os.Mkdir(link, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(link, "mine"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := replace(link, b, "test"); err == nil {
			t.Fatal("no error when a non-empty directory is in the way")
		}
		if _, err := os.Lstat(link + ".new"); err == nil {
			t.Error("the .new link was left behind by a failed replace")
		}
		if fi, err := os.Lstat(link); err != nil || !fi.IsDir() {
			t.Errorf("the directory is gone: %v", err)
		}
	})
}

func TestFilePrintingNoVersionIsReplaced(t *testing.T) {
	root := t.TempDir()
	b := bundle(t, root)
	link := script(t, filepath.Join(root, "adhd-core"), "exit 0")
	r, err := Ensure(link, b)
	if err != nil || r.Action != Linked {
		t.Fatalf("Ensure = %+v, %v", r, err)
	}
	if got := target(t, link); got != b {
		t.Errorf("points at %s, want %s", got, b)
	}
	if !strings.Contains(r.Reason, "unknown") {
		t.Errorf("reason = %q, want an unknown version", r.Reason)
	}
	if kept, err := os.ReadFile(link + ".old"); err != nil || string(kept) != "#!/bin/sh\nexit 0\n" {
		t.Errorf("the replaced file is not kept as %s.old: %q, %v", link, kept, err)
	}
}

func TestVersionOrderWithUnparseableVersions(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		less bool
	}{
		{"dev", "0.5.0", true}, {"0.5.0", "dev", false}, {"dev", "nightly", false},
		{"", "0.5.0", true},
	} {
		if got := older(tt.a, tt.b); got != tt.less {
			t.Errorf("older(%q, %q) = %v", tt.a, tt.b, got)
		}
	}
}

func TestVersionOrder(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		less bool
	}{
		{"0.4.0", "0.5.0", true}, {"0.9.0", "0.10.0", true}, {"0.5.0", "0.5.0", false},
		{"1.0.0", "0.5.0", false}, {"0.5", "0.5.1", true},
	} {
		if got := older(tt.a, tt.b); got != tt.less {
			t.Errorf("older(%s, %s) = %v", tt.a, tt.b, got)
		}
	}
}
