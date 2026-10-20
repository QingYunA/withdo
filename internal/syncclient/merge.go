package syncclient

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// mergeReferences are every local column that points at a synced row. All of them follow the row
// when a merge renumbers it, including task_images, which never syncs but would otherwise cascade
// away with a task that maps onto one of the account's.
var mergeReferences = []struct {
	Table  string
	Column string
	Target string
}{
	{"tasks", "category_id", syncproto.EntityCategories},
	{"tasks", "project_id", syncproto.EntityProjects},
	{"tasks", "milestone_id", syncproto.EntityMilestones},
	{"tasks", "parent_id", syncproto.EntityTasks},
	{"milestones", "project_id", syncproto.EntityProjects},
	{"reports", "task_id", syncproto.EntityTasks},
	{"projects", "parent_id", syncproto.EntityProjects},
	{"task_images", "task_id", syncproto.EntityTasks},
}

// LinkMerge joins a workspace that already has rows without losing this database's data. It
// registers a joining device, leases id blocks for the local rows, pulls the whole account into
// memory, then in one local write moves every local row to the account: a built-in category maps to
// the account category with the same builtin, a user category to the live category with the same
// name, a project with a repo_slug to the live project with the same slug, and every other row to a
// fresh id. Rows that map onto an account row are dropped after their references moved; the account
// rows are applied; local-only columns of a merged project stay. Every local row with a fresh id is
// queued as a full put, and the merge ends when those puts are uploaded. It returns the old to new
// task ids for the app's files keyed by task id.
func (c *Client) LinkMerge(ctx context.Context, name string) (map[int64]int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("syncclient: device name is required")
	}
	if err := c.ensureUnlinked(ctx); err != nil {
		return nil, err
	}
	// A merging device joins: it must not take over a snapshot another device is seeding.
	dev, err := c.register(ctx, name, false)
	if err != nil {
		if isStatus(err, 409) {
			return nil, ErrSeeding
		}
		return nil, err
	}
	if dev.Seeding {
		// LinkMerge is for a workspace that already has rows; an empty one should be seeded with
		// Link. End the seeding this registration started instead of leaving it behind.
		_ = c.markSeeded(ctx, dev.DeviceID)
		return nil, errors.New("syncclient: the workspace is empty; link without --merge to seed it")
	}
	if err := c.leaseMergeBlocks(ctx, dev.DeviceID); err != nil {
		return nil, err
	}
	account, cookie, err := c.pullAccount(ctx, dev.DeviceID)
	if err != nil {
		return nil, err
	}
	tasks, err := c.mergeLocal(ctx, dev, account, cookie)
	if err != nil {
		return nil, err
	}
	if _, err := c.pushAll(ctx); err != nil {
		return tasks, err
	}
	if err := c.refill(ctx); err != nil {
		return tasks, err
	}
	return tasks, nil
}

// LinkReplace backs this database up, deletes every synced row and every task image, and joins the
// account as an empty device, pulling everything. It returns the backup file's path. The backup is
// taken before anything is deleted.
func (c *Client) LinkReplace(ctx context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("syncclient: device name is required")
	}
	if err := c.ensureUnlinked(ctx); err != nil {
		return "", err
	}
	dev, err := c.register(ctx, name, false)
	if err != nil {
		if isStatus(err, 409) {
			return "", ErrSeeding
		}
		return "", err
	}
	dbPath, err := c.core.DatabasePath(ctx)
	if err != nil {
		return "", err
	}
	if dbPath == "" {
		return "", errors.New("syncclient: cannot back up an in-memory database")
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return "", fmt.Errorf("syncclient: resolve database path: %w", err)
	}
	backupDir := filepath.Join(filepath.Dir(abs), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("syncclient: create backup folder: %w", err)
	}
	backupPath := filepath.Join(backupDir,
		"adhd-todo-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".db")
	if err := c.core.BackupTo(ctx, backupPath); err != nil {
		return "", err
	}
	err = c.core.ApplyChanges(ctx, func(q store.Querier) error {
		if _, err := q.ExecContext(ctx, "PRAGMA defer_foreign_keys = ON"); err != nil {
			return fmt.Errorf("defer foreign keys: %w", err)
		}
		// Children first; task_images goes before tasks even though the cascade would take it.
		for _, table := range []string{"task_images", "reports", "tasks", "milestones", "categories", "projects"} {
			if _, err := q.ExecContext(ctx, "DELETE FROM "+quoteIdent(table)); err != nil {
				return fmt.Errorf("empty %s: %w", table, err)
			}
		}
		// A row queued by an abandoned earlier attempt must not outlive the rows it names, and id
		// blocks belong to this Mac's old device and workspace: the empty join leases new ones.
		for _, table := range []string{"sync_outbox", "sync_id_blocks"} {
			if _, err := q.ExecContext(ctx, "DELETE FROM "+table); err != nil {
				return fmt.Errorf("empty %s: %w", table, err)
			}
		}
		return c.storeLinkState(ctx, q, dev)
	})
	if err != nil {
		return backupPath, fmt.Errorf("replace the local data: %w", err)
	}
	if dev.Seeding {
		// An empty workspace: the device was registered as its seeder, and an empty device has
		// nothing to seed.
		if err := c.endSeeding(ctx); err != nil {
			return backupPath, err
		}
	}
	if _, err := c.pullAll(ctx); err != nil {
		return backupPath, err
	}
	if err := c.refill(ctx); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}

// ensureUnlinked refuses a merge or replace on a database Link already linked.
func (c *Client) ensureUnlinked(ctx context.Context) error {
	id, found, err := c.core.SyncState(ctx, stateDeviceID)
	if err != nil {
		return err
	}
	if found && id != "" {
		return errors.New("syncclient: this database is already linked")
	}
	return nil
}

// storeLinkState records a device on the local database. It runs inside the write that makes the
// rows ready for it.
func (c *Client) storeLinkState(ctx context.Context, q store.Querier, dev syncproto.CreateDeviceResponse) error {
	if err := setState(ctx, q, stateDeviceID, dev.DeviceID); err != nil {
		return err
	}
	if err := setState(ctx, q, stateWorkspaceID, strconv.FormatInt(dev.WorkspaceID, 10)); err != nil {
		return err
	}
	if err := setState(ctx, q, stateEnabled, "1"); err != nil {
		return err
	}
	return setState(ctx, q, stateCookie, "0")
}

// leaseMergeBlocks leases enough ids for every local row of each entity, so the one merge write can
// give every row that does not map onto an account row a fresh id. Rows that do map leave their
// unused ids in sync_id_blocks for later writes.
func (c *Client) leaseMergeBlocks(ctx context.Context, deviceID string) error {
	counts, err := c.core.SyncCounts(ctx)
	if err != nil {
		return err
	}
	for _, entity := range entities {
		left := counts[entity]
		for left > 0 {
			count := left
			if count > syncproto.MaxIDsPerRequest {
				count = syncproto.MaxIDsPerRequest
			}
			block, err := c.leaseIDs(ctx, deviceID, entity, int(count))
			if err != nil {
				if isStatus(err, 409) {
					return ErrSeeding
				}
				return err
			}
			if err := c.storeBlock(ctx, entity, block.Start, block.End); err != nil {
				return err
			}
			left -= count
		}
	}
	return nil
}

// pullAccount pages through the whole account from zero and keeps the newest version of every row.
// Tombstones are remembered as absent, so a row deleted and later restored is live again.
func (c *Client) pullAccount(ctx context.Context, deviceID string) (map[string]syncproto.Change, int64, error) {
	account := map[string]syncproto.Change{}
	cookie := int64(0)
	for {
		page, err := c.pull(ctx, deviceID, cookie, syncproto.DefaultPullLimit)
		if err != nil {
			if isStatus(err, 409) {
				return nil, 0, ErrSeeding
			}
			return nil, 0, err
		}
		for _, change := range page.Changes {
			key := change.Entity + "/" + strconv.FormatInt(change.RowID, 10)
			if change.Deleted {
				delete(account, key)
				continue
			}
			account[key] = change
		}
		cookie = page.Cookie
		if !page.More {
			return account, cookie, nil
		}
	}
}

// mergeLocal runs the one local write of a merge and returns the task id map.
func (c *Client) mergeLocal(ctx context.Context, dev syncproto.CreateDeviceResponse,
	account map[string]syncproto.Change, cookie int64) (map[int64]int64, error) {
	var tasks map[int64]int64
	err := c.core.ApplyChanges(ctx, func(q store.Querier) error {
		if _, err := q.ExecContext(ctx, "PRAGMA defer_foreign_keys = ON"); err != nil {
			return fmt.Errorf("defer foreign keys: %w", err)
		}
		var err error
		tasks, err = mergeInWrite(ctx, q, dev, account, cookie)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("merge the local data: %w", err)
	}
	return tasks, nil
}

type localCategory struct {
	id      int64
	builtin sql.NullString
	name    string
}

type localProject struct {
	id       int64
	slug     sql.NullString
	repoPath sql.NullString
}

// mergeInWrite is the one transaction of LinkMerge: renumber rows and references, drop the rows the
// account already has, apply the account's rows, keep merged projects' device-local paths, and queue
// the renumbered rows for upload.
func mergeInWrite(ctx context.Context, q store.Querier, dev syncproto.CreateDeviceResponse,
	account map[string]syncproto.Change, cookie int64) (map[int64]int64, error) {
	local := make(map[string][]int64, len(entities))
	for _, entity := range entities {
		ids, err := readLocalIDs(ctx, q, entity)
		if err != nil {
			return nil, err
		}
		local[entity] = ids
	}
	categories, err := readLocalCategories(ctx, q)
	if err != nil {
		return nil, err
	}
	projects, err := readLocalProjects(ctx, q)
	if err != nil {
		return nil, err
	}
	accountCategoriesByBuiltin, accountCategoriesByName, accountProjectsBySlug := accountKeys(account)

	targets := make(map[string]map[int64]int64, len(entities))
	merged := make(map[string]map[int64]bool, len(entities))
	for _, entity := range entities {
		targets[entity] = map[int64]int64{}
		merged[entity] = map[int64]bool{}
	}
	for _, category := range categories {
		id, ok := accountCategoryID(category, accountCategoriesByBuiltin, accountCategoriesByName)
		if !ok {
			continue
		}
		targets[syncproto.EntityCategories][category.id] = id
		merged[syncproto.EntityCategories][category.id] = true
	}
	for _, project := range projects {
		if !project.slug.Valid {
			continue
		}
		if id, ok := accountProjectsBySlug[project.slug.String]; ok {
			targets[syncproto.EntityProjects][project.id] = id
			merged[syncproto.EntityProjects][project.id] = true
		}
	}

	// Fresh ids only for the rows that do not map onto the account. The leased blocks were sized
	// for every local row, so there are always enough.
	for _, entity := range entities {
		need := len(local[entity]) - len(merged[entity])
		if need == 0 {
			continue
		}
		ids, err := allocateIDs(ctx, q, entity, need)
		if err != nil {
			return nil, err
		}
		next := 0
		switch entity {
		case syncproto.EntityCategories:
			for _, category := range categories {
				if merged[entity][category.id] {
					continue
				}
				targets[entity][category.id] = ids[next]
				next++
			}
		case syncproto.EntityProjects:
			for _, project := range projects {
				if merged[entity][project.id] {
					continue
				}
				targets[entity][project.id] = ids[next]
				next++
			}
		default:
			for _, id := range local[entity] {
				if merged[entity][id] {
					continue
				}
				targets[entity][id] = ids[next]
				next++
			}
		}
		if next != need {
			return nil, fmt.Errorf("syncclient: mapped %d of %d %s rows", next, need, entity)
		}
	}

	// Move every local row out of the positive id space first, then move the references with it.
	// The second pass goes from those negative ids to the final ids, so no remap can land on an id
	// an earlier remap still has to read: without the two passes, one row's target could be
	// another row's old id.
	for _, entity := range entities {
		for _, old := range local[entity] {
			if _, err := q.ExecContext(ctx,
				"UPDATE "+quoteIdent(entity)+" SET id = ? WHERE id = ?", tempID(old), old); err != nil {
				return nil, fmt.Errorf("move %s %d aside: %w", entity, old, err)
			}
		}
	}
	for _, ref := range mergeReferences {
		if err := remapReferences(ctx, q, ref.Table, ref.Column, targets[ref.Target],
			func(old, _ int64) (int64, int64) { return old, tempID(old) }); err != nil {
			return nil, err
		}
	}
	for _, ref := range mergeReferences {
		if err := remapReferences(ctx, q, ref.Table, ref.Column, targets[ref.Target],
			func(old, target int64) (int64, int64) { return tempID(old), target }); err != nil {
			return nil, err
		}
	}
	// Only merged rows still sit at a temp id now; no reference points at it any more.
	for _, entity := range entities {
		for old := range targets[entity] {
			if !merged[entity][old] {
				continue
			}
			if _, err := q.ExecContext(ctx,
				"DELETE FROM "+quoteIdent(entity)+" WHERE id = ?", tempID(old)); err != nil {
				return nil, fmt.Errorf("drop merged %s %d: %w", entity, old, err)
			}
		}
	}

	// The account's rows replace the merged local ones. A project's repo_path is device-local, so
	// it comes back from this Mac's row, not from the account.
	repoPaths := map[int64]sql.NullString{}
	for _, project := range projects {
		if merged[syncproto.EntityProjects][project.id] {
			repoPaths[targets[syncproto.EntityProjects][project.id]] = project.repoPath
		}
	}
	for _, key := range sortedChangeKeys(account) {
		change := account[key]
		if change.Deleted {
			continue
		}
		if err := upsertRow(ctx, q, change); err != nil {
			return nil, err
		}
	}
	for id, path := range repoPaths {
		var err error
		if path.Valid {
			_, err = q.ExecContext(ctx, "UPDATE projects SET repo_path = ? WHERE id = ?", path.String, id)
		} else {
			_, err = q.ExecContext(ctx, "UPDATE projects SET repo_path = NULL WHERE id = ?", id)
		}
		if err != nil {
			return nil, fmt.Errorf("restore project %d's path: %w", id, err)
		}
	}

	// Rows that stay move from their temp id to the fresh id; rows that joined the account are gone.
	for _, entity := range entities {
		for _, old := range local[entity] {
			if merged[entity][old] {
				continue
			}
			if _, err := q.ExecContext(ctx,
				"UPDATE "+quoteIdent(entity)+" SET id = ? WHERE id = ?", targets[entity][old], tempID(old)); err != nil {
				return nil, fmt.Errorf("renumber %s %d: %w", entity, old, err)
			}
		}
	}
	// One full put for every row that got a new id. A merged row is the account's already and must
	// not upload this Mac's copy over it.
	now := "strftime('%Y-%m-%dT%H:%M:%fZ', 'now')"
	for _, entity := range entities {
		for _, old := range local[entity] {
			if merged[entity][old] {
				continue
			}
			if _, err := q.ExecContext(ctx,
				"INSERT INTO sync_outbox (entity, row_id, op, created_at) VALUES (?, ?, 'put', "+now+")",
				entity, targets[entity][old]); err != nil {
				return nil, fmt.Errorf("queue merged %s %d: %w", entity, targets[entity][old], err)
			}
		}
	}
	if err := setState(ctx, q, stateDeviceID, dev.DeviceID); err != nil {
		return nil, err
	}
	if err := setState(ctx, q, stateWorkspaceID, strconv.FormatInt(dev.WorkspaceID, 10)); err != nil {
		return nil, err
	}
	if err := setState(ctx, q, stateEnabled, "1"); err != nil {
		return nil, err
	}
	if err := setState(ctx, q, stateCookie, strconv.FormatInt(cookie, 10)); err != nil {
		return nil, err
	}

	tasks := make(map[int64]int64, len(local[syncproto.EntityTasks]))
	for old, target := range targets[syncproto.EntityTasks] {
		tasks[old] = target
	}
	return tasks, nil
}

// readLocalIDs reads one entity's ids in ascending order, the order a merge renumbers them in.
func readLocalIDs(ctx context.Context, q store.Querier, entity string) ([]int64, error) {
	rows, err := q.QueryContext(ctx, "SELECT id FROM "+quoteIdent(entity)+" ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read %s ids: %w", entity, err)
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read %s id: %w", entity, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read %s ids: %w", entity, err)
	}
	return ids, nil
}

func readLocalCategories(ctx context.Context, q store.Querier) ([]localCategory, error) {
	rows, err := q.QueryContext(ctx, "SELECT id, builtin, name FROM categories ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read local categories: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []localCategory
	for rows.Next() {
		var category localCategory
		if err := rows.Scan(&category.id, &category.builtin, &category.name); err != nil {
			return nil, fmt.Errorf("read local category: %w", err)
		}
		out = append(out, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read local categories: %w", err)
	}
	return out, nil
}

func readLocalProjects(ctx context.Context, q store.Querier) ([]localProject, error) {
	rows, err := q.QueryContext(ctx, "SELECT id, repo_slug, repo_path FROM projects ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read local projects: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []localProject
	for rows.Next() {
		var project localProject
		if err := rows.Scan(&project.id, &project.slug, &project.repoPath); err != nil {
			return nil, fmt.Errorf("read local project: %w", err)
		}
		out = append(out, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read local projects: %w", err)
	}
	return out, nil
}

// accountKeys indexes the account's live categories by builtin and by NOCASE name, and its live
// projects by repo_slug, the keys the merge maps by.
func accountKeys(account map[string]syncproto.Change) (byBuiltin, byName, bySlug map[string]int64) {
	byBuiltin = map[string]int64{}
	byName = map[string]int64{}
	bySlug = map[string]int64{}
	for _, change := range account {
		if change.Deleted {
			continue
		}
		switch change.Entity {
		case syncproto.EntityCategories:
			if builtin, ok := stringValue(change.Data["builtin"]); ok && builtin != "" {
				byBuiltin[builtin] = change.RowID
			}
			if name, ok := stringValue(change.Data["name"]); ok && name != "" {
				byName[nocaseKey(name)] = change.RowID
			}
		case syncproto.EntityProjects:
			if slug, ok := stringValue(change.Data["repo_slug"]); ok && slug != "" {
				bySlug[slug] = change.RowID
			}
		}
	}
	return byBuiltin, byName, bySlug
}

// accountCategoryID maps a local category: a built-in by its builtin name, a user category by its
// name in any case.
func accountCategoryID(category localCategory, byBuiltin, byName map[string]int64) (int64, bool) {
	if category.builtin.Valid && category.builtin.String != "" {
		id, ok := byBuiltin[category.builtin.String]
		return id, ok
	}
	id, ok := byName[nocaseKey(category.name)]
	return id, ok
}

// nocaseKey folds the ASCII letters the way SQLite's NOCASE collation on categories.name does. A
// strings.ToLower would also fold non-ASCII letters that the unique index keeps apart.
func nocaseKey(value string) string {
	raw := []byte(value)
	for i, b := range raw {
		if b >= 'A' && b <= 'Z' {
			raw[i] = b + ('a' - 'A')
		}
	}
	return string(raw)
}

// stringValue reads a string from a JSON-decoded row value.
func stringValue(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok
}

// tempID is the negative id a merge parks a local row at. Negating and subtracting one keeps it
// away from zero and from every account id, which are positive.
func tempID(id int64) int64 { return -id - 1 }

// remapReferences rewrites one referencing column through mapping. It runs twice per merge: first
// local ids to temp ids, then temp ids to final ids, so no rewritten value can be mistaken for a
// value a later rewrite still has to change.
func remapReferences(ctx context.Context, q store.Querier, table, column string, mapping map[int64]int64,
	move func(old, target int64) (from, to int64)) error {
	for old, target := range mapping {
		from, to := move(old, target)
		if _, err := q.ExecContext(ctx,
			"UPDATE "+quoteIdent(table)+" SET "+quoteIdent(column)+" = ? WHERE "+quoteIdent(column)+" = ?",
			to, from); err != nil {
			return fmt.Errorf("remap %s.%s %d to %d: %w", table, column, from, to, err)
		}
	}
	return nil
}

// allocateIDs takes n ids from the local blocks, smallest block first, and spends them. It mirrors
// domain's id allocation, but runs inside the merge's own write.
func allocateIDs(ctx context.Context, q store.Querier, entity string, n int) ([]int64, error) {
	out := make([]int64, 0, n)
	for len(out) < n {
		var (
			rowid int64
			next  int64
			end   int64
		)
		err := q.QueryRowContext(ctx,
			"SELECT rowid, next, end FROM sync_id_blocks WHERE entity = ? AND end > next ORDER BY next, rowid LIMIT 1",
			entity).Scan(&rowid, &next, &end)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w for %s", domain.ErrNoSyncIDs, entity)
		}
		if err != nil {
			return nil, fmt.Errorf("read %s id block: %w", entity, err)
		}
		take := int64(n - len(out))
		if end-next < take {
			take = end - next
		}
		for i := int64(0); i < take; i++ {
			out = append(out, next+i)
		}
		next += take
		if next >= end {
			if _, err := q.ExecContext(ctx, "DELETE FROM sync_id_blocks WHERE rowid = ?", rowid); err != nil {
				return nil, fmt.Errorf("drop spent %s id block: %w", entity, err)
			}
			continue
		}
		if _, err := q.ExecContext(ctx, "UPDATE sync_id_blocks SET next = ? WHERE rowid = ?", next, rowid); err != nil {
			return nil, fmt.Errorf("advance %s id block: %w", entity, err)
		}
	}
	return out, nil
}

// sortedChangeKeys applies account rows in a stable order, which keeps a failing merge reproducible.
func sortedChangeKeys(account map[string]syncproto.Change) []string {
	keys := make([]string, 0, len(account))
	for key := range account {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
