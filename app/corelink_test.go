//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkBundledCoreLinksOnlyFromInsideAnInstalledAppBundle(t *testing.T) {
	root := t.TempDir()
	dev := filepath.Join(root, "build", "app")
	bundled := filepath.Join(root, "ADHD Todo.app", "Contents", "MacOS", "ADHD Todo")
	for _, exe := range []string{dev, bundled} {
		if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
			t.Fatal(err)
		}
		script := "#!/bin/sh\necho withdo 0.5.0\n"
		if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name, exe string
		linked    bool
	}{
		{"a run outside an app bundle", dev, false},
		{"a disk image", "/Volumes/ADHD Todo/ADHD Todo.app/Contents/MacOS/ADHD Todo", false},
		{"a translocated app", "/private/var/folders/xx/T/AppTranslocation/0A1B2C3D-0000-0000-0000-000000000000/d/ADHD Todo.app/Contents/MacOS/ADHD Todo", false},
		{"an installed app", bundled, true},
	}
	for _, c := range cases {
		link := filepath.Join(t.TempDir(), "bin", "withdo")
		linkBundledCore(c.exe, link)
		got, err := os.Readlink(link)
		switch {
		case c.linked && got != c.exe:
			t.Errorf("%s: link points at %q", c.name, got)
		case !c.linked && err == nil:
			t.Errorf("%s: linked the agents to %q", c.name, got)
		}
		if !c.linked {
			if _, err := os.Lstat(link); err == nil {
				t.Errorf("%s: %s exists", c.name, link)
			}
		}
	}
}

func TestLinkOwnExecutableKeepsTheLegacyNameLinkedToo(t *testing.T) {
	root := t.TempDir()
	bundled := filepath.Join(root, "Withdo.app", "Contents", "MacOS", "Withdo")
	if err := os.MkdirAll(filepath.Dir(bundled), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundled, []byte("#!/bin/sh\necho withdo 0.5.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "bin", "withdo")
	linkBundledCores(bundled, link)
	for _, name := range []string{"withdo", legacyCoreName} {
		if got, err := os.Readlink(filepath.Join(filepath.Dir(link), name)); err != nil || got != bundled {
			t.Errorf("%s points at %q (%v), want %q", name, got, err, bundled)
		}
	}
}
