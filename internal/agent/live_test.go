package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/paths"
	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/QingYunA/adhd-todo/internal/testutil"
	"github.com/cloudwego/eino/components/model"
)

// TestLiveModel talks to a real model. It is skipped unless ADHD_TODO_LIVE_MODEL=1.
//
//	ADHD_TODO_LIVE_MODEL=1 go test ./internal/agent -run TestLiveModel -v
//
// The provider is the active one of the app's models.json, with the key in ADHD_TODO_LIVE_KEY (the
// app's Keychain item is never read from tests). To use a preset without the file, set
// ADHD_TODO_LIVE_PRESET (for example "magpie"); ADHD_TODO_LIVE_MODEL_NAME overrides the model. Tasks go to a throwaway
// database, never the real one.
func TestLiveModel(t *testing.T) {
	if os.Getenv("ADHD_TODO_LIVE_MODEL") != "1" {
		t.Skip("set ADHD_TODO_LIVE_MODEL=1 to run against a real model")
	}
	testutil.IsolateGit(t)
	db, _ := testutil.OpenDB(t)
	core := domain.New(db, domain.Options{})
	ag := agent.New(agent.Config{Core: core, Model: liveModel(t), Now: func() time.Time { return time.Now().In(agent.LocalZone()) }})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	res, err := ag.Ask(ctx, "明天九点提醒我开会")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	t.Logf("reply: %q receipts: %+v question: %+v", res.Reply, res.Receipts, res.Question)
	if len(res.Receipts) != 1 || res.Receipts[0].Title == "" {
		t.Fatalf("want one receipt, got %+v", res.Receipts)
	}
	task, _ := core.GetTask(ctx, res.Receipts[0].TaskID)
	if task == nil || task.RemindAt == nil {
		t.Fatalf("no reminder stored: %+v", task)
	}
	remind, _ := time.Parse(time.RFC3339Nano, *task.RemindAt)
	local := remind.In(agent.LocalZone())
	t.Logf("stored %s, local %s", *task.RemindAt, local.Format(time.RFC3339))
	if local.Hour() != 9 || local.Minute() != 0 || !local.After(time.Now()) {
		t.Errorf("reminder at %s, want tomorrow 09:00 local", local)
	}
	if err := ag.Undo(ctx, res.Receipts[0].ID); err != nil {
		t.Errorf("Undo: %v", err)
	}
}

func liveModel(t *testing.T) agent.ModelFunc {
	if id := os.Getenv("ADHD_TODO_LIVE_PRESET"); id != "" {
		r, ok := providers.Config{}.With(id, providers.Entry{Model: os.Getenv("ADHD_TODO_LIVE_MODEL_NAME")}).Resolve(id)
		if !ok {
			t.Fatalf("unknown preset %q", id)
		}
		return func(ctx context.Context) (model.ToolCallingChatModel, error) {
			return providers.NewModel(ctx, r, os.Getenv("ADHD_TODO_LIVE_KEY"), nil)
		}
	}
	db, err := paths.DefaultDBPath()
	if err != nil {
		t.Fatal(err)
	}
	store := providers.NewStore(filepath.Join(filepath.Dir(db), providers.ConfigFile))
	keys := providers.NewMemoryKeyring()
	if key := os.Getenv("ADHD_TODO_LIVE_KEY"); key != "" {
		cfg, _ := store.Load()
		_ = keys.Set(cfg.Active, key)
	}
	mgr, err := providers.NewManager(store, keys, nil)
	if err != nil || !mgr.Configured() {
		t.Skipf("no model configured in the app (%v)", err)
	}
	return mgr.Model
}
