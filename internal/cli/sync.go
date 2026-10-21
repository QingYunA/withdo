package cli

import (
	"bufio"
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
  link --server URL --token-file FILE --name NAME [--merge|--replace]
             Link this device to a sync server with a static token (development)
  login --server URL --user LOGIN --password-stdin --name NAME [--issuer URL --client-id ID [--scope SCOPE]...] [--merge|--replace]
             Sign in with an account and password, then link this device. The server says how to sign
             in; --issuer and --client-id force an OIDC sign-in instead
  logout     Forget the sign-in on this device
  status     Show the device, workspace, cookie, pending changes and id blocks
  now        Sync once and print the counts pushed and pulled

--password-stdin reads the password from the first line of stdin, like docker login.
--merge and --replace choose what happens when both this device and the account have tasks:
--merge joins this device's tasks to the account; --replace backs this device up and uses the account's data.
`

// runSync runs one `adhd-core sync` subcommand.
func runSync(ctx context.Context, args []string, d deps) error {
	if len(args) == 0 {
		return errors.New("usage: adhd-core sync link|login|logout|status|now")
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
		return syncLogout(args[1:], dbPath, d)
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
	merge := fs.Bool("merge", false, "join this device's tasks to the account's when both have data")
	replace := fs.Bool("replace", false, "back this device up and use the account's data when both have data")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("sync link: %w", err)
	}
	if *merge && *replace {
		return errors.New("sync link: --merge and --replace cannot be used together")
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
	seedRows, err := localRowCount(ctx, core)
	if err != nil {
		return err
	}
	return linkAndSave(ctx, client, dbPath, syncclient.Config{Server: *server, Token: token},
		*name, hasData, seedRows, *merge, *replace, d)
}

// syncLogin signs in with an account and password read from stdin, writes sync.json without a
// token and links this Mac like sync link does. The refresh token goes to the keychain, never to
// the file.
func syncLogin(ctx context.Context, args []string, dbPath string, core *domain.Core, d deps) error {
	fs := flag.NewFlagSet("sync login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	server := fs.String("server", "", "sync server URL")
	issuer := fs.String("issuer", "", "OIDC issuer URL")
	clientID := fs.String("client-id", "", "OIDC client id")
	user := fs.String("user", "", "account login name")
	passwordStdin := fs.Bool("password-stdin", false, "read the password from the first line of stdin")
	name := fs.String("name", "", "device name")
	merge := fs.Bool("merge", false, "join this device's tasks to the account's when both have data")
	replace := fs.Bool("replace", false, "back this device up and use the account's data when both have data")
	var scopes scopeList
	fs.Var(&scopes, "scope", "an extra OIDC scope, repeatable")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("sync login: %w", err)
	}
	if *merge && *replace {
		return errors.New("sync login: --merge and --replace cannot be used together")
	}
	if strings.TrimSpace(*server) == "" {
		return errors.New("sync login: --server is required")
	}
	if (strings.TrimSpace(*issuer) == "") != (strings.TrimSpace(*clientID) == "") {
		return errors.New("sync login: --issuer and --client-id go together; leave both out to use what the server says")
	}
	if strings.TrimSpace(*user) == "" {
		return errors.New("sync login: --user is required")
	}
	if !*passwordStdin {
		return errors.New("sync login: --password-stdin is required")
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("sync login: --name is required")
	}
	if d.secrets == nil {
		return errors.New("sync login: no keychain available; use the desktop app")
	}
	password, err := readPassword(d.stdin)
	if err != nil {
		return err
	}
	// An issuer given on the command line is an OIDC sign-in; without one the server's
	// GET /v1/config says whether to use its PocketBase or an issuer.
	cfg := syncclient.Config{Server: *server}
	if strings.TrimSpace(*issuer) != "" {
		cfg.Issuer, cfg.ClientID, cfg.Scopes = strings.TrimRight(strings.TrimSpace(*issuer), "/"), strings.TrimSpace(*clientID), scopes
	} else {
		public, err := syncclient.FetchConfig(ctx, *server, nil)
		if err != nil {
			return fmt.Errorf("sync login: %w", err)
		}
		if err := cfg.ApplyPublic(public); err != nil {
			return syncFriendlyError(err)
		}
	}
	source := syncclient.NewAccountSource(cfg, d.secrets, dbPath)
	if err := source.SignIn(ctx, *server, *user, password); err != nil {
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
	seedRows, err := localRowCount(ctx, core)
	if err != nil {
		return err
	}
	if account, ok := source.Account(); ok {
		cfg.Account = &account
	}
	_, _ = io.WriteString(d.stdout, "signed in\n")
	return linkAndSave(ctx, client, dbPath, cfg, *name, hasData, seedRows, *merge, *replace, d)
}

// readPassword reads the password from the first line of stdin and drops its line ending, as
// docker login does. The password is not trimmed.
func readPassword(stdin io.Reader) (string, error) {
	if stdin == nil {
		return "", errors.New("sync login: no stdin to read the password from")
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("sync login: read the password: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// syncLogout forgets the refresh token and the PocketBase token. The sync.json stays, so signing in again needs no link.
func syncLogout(args []string, dbPath string, d deps) error {
	if len(args) > 0 {
		return errors.New("usage: adhd-core sync logout")
	}
	if d.secrets == nil {
		return errors.New("sync logout: no keychain available; use the desktop app")
	}
	if err := syncclient.DeleteRefreshTokenAt(d.secrets, dbPath); err != nil {
		return err
	}
	if err := syncclient.DeletePocketBaseToken(d.secrets, dbPath); err != nil {
		return err
	}
	_, _ = io.WriteString(d.stdout, "signed out\n")
	return nil
}

// linkAndSave links the database, writes sync.json and prints what happened. When both this Mac and
// the account have data, Link returns ErrLocalData and --merge or --replace chooses the way on. The
// config is written only once the link exists on this Mac, so a refused link leaves it unconfigured.
func linkAndSave(ctx context.Context, client *syncclient.Client, dbPath string, cfg syncclient.Config,
	name string, hasData bool, seedRows int64, merge, replace bool, d deps) error {
	name = strings.TrimSpace(name)
	err := client.Link(ctx, name)
	if errors.Is(err, syncclient.ErrLocalData) {
		switch {
		case merge:
			tasks, mergeErr := client.LinkMerge(ctx, name)
			if tasks != nil {
				// The local merge committed; sync.json must exist even if the first push failed.
				if saveErr := saveLinkConfig(dbPath, cfg); saveErr != nil {
					return saveErr
				}
			}
			if mergeErr != nil {
				return syncFriendlyError(mergeErr)
			}
			if err := drainLink(ctx, client); err != nil {
				return err
			}
			if err := printLinkedDevice(ctx, client, d); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(d.stdout, "merged this device's %d tasks into the account\n", len(tasks))
			return nil
		case replace:
			backup, replaceErr := client.LinkReplace(ctx, name)
			// The backup is made before the wipe, so its presence alone does not mean this Mac
			// joined; sync.json waits for the device id.
			if backup != "" {
				if _, idErr := client.DeviceID(ctx); idErr == nil {
					if saveErr := saveLinkConfig(dbPath, cfg); saveErr != nil {
						return saveErr
					}
				}
			}
			if replaceErr != nil {
				return syncFriendlyError(replaceErr)
			}
			if err := drainLink(ctx, client); err != nil {
				return err
			}
			if err := printLinkedDevice(ctx, client, d); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(d.stdout, "replaced this device's data with the account's; backup at %s\n", backup)
			return nil
		default:
			return syncFriendlyError(err)
		}
	}
	if err != nil {
		return syncFriendlyError(err)
	}
	if err := saveLinkConfig(dbPath, cfg); err != nil {
		return err
	}
	if err := drainLink(ctx, client); err != nil {
		return err
	}
	if err := printLinkedDevice(ctx, client, d); err != nil {
		return err
	}
	if hasData {
		_, _ = fmt.Fprintf(d.stdout, "uploaded this device's data (%d rows)\n", seedRows)
	} else {
		_, _ = io.WriteString(d.stdout, "joined the account\n")
	}
	return nil
}

func saveLinkConfig(dbPath string, cfg syncclient.Config) error {
	return syncclient.SaveConfigAt(syncclient.ConfigPath(dbPath), cfg)
}

// drainLink uploads whatever another process wrote during the link. A write made by another process
// while the link ran is still in the outbox.
func drainLink(ctx context.Context, client *syncclient.Client) error {
	pending, err := client.Pending(ctx)
	if err != nil || pending == 0 {
		return err
	}
	if _, err := client.SyncWithStats(ctx); err != nil {
		return syncFriendlyError(err)
	}
	return nil
}

func printLinkedDevice(ctx context.Context, client *syncclient.Client, d deps) error {
	device, err := client.DeviceID(ctx)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(d.stdout, "linked device %s\n", device)
	return nil
}

// localRowCount is the number of synced rows `sync link` reports as uploaded after seeding.
func localRowCount(ctx context.Context, core *domain.Core) (int64, error) {
	counts, err := core.SyncCounts(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, n := range counts {
		total += n
	}
	return total, nil
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
	case cfg.SignsInWithAccount():
		if d.secrets == nil {
			return errors.New("sync now: account sign-in needs the desktop app's keychain")
		}
		tokens = syncclient.NewAccountSource(cfg, d.secrets, dbPath)
	case cfg.Token != "":
		tokens = syncclient.StaticToken(cfg.Token)
	default:
		return errors.New("sync now: sync.json has no token or account sign-in")
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
// ErrLocalData line names the two flags that choose what happens when both sides have data.
func syncFriendlyError(err error) error {
	switch {
	case errors.Is(err, syncclient.ErrLocalData):
		return errors.New("this device already has tasks and the account already has data; use --merge to join them or --replace to use the account's data (this device is backed up first)")
	case errors.Is(err, syncclient.ErrSeeding):
		return errors.New("the workspace is still uploading its first snapshot; try again later")
	case errors.Is(err, syncclient.ErrNotLinked):
		return errors.New("this database is not linked; run adhd-core sync link first")
	case errors.Is(err, syncclient.ErrInvalidCredentials):
		return errors.New("the login name or password is wrong")
	case errors.Is(err, syncclient.ErrAccountLocked):
		return errors.New("this account is locked or disabled")
	case errors.Is(err, syncclient.ErrSecondFactor):
		return errors.New("this account uses a second factor; in-app sign-in supports passwords only")
	case errors.Is(err, syncclient.ErrTooManyAttempts):
		return errors.New("too many sign-in attempts; wait a few minutes and try again")
	case errors.Is(err, syncclient.ErrSignInUnsupported):
		return errors.New("this sync server does not support in-app sign-in; update it")
	case syncclient.IsSignedOut(err):
		return errors.New("not signed in; run adhd-core sync login")
	case syncclient.IsUnauthorized(err):
		return errors.New("the sync server refused the token; run adhd-core sync login again")
	default:
		return err
	}
}

// scopeList collects a repeated --scope flag.
type scopeList []string

func (l *scopeList) String() string { return strings.Join(*l, " ") }

func (l *scopeList) Set(v string) error {
	*l = append(*l, v)
	return nil
}
