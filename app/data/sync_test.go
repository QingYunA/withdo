package data

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// appSyncServer answers enough of the sync protocol for the app's runner to start.
type appSyncServer struct {
	*httptest.Server
	pulls    atomic.Int64
	started  chan struct{}
	finished chan struct{}
}

func newAppSyncServer(t *testing.T) *appSyncServer {
	t.Helper()
	s := &appSyncServer{started: make(chan struct{}, 8), finished: make(chan struct{}, 8)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/pull", func(w http.ResponseWriter, _ *http.Request) {
		s.pulls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"cookie":0,"last_mutation_id":0,"changes":[],"more":false}`)
	})
	mux.HandleFunc("POST /v1/push", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"last_mutation_id":0}`)
	})
	mux.HandleFunc("POST /v1/ids", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"start":1000,"end":2000}`)
	})
	mux.HandleFunc("GET /v1/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flush", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()
		select {
		case s.started <- struct{}{}:
		default:
		}
		<-r.Context().Done()
		select {
		case s.finished <- struct{}{}:
		default:
		}
	})
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func (s *appSyncServer) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-s.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the app never opened the sync event stream")
	}
}

func (s *appSyncServer) waitFinished(t *testing.T) {
	t.Helper()
	select {
	case <-s.finished:
	case <-time.After(10 * time.Second):
		t.Fatal("the app did not stop the sync runner")
	}
}

// linkedAppDB makes an app database and, when asked, marks it linked and writes sync.json.
func linkedAppDB(t *testing.T, server *appSyncServer, linked, configured bool) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if linked {
		for key, value := range map[string]string{
			"device_id": "app-device", "workspace_id": "7", "enabled": "1", "cookie": "0",
		} {
			if _, err := db.ExecContext(ctx,
				"INSERT INTO sync_state (key, value) VALUES (?, ?)", key, value); err != nil {
				_ = db.Close()
				t.Fatalf("prepare %s: %v", key, err)
			}
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if configured {
		if err := syncclient.SaveConfig(path, syncclient.Config{Server: server.URL, Token: "token"}); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// TestS3Core3AppStartsAndStopsRunner is acceptance criterion 3: data.Real starts the runner after
// the watcher only when sync.json exists and the database is linked, and Close stops it.
func TestS3Core3AppStartsAndStopsRunner(t *testing.T) {
	ctx := context.Background()
	t.Run("starts with sync.json and a linked database", func(t *testing.T) {
		server := newAppSyncServer(t)
		path := linkedAppDB(t, server, true, true)
		real, err := OpenReal(ctx, path, RealOptions{DataFolder: filepath.Dir(path), Interval: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		real.Start()
		if real.syncRunner() == nil {
			_ = real.Close()
			t.Fatal("the app did not start a sync runner")
		}
		server.waitStarted(t)
		if err := real.Close(); err != nil {
			t.Fatal(err)
		}
		server.waitFinished(t)
		if real.syncRunner() != nil {
			t.Fatal("Close left the runner running")
		}
	})

	t.Run("no sync.json means no runner", func(t *testing.T) {
		server := newAppSyncServer(t)
		path := linkedAppDB(t, server, true, false)
		real, err := OpenReal(ctx, path, RealOptions{DataFolder: filepath.Dir(path), Interval: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		real.Start()
		defer func() { _ = real.Close() }()
		if real.syncRunner() != nil {
			t.Fatal("the app started a runner without sync.json")
		}
	})

	t.Run("sync.json without a linked database means no runner", func(t *testing.T) {
		server := newAppSyncServer(t)
		path := linkedAppDB(t, server, false, true)
		real, err := OpenReal(ctx, path, RealOptions{DataFolder: filepath.Dir(path), Interval: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		real.Start()
		defer func() { _ = real.Close() }()
		if real.syncRunner() != nil {
			t.Fatal("the app started a runner on an unlinked database")
		}
	})
}
