package syncclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFile is the name of the sync configuration next to the database.
const ConfigFile = "sync.json"

// Config is sync.json: the server to talk to, and either the development bearer token, "auth":
// "pocketbase" for the server's own accounts, or the OIDC issuer and client id of the account, with any scopes the issuer needs beyond "openid
// offline_access" (Zitadel's project audience). Account is the signed-in name and email, so the
// Settings pane can show who this Mac belongs to offline. The refresh token is never written here.
type Config struct {
	Server   string   `json:"server"`
	Auth     string   `json:"auth,omitempty"`
	Token    string   `json:"token,omitempty"`
	Issuer   string   `json:"issuer,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	Account  *Account `json:"account,omitempty"`
}

// ConfigPath returns the sync.json path next to the database at dbPath.
func ConfigPath(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), ConfigFile)
}

// LoadConfig reads sync.json next to dbPath. found is false when the file is not there, which is
// how a Mac that has not linked looks.
func LoadConfig(dbPath string) (cfg Config, found bool, err error) {
	return LoadConfigAt(ConfigPath(dbPath))
}

// LoadConfigAt reads the sync.json at path. found is false when it does not exist.
func LoadConfigAt(path string) (cfg Config, found bool, err error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	cfg.Server = strings.TrimRight(strings.TrimSpace(cfg.Server), "/")
	if cfg.Server == "" {
		return Config{}, false, fmt.Errorf("read %s: server is empty", path)
	}
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.Issuer = strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	return cfg, true, nil
}

// SaveConfig writes sync.json next to dbPath with mode 0600.
func SaveConfig(dbPath string, cfg Config) error {
	return SaveConfigAt(ConfigPath(dbPath), cfg)
}

// SaveConfigAt writes the sync.json at path with mode 0600. The file holds a token, so it is never
// readable by other users, also when it already existed with a wider mode.
func SaveConfigAt(path string, cfg Config) error {
	cfg.Server = strings.TrimRight(strings.TrimSpace(cfg.Server), "/")
	if cfg.Server == "" {
		return errors.New("syncclient: server is required")
	}
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.Issuer = strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create sync config folder: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect %s: %w", path, err)
	}
	return nil
}
