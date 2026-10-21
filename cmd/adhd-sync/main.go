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
	"strings"
	"syscall"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
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
	issuer := fs.String("oidc-issuer", "", "OIDC issuer URL whose access tokens are accepted")
	audience := fs.String("oidc-audience", "", "audience the OIDC access tokens must carry")
	clientID := fs.String("oidc-client-id", "", "OIDC client id GET /v1/config tells clients to sign in with")
	scopes := fs.String("oidc-scopes", "", "extra OIDC scopes for GET /v1/config, separated by spaces")
	loginPATFile := fs.String("zitadel-login-pat-file", "", "file holding the Zitadel login machine user's PAT, enabling POST /v1/sign-in")
	zitadelInternalURL := fs.String("zitadel-internal-url", "", "base URL for adhd-sync's Zitadel requests, such as http://proxy")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *databaseURL == "" {
		return errors.New("-database-url is required")
	}
	if (*issuer == "") != (*audience == "") {
		return errors.New("-oidc-issuer and -oidc-audience must be given together")
	}
	var loginPAT string
	if *loginPATFile != "" {
		if *issuer == "" || *clientID == "" {
			return errors.New("-zitadel-login-pat-file requires -oidc-issuer and -oidc-client-id")
		}
		raw, err := os.ReadFile(*loginPATFile)
		if err != nil {
			return fmt.Errorf("read -zitadel-login-pat-file: %w", err)
		}
		loginPAT = strings.TrimSpace(string(raw))
		if loginPAT == "" {
			return errors.New("-zitadel-login-pat-file is empty")
		}
	}
	if *tokensPath == "" && *issuer == "" {
		return errors.New("one of -tokens or -oidc-issuer is required")
	}
	var auth syncserver.Authenticator
	if *tokensPath != "" {
		tokens, err := syncserver.LoadTokens(*tokensPath)
		if err != nil {
			return err
		}
		auth = tokens
	}
	if *issuer != "" {
		oidcAuth, err := syncserver.NewOIDCAuthenticator(ctx, *issuer, *audience)
		if err != nil {
			return err
		}
		if auth == nil {
			auth = oidcAuth
		} else {
			// A static token is tried first; the OIDC verifier handles everything else.
			auth = syncserver.Chain{auth, oidcAuth}
		}
	}
	server, err := syncserver.Open(ctx, *databaseURL, auth)
	if err != nil {
		return err
	}
	defer server.Close()
	server.SetPublicConfig(syncserver.PublicConfig{
		Issuer: *issuer, ClientID: *clientID, Scopes: strings.Fields(*scopes),
	})
	if loginPAT != "" {
		if err := server.EnablePasswordSignIn(syncserver.PasswordSignInOptions{
			Issuer: *issuer, ClientID: *clientID, PAT: loginPAT,
			InternalURL: *zitadelInternalURL, Scopes: strings.Fields(*scopes),
		}); err != nil {
			return err
		}
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *addr, err)
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
