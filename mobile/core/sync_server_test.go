package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// TestSyncSetServerReadsThePublicConfig: the Change field's save asks /v1/config and writes the
// issuer, client id and scopes the server names, into a file the status then reads.
func TestSyncSetServerReadsThePublicConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/config" {
			http.NotFound(w, r)
			return
		}
		writeTestJSON(w, map[string]any{
			"issuer":    "https://issuer.example/",
			"client_id": "phone-client",
			"scopes":    []string{"openid", "offline_access"},
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	phone, err := open(dir, newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()

	if err := phone.SyncSetServer(server.URL + "/"); err != nil {
		t.Fatalf("SyncSetServer: %v", err)
	}
	cfg, found, err := syncclient.LoadConfigAt(filepath.Join(dir, "sync.json"))
	if err != nil || !found {
		t.Fatalf("read sync.json: found %v, err %v", found, err)
	}
	if cfg.Server != server.URL || cfg.Issuer != "https://issuer.example" || cfg.ClientID != "phone-client" {
		t.Errorf("sync.json = %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.Scopes, []string{"openid", "offline_access"}) {
		t.Errorf("scopes = %v", cfg.Scopes)
	}

	var status statusJSON
	if err := json.Unmarshal([]byte(phone.SyncStatus()), &status); err != nil {
		t.Fatalf("decode SyncStatus: %v", err)
	}
	if status.Server != server.URL || status.Linked || status.SignedIn {
		t.Errorf("status = %+v, want the server and no link", status)
	}
}

// TestSyncSetServerWritesNothingOnAFailedFetch: a wrong address or a server without sign-in is
// reported before sync.json changes, so the old settings keep working.
func TestSyncSetServerWritesNothingOnAFailedFetch(t *testing.T) {
	noSignIn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(w, map[string]any{"issuer": "", "client_id": "", "scopes": []string{}})
	}))
	defer noSignIn.Close()

	dir := t.TempDir()
	phone, err := open(dir, newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()
	if err := phone.SyncConfigure("https://old.example.com", "https://old-issuer.example", "old-client", ""); err != nil {
		t.Fatalf("SyncConfigure: %v", err)
	}

	for _, server := range []string{"http://example.com", "not a url", noSignIn.URL} {
		if err := phone.SyncSetServer(server); err == nil {
			t.Errorf("SyncSetServer(%q) succeeded", server)
		}
	}
	cfg, found, err := syncclient.LoadConfigAt(filepath.Join(dir, "sync.json"))
	if err != nil || !found {
		t.Fatalf("read sync.json: found %v, err %v", found, err)
	}
	if cfg.Server != "https://old.example.com" || cfg.Issuer != "https://old-issuer.example" {
		t.Errorf("a failed SyncSetServer changed sync.json: %+v", cfg)
	}
}

// TestStatusCarriesTheSettingsTheSyncScreenDraws: the default server before sync.json, a token as
// signed in, a failure's start time and its retry time.
func TestStatusCarriesTheSettingsTheSyncScreenDraws(t *testing.T) {
	phone, err := open(t.TempDir(), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()

	var fresh statusJSON
	if err := json.Unmarshal([]byte(phone.SyncStatus()), &fresh); err != nil {
		t.Fatalf("decode fresh SyncStatus: %v", err)
	}
	if fresh.Server != data.DefaultSyncServer || fresh.Linked || fresh.SignedIn {
		t.Errorf("fresh status = %+v", fresh)
	}

	if err := phone.SyncConfigure("https://sync.example.com", "", "", ""); err != nil {
		t.Fatalf("SyncConfigure: %v", err)
	}
	// A development token is the only credential SyncConfigure can leave; write the file directly
	// the way the desktop's dev setup does.
	cfg, _, err := syncclient.LoadConfigAt(filepath.Join(phone.dataDir, "sync.json"))
	if err != nil {
		t.Fatalf("read sync.json: %v", err)
	}
	cfg.Token = "dev-token"
	if err := syncclient.SaveConfigAt(filepath.Join(phone.dataDir, "sync.json"), cfg); err != nil {
		t.Fatalf("write sync.json: %v", err)
	}
	var signedIn statusJSON
	if err := json.Unmarshal([]byte(phone.SyncStatus()), &signedIn); err != nil {
		t.Fatalf("decode signed-in SyncStatus: %v", err)
	}
	if !signedIn.SignedIn {
		t.Errorf("status with a token = %+v, want signed_in", signedIn)
	}

	retry := demoNow.Add(2 * time.Minute)
	phone.runnerStatus(syncclient.Status{
		State: syncclient.StateError, LastError: "the server did not answer",
		Pending: 3, RetryAt: retry,
	})
	var failed statusJSON
	if err := json.Unmarshal([]byte(phone.SyncStatus()), &failed); err != nil {
		t.Fatalf("decode failed SyncStatus: %v", err)
	}
	if failed.State != "error" || failed.RetryAt == "" || failed.FailedSince == "" {
		t.Errorf("failed status = %+v", failed)
	}
	if failed.LastError != "the server did not answer" {
		t.Errorf("last_error = %q", failed.LastError)
	}
}
