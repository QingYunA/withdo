package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// cliSyncServer is a small sync server for the CLI: it registers one device, accepts pushes and
// leases id blocks. It does not store rows, which is enough for link, status and now.
type cliSyncServer struct {
	*httptest.Server
	seeding bool

	mu        sync.Mutex
	next      int64
	pushes    int
	mutations int
}

func newCLISyncServer(t *testing.T, seeding bool) *cliSyncServer {
	t.Helper()
	s := &cliSyncServer{seeding: seeding, next: 1000}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(syncproto.CreateDeviceResponse{
			DeviceID: "cli-device", WorkspaceID: 7, Empty: !s.seeding, Seeding: s.seeding,
		})
	})
	mux.HandleFunc("POST /v1/devices/seeded", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"seeded":true}`))
	})
	mux.HandleFunc("POST /v1/push", func(w http.ResponseWriter, r *http.Request) {
		var req syncproto.PushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.pushes++
		s.mutations += len(req.Mutations)
		last := int64(0)
		for _, mutation := range req.Mutations {
			if mutation.ID > last {
				last = mutation.ID
			}
		}
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(syncproto.PushResponse{LastMutationID: last})
	})
	mux.HandleFunc("POST /v1/ids", func(w http.ResponseWriter, r *http.Request) {
		var req syncproto.IDsRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		count := int64(req.Count)
		if count <= 0 {
			count = 1000
		}
		s.mu.Lock()
		start := s.next
		s.next += count
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(syncproto.IDsResponse{Start: start, End: start + count})
	})
	mux.HandleFunc("GET /v1/pull", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cookie":0,"last_mutation_id":0,"changes":[],"more":false}`))
	})
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func (s *cliSyncServer) pushed() (pushes, mutations int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pushes, s.mutations
}

func writeTokenFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("  cli-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runCLI(t *testing.T, h *harness, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	h.out.Reset()
	h.err.Reset()
	code = run(context.Background(), args, h.d)
	return code, h.out.String(), h.err.String()
}

func oneLineError(t *testing.T, stderr string) {
	t.Helper()
	if got := strings.Count(strings.TrimSpace(stderr), "\n"); got != 0 {
		t.Fatalf("stderr is not one line: %q", stderr)
	}
	if !strings.HasPrefix(stderr, "withdo: ") {
		t.Fatalf("stderr = %q, want the withdo prefix", stderr)
	}
}

// TestS3Core4CLI is acceptance criterion 4: link joins an empty database and seeds one with data,
// status and now print their fields, a missing token file is one line, and ErrLocalData names the
// two flags that choose what happens when both sides have data.
func TestS3Core4CLI(t *testing.T) {
	t.Run("link joins an empty database and status and now print their fields", func(t *testing.T) {
		server := newCLISyncServer(t, true)
		path := filepath.Join(t.TempDir(), "empty.db")
		h := newHarness(t, path)
		token := writeTokenFile(t)

		code, out, errOut := runCLI(t, h, "sync", "link", "--server", server.URL, "--token-file", token, "--name", "Mac")
		if code != 0 {
			t.Fatalf("link exit %d: %s", code, errOut)
		}
		for _, want := range []string{"linked device cli-device", "joined the account"} {
			if !strings.Contains(out, want) {
				t.Errorf("link output lacks %q:\n%s", want, out)
			}
		}
		configPath := syncclient.ConfigPath(path)
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("sync.json: %v", err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
			t.Errorf("sync.json mode = %o, want 600", info.Mode().Perm())
		}

		code, out, errOut = runCLI(t, h, "sync", "status")
		if code != 0 {
			t.Fatalf("status exit %d: %s", code, errOut)
		}
		for _, want := range []string{
			"device: cli-device", "workspace: 7", "cookie: 0", "pending: 0",
			"id blocks left:", "projects 1000", "tasks 1000", "last error: none",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("status output lacks %q:\n%s", want, out)
			}
		}

		code, out, errOut = runCLI(t, h, "sync", "now")
		if code != 0 {
			t.Fatalf("now exit %d: %s", code, errOut)
		}
		if out != "pushed: 0\npulled: 0\n" {
			t.Errorf("now output = %q", out)
		}

		db, err := store.Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		task, err := domain.New(db, domain.Options{}).CreateTask(context.Background(),
			domain.CreateTaskInput{Title: "local"})
		_ = db.Close()
		if err != nil {
			t.Fatal(err)
		}
		if task.ID == 0 {
			t.Fatal("the linked database assigned no id from its block")
		}
		code, out, errOut = runCLI(t, h, "sync", "now")
		if code != 0 {
			t.Fatalf("now after a write exit %d: %s", code, errOut)
		}
		if !strings.Contains(out, "pushed: 1") {
			t.Errorf("now after a write = %q, want one push", out)
		}
	})

	t.Run("link seeds a database with data", func(t *testing.T) {
		server := newCLISyncServer(t, true)
		path := filepath.Join(t.TempDir(), "data.db")
		db, err := store.Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := domain.New(db, domain.Options{}).CreateTask(context.Background(),
			domain.CreateTaskInput{Title: "seed me"}); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		h := newHarness(t, path)

		code, out, errOut := runCLI(t, h, "sync", "link", "--server", server.URL, "--token-file", writeTokenFile(t), "--name", "Mac")
		if code != 0 {
			t.Fatalf("link exit %d: %s", code, errOut)
		}
		if !strings.Contains(out, "uploaded this device's data (") {
			t.Errorf("link output = %q, want the uploaded rows line", out)
		}
		if pushes, mutations := server.pushed(); pushes == 0 || mutations == 0 {
			t.Errorf("server got %d pushes and %d mutations, want the snapshot", pushes, mutations)
		}
	})

	t.Run("local data and account data explain Settings", func(t *testing.T) {
		server := newCLISyncServer(t, false)
		path := filepath.Join(t.TempDir(), "both.db")
		db, err := store.Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := domain.New(db, domain.Options{}).CreateTask(context.Background(),
			domain.CreateTaskInput{Title: "already here"}); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		h := newHarness(t, path)

		code, _, errOut := runCLI(t, h, "sync", "link", "--server", server.URL, "--token-file", writeTokenFile(t), "--name", "Mac")
		if code == 0 {
			t.Fatal("linking two databases with data should fail")
		}
		if !strings.Contains(errOut, "--merge") || !strings.Contains(errOut, "--replace") {
			t.Errorf("stderr = %q, want both ways on named", errOut)
		}
		oneLineError(t, errOut)
		if _, err := os.Stat(syncclient.ConfigPath(path)); err == nil {
			t.Error("a failed link wrote sync.json")
		}
	})

	t.Run("a missing token file is one line", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.db")
		h := newHarness(t, path)
		code, _, errOut := runCLI(t, h, "sync", "link", "--server", "http://127.0.0.1:1",
			"--token-file", filepath.Join(t.TempDir(), "absent"), "--name", "Mac")
		if code == 0 {
			t.Fatal("link without a token file should fail")
		}
		if !strings.Contains(errOut, "token file") {
			t.Errorf("stderr = %q, want the token file error", errOut)
		}
		oneLineError(t, errOut)
		if _, err := os.Stat(syncclient.ConfigPath(path)); err == nil {
			t.Error("a link without a token file wrote sync.json")
		}
	})
}
