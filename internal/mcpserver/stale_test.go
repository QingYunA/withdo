package mcpserver_test

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/mcpserver"
	"github.com/QingYunA/adhd-todo/internal/paths"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

func TestToolCallsAreRefusedOnceTheBinaryIsReplaced(t *testing.T) {
	testutil.IsolateGit(t)
	db, dbPath := testutil.OpenDB(t)
	exe := filepath.Join(t.TempDir(), "adhd-core")
	if err := os.WriteFile(exe, []byte("old build"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := mcpserver.New(domain.New(db, domain.Options{}), mcpserver.Options{
		Executable: exe, ImagesDir: paths.ImagesDir(filepath.Dir(dbPath)),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	serverT, clientT := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-agent", Version: "1.0.0"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close(); _ = serverSession.Close() })
	r := &rig{t: t, session: session}

	if res := r.call("list_categories", nil); res.IsError {
		t.Fatalf("a fresh binary was refused: %s", res.Text)
	}

	time.Sleep(2100 * time.Millisecond) // past the check interval
	if err := os.WriteFile(exe, []byte("new build, longer"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := r.call("list_categories", nil)
	if !res.IsError || !strings.Contains(res.Text, "adhd-core was updated") {
		t.Fatalf("a replaced binary was not refused: %+v", res)
	}
}
