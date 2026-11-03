package syncclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRawConfig(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ConfigFile)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A config saved with the official server's old name moves to the new one, on disk too, and keeps
// the rest of the file.
func TestLoadConfigMovesLegacyServerToOfficial(t *testing.T) {
	for _, saved := range []string{
		"https://todo.sri-ai.xyz",
		"https://todo.sri-ai.xyz/",
		"  HTTPS://Todo.Sri-AI.xyz/ ",
	} {
		raw := `{"server":"` + saved + `","auth":"pocketbase","account":{"name":"Cyan","email":"c@example.com"}}`
		path := writeRawConfig(t, raw)
		cfg, found, err := LoadConfigAt(path)
		if err != nil || !found {
			t.Fatalf("LoadConfigAt(%q) = %v, %v", saved, found, err)
		}
		if cfg.Server != OfficialServer || cfg.Auth != AuthPocketBase || cfg.Account == nil || cfg.Account.Email != "c@example.com" {
			t.Errorf("%q loaded as %+v, want the official server with auth and account kept", saved, cfg)
		}
		again, _, err := LoadConfigAt(path)
		if err != nil || again.Server != OfficialServer {
			t.Errorf("%q: second load = %+v, %v", saved, again, err)
		}
		onDisk, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(onDisk) == raw || !strings.Contains(string(onDisk), OfficialServer) || !strings.Contains(string(onDisk), "c@example.com") {
			t.Errorf("%q: file not rewritten: %s", saved, onDisk)
		}
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v, want 0600", info.Mode().Perm())
		}
	}
}

func TestLoadConfigLeavesOtherServersAlone(t *testing.T) {
	for _, server := range []string{OfficialServer, "https://todo.example.com", "http://127.0.0.1:18080", "http://todo.sri-ai.xyz"} {
		raw := `{"server":"` + server + `/","auth":"pocketbase"}`
		path := writeRawConfig(t, raw)
		cfg, _, err := LoadConfigAt(path)
		if err != nil || cfg.Server != server {
			t.Errorf("LoadConfigAt(%q) = %q, %v; want the same server", server, cfg.Server, err)
		}
		if onDisk, _ := os.ReadFile(path); string(onDisk) != raw {
			t.Errorf("%q: file was rewritten: %s", server, onDisk)
		}
	}
}

func TestIsOfficialServerTakesBothNames(t *testing.T) {
	for server, want := range map[string]bool{
		OfficialServer: true, LegacyServer: true, " " + LegacyServer + "/ ": true, "HTTPS://API.WITHDO.SI/": true,
		"https://other.example": false, "http://api.withdo.si": false, "": false,
	} {
		if got := IsOfficialServer(server); got != want {
			t.Errorf("IsOfficialServer(%q) = %v, want %v", server, got, want)
		}
	}
}
