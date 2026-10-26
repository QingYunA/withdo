package mcpserver

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// An MCP server lives as long as the agent session, often days, while the app that ships
// adhd-core is upgraded under it. The old process then keeps writing to the shared database with
// its old rules (once: ids from SQLite instead of the sync id blocks, which collided with rows of
// the new build). refuseWhenStale makes such a process refuse every tool call instead.

const (
	staleMessage = "adhd-core was updated; this MCP server is an old copy and no longer writes to the database. " +
		"Restart the agent session (or reconnect the adhd-todo MCP server) to load the new one."
	// staleCheckEvery bounds how often a busy agent makes us stat the binary.
	staleCheckEvery = 2 * time.Second
)

// binaryWatch remembers what the executable looked like at startup.
type binaryWatch struct {
	path    string
	size    int64
	modTime time.Time

	mu      sync.Mutex
	checked time.Time
	stale   bool
}

// watchBinary records the identity of the running executable, or returns nil when it cannot be
// found (then nothing is ever treated as stale). An empty path means os.Executable().
func watchBinary(path string) *binaryWatch {
	if path == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil
		}
		path = exe
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return &binaryWatch{path: path, size: info.Size(), modTime: info.ModTime(), checked: time.Now()}
}

// isStale reports whether the file at path is gone or no longer the one that was started. Once
// stale it stays stale; the result is cached for staleCheckEvery.
func (w *binaryWatch) isStale() bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stale || time.Since(w.checked) < staleCheckEvery {
		return w.stale
	}
	w.checked = time.Now()
	info, err := os.Stat(w.path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		w.stale = true
	case err == nil:
		w.stale = info.Size() != w.size || !info.ModTime().Equal(w.modTime)
	}
	return w.stale
}

// refuseWhenStale is receiving middleware: a tools/call to a replaced binary answers with an
// error and never reaches a handler. Other methods (initialize, tools/list) still work, so the
// agent can show the error instead of a dead connection.
func refuseWhenStale(w *binaryWatch) func(mcp.MethodHandler) mcp.MethodHandler {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "tools/call" && w.isStale() {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: staleMessage}}, IsError: true}, nil
			}
			return next(ctx, method, req)
		}
	}
}
