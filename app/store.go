package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/ghpoll"
	"github.com/QingYunA/adhd-todo/internal/paths"
	"github.com/QingYunA/adhd-todo/internal/providers"
)

// seenFile is the record of the handed-over rows the user has seen, in the data folder.
const seenFile = "seen.json"

// ghRunner runs the gh CLI for the pull request poller. Tests replace it: they never reach GitHub.
var ghRunner ghpoll.Runner = &ghpoll.ExecRunner{}

// backend is the store the app runs on, with what only the real or the fake one needs.
type backend struct {
	store      data.Store
	dataFolder string
	// start runs once the subscriptions are in place: it starts the watcher, or the fake's timer.
	start func(stop <-chan struct{})
	// close releases the store; nil for the fake.
	close func() error
	// assistant is the built-in agent behind the ask field and models its Settings tab. The demo
	// has no assistant, so its ask field adds Inbox tasks.
	assistant data.Assistant
	models    data.ModelSettings
	// github is the pull request poller behind Settings > GitHub.
	github data.GitHub
	// seen keeps which handed-over rows of Mine the user has seen, between runs.
	seen data.SeenStore
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
	dir := filepath.Dir(path)
	mgr, err := providers.NewManager(providers.NewStore(filepath.Join(dir, providers.ConfigFile)), newSecretKeyring(), nil)
	if err != nil {
		log.Printf("model settings: %v", err) // the app still runs, with no model connected
	}
	loc := agent.LocalZone()
	ag := agent.New(agent.Config{
		Core: r.Core(), Model: mgr.Model, Now: func() time.Time { return time.Now().In(loc) },
	})
	poller := ghpoll.New(ghpoll.Config{Runner: ghRunner, Store: r.Core()})
	return &backend{
		store:      r,
		dataFolder: dir,
		start:      func(<-chan struct{}) { r.Start(); poller.Start() },
		close:      func() error { poller.Stop(); return r.Close() },
		assistant:  &data.AgentAssistant{Agent: ag, Provider: mgr},
		models:     &data.ProviderSettings{Provider: mgr},
		github:     data.PollerGitHub{Poller: poller},
		seen:       data.FileSeen{Path: filepath.Join(dir, seenFile)},
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
		models:     demoModels(),
		seen:       &data.MemorySeen{},
		github:     data.NewFakeGitHub(data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: time.Now()}),
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

// demoModels is Settings > Models for the demo: provider file in a temp folder, keys in memory.
func demoModels() data.ModelSettings {
	dir, err := os.MkdirTemp("", "adhd-todo-demo-")
	if err != nil {
		return nil
	}
	mgr, _ := providers.NewManager(providers.NewStore(filepath.Join(dir, providers.ConfigFile)), providers.NewMemoryKeyring(), nil)
	return &data.FakeModels{ProviderSettings: data.ProviderSettings{Provider: mgr}}
}
