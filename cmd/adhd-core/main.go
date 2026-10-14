// Command adhd-core is adhd-todo's command line: an MCP server for coding
// agents (mcp) and a plain-text report of what they are doing (overview).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], defaultDeps())
	stop()
	os.Exit(code)
}
