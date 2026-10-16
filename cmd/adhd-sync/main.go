// adhd-sync serves the standalone single-owner mobile sync database.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QingYunA/adhd-todo/internal/mobilesync"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "import-desktop" {
		return runImportDesktop(os.Args[2:])
	}
	listen := flag.String("listen", "127.0.0.1:8787", "HTTP listen address; use TLS termination for remote access")
	path := flag.String("db", "adhd-sync.db", "dedicated mobile sync SQLite file (never the desktop database)")
	token := flag.String("token", os.Getenv("ADHD_SYNC_TOKEN"), "single-owner bearer secret; prefer ADHD_SYNC_TOKEN environment variable")
	resetEpoch := flag.Bool("reset-epoch", false, "force device resnapshots after restoring a database backup")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := mobilesync.Open(ctx, *path)
	if err != nil {
		return fmt.Errorf("open sync database: %w", err)
	}
	defer func() { _ = db.Close() }()
	handler, err := mobilesync.NewHandler(db, *token)
	if err != nil {
		return err
	}
	if *resetEpoch {
		if err := db.ResetEpoch(ctx); err != nil {
			return fmt.Errorf("reset sync epoch: %w", err)
		}
	}
	server := &http.Server{
		Addr: *listen, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Printf("mobile sync listening on %s", *listen)
	select {
	case err := <-done:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}

func runImportDesktop(args []string) error {
	flags := flag.NewFlagSet("adhd-sync import-desktop", flag.ContinueOnError)
	source := flags.String("source", "", "checkpointed desktop SQLite snapshot; read-only, never migrated")
	sourceID := flags.String("source-id", "", "stable identity for this desktop database; reuse for every import")
	serverURL := flags.String("url", "", "sync service base URL")
	token := flags.String("token", os.Getenv("ADHD_SYNC_TOKEN"), "single-owner bearer secret; prefer ADHD_SYNC_TOKEN environment variable")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	summary, err := mobilesync.ImportDesktop(ctx, *source, *sourceID, *serverURL, *token)
	if encodeErr := json.NewEncoder(os.Stdout).Encode(summary); encodeErr != nil {
		return encodeErr
	}
	if err != nil {
		return err
	}
	if summary.Conflicts != 0 || summary.Rejected != 0 {
		return fmt.Errorf("import requires reconciliation: %d conflicts, %d rejected", summary.Conflicts, summary.Rejected)
	}
	return nil
}
