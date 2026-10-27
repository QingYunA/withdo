package main

import (
	"path/filepath"
	"testing"
)

func TestTargetIsTheAppNextToTheLauncher(t *testing.T) {
	self := filepath.Join("Programs", "ADHD Todo", "ADHD Todo.exe")
	want := filepath.Join("Programs", "ADHD Todo", "Withdo.exe")
	if got := target(self); got != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}
