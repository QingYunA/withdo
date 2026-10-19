//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkBundledCoreLinksOnlyFromInsideAnAppBundle(t *testing.T) {
	root := t.TempDir()
	dev := filepath.Join(root, "resources")
	app := filepath.Join(root, "ADHD Todo.app", "Contents", "Resources")
	for _, dir := range []string{dev, app} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		script := "#!/bin/sh\necho adhd-core 0.5.0\n"
		if err := os.WriteFile(filepath.Join(dir, "adhd-core"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "bin", "adhd-core")

	linkBundledCore(dev, link)
	if _, err := os.Lstat(link); err == nil {
		t.Fatal("a run outside an app bundle linked the agents to its resources")
	}
	linkBundledCore(app, link)
	if got, _ := os.Readlink(link); got != filepath.Join(app, "adhd-core") {
		t.Errorf("link points at %q", got)
	}
}
