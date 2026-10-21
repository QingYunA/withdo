// Package pbauth embeds PocketBase in adhd-sync as the account system (M13): PocketBase owns the
// users, their passwords and the sign-in endpoints; this package makes it an Authenticator for the
// sync routes and serves both from one listener. Only cmd/adhd-sync imports it, so PocketBase
// never reaches the desktop app or the phone core.
package pbauth

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Open creates the PocketBase app with its data in dir, opens the databases and applies the
// migrations, ours included. Its own CLI is not used; the caller owns the flags. Close the app
// with Close.
func Open(dir string) (*pocketbase.PocketBase, error) {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:  dir,
		HideStartBanner: true,
	})
	if err := app.Bootstrap(); err != nil {
		return nil, fmt.Errorf("bootstrap PocketBase in %s: %w", dir, err)
	}
	if err := app.RunAllMigrations(); err != nil {
		_ = app.ClearBootstrap()
		return nil, fmt.Errorf("migrate PocketBase in %s: %w", dir, err)
	}
	return app, nil
}

// Close releases the app's databases.
func Close(app core.App) error { return app.ClearBootstrap() }

// Serve serves PocketBase's API and handler (the sync server's routes, /v1/ and /healthz) on ln
// until ctx is cancelled. Sync requests skip PocketBase's request log and its token loading: they
// authenticate through the Authenticator, and logging every poke would fill the log database.
func Serve(ctx context.Context, app core.App, ln net.Listener, handler http.Handler) error {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Listener = ln
		// The event stream outlives PocketBase's five minute write timeout.
		e.Server.WriteTimeout = 0
		for _, pattern := range []string{"/v1/{path...}", "/healthz"} {
			e.Router.Any(pattern, apis.WrapStdHandler(handler)).
				Unbind(apis.DefaultActivityLoggerMiddlewareId, apis.DefaultLoadAuthTokenMiddlewareId)
		}
		return e.Next()
	})
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			// Terminate stops the HTTP server first, then closes the databases.
			_ = app.OnTerminate().Trigger(&core.TerminateEvent{App: app}, func(e *core.TerminateEvent) error {
				return nil
			})
		case <-done:
		}
	}()
	return apis.Serve(app, apis.ServeConfig{HttpAddr: ln.Addr().String()})
}
