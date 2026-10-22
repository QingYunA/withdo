package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestS2a9CommandStartsAndHealthz is acceptance criterion 9: adhd-sync starts with -addr,
// -database-url and -tokens, and GET /healthz answers without a bearer token.
func TestS2a9CommandStartsAndHealthz(t *testing.T) {
	databaseURL := os.Getenv("ADHD_SYNC_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ADHD_SYNC_TEST_DATABASE_URL is not set")
	}
	t.Parallel()

	ctx := context.Background()
	schema := fmt.Sprintf("s2a_cmd_%d", time.Now().UnixNano())
	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Postgres: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(),
			"DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		_ = admin.Close(context.Background())
	})

	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	tokensPath := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(tokensPath, []byte(`{"test-token":"test-subject"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(runCtx, []string{
			"-addr", addr,
			"-database-url", u.String(),
			"-tokens", tokensPath,
			"-oidc-client-id", "adhd-todo",
			"-oidc-scopes", "openid offline_access urn:example:audience",
		})
	}()

	deadline := time.Now().Add(10 * time.Second)
	var body []byte
	for {
		resp, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			body, err = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				t.Fatalf("read /healthz: %v", err)
			}
			if resp.StatusCode != http.StatusOK || string(body) != "ok" {
				t.Fatalf("GET /healthz = %d %q, want 200 ok", resp.StatusCode, body)
			}
			resp, err = http.Get("http://" + addr + "/v1/config")
			if err != nil {
				t.Fatalf("GET /v1/config: %v", err)
			}
			body, err = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				t.Fatalf("read /v1/config: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET /v1/config = %d %q", resp.StatusCode, body)
			}
			var cfg struct {
				ClientID string   `json:"client_id"`
				Scopes   []string `json:"scopes"`
			}
			if err := json.Unmarshal(body, &cfg); err != nil {
				t.Fatalf("decode /v1/config: %v", err)
			}
			if cfg.ClientID != "adhd-todo" || len(cfg.Scopes) != 3 || cfg.Scopes[2] != "urn:example:audience" {
				t.Fatalf("GET /v1/config = %s", body)
			}
			break
		}
		select {
		case err := <-errCh:
			t.Fatalf("adhd-sync exited before /healthz: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET /healthz did not become ready: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("adhd-sync exited with %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("adhd-sync did not shut down")
	}
}

// TestS4aCommandOIDCFlags checks the flag combinations: a static token and OIDC are each enough,
// both are allowed, the OIDC pair must be given together, and the login PAT needs an issuer and a
// client id.
func TestS4aCommandOIDCFlags(t *testing.T) {
	ctx := context.Background()
	tokensPath := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(tokensPath, []byte(`{"test-token":"test-subject"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyPAT := filepath.Join(t.TempDir(), "empty.pat")
	if err := os.WriteFile(emptyPAT, []byte(" \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	missingPAT := filepath.Join(t.TempDir(), "missing.pat")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no authenticator", []string{"-database-url", "postgres://ignored"}, "one of -tokens"},
		{"issuer without audience", []string{"-database-url", "postgres://ignored", "-oidc-issuer", "https://issuer.example"}, "must be given together"},
		{"audience without issuer", []string{"-database-url", "postgres://ignored", "-oidc-audience", "client"}, "must be given together"},
		{"login PAT without OIDC", []string{"-database-url", "postgres://ignored", "-tokens", tokensPath,
			"-zitadel-login-pat-file", missingPAT}, "requires -oidc-issuer and -oidc-client-id"},
		{"login PAT without client id", []string{"-database-url", "postgres://ignored",
			"-oidc-issuer", "https://issuer.example", "-oidc-audience", "client",
			"-zitadel-login-pat-file", missingPAT}, "requires -oidc-issuer and -oidc-client-id"},
		{"empty login PAT", []string{"-database-url", "postgres://ignored",
			"-oidc-issuer", "https://issuer.example", "-oidc-audience", "client", "-oidc-client-id", "adhd-todo",
			"-zitadel-login-pat-file", emptyPAT}, "is empty"},
		{"missing login PAT", []string{"-database-url", "postgres://ignored",
			"-oidc-issuer", "https://issuer.example", "-oidc-audience", "client", "-oidc-client-id", "adhd-todo",
			"-zitadel-login-pat-file", missingPAT}, "read -zitadel-login-pat-file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(ctx, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("run error = %v, want %q", err, tt.want)
			}
		})
	}
}
