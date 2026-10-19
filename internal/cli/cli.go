// Package cli is adhd-core, the command line: an MCP server for coding agents (mcp) and a plain-text
// report of what they are doing (overview). The standalone cmd/adhd-core and the app, which is
// adhd-core too when run under that name or with one of its commands, both call Main.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/mcpserver"
	"github.com/QingYunA/adhd-todo/internal/paths"
	"github.com/QingYunA/adhd-todo/internal/store"
)

const usage = `Usage: adhd-core <command>

Commands:
  mcp        Run the MCP server on stdio
  overview   Print active tasks per project, with stale flags
  guide      Print a setup guide to paste to any coding agent, so it adds this MCP server itself
  version    Print the version (also --version)
`

// Exit codes.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// commands are the first arguments adhd-core answers to; the app uses them to tell a command line from a launch.
var commands = []string{"mcp", "overview", "guide", "version", "--version", "help", "--help", "-h"}

// IsCommand reports whether arg is one of adhd-core's own commands or flags.
func IsCommand(arg string) bool { return slices.Contains(commands, arg) }

// Main runs adhd-core on the process's environment until it ends or is interrupted, and returns
// the exit code.
func Main(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, args, defaultDeps())
}

// deps is everything run needs from the outside world, so tests can fake it.
type deps struct {
	getenv func(string) string
	home   string
	stdout io.Writer
	stderr io.Writer
	// serve runs the MCP server until its client goes away.
	serve func(context.Context, *mcp.Server) error
	// location is the zone the overview prints times in; the machine's when nil.
	location *time.Location
}

// defaultDeps wires run to the real process environment.
func defaultDeps() deps {
	home, _ := os.UserHomeDir() // ADHD_TODO_DB alone is enough without a home directory
	return deps{getenv: os.Getenv, home: home, stdout: os.Stdout, stderr: os.Stderr, serve: mcpserver.Run}
}

// run executes one command and returns the process exit code. stdout carries
// only command output (and, for mcp, the protocol); diagnostics go to stderr.
func run(ctx context.Context, args []string, d deps) int {
	if len(args) == 0 {
		_, _ = io.WriteString(d.stderr, usage)
		return exitUsage
	}
	if wantsHelp(args) {
		_, _ = io.WriteString(d.stdout, usage)
		return exitOK
	}
	switch args[0] {
	case "version", "--version":
		_, _ = fmt.Fprintf(d.stdout, "adhd-core %s\n", mcpserver.Version)
		return exitOK
	case "guide":
		_, _ = io.WriteString(d.stdout, agentsetup.New(d.home).Guide())
		return exitOK
	case "mcp", "overview":
		if err := execute(ctx, args[0], d); err != nil {
			_, _ = fmt.Fprintf(d.stderr, "adhd-core: %v\n", err)
			return exitError
		}
		return exitOK
	default:
		_, _ = fmt.Fprintf(d.stderr, "Unknown command: %s\n%s", args[0], usage)
		return exitUsage
	}
}

// wantsHelp reports whether args ask for the usage: help, --help or -h on their own, or --help or
// -h after a command. There is one usage for all commands, so that is what each of them prints.
func wantsHelp(args []string) bool {
	switch args[0] {
	case "help", "--help", "-h":
		return true
	case "mcp", "overview", "guide":
		return len(args) > 1 && (args[1] == "--help" || args[1] == "-h")
	}
	return false
}

func execute(ctx context.Context, command string, d deps) error {
	db, err := store.Open(ctx, paths.DBPath(d.getenv, runtime.GOOS, d.home))
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	core := domain.New(db, domain.Options{})
	if command == "mcp" {
		_, _ = io.WriteString(d.stderr, "[adhd-todo] MCP server ready on stdio\n")
		return d.serve(ctx, mcpserver.New(core, mcpserver.Options{}))
	}
	overview, err := core.GetOverview(ctx, domain.OverviewOptions{})
	if err != nil {
		return err
	}
	loc := d.location
	if loc == nil {
		loc = time.Local
	}
	_, err = fmt.Fprintln(d.stdout, domain.FormatOverviewIn(overview, loc))
	return err
}
