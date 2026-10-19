package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/ghpoll"
	"github.com/QingYunA/adhd-todo/internal/paths"
	"github.com/QingYunA/adhd-todo/internal/providers"
)

// seenFile is the record of the handed-over rows the user has seen, in the data folder.
const seenFile = "seen.json"

// dismissedFile is the record of the handed-over rows the user put aside with Not now, in the data folder.
const dismissedFile = "dismissed.json"

// prefsFile is the app's own choices from Settings, in the data folder.
const prefsFile = "prefs.json"

// recentFile is what the assistant wrote about the Recent changes page, in the data folder.
const recentFile = "recent-changes.json"

// memoryFile is the assistant's notes about the user, a Markdown file the user can edit, in the data folder.
const memoryFile = "assistant-memory.md"

// ghRunner runs the gh CLI for the pull request poller. Tests replace it: they never reach GitHub.
var ghRunner ghpoll.Runner = &ghpoll.ExecRunner{}

// backend is the store the app runs on, with what only the real or the demo one needs.
type backend struct {
	store      data.Store
	dataFolder string
	// memoryFile is the assistant's memory file; empty when it has none (the demo).
	memoryFile string
	// start runs once the subscriptions are in place: it starts the watcher.
	start func(stop <-chan struct{})
	// close releases the store.
	close func() error
	// assistant is the built-in agent behind the ask field and models its Settings tab. The demo
	// has no assistant, so its ask field adds Inbox tasks.
	assistant data.Assistant
	models    data.ModelSettings
	// github is the pull request poller behind Settings > GitHub.
	github data.GitHub
	// seen keeps which handed-over rows of Mine the user has seen, between runs.
	seen data.SeenStore
	// dismissed keeps which handed-over rows of Mine the user put aside, between runs.
	dismissed data.SeenStore
	// prefs keeps the choices made in Settings that the app keeps itself, between runs.
	prefs data.PrefsStore
	// recent reads the repositories for the Recent changes page, and recentCache keeps what the
	// assistant wrote about them.
	recent      data.Recent
	recentCache *data.RecentCache
}

// openBackend returns the demo store (with seed data, or empty) when asked for, otherwise the shared
// database at the path from internal/paths, which honours ADHD_TODO_DB.
func openBackend(fake, empty bool) (*backend, error) {
	if fake || empty {
		return openDemo(empty)
	}
	path, err := paths.DefaultDBPath()
	if err != nil {
		return nil, err
	}
	return openReal(path)
}

func openReal(path string) (*backend, error) {
	dir := filepath.Dir(path)
	r, err := data.OpenReal(context.Background(), path, data.RealOptions{DataFolder: dir})
	if err != nil {
		return nil, err
	}
	mgr, err := providers.NewManager(providers.NewStore(filepath.Join(dir, providers.ConfigFile)), newSecretKeyring(), nil)
	if err != nil {
		log.Printf("model settings: %v", err) // the app still runs, with no model connected
	}
	loc := agent.LocalZone()
	ag := agent.New(agent.Config{
		Core: r.Core(), Model: mgr.Model, Memory: agent.NewMemory(filepath.Join(dir, memoryFile)), Now: func() time.Time { return time.Now().In(loc) },
	})
	poller := ghpoll.New(ghpoll.Config{Runner: ghRunner, Store: r.Core(), Housekeeping: closeFinishedSubtasks(r.Core())})
	return &backend{
		store:       r,
		dataFolder:  dir,
		memoryFile:  filepath.Join(dir, memoryFile),
		start:       func(<-chan struct{}) { r.Start(); r.CollectSoon(); poller.Start() }, // CollectSoon: pictures another process let go
		close:       func() error { poller.Stop(); return r.Close() },
		assistant:   &data.AgentAssistant{Agent: ag, Provider: mgr, Images: r},
		models:      &data.ProviderSettings{Provider: mgr},
		github:      data.PollerGitHub{Poller: poller},
		seen:        data.FileSeen{Path: filepath.Join(dir, seenFile)},
		dismissed:   data.FileSeen{Path: filepath.Join(dir, dismissedFile)},
		prefs:       data.FilePrefs{Path: filepath.Join(dir, prefsFile)},
		recent:      data.GHRecent{Runner: ghRunner},
		recentCache: data.NewRecentCache(filepath.Join(dir, recentFile)),
	}, nil
}

// closeFinishedSubtasks is the poller's housekeeping: at launch, every few minutes and when the window
// comes back, subtasks whose branch was merged or deleted become done. The store's watcher shows it.
func closeFinishedSubtasks(core *domain.Core) func(context.Context) {
	return func(ctx context.Context) {
		if n, err := core.CloseFinishedSubtasks(ctx); err != nil {
			log.Printf("close finished subtasks: %v", err)
		} else if n > 0 {
			log.Printf("closed %d subtasks whose branch was merged or deleted", n)
		}
	}
}

// openDemo opens the demo store: a Real on a temporary SQLite file, seeded unless empty, which
// close removes.
func openDemo(empty bool) (*backend, error) {
	r, err := data.OpenDemo(time.Now, !empty)
	if err != nil {
		return nil, err
	}
	return &backend{
		store:       r,
		dataFolder:  "(temporary demo data)",
		start:       func(<-chan struct{}) { r.Start() },
		close:       r.Close,
		models:      demoModels(),
		seen:        &data.MemorySeen{},
		dismissed:   &data.MemorySeen{},
		prefs:       &data.MemoryPrefs{},
		github:      data.NewFakeGitHub(data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: time.Now()}),
		recent:      data.DemoRecent(time.Now()),
		recentCache: data.NewRecentCache(""),
	}, nil
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
