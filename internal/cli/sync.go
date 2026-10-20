package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/paths"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// syncEntities is the order `sync status` lists the id blocks in.
var syncEntities = []string{
	syncproto.EntityProjects, syncproto.EntityCategories, syncproto.EntityMilestones,
	syncproto.EntityTasks, syncproto.EntityReports,
}

const syncUsage = `Usage: adhd-core sync <command>

Commands:
  link --server URL --token-file FILE --name NAME
             Link this Mac to a sync server with a static token (development)
  login --server URL --issuer URL --client-id ID --name NAME [--scope SCOPE]...
             Sign in to an OIDC account and link this Mac
  logout     Forget the OIDC sign-in on this Mac
  status     Show the device, workspace, cookie, pending changes and id blocks
  now        Sync once and print the counts pushed and pulled
`

// runSync runs one `adhd-core sync` subcommand.
func runSync(ctx context.Context, args []string, d deps) error {
	if len(args) == 0 {
		return errors.New("usage: adhd-core sync link|status|now")
	}
	dbPath := paths.DBPath(d.getenv, runtime.GOOS, d.home)
	db, err := store.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	core := domain.New(db, domain.Options{})
	switch args[0] {
	case "link":
		return syncLink(ctx, args[1:], dbPath, core, d)
	case "login":
		return syncLogin(ctx, args[1:], dbPath, core, d)
	case "logout":
		return syncLogout(args[1:], d)
	case "status":
		return syncStatus(ctx, core, d)
	case "now":
		return syncNow(ctx, dbPath, core, d)
	default:
		return fmt.Errorf("unknown sync command: %s", args[0])
	}
}

// syncLink registers this database, writes sync.json and drains whatever the link queued.
func syncLink(ctx context.Context, args []string, dbPath string, core *domain.Core, d deps) error {
	fs := flag.NewFlagSet("sync link", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	server := fs.String("server", "", "sync server URL")
	tokenFile := fs.String("token-file", "", "file holding the bearer token")
	name := fs.String("name", "", "device name")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("sync link: %w", err)
	}
	if strings.TrimSpace(*server) == "" {
		return errors.New("sync link: --server is required")
	}
	if strings.TrimSpace(*tokenFile) == "" {
		return errors.New("sync link: --token-file is required")
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("sync link: --name is required")
	}
	raw, err := os.ReadFile(*tokenFile)
	if err != nil {
		return fmt.Errorf("sync link: read token file: %w", err)
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return errors.New("sync link: the token file is empty")
	}
	client, err := syncclient.New(core, syncclient.Options{BaseURL: *server, TokenSource: syncclient.StaticToken(token)})
	if err != nil {
		return err
	}
	hasData, err := core.SyncHasLocalData(ctx)
	if err != nil {
		return err
	}
	return linkAndSave(ctx, client, dbPath, syncclient.Config{Server: *server, Token: token}, *name, hasData, d)
}

// syncLogin signs in through the system browser, writes sync.json without a token and links this
// Mac like sync link does. The refresh token goes to the keychain, never to the file.
func syncLogin(ctx context.Context, args []string, dbPath string, core *domain.Core, d deps) error {
	fs := flag.NewFlagSet("sync login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	server := fs.String("server", "", "sync server URL")
	issuer := fs.String("issuer", "", "OIDC issuer URL")
	clientID := fs.String("client-id", "", "OIDC client id")
	name := fs.String("name", "", "device name")
	var scopes scopeList
	fs.Var(&scopes, "scope", "an extra OIDC scope, repeatable")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("sync login: %w", err)
	}
	if strings.TrimSpace(*server) == "" {
		return errors.New("sync login: --server is required")
	}
	if strings.TrimSpace(*issuer) == "" {
		return errors.New("sync login: --issuer is required")
	}
	if strings.TrimSpace(*clientID) == "" {
		return errors.New("sync login: --client-id is required")
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("sync login: --name is required")
	}
	if d.secrets == nil {
		return errors.New("sync login: no keychain available; use the desktop app")
	}
	source := syncclient.NewOIDCSource(*issuer, *clientID, scopes, d.secrets)
	open := d.openBrowser
	if open == nil {
		open = openBrowser
	}
	if err := source.Login(ctx, func(url string) error {
		_, _ = fmt.Fprintf(d.stdout, "open this URL to sign in:\n%s\n", url)
		return open(url)
	}); err != nil {
		return syncFriendlyError(err)
	}
	client, err := syncclient.New(core, syncclient.Options{BaseURL: *server, TokenSource: source})
	if err != nil {
		return err
	}
	hasData, err := core.SyncHasLocalData(ctx)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(d.stdout, "signed in\n")
	return linkAndSave(ctx, client, dbPath,
		syncclient.Config{Server: *server, Issuer: *issuer, ClientID: *clientID, Scopes: scopes}, *name, hasData, d)
}

// syncLogout forgets the refresh token. The sync.json stays, so signing in again needs no link.
func syncLogout(args []string, d deps) error {
	if len(args) > 0 {
		return errors.New("usage: adhd-core sync logout")
	}
	if d.secrets == nil {
		return errors.New("sync logout: no keychain available; use the desktop app")
	}
	if err := syncclient.DeleteRefreshToken(d.secrets); err != nil {
		return err
	}
	_, _ = io.WriteString(d.stdout, "signed out\n")
	return nil
}

// linkAndSave links the database, writes sync.json and prints what happened. The config is written
// only after the server accepted the link, so a failed link leaves the Mac unconfigured.
func linkAndSave(ctx context.Context, client *syncclient.Client, dbPath string, cfg syncclient.Config,
	name string, hasData bool, d deps) error {
	name = strings.TrimSpace(name)
	if err := client.Link(ctx, name); err != nil {
		return syncFriendlyError(err)
	}
	if err := syncclient.SaveConfigAt(syncclient.ConfigPath(dbPath), cfg); err != nil {
		return err
	}
	// A write made by another process during the link is still in the outbox.
	if pending, err := client.Pending(ctx); err == nil && pending > 0 {
		if _, err := client.SyncWithStats(ctx); err != nil {
			return syncFriendlyError(err)
		}
	}
	device, err := client.DeviceID(ctx)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(d.stdout, "linked device %s\n", device)
	if hasData {
		_, _ = io.WriteString(d.stdout, "seeded workspace\n")
	} else {
		_, _ = io.WriteString(d.stdout, "joined workspace\n")
	}
	return nil
}

// syncStatus prints the local sync state. It reads only this Mac, so it works offline.
func syncStatus(ctx context.Context, core *domain.Core, d deps) error {
	device, found, err := core.SyncState(ctx, "device_id")
	if err != nil {
		return err
	}
	if !found || device == "" {
		return errors.New("sync status: this database is not linked")
	}
	workspace, _, err := core.SyncState(ctx, "workspace_id")
	if err != nil {
		return err
	}
	if workspace == "" {
		workspace = "unknown"
	}
	cookie, found, err := core.SyncState(ctx, "cookie")
	if err != nil {
		return err
	}
	if !found || cookie == "" {
		cookie = "0"
	}
	if _, err := strconv.ParseInt(cookie, 10, 64); err != nil {
		cookie = "0"
	}
	pending, err := core.SyncPending(ctx)
	if err != nil {
		return err
	}
	lastError, found, err := core.SyncState(ctx, "last_error")
	if err != nil {
		return err
	}
	if !found || lastError == "" {
		lastError = "none"
	}
	_, _ = fmt.Fprintf(d.stdout, "device: %s\nworkspace: %s\ncookie: %s\npending: %d\n",
		device, workspace, cookie, pending)
	left := make([]string, 0, len(syncEntities))
	for _, entity := range syncEntities {
		n, err := core.SyncIDsLeft(ctx, entity)
		if err != nil {
			return err
		}
		left = append(left, fmt.Sprintf("%s %d", entity, n))
	}
	_, _ = fmt.Fprintf(d.stdout, "id blocks left: %s\nlast error: %s\n", strings.Join(left, ", "), lastError)
	return nil
}

// syncNow runs one Sync and prints how many mutations and changes went each way.
func syncNow(ctx context.Context, dbPath string, core *domain.Core, d deps) error {
	cfg, found, err := syncclient.LoadConfigAt(syncclient.ConfigPath(dbPath))
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("sync now: %s is missing", syncclient.ConfigPath(dbPath))
	}
	var tokens syncclient.TokenSource
	switch {
	case cfg.Issuer != "":
		if d.secrets == nil {
			return errors.New("sync now: OIDC sign-in needs the desktop app's keychain")
		}
		tokens = syncclient.NewOIDCSource(cfg.Issuer, cfg.ClientID, cfg.Scopes, d.secrets)
	case cfg.Token != "":
		tokens = syncclient.StaticToken(cfg.Token)
	default:
		return errors.New("sync now: sync.json has no token or OIDC issuer")
	}
	client, err := syncclient.New(core, syncclient.Options{BaseURL: cfg.Server, TokenSource: tokens})
	if err != nil {
		return err
	}
	stats, err := client.SyncWithStats(ctx)
	client.RecordResult(ctx, err)
	if err != nil {
		return syncFriendlyError(err)
	}
	_, _ = fmt.Fprintf(d.stdout, "pushed: %d\npulled: %d\n", stats.Pushed, stats.Pulled)
	return nil
}

// syncFriendlyError turns the client's exported errors into the one line the user reads. The
// ErrLocalData line explains the choice S3-UI will offer.
func syncFriendlyError(err error) error {
	switch {
	case errors.Is(err, syncclient.ErrLocalData):
		return errors.New("this Mac already has tasks and the account already has data; merging comes with the Settings page")
	case errors.Is(err, syncclient.ErrSeeding):
		return errors.New("the workspace is still uploading its first snapshot; try again later")
	case errors.Is(err, syncclient.ErrNotLinked):
		return errors.New("this database is not linked; run adhd-core sync link first")
	case syncclient.IsSignedOut(err):
		return errors.New("not signed in; run adhd-core sync login")
	case syncclient.IsUnauthorized(err):
		return errors.New("the sync server refused the token; run adhd-core sync login again")
	default:
		return err
	}
}

// openBrowser opens url in the platform's browser. Tests and the app replace it.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}

// scopeList collects a repeated --scope flag.
type scopeList []string

func (l *scopeList) String() string { return strings.Join(*l, " ") }

func (l *scopeList) Set(v string) error {
	*l = append(*l, v)
	return nil
}
