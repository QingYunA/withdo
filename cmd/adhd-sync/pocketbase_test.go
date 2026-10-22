package main

import (
	"bytes"
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

// schemaURL returns a database URL on a fresh Postgres schema, dropped at the end of the test.
func schemaURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("ADHD_SYNC_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ADHD_SYNC_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("m13_cmd_%d", time.Now().UnixNano())
	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Postgres: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		_ = admin.Close(context.Background())
	})
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

func call(t *testing.T, method, rawURL, token string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

// TestPocketBaseAccountsEndToEnd imports a user, serves with -pb-dir, signs in through PocketBase,
// syncs with the token, and moves an old subject's workspace to the new identity.
func TestPocketBaseAccountsEndToEnd(t *testing.T) {
	databaseURL := schemaURL(t)
	pbDir := filepath.Join(t.TempDir(), "pb_data")
	tokensPath := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(tokensPath, []byte(`{"old-token":"zitadel-user-1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	handled, err := runAccountCommand("pb-import-user",
		[]string{"-pb-dir", pbDir, "-email", "ada@example.com", "-name", "Ada"},
		strings.NewReader("a long enough password\n"), &out)
	if !handled || err != nil {
		t.Fatalf("pb-import-user: handled=%v err=%v", handled, err)
	}
	subject := strings.TrimSpace(out.String())
	if !strings.HasPrefix(subject, "pb:") {
		t.Fatalf("pb-import-user printed %q", subject)
	}
	if strings.Contains(out.String(), "password") {
		t.Fatalf("pb-import-user output mentions the password: %q", out.String())
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, []string{"-addr", addr, "-database-url", databaseURL, "-pb-dir", pbDir, "-tokens", tokensPath})
	}()
	base := "http://" + addr
	deadline := time.Now().Add(20 * time.Second)
	for {
		if resp, err := http.Get(base + "/healthz"); err == nil {
			_ = resp.Body.Close()
			break
		}
		select {
		case err := <-errCh:
			t.Fatalf("adhd-sync exited early: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("adhd-sync did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}

	status, raw := call(t, http.MethodGet, base+"/v1/config", "", nil)
	if status != http.StatusOK || !strings.Contains(string(raw), `"auth":"pocketbase"`) {
		t.Fatalf("/v1/config = %d %s", status, raw)
	}
	status, raw = call(t, http.MethodPost, base+"/api/collections/users/auth-with-password", "",
		map[string]string{"identity": "ada@example.com", "password": "a long enough password"})
	var signedIn struct {
		Token  string `json:"token"`
		Record struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"record"`
	}
	if err := json.Unmarshal(raw, &signedIn); err != nil || status != http.StatusOK || signedIn.Token == "" {
		t.Fatalf("sign in = %d %s", status, raw)
	}
	if "pb:"+signedIn.Record.ID != subject || signedIn.Record.Name != "Ada" {
		t.Fatalf("signed in as %+v, imported %s", signedIn.Record, subject)
	}

	// The old Zitadel identity still works through the chain and owns a device.
	status, _ = call(t, http.MethodPost, base+"/v1/devices", "old-token", map[string]string{"name": "laptop"})
	if status != http.StatusOK {
		t.Fatalf("device with the old token = %d", status)
	}
	// The PocketBase identity starts with an empty workspace, then inherits the old one.
	listDevices := func() string {
		status, raw := call(t, http.MethodGet, base+"/v1/devices", signedIn.Token, nil)
		if status != http.StatusOK {
			t.Fatalf("GET /v1/devices = %d %s", status, raw)
		}
		return string(raw)
	}
	if got := listDevices(); strings.Contains(got, "laptop") {
		t.Fatalf("new identity already sees the laptop: %s", got)
	}
	var moved bytes.Buffer
	handled, err = runAccountCommand("move-subject",
		[]string{"-database-url", databaseURL, "-from", "zitadel-user-1", "-to", subject}, nil, &moved)
	if !handled || err != nil {
		t.Fatalf("move-subject: handled=%v err=%v", handled, err)
	}
	if got := listDevices(); !strings.Contains(got, "laptop") {
		t.Fatalf("after the move the new identity sees %s", got)
	}
	// The old identity is kept as an alias: list-subjects shows it, and the old token still works.
	var listed bytes.Buffer
	if handled, err := runAccountCommand("list-subjects", []string{"-database-url", databaseURL}, nil, &listed); !handled || err != nil {
		t.Fatalf("list-subjects: handled=%v err=%v", handled, err)
	}
	if !strings.Contains(listed.String(), "zitadel-user-1 alias 1 ") || !strings.Contains(listed.String(), subject+" primary 1 ") {
		t.Fatalf("list-subjects printed:\n%s", listed.String())
	}
	if status, raw := call(t, http.MethodGet, base+"/v1/devices", "old-token", nil); status != http.StatusOK || !strings.Contains(string(raw), "laptop") {
		t.Fatalf("old token after the move = %d %s", status, raw)
	}
	if handled, err := runAccountCommand("drop-alias", []string{"-database-url", databaseURL, "-subject", "zitadel-user-1"}, nil, io.Discard); !handled || err != nil {
		t.Fatalf("drop-alias: handled=%v err=%v", handled, err)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("adhd-sync exited with %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("adhd-sync did not shut down")
	}
}

func TestAccountCommandsValidateTheirFlags(t *testing.T) {
	if handled, _ := runAccountCommand("serve", nil, nil, io.Discard); handled {
		t.Fatal("an unknown command should not be handled")
	}
	for name, args := range map[string][]string{
		"pb-import-user": {"-pb-dir", t.TempDir()},
		"pb-superuser":   {"-email", "a@example.com"},
		"move-subject":   {"-from", "a"},
		"add-alias":      {"-subject", "a"},
		"drop-alias":     {"-subject", "a"},
		"list-subjects":  {},
	} {
		if _, err := runAccountCommand(name, args, strings.NewReader("pw\n"), io.Discard); err == nil {
			t.Errorf("%s with missing flags succeeded", name)
		}
	}
	if _, err := runAccountCommand("pb-superuser", []string{"-pb-dir", t.TempDir(), "-email", "a@example.com"},
		strings.NewReader("\n"), io.Discard); err == nil {
		t.Error("an empty password was accepted")
	}
}
