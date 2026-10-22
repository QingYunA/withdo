// Command adhd-sync is adhd-todo's self-hosted sync server: an HTTP API over Postgres for the
// desktop and Android clients (M11).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

// run parses the flags, opens the database and serves until ctx is cancelled.
func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("adhd-sync", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "address to listen on")
	databaseURL := fs.String("database-url", "", "Postgres connection URL")
	tokensPath := fs.String("tokens", "", "JSON file mapping bearer tokens to user subjects (development)")
	pbDir := fs.String("pb-dir", "", "PocketBase data directory; enables PocketBase accounts (sign-in by email and password)")
	if len(args) > 0 {
		if handled, err := runAccountCommand(args[0], args[1:], os.Stdin, os.Stdout); handled {
			return err
		}
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *databaseURL == "" {
		return errors.New("-database-url is required")
	}
	if *tokensPath == "" && *pbDir == "" {
		return errors.New("one of -tokens or -pb-dir is required")
	}
	// PocketBase is tried first, then the development tokens.
	var chain syncserver.Chain
	var pb core.App
	publicAuth := ""
	if *pbDir != "" {
		app, err := pbauth.Open(*pbDir)
		if err != nil {
			return err
		}
		defer func() { _ = pbauth.Close(app) }()
		pb, publicAuth = app, "pocketbase"
		chain = append(chain, pbauth.NewAuthenticator(app))
	}
	if *tokensPath != "" {
		tokens, err := syncserver.LoadTokens(*tokensPath)
		if err != nil {
			return err
		}
		chain = append(chain, tokens)
	}
	var auth syncserver.Authenticator = chain
	if len(chain) == 1 {
		auth = chain[0]
	}
	server, err := syncserver.Open(ctx, *databaseURL, auth)
	if err != nil {
		return err
	}
	defer server.Close()
	server.SetPublicConfig(syncserver.PublicConfig{Auth: publicAuth})

	// Voice input: the recognizer's key comes from the environment, never from a flag or a file in
	// the repository. Without it POST /v1/transcribe answers 503 voice_disabled.
	voiceLimit := 0
	if raw := os.Getenv("VOICE_DAILY_LIMIT"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return errors.New("VOICE_DAILY_LIMIT must be a positive number")
		}
		voiceLimit = n
	}
	server.EnableVoice(syncserver.VoiceOptions{
		APIKey: os.Getenv("MIMO_API_KEY"), BaseURL: os.Getenv("MIMO_BASE_URL"), DailyLimit: voiceLimit,
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *addr, err)
	}
	if pb != nil {
		log.Printf("adhd-sync listening on %s (PocketBase accounts in %s)", ln.Addr(), *pbDir)
		return pbauth.Serve(ctx, pb, ln, server.Handler())
	}
	httpServer := &http.Server{
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(ln) }()
	log.Printf("adhd-sync listening on %s", ln.Addr())

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
