package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/notify"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/ghpoll"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/egoist/mygo/ui"
)

// The poller finds a pull request merged in the shared database; the app tells it as an event once,
// with the PR number, the task and its project.
func TestAMergedPullRequestReachesTheAppAsAnEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "my.db")
	t.Setenv("ADHD_TODO_DB", path)
	url := "https://github.com/acme/rocket/pull/7"
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	core := domain.New(db, domain.Options{})
	p, err := core.UpsertProject(context.Background(), domain.UpsertProjectInput{RepoSlug: "acme/rocket"})
	if err != nil {
		t.Fatal(err)
	}
	made, err := core.CreateTask(context.Background(),
		domain.CreateTaskInput{Title: "Ship it", Source: domain.SourceAgent, Status: domain.StatusReview, PRURL: &url, ProjectID: &p.ID})
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer func(old ghpoll.Runner) { ghRunner = old }(ghRunner)
	ghRunner = &stubGH{reply: func(args []string) (ghpoll.Output, error) {
		if args[0] == "auth" {
			return ghpoll.Output{Stdout: []byte("github.com\n  ✓ Logged in to github.com account QingYunA (keyring)\n")}, nil
		}
		return ghpoll.Output{Stdout: []byte(mergedPR)}, nil
	}}

	be, err := openBackend(false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = be.close() }()
	got := make(chan data.Event, 4)
	be.store.SubscribeEvents(func(e data.Event) { got <- e })
	be.start(nil)

	select {
	case e := <-got:
		want := data.Event{Kind: data.EventMerged, TaskID: int(made.ID), Title: "Ship it", ProjectName: "rocket", PRNumber: 7}
		if e != want {
			t.Errorf("event = %+v, want %+v", e, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no event for the merged pull request")
	}
	select {
	case e := <-got:
		t.Errorf("told again: %+v", e)
	case <-time.After(300 * time.Millisecond):
	}
}

// A click on the summary opens Mine; one on a task's notification opens the task.
func TestClickingTheSummaryNotificationOpensMine(t *testing.T) {
	be, err := openBackend(true, false)
	if err != nil {
		t.Fatal(err)
	}
	app := views.NewApp(be.store, nil, views.Config{})
	m := views.NewMain(app)
	s := &shell{app: app, main: m}
	tt := ui.NewTester(m.View, 1100, 720)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	tt.Frame()
	if err := tt.Click("Agents"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.ShowingMine() {
		t.Fatal("the test did not leave Mine")
	}
	s.openFromNotification(notify.SummaryID)
	tt.Frame()
	if !m.ShowingMine() {
		t.Error("the summary did not open Mine")
	}
	s.openFromNotification("not-a-notification")
}
