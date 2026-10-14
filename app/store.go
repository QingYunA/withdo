package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/paths"
)

// backend is the store the app runs on, with what only the real or the fake one needs.
type backend struct {
	store      data.Store
	dataFolder string
	// start runs once the subscriptions are in place: it starts the watcher, or the fake's timer.
	start func(stop <-chan struct{})
	// close releases the store; nil for the fake.
	close func() error
}

// openBackend returns the fake (with seed data, or empty) when asked for, otherwise the shared
// database at the path from internal/paths, which honours ADHD_TODO_DB.
func openBackend(fake, empty bool) (*backend, error) {
	if fake || empty {
		return openFake(empty), nil
	}
	path, err := paths.DefaultDBPath()
	if err != nil {
		return nil, err
	}
	return openReal(path)
}

func openReal(path string) (*backend, error) {
	r, err := data.OpenReal(context.Background(), path, data.RealOptions{})
	if err != nil {
		return nil, err
	}
	return &backend{
		store:      r,
		dataFolder: filepath.Dir(path),
		start:      func(<-chan struct{}) { r.Start() },
		close:      r.Close,
	}, nil
}

func openFake(empty bool) *backend {
	var f *data.Fake
	if empty {
		f = data.NewEmptyFake(time.Now)
	} else {
		f = data.NewFake(time.Now)
	}
	return &backend{
		store:      f,
		dataFolder: "(in-memory demo data)",
		start:      func(stop <-chan struct{}) { go tickFake(f, stop) },
	}
}

// tickFake asks the fake to send due reminders; the real store does this from its watcher.
func tickFake(f *data.Fake, stop <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			f.CheckReminders()
		case <-stop:
			return
		}
	}
}
