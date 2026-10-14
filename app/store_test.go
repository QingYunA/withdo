package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

func TestOpenBackendUsesTheEnvironmentDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ADHD_TODO_DB", filepath.Join(dir, "x", "my.db"))
	be, err := openBackend(false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = be.close() }()
	if be.dataFolder != filepath.Join(dir, "x") {
		t.Errorf("dataFolder = %q", be.dataFolder)
	}
	if _, ok := be.store.(*data.Real); !ok {
		t.Fatalf("store is %T, want *data.Real", be.store)
	}
	got := make(chan struct{}, 1)
	be.store.Subscribe(func() { got <- struct{}{} })
	be.start(nil)
	time.Sleep(100 * time.Millisecond) // let the watcher take its baseline
	if _, err := be.store.CreateTask(data.CreateTaskInput{Title: "hello"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(3 * time.Second):
		t.Error("no change event from the watcher")
	}
}

func TestOpenBackendFakes(t *testing.T) {
	for _, tc := range []struct {
		fake, empty bool
		wantTasks   bool
	}{{true, false, true}, {false, true, false}} {
		be, err := openBackend(tc.fake, tc.empty)
		if err != nil {
			t.Fatal(err)
		}
		ts, _ := be.store.ListTasks(data.Filter{})
		if (len(ts) > 0) != tc.wantTasks {
			t.Errorf("fake=%v empty=%v: %d tasks", tc.fake, tc.empty, len(ts))
		}
		stop := make(chan struct{})
		be.start(stop)
		close(stop)
	}
}

func TestOpenBackendFailsOnABadPath(t *testing.T) {
	t.Setenv("ADHD_TODO_DB", "/dev/null/nope/x.db")
	if _, err := openBackend(false, false); err == nil {
		t.Error("expected an error")
	}
}
