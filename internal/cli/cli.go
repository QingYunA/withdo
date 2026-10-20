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
	"path/filepath"
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
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

const usage = `Usage: adhd-core <command>

Commands:
  mcp        Run the MCP server on stdio
  overview   Print active tasks per project, with stale flags
  sync       Link this device to a sync server, show its status, or sync once
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
var commands = []string{"mcp", "overview", "sync", "guide", "version", "--version", "help", "--help", "-h"}

// IsCommand reports whether arg is one of adhd-core's own commands or flags.
func IsCommand(arg string) bool { return slices.Contains(commands, arg) }

// Options inject what only the desktop app can provide: the Keychain-backed sync secrets. The
// standalone adhd-core passes nothing.
type Options struct {
	Secrets syncclient.Secrets
}

// Main runs adhd-core on the process's environment until it ends or is interrupted, and returns
// the exit code.
func Main(args []string, options ...Options) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	d := defaultDeps()
	if len(options) > 0 {
		if options[0].Secrets != nil {
			d.secrets = options[0].Secrets
		}
	}
	return run(ctx, args, d)
}

// deps is everything run needs from the outside world, so tests can fake it.
type deps struct {
	getenv func(string) string
	home   string
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
	// serve runs the MCP server until its client goes away.
	serve func(context.Context, *mcp.Server) error
	// location is the zone the overview prints times in; the machine's when nil.
	location *time.Location
	// secrets is the keychain OIDC sync tokens live in; nil when this binary has no keychain.
	secrets syncclient.Secrets
}

// defaultDeps wires run to the real process environment.
func defaultDeps() deps {
	home, _ := os.UserHomeDir() // ADHD_TODO_DB alone is enough without a home directory
	return deps{
		getenv: os.Getenv, home: home, stdout: os.Stdout, stderr: os.Stderr,
		stdin: os.Stdin, serve: mcpserver.Run,
	}
}

// run executes one command and returns the process exit code. stdout carries
// only command output (and, for mcp, the protocol); diagnostics go to stderr.
func run(ctx context.Context, args []string, d deps) int {
	if len(args) == 0 {
		_, _ = io.WriteString(d.stderr, usage)
		return exitUsage
	}
	if wantsHelp(args) {
		if args[0] == "sync" {
			_, _ = io.WriteString(d.stdout, syncUsage)
		} else {
			_, _ = io.WriteString(d.stdout, usage)
		}
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
	case "sync":
		if err := runSync(ctx, args[1:], d); err != nil {
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
// -h after a command. Each command prints the usage, except sync, which has its own.
func wantsHelp(args []string) bool {
	switch args[0] {
	case "help", "--help", "-h":
		return true
	case "mcp", "overview", "sync", "guide":
		return len(args) > 1 && (args[1] == "--help" || args[1] == "-h")
	}
	return false
}

func execute(ctx context.Context, command string, d deps) error {
	dbPath := paths.DBPath(d.getenv, runtime.GOOS, d.home)
	db, err := store.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	core := domain.New(db, domain.Options{})
	if command == "mcp" {
		_, _ = io.WriteString(d.stderr, "[adhd-todo] MCP server ready on stdio\n")
		return d.serve(ctx, mcpserver.New(core, mcpserver.Options{ImagesDir: paths.ImagesDir(filepath.Dir(dbPath))}))
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
