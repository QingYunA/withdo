package mobilesync

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"

	"github.com/QingYunA/adhd-todo/internal/store"
)

const (
	MaxMutations     = 100
	MaxPullLimit     = 200
	MaxEntityBytes   = 64 << 10
	maxSnapshotBytes = 8 << 20
	maxSnapshotCount = 10000
	maxSafeInteger   = int64(9007199254740991)
)

var (
	ErrResnapshot       = errors.New("resnapshot_required")
	ErrSnapshotTooLarge = errors.New("snapshot_too_large")
)

type Store struct{ db *sql.DB }

// Open uses a dedicated database. Refusing unrelated tables avoids accidental
// writes to a desktop database passed as the sync service's -db argument.
func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("sync database path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	fail := func(err error) (*Store, error) { _ = db.Close(); return nil, err }
	var other int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master
		WHERE type='table' AND name NOT IN ('sync_meta','sync_entities','sync_changes','sync_mutations','sqlite_sequence')`).Scan(&other); err != nil {
		return fail(err)
	}
	if other != 0 {
		return fail(errors.New("sync service requires a dedicated database"))
	}
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return fail(err)
	}
	var epoch [16]byte
	if _, err := rand.Read(epoch[:]); err != nil {
		return fail(err)
	}
	err = store.Immediate(ctx, db, func(q store.Querier) error {
		_, err := q.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sync_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS sync_entities (
		  type TEXT NOT NULL, id TEXT NOT NULL, version INTEGER NOT NULL,
		  deleted INTEGER NOT NULL, data BLOB, PRIMARY KEY(type,id));
		CREATE TABLE IF NOT EXISTS sync_changes (
		  seq INTEGER PRIMARY KEY AUTOINCREMENT, record BLOB NOT NULL);
		CREATE TABLE IF NOT EXISTS sync_mutations (
		  id TEXT PRIMARY KEY, hash TEXT NOT NULL, result BLOB NOT NULL);`)
		if err != nil {
			return err
		}
		_, err = q.ExecContext(ctx, "INSERT OR IGNORE INTO sync_meta(key,value) VALUES('epoch',?)", hex.EncodeToString(epoch[:]))
		return err
	})
	if err != nil {
		return fail(err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// ResetEpoch must be called after restoring a backup, before accepting clients.
// It leaves records and receipts intact while forcing every device to snapshot.
func (s *Store) ResetEpoch(ctx context.Context) error {
	var epoch [16]byte
	if _, err := rand.Read(epoch[:]); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE sync_meta SET value=? WHERE key='epoch'", hex.EncodeToString(epoch[:]))
	return err
}

func getRecord(ctx context.Context, q store.Querier, kind, id string) (*Record, error) {
	r := &Record{Type: kind, ID: id}
	err := q.QueryRowContext(ctx, "SELECT version,deleted,data FROM sync_entities WHERE type=? AND id=?", kind, id).Scan(&r.Version, &r.Deleted, &r.Data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// Push commits each independent mutation and its durable receipt together.
// A dropped HTTP response is recovered by replaying the identical mutation ID.
func (s *Store) Push(ctx context.Context, request PushRequest) (PushResponse, error) {
	out := PushResponse{Results: make([]PushResult, 0, len(request.Mutations))}
	for _, m := range request.Mutations {
		var result PushResult
		err := store.Immediate(ctx, s.db, func(q store.Querier) error {
			payload, err := json.Marshal(struct {
				DeviceID string   `json:"deviceId"`
				Mutation Mutation `json:"mutation"`
			}{request.DeviceID, m})
			if err != nil {
				return err
			}
			digest := sha256.Sum256(payload)
			hash := hex.EncodeToString(digest[:])
			var oldHash string
			var receipt []byte
			err = q.QueryRowContext(ctx, "SELECT hash,result FROM sync_mutations WHERE id=?", m.ID).Scan(&oldHash, &receipt)
			if err == nil {
				if hash != oldHash {
					result = PushResult{MutationID: m.ID, Status: "rejected", Error: "mutation_payload_mismatch"}
					return nil
				}
				return json.Unmarshal(receipt, &result)
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			result, err = applyMutation(ctx, q, m)
			if err != nil {
				return err
			}
			receipt, err = json.Marshal(result)
			if err != nil {
				return err
			}
			_, err = q.ExecContext(ctx, "INSERT INTO sync_mutations(id,hash,result) VALUES(?,?,?)", m.ID, hash, receipt)
			return err
		})
		if err != nil {
			return PushResponse{}, err
		}
		out.Results = append(out.Results, result)
	}
	return out, nil
}

func applyMutation(ctx context.Context, q store.Querier, m Mutation) (PushResult, error) {
	r := PushResult{MutationID: m.ID, Status: "rejected"}
	if !validID(m.ID) || !validID(m.EntityID) || !validType(m.Type) || m.BaseVersion < 0 || m.BaseVersion >= maxSafeInteger || len(m.Data) == 0 || len(m.Data) > MaxEntityBytes {
		r.Error = "invalid_mutation"
		return r, nil
	}
	cur, err := getRecord(ctx, q, m.Type, m.EntityID)
	if err != nil {
		return r, err
	}
	version := int64(0)
	if cur != nil {
		version = cur.Version
	}
	if m.BaseVersion != version {
		r.Status, r.Record = "conflict", cur
		return r, nil
	}
	deleted := bytes.Equal(bytes.TrimSpace(m.Data), []byte("null"))
	if m.Type == "report" && (deleted || cur != nil) {
		r.Error = "reports_are_append_only"
		return r, nil
	}
	if deleted && (cur == nil || cur.Deleted) {
		r.Error = "entity_not_found"
		return r, nil
	}
	code, err := validateMutation(ctx, q, m, cur, deleted)
	if err != nil {
		return r, err
	}
	if code != "" {
		r.Error = code
		return r, nil
	}
	next := Record{Type: m.Type, ID: m.EntityID, Version: version + 1, Deleted: deleted, Data: m.Data}
	_, err = q.ExecContext(ctx, `INSERT INTO sync_entities(type,id,version,deleted,data) VALUES(?,?,?,?,?)
		ON CONFLICT(type,id) DO UPDATE SET version=excluded.version,deleted=excluded.deleted,data=excluded.data`, next.Type, next.ID, next.Version, next.Deleted, []byte(next.Data))
	if err != nil {
		return r, err
	}
	encoded, err := json.Marshal(next)
	if err != nil {
		return r, err
	}
	res, err := q.ExecContext(ctx, "INSERT INTO sync_changes(record) VALUES(?)", encoded)
	if err != nil {
		return r, err
	}
	seq, err := res.LastInsertId()
	if err != nil {
		return r, err
	}
	if seq > maxSafeInteger {
		return r, errors.New("sync sequence exhausted")
	}
	r.Status, r.Record = "accepted", &next
	return r, nil
}

func readHead(ctx context.Context, q store.Querier) (string, int64, error) {
	var epoch string
	var cursor int64
	if err := q.QueryRowContext(ctx, "SELECT value FROM sync_meta WHERE key='epoch'").Scan(&epoch); err != nil {
		return "", 0, err
	}
	err := q.QueryRowContext(ctx, "SELECT coalesce(max(seq),0) FROM sync_changes").Scan(&cursor)
	return epoch, cursor, err
}

func (s *Store) Pull(ctx context.Context, cursor int64, limit int, knownEpoch string) (PullResponse, error) {
	out := PullResponse{Changes: []Change{}, Cursor: cursor}
	if cursor < 0 || cursor > maxSafeInteger || limit < 1 || limit > MaxPullLimit {
		return out, errors.New("invalid pull parameters")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	epoch, head, err := readHead(ctx, tx)
	if err != nil {
		return out, err
	}
	if (knownEpoch != "" && knownEpoch != epoch) || cursor > head {
		return out, ErrResnapshot
	}
	out.Epoch = epoch
	rows, err := tx.QueryContext(ctx, "SELECT seq,record FROM sync_changes WHERE seq>? ORDER BY seq LIMIT ?", cursor, limit)
	if err != nil {
		return out, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var c Change
		var raw []byte
		if err := rows.Scan(&c.Seq, &raw); err != nil {
			return out, err
		}
		if err := json.Unmarshal(raw, &c.Record); err != nil {
			return out, err
		}
		out.Changes, out.Cursor = append(out.Changes, c), c.Seq
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}
	out.HasMore = out.Cursor < head
	return out, tx.Commit()
}

func (s *Store) Snapshot(ctx context.Context) (SnapshotResponse, error) {
	out := SnapshotResponse{Records: []Record{}}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	out.Epoch, out.Cursor, err = readHead(ctx, tx)
	if err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT type,id,version,deleted,data FROM sync_entities ORDER BY type,id")
	if err != nil {
		return out, err
	}
	defer func() { _ = rows.Close() }()
	bytes := 0
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Type, &r.ID, &r.Version, &r.Deleted, &r.Data); err != nil {
			return out, err
		}
		bytes += len(r.Data) + 200
		if bytes > maxSnapshotBytes || len(out.Records) >= maxSnapshotCount {
			return out, ErrSnapshotTooLarge
		}
		out.Records = append(out.Records, r)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}
	return out, tx.Commit()
}
