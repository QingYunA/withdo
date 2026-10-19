package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
             Link this Mac to a sync server
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
	client, err := syncclient.New(core, syncclient.Options{BaseURL: *server, Token: token})
	if err != nil {
		return err
	}
	hasData, err := core.SyncHasLocalData(ctx)
	if err != nil {
		return err
	}
	if err := client.Link(ctx, strings.TrimSpace(*name)); err != nil {
		return syncFriendlyError(err)
	}
	if err := syncclient.SaveConfigAt(syncclient.ConfigPath(dbPath),
		syncclient.Config{Server: *server, Token: token}); err != nil {
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
	client, err := syncclient.New(core, syncclient.Options{BaseURL: cfg.Server, Token: cfg.Token})
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
	case syncclient.IsUnauthorized(err):
		return errors.New("the sync server refused the token; run adhd-core sync link again")
	default:
		return err
	}
}
