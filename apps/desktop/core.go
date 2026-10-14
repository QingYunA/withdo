package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// CoreError is sent to every page when adhd-core died and could not be restarted.
var CoreError = mygo.NewEvent[string]("core-error")

// CoreConn is where the page finds adhd-core: http://127.0.0.1:<port> with the bearer token.
type CoreConn struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

// Core is the page's view of the sidecar.
type Core struct{ sidecar *Sidecar }

// Conn waits until adhd-core is ready and returns its port and token. It rejects when the
// sidecar failed for good.
func (c *Core) Conn(ctx context.Context) (CoreConn, error) {
	ep, err := c.sidecar.Wait(ctx)
	if err != nil {
		return CoreConn{}, err
	}
	return CoreConn{Port: ep.Port, Token: ep.Token}, nil
}

// sidecarPath is where mygo puts the helper binary: Contents/Resources/bin/adhd-core in the
// bundle, resources/<os>-<arch>/bin/adhd-core under `go run`.
func sidecarPath() (string, error) {
	dir, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "bin", "adhd-core")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return filepath.Join(dir, "darwin-arm64", "bin", "adhd-core"), nil
}
