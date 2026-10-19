//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkBundledCoreLinksOnlyFromInsideAnAppBundle(t *testing.T) {
	root := t.TempDir()
	dev := filepath.Join(root, "build", "app")
	bundled := filepath.Join(root, "ADHD Todo.app", "Contents", "MacOS", "ADHD Todo")
	for _, exe := range []string{dev, bundled} {
		if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
			t.Fatal(err)
		}
		script := "#!/bin/sh\necho adhd-core 0.5.0\n"
		if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "bin", "adhd-core")

	linkBundledCore(dev, link)
	if _, err := os.Lstat(link); err == nil {
		t.Fatal("a run outside an app bundle linked the agents to its executable")
	}
	linkBundledCore(bundled, link)
	if got, _ := os.Readlink(link); got != bundled {
		t.Errorf("link points at %q", got)
	}
}
