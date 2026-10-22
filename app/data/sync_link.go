package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// SyncSignIn signs in with an account and password through the sync server, links this database
// and starts the runner. When both sides have tasks it stops at the conflict the pane asks about.
func (r *Real) SyncSignIn(loginName, password string) {
	if r.syncPath == "" || r.opts.Secrets == nil {
		r.setSyncError(errors.New("sync is not set up on this Mac"))
		return
	}
	r.mu.Lock()
	if r.sync.signingIn {
		r.mu.Unlock()
		return
	}
	r.loadSyncLocked()
	r.sync.signingIn = true
	r.sync.signInStage = SignInStageChecking
	r.sync.conflict = nil
	r.sync.status.LastError = ""
	r.sync.failedSince = time.Time{}
	r.sync.signInFailure = SignInFailureNone
	server := r.sync.server
	r.mu.Unlock()
	r.syncChanged()
	go r.signIn(server, loginName, password)
}

// signIn is SyncSignIn off the UI thread. The password lives only in this call, never in the
// config, the snapshot or a log line.
func (r *Real) signIn(server, loginName, password string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	public, err := syncclient.FetchConfig(ctx, server, nil)
	if err != nil {
		r.failSignIn(err)
		return
	}
	cfg := syncclient.Config{Server: server}
	if err := cfg.ApplyPublic(public); err != nil {
		r.failSignIn(err)
		return
	}
	tokens := r.accountSource(cfg)
	if err := tokens.SignIn(ctx, server, loginName, password); err != nil {
		r.failSignIn(err)
		return
	}
	r.finishSignIn(ctx, server, cfg, tokens)
}

// finishSignIn is everything after a token is stored, shared by the password and the code
// sign-in: save the account, link the database (or resume it), and stop at the conflict when both
// sides have tasks.
func (r *Real) finishSignIn(ctx context.Context, server string, cfg syncclient.Config, tokens syncclient.AccountSource) {
	r.setSignInStage(SignInStageDownloading)
	account, hasAccount := tokens.Account()
	if hasAccount {
		cfg.Account = &account
	}
	client, err := syncclient.New(r.core, syncclient.Options{BaseURL: server, TokenSource: tokens})
	if err != nil {
		r.failSignIn(err)
		return
	}
	if err := syncclient.SaveConfigAt(r.syncPath, cfg); err != nil {
		r.failSignIn(err)
		return
	}
	r.mu.Lock()
	r.sync.cfg, r.sync.server = cfg, server
	r.sync.account, r.sync.hasAccount = account, hasAccount
	r.sync.tokens, r.sync.client = tokens, client

	r.sync.seeding = false
	r.sync.seedTotal, r.sync.pending = 0, 0
	linked := r.sync.linked
	r.mu.Unlock()
	if linked {
		// The database kept its device when the user signed out: signing in again resumes it.
		r.mu.Lock()
		r.sync.signingIn = false
		r.sync.signInStage = SignInStageNone
		r.sync.signInFailure = SignInFailureNone
		r.sync.code = codeState{}
		r.sync.status = syncclient.Status{State: syncclient.StateSyncing}
		r.sync.failedSince = time.Time{}
		r.mu.Unlock()
		r.startSync(context.Background())
		r.syncChanged()
		return
	}

	stop := make(chan struct{})
	go r.watchSeed(ctx, client, stop)
	err = client.Link(ctx, r.syncDeviceName())
	close(stop)
	r.mu.Lock()
	r.sync.seeding = false
	r.mu.Unlock()
	if errors.Is(err, syncclient.ErrLocalData) {
		r.conflictAfterLink(ctx, client)
		return
	}
	if err != nil {
		r.failSignIn(err)
		return
	}
	r.mu.Lock()
	r.sync.linked = true
	r.sync.signingIn = false
	r.sync.signInStage = SignInStageNone
	r.sync.conflict = nil
	r.sync.signInFailure = SignInFailureNone
	r.sync.code = codeState{}
	r.sync.status = syncclient.Status{State: syncclient.StateSyncing}
	r.sync.failedSince = time.Time{}
	r.mu.Unlock()
	r.startSync(context.Background())
	r.syncChanged()
}

// watchSeed keeps the upload count up to date while Link pushes the first snapshot.
func (r *Real) watchSeed(ctx context.Context, client *syncclient.Client, stop <-chan struct{}) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			total, err := client.SeedTotal(ctx)
			if err != nil || total == 0 {
				continue
			}
			pending, err := client.Pending(ctx)
			if err != nil {
				continue
			}
			r.mu.Lock()
			changed := r.sync.seedTotal != total || r.sync.pending != pending
			r.sync.seedTotal, r.sync.pending = total, pending
			r.mu.Unlock()
			if changed {
				r.syncChanged()
			}
		}
	}
}

// conflictAfterLink asks the pane which side wins when both this Mac and the account have tasks.
func (r *Real) conflictAfterLink(ctx context.Context, client *syncclient.Client) {
	counts, err := r.core.SyncCounts(ctx)
	if err != nil {
		r.failSignIn(err)
		return
	}
	devices, _ := client.Devices(ctx)
	// The device this sign-in just registered has never synced; it is not one of the devices
	// whose data the account already has.
	seen := 0
	for _, d := range devices {
		if d.LastSeenAt != nil {
			seen++
		}
	}
	r.mu.Lock()
	r.sync.signingIn = false
	r.sync.signInStage = SignInStageNone
	r.sync.seeding = false
	r.sync.conflict = &SyncConflict{
		LocalTasks: int(counts["tasks"]), LocalProjects: int(counts["projects"]), AccountDevices: seen,
	}
	r.sync.devices = r.syncDevicesLocked(devices)
	r.sync.status.LastError = ""
	r.mu.Unlock()
	r.syncChanged()
}

// SyncMerge merges this Mac's tasks into the account, then renames the ids in the files the app
// keeps by task id.
func (r *Real) SyncMerge() {
	client, ok := r.syncClientForConflict()
	if !ok {
		return
	}
	r.mu.Lock()
	r.sync.signingIn = true
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		ids, err := client.LinkMerge(ctx, r.syncDeviceName())
		if err != nil {
			r.failSignIn(err)
			return
		}
		if err := rewriteSyncTaskIDs(filepath.Dir(r.syncPath), ids); err != nil {
			// The merge is done; a file the app can rebuild is not worth undoing it for.
			r.setSyncError(err)
		}
		r.finishLink(client)
	}()
}

// SyncUseAccount backs this Mac up and replaces its data with the account's.
func (r *Real) SyncUseAccount() {
	client, ok := r.syncClientForConflict()
	if !ok {
		return
	}
	r.mu.Lock()
	r.sync.signingIn = true
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if _, err := client.LinkReplace(ctx, r.syncDeviceName()); err != nil {
			r.failSignIn(err)
			return
		}
		r.finishLink(client)
	}()
}

// finishLink records a successful LinkMerge or LinkReplace and starts the runner.
func (r *Real) finishLink(client *syncclient.Client) {
	r.mu.Lock()
	r.sync.linked = true
	r.sync.signingIn = false
	r.sync.signInStage = SignInStageNone
	r.sync.conflict = nil
	r.sync.code = codeState{}
	r.sync.status = syncclient.Status{State: syncclient.StateSyncing}
	r.sync.failedSince = time.Time{}
	r.sync.client = client
	r.mu.Unlock()
	r.startSync(context.Background())
	r.syncChanged()
}

// SyncCancel abandons a sign-in that found tasks on both sides. The data stays as it was and the
// token is deleted again.
func (r *Real) SyncCancel() {
	r.mu.Lock()
	conflict := r.sync.conflict
	r.sync.conflict = nil
	r.sync.code = codeState{}
	r.sync.signingIn = false
	r.sync.signInFailure = SignInFailureNone
	r.sync.status.LastError = ""
	r.sync.failedSince = time.Time{}
	tokens := r.sync.tokens
	cfg := r.sync.cfg
	cfg.Account = nil
	r.sync.cfg = cfg
	r.sync.account, r.sync.hasAccount = syncclient.Account{}, false
	r.sync.tokens, r.sync.client = nil, nil
	r.mu.Unlock()
	if conflict == nil {
		return
	}
	if source, ok := tokens.(syncclient.AccountSource); ok {
		_ = source.Logout()
	}
	if r.syncPath != "" && cfg.Server != "" {
		_ = syncclient.SaveConfigAt(r.syncPath, cfg)
	}
	r.syncChanged()
}

// rewriteSyncTaskIDs renames task ids in the three files the app keys by task id. The files are
// beside the database; a missing one is skipped. recent-changes.json is keyed by URL today, but
// the walk also renames any numeric key or task id it finds there.
func rewriteSyncTaskIDs(dir string, ids map[int64]int64) error {
	if len(ids) == 0 {
		return nil
	}
	mapping := make(map[string]string, len(ids))
	for old, id := range ids {
		mapping[strconv.FormatInt(old, 10)] = strconv.FormatInt(id, 10)
	}
	var errs []error
	for _, name := range []string{syncSeenFile, syncDismissed, syncRecentCache} {
		if err := rewriteTaskIDsInFile(filepath.Join(dir, name), mapping); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// rewriteTaskIDsInFile rewrites one JSON file through a temporary one, so a crash leaves the old
// file whole.
func rewriteTaskIDsInFile(path string, mapping map[string]string) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var value any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	value = rewriteTaskIDs(value, "", mapping)
	out, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// rewriteTaskIDs walks a decoded JSON value: numeric object keys are task ids in the two seen
// files, and a value under a task-ish key is a task id elsewhere.
func rewriteTaskIDs(value any, key string, mapping map[string]string) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, child := range v {
			next := k
			if id, ok := mapping[k]; ok {
				if _, err := strconv.Atoi(k); err == nil {
					next = id
				}
			}
			out[next] = rewriteTaskIDs(child, k, mapping)
		}
		return out
	case []any:
		for i, child := range v {
			v[i] = rewriteTaskIDs(child, key, mapping)
		}
		return v
	case json.Number:
		if taskKey(key) {
			if id, ok := mapping[v.String()]; ok {
				return json.Number(id)
			}
		}
		return v
	case string:
		if taskKey(key) {
			if id, ok := mapping[v]; ok {
				return id
			}
		}
		return v
	case float64:
		if taskKey(key) {
			raw := strconv.FormatInt(int64(v), 10)
			if id, ok := mapping[raw]; ok {
				if n, err := strconv.ParseFloat(id, 64); err == nil {
					return n
				}
			}
		}
		return v
	default:
		return value
	}
}

// taskKey says whether a JSON key names a task id, such as "task_id" or "taskId".
func taskKey(key string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "task") && (strings.HasSuffix(k, "id") || k == "task")
}
