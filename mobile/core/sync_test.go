package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

const syncTestToken = "e30.eyJleHAiOjQxMDI0NDQ4MDB9.sig" // a JWT-shaped token that expires in 2100

var syncSchemaCounter atomic.Uint64

// syncHarness is the S2a server on a fresh Postgres schema.
type syncHarness struct {
	server *syncserver.Server
	http   *httptest.Server
}

func newSyncHarness(t *testing.T) *syncHarness {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test against Postgres; skipped with -short")
	}
	databaseURL := os.Getenv("ADHD_SYNC_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ADHD_SYNC_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("mobile_%d_%d", os.Getpid(), syncSchemaCounter.Add(1))
	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Postgres: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create schema: %v", err)
	}
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse the database URL: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	server, err := syncserver.Open(ctx, u.String(), syncserver.StaticTokens{syncTestToken: "mobile-core-subject"})
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		_ = admin.Close(ctx)
		t.Fatalf("open the sync server: %v", err)
	}
	h := &syncHarness{server: server, http: httptest.NewServer(server.Handler())}
	t.Cleanup(func() {
		h.http.Close()
		h.server.Close()
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		_ = admin.Close(context.Background())
	})
	return h
}

// TestSyncWithADesktop is acceptance 4: a desktop domain.Core and a phone core linked to the same
// workspace with a static token. A task made on the phone reaches the desktop, and a reply made on
// the phone reaches the desktop's activity.
func TestSyncWithADesktop(t *testing.T) {
	h := newSyncHarness(t)
	ctx := context.Background()
	fixed := func() time.Time { return demoNow }

	desktopDB, _ := testutil.OpenDB(t)
	desktop := domain.New(desktopDB, domain.Options{Now: fixed})
	desktopClient, err := syncclient.New(desktop, syncclient.Options{BaseURL: h.http.URL, Token: syncTestToken})
	if err != nil {
		t.Fatalf("desktop sync client: %v", err)
	}
	if err := desktopClient.Link(ctx, "desktop"); err != nil {
		t.Fatalf("link the desktop: %v", err)
	}

	phoneDir := t.TempDir()
	phone, err := open(phoneDir, nil, fixed)
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	defer func() { _ = phone.Close() }()
	if err := syncclient.SaveConfigAt(filepath.Join(phoneDir, "sync.json"),
		syncclient.Config{Server: h.http.URL, Token: syncTestToken}); err != nil {
		t.Fatalf("write sync.json: %v", err)
	}
	phoneClient, err := syncclient.New(phone.dom, syncclient.Options{BaseURL: h.http.URL, Token: syncTestToken})
	if err != nil {
		t.Fatalf("phone sync client: %v", err)
	}
	if err := phoneClient.Link(ctx, "phone"); err != nil {
		t.Fatalf("link the phone: %v", err)
	}

	raw, err := phone.CreateTask(`{"title":"Made on the phone"}`)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	var made data.Task
	if err := json.Unmarshal([]byte(raw), &made); err != nil {
		t.Fatalf("decode the created task: %v", err)
	}
	if err := phone.SyncNow(); err != nil {
		t.Fatalf("phone SyncNow: %v", err)
	}
	if err := desktopClient.Sync(ctx); err != nil {
		t.Fatalf("desktop sync: %v", err)
	}
	onDesktop, err := desktop.GetTask(ctx, int64(made.ID))
	if err != nil || onDesktop == nil {
		t.Fatalf("the phone's task did not reach the desktop: %v", err)
	}
	if onDesktop.Title != "Made on the phone" {
		t.Errorf("desktop task = %+v", onDesktop)
	}

	if err := phone.Reply(int64(made.ID), "A reply from the phone"); err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if err := phone.SyncNow(); err != nil {
		t.Fatalf("phone SyncNow after the reply: %v", err)
	}
	if err := desktopClient.Sync(ctx); err != nil {
		t.Fatalf("desktop sync after the reply: %v", err)
	}
	reports, err := desktop.ListReports(ctx, int64(made.ID))
	if err != nil {
		t.Fatalf("desktop reports: %v", err)
	}
	found := false
	for _, report := range reports {
		if report.Agent == domain.ReplyAgent && report.Summary == "A reply from the phone" {
			found = true
		}
	}
	if !found {
		t.Errorf("the reply did not reach the desktop's activity: %+v", reports)
	}

	// The status JSON stays the shape Kotlin decodes.
	var status struct {
		State   string `json:"state"`
		Pending int64  `json:"pending"`
	}
	if err := json.Unmarshal([]byte(phone.SyncStatus()), &status); err != nil {
		t.Errorf("decode SyncStatus: %v", err)
	}
	if status.State == "" {
		t.Errorf("SyncStatus has no state: %s", phone.SyncStatus())
	}

	// The Runner reports its status to the listener, and StopSync waits for it.
	statuses := make(chan string, 16)
	phone.SetListener(&statusListener{statuses: statuses})
	if err := phone.StartSync(); err != nil {
		t.Fatalf("StartSync: %v", err)
	}
	select {
	case raw := <-statuses:
		if !json.Valid([]byte(raw)) {
			t.Errorf("OnStatus got invalid JSON: %s", raw)
		}
	case <-time.After(5 * time.Second):
		t.Error("OnStatus was not called after StartSync")
	}
	if err := phone.StopSync(); err != nil {
		t.Errorf("StopSync: %v", err)
	}
}

// statusListener records the Runner's OnStatus JSON.
type statusListener struct{ statuses chan string }

func (l *statusListener) OnChange() {}

func (l *statusListener) OnStatus(status string) {
	select {
	case l.statuses <- status:
	default:
	}
}
