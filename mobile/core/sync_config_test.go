package core

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// TestSyncConfigureWritesTheSettings pins SyncConfigure's file and the initial status shape.
func TestSyncConfigureWritesTheSettings(t *testing.T) {
	dir := t.TempDir()
	c, err := open(dir, nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = c.Close() }()

	if err := c.SyncConfigure("https://sync.example.com/", "https://issuer.example/", "client-id", " openid  profile "); err != nil {
		t.Fatalf("SyncConfigure: %v", err)
	}
	cfg, found, err := syncclient.LoadConfigAt(filepath.Join(dir, "sync.json"))
	if err != nil || !found {
		t.Fatalf("read sync.json: found %v, err %v", found, err)
	}
	if cfg.Server != "https://sync.example.com" || cfg.Issuer != "https://issuer.example" || cfg.ClientID != "client-id" {
		t.Errorf("sync.json = %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.Scopes, []string{"openid", "profile"}) {
		t.Errorf("scopes = %v", cfg.Scopes)
	}
	if err := c.SyncConfigure("", "", "", ""); err == nil {
		t.Error("SyncConfigure accepted an empty server")
	}

	var status statusJSON
	if err := json.Unmarshal([]byte(c.SyncStatus()), &status); err != nil {
		t.Fatalf("decode SyncStatus: %v", err)
	}
	if status.State != string(syncclient.StateIdle) || status.Pending != 0 || status.LastSynced != "" {
		t.Errorf("initial status = %+v", status)
	}
}
