package store

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// The DSN must be a file:/// URI with an empty authority. On Windows a path
// like C:\...\t.db must not become file://C:/..., which SQLite rejects with
// "invalid uri authority: C:".
func TestDSNHasEmptyURIAuthority(t *testing.T) {
	d := dsn(filepath.Join(t.TempDir(), "nested", "t.db"))
	u, err := url.Parse(d)
	if err != nil {
		t.Fatalf("dsn %q does not parse: %v", d, err)
	}
	if u.Host != "" {
		t.Fatalf("dsn %q has URI authority %q; want file:/// form", d, u.Host)
	}
	if !strings.HasPrefix(d, "file:///") {
		t.Errorf("dsn %q should start with file:///", d)
	}
}
