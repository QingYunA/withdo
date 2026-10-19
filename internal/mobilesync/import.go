package mobilesync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ImportIssue struct {
	Type     string `json:"type"`
	EntityID string `json:"entityId"`
	Status   string `json:"status"`
	Error    string `json:"error"`
}

type ImportSummary struct {
	Accepted  int           `json:"accepted"`
	Conflicts int           `json:"conflicts"`
	Rejected  int           `json:"rejected"`
	Issues    []ImportIssue `json:"issues"`
}

// importID is a namespaced, deterministic UUIDv8. Explicit source identity makes
// the same desktop integer ID distinct across independently created databases.
func importID(source string, parts ...string) string {
	encoded, _ := json.Marshal(append([]string{"adhd-todo/desktop-import/v1", source}, parts...))
	h := sha256.Sum256(encoded)
	h[6], h[8] = h[6]&0x0f|0x80, h[8]&0x3f|0x80
	s := hex.EncodeToString(h[:16])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}

func sourceEntityID(source, kind string, id int64) string {
	return importID(source, kind, strconv.FormatInt(id, 10))
}

// ImportDesktop reads a checkpointed desktop snapshot without opening it for
// writing, then uses the public push API. Every item starts at baseVersion 0;
// later source changes conflict instead of replacing edits made on a phone.
func ImportDesktop(ctx context.Context, source, sourceID, serverURL, token string) (ImportSummary, error) {
	summary := ImportSummary{Issues: []ImportIssue{}}
	if strings.TrimSpace(sourceID) == "" || strings.TrimSpace(sourceID) != sourceID || len(sourceID) > 200 {
		return summary, errors.New("a stable nonempty -source-id of at most 200 bytes is required")
	}
	if strings.TrimSpace(token) == "" || strings.TrimSpace(token) != token || strings.ContainsAny(token, "\r\n") {
		return summary, errors.New("a nonempty bearer token is required")
	}
	endpoint, err := url.Parse(serverURL)
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return summary, errors.New("-url must be an HTTP(S) server base URL without credentials, query or fragment")
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/v1/sync/push"
	endpoint.RawPath = ""
	mutations, err := desktopMutations(ctx, source, sourceID)
	if err != nil {
		return summary, err
	}
	deviceID := importID(sourceID, "device")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for at := 0; at < len(mutations); {
		batch := PushRequest{DeviceID: deviceID, Mutations: []Mutation{}}
		base, _ := json.Marshal(batch)
		size := len(base)
		for at < len(mutations) && len(batch.Mutations) < MaxMutations {
			raw, _ := json.Marshal(mutations[at])
			if size+len(raw)+1 > MaxRequestBytes {
				break
			}
			batch.Mutations = append(batch.Mutations, mutations[at])
			size += len(raw) + 1
			at++
		}
		if len(batch.Mutations) == 0 {
			return summary, errors.New("source record exceeds sync request limit")
		}
		body, _ := json.Marshal(batch)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
		if err != nil {
			return summary, errors.New("could not create sync request")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if err != nil {
			return summary, fmt.Errorf("sync request failed: %w", err)
		}
		raw, readErr := io.ReadAll(io.LimitReader(response.Body, 2*MaxRequestBytes+4097))
		_ = response.Body.Close()
		if readErr != nil || len(raw) > 2*MaxRequestBytes+4096 {
			return summary, errors.New("invalid or oversized sync response")
		}
		if response.StatusCode != http.StatusOK {
			return summary, fmt.Errorf("sync server returned HTTP %d", response.StatusCode)
		}
		var result PushResponse
		if err := decodeStrict(raw, &result); err != nil || len(result.Results) != len(batch.Mutations) {
			return summary, errors.New("invalid sync result")
		}
		for i, r := range result.Results {
			m := batch.Mutations[i]
			if r.MutationID != m.ID {
				return summary, errors.New("sync result mutation ID mismatch")
			}
			switch r.Status {
			case "accepted":
				summary.Accepted++
			case "conflict", "rejected":
				if r.Status == "conflict" {
					summary.Conflicts++
				} else {
					summary.Rejected++
				}
				summary.Issues = append(summary.Issues, ImportIssue{Type: m.Type, EntityID: m.EntityID, Status: r.Status, Error: r.Error})
			default:
				return summary, errors.New("unknown sync result status")
			}
		}
	}
	return summary, nil
}

func desktopMutations(ctx context.Context, path, sourceID string) ([]Mutation, error) {
	if path == "" {
		return nil, errors.New("-source is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	before, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("source must be a checkpointed SQLite file")
	}
	if err := requireCheckpointed(abs); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	q := url.Values{"mode": {"ro"}, "immutable": {"1"}}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("open read-only desktop snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	mutations := []Mutation{}
	bytes := 0
	appendEntity := func(kind, id string, value any) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		bytes += len(data) + 200
		if len(mutations) >= maxSnapshotCount || bytes > maxSnapshotBytes {
			return errors.New("desktop snapshot exceeds initial mobile snapshot capacity")
		}
		digest := sha256.Sum256(data)
		mutations = append(mutations, Mutation{ID: importID(sourceID, "mutation", kind, id, hex.EncodeToString(digest[:])), Type: kind, EntityID: id, Data: data})
		return nil
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,name,coalesce(repo_slug,''),archived_at,created_at,updated_at FROM projects ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read desktop projects: %w", err)
	}
	projects := map[int64]bool{}
	for rows.Next() {
		var p project
		var id int64
		if err := rows.Scan(&id, &p.Name, &p.RepoSlug, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		p.ID, projects[id] = sourceEntityID(sourceID, "project", id), true
		if err := appendEntity("project", p.ID, p); err != nil {
			_ = rows.Close()
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	// The first three desktop schema versions predate parent_id. They remain
	// readable without applying migration 004 to the user's source database.
	var hasParent int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM pragma_table_info('tasks') WHERE name='parent_id'").Scan(&hasParent); err != nil {
		return nil, err
	}
	parentColumn := "NULL"
	if hasParent != 0 {
		parentColumn = "parent_id"
	}
	rows, err = tx.QueryContext(ctx, `SELECT id,project_id,`+parentColumn+`,title,coalesce(notes,''),status,priority,source,
		due_at,remind_at,coalesce(branch,''),coalesce(pr_url,''),created_at,updated_at,completed_at FROM tasks ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("read desktop tasks: %w", err)
	}
	type sourceTask struct {
		data   task
		parent *int64
	}
	tasks, order := map[int64]sourceTask{}, []int64{}
	taskBytes := 0
	for rows.Next() {
		var t task
		var id int64
		var projectID, parentID *int64
		if err := rows.Scan(&id, &projectID, &parentID, &t.Title, &t.Notes, &t.Status, &t.Priority, &t.Source, &t.DueAt, &t.RemindAt, &t.Branch, &t.PRURL, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		t.ID = sourceEntityID(sourceID, "task", id)
		if projectID != nil {
			if !projects[*projectID] {
				_ = rows.Close()
				return nil, errors.New("desktop task references a missing project")
			}
			mapped := sourceEntityID(sourceID, "project", *projectID)
			t.ProjectID = &mapped
		}
		if parentID != nil {
			mapped := sourceEntityID(sourceID, "task", *parentID)
			t.ParentID = &mapped
		}
		tasks[id], order = sourceTask{t, parentID}, append(order, id)
		taskBytes += len(t.Title) + len(t.Notes) + len(t.Branch) + len(t.PRURL) + 500
		if len(tasks)+len(mutations) > maxSnapshotCount || taskBytes+bytes > maxSnapshotBytes {
			_ = rows.Close()
			return nil, errors.New("desktop snapshot exceeds initial mobile snapshot capacity")
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	visited := map[int64]int{}
	var visit func(int64, int) error
	visit = func(id int64, depth int) error {
		if visited[id] == 2 {
			return nil
		}
		t, ok := tasks[id]
		if !ok || visited[id] == 1 || depth > 100 {
			return errors.New("desktop snapshot has invalid task ancestry")
		}
		visited[id] = 1
		if t.parent != nil {
			if err := visit(*t.parent, depth+1); err != nil {
				return err
			}
		}
		visited[id] = 2
		return appendEntity("task", t.data.ID, t.data)
	}
	for _, id := range order {
		if err := visit(id, 0); err != nil {
			return nil, err
		}
	}
	rows, err = tx.QueryContext(ctx, "SELECT id,task_id,agent,status,summary,coalesce(next_step,''),created_at FROM reports ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read desktop reports: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var r report
		var id, taskID int64
		if err := rows.Scan(&id, &taskID, &r.Agent, &r.Status, &r.Summary, &r.NextStep, &r.CreatedAt); err != nil {
			return nil, err
		}
		if _, ok := tasks[taskID]; !ok {
			return nil, errors.New("desktop report references a missing task")
		}
		r.ID, r.TaskID = sourceEntityID(sourceID, "report", id), sourceEntityID(sourceID, "task", taskID)
		if err := appendEntity("report", r.ID, r); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	after, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return nil, errors.New("source changed during import; use a stopped, checkpointed backup")
	}
	if err := requireCheckpointed(abs); err != nil {
		return nil, err
	}
	return mutations, tx.Commit()
}

func requireCheckpointed(path string) error {
	if wal, err := os.Stat(path + "-wal"); err == nil && wal.Size() != 0 {
		return errors.New("source has a WAL; use a checkpointed SQLite backup after stopping desktop/CLI/MCP writers")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
