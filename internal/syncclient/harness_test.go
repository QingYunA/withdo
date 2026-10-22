package syncclient_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

const (
	testToken  = "s2b-token"
	otherToken = "s2b-other-token"
)

var testTokens = syncserver.StaticTokens{
	testToken:  "s2b-subject",
	otherToken: "s2b-other-subject",
}

var testCtx = context.Background()

var schemaCounter atomic.Uint64

// syncedEntities is the order the tests compare the five synced tables in.
var syncedEntities = []string{
	syncproto.EntityProjects,
	syncproto.EntityCategories,
	syncproto.EntityMilestones,
	syncproto.EntityTasks,
	syncproto.EntityReports,
}

// harness is one S2a server on a fresh Postgres schema, shared by the devices of one test.
type harness struct {
	server *syncserver.Server
	http   *httptest.Server
	// admin is a Postgres connection with the test's search_path, for reading the server's rows.
	admin *pgx.Conn
	// databaseURL is the URL of the test's schema, for a second server over the same data.
	databaseURL string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	// These run whole devices against Postgres for a minute; -short leaves them to the race pass
	// of check.sh full, so they do not slow the quick pass alongside the UI tests.
	if testing.Short() {
		t.Skip("integration test against Postgres; skipped with -short")
	}
	databaseURL := os.Getenv("ADHD_SYNC_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ADHD_SYNC_TEST_DATABASE_URL is not set")
	}
	ctx := testCtx
	schema := fmt.Sprintf("s2b_%d_%d", os.Getpid(), schemaCounter.Add(1))
	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Postgres: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create schema: %v", err)
	}
	if _, err := admin.Exec(ctx, "SET search_path TO "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		_ = admin.Close(ctx)
		t.Fatalf("set search_path: %v", err)
	}
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	server, err := syncserver.Open(ctx, u.String(), testTokens)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		_ = admin.Close(ctx)
		t.Fatalf("open sync server: %v", err)
	}
	h := &harness{server: server, http: httptest.NewServer(server.Handler()), admin: admin, databaseURL: u.String()}
	t.Cleanup(func() {
		h.http.Close()
		server.Close()
		if _, err := admin.Exec(context.Background(),
			"DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		_ = admin.Close(context.Background())
	})
	return h
}

// device is one local database and its client on a harness.
type device struct {
	db     *sql.DB
	path   string
	core   *domain.Core
	client *syncclient.Client
}

// newDevice opens a fresh migrated SQLite database for the test. httpClient may be nil.
func (h *harness) newDevice(t *testing.T, httpClient *http.Client) *device {
	t.Helper()
	return h.newDeviceWithToken(t, httpClient, testToken)
}

// newDeviceWithToken opens a device for another account, whose data must never show up in the
// first account's workspace.
func (h *harness) newDeviceWithToken(t *testing.T, httpClient *http.Client, token string) *device {
	t.Helper()
	db, path := testutil.OpenDB(t)
	core := domain.New(db, domain.Options{})
	client, err := syncclient.New(core, syncclient.Options{
		BaseURL: h.http.URL,
		Token:   token,
		HTTP:    httpClient,
	})
	if err != nil {
		t.Fatalf("new sync client: %v", err)
	}
	return &device{db: db, path: path, core: core, client: client}
}

func (d *device) link(t *testing.T, name string) {
	t.Helper()
	if err := d.client.Link(testCtx, name); err != nil {
		t.Fatalf("link %s: %v", name, err)
	}
}

func (d *device) sync(t *testing.T) {
	t.Helper()
	if err := d.client.Sync(testCtx); err != nil {
		t.Fatalf("sync: %v", err)
	}
}

// assertSyncedEqual checks every synced column of every row of the five tables. Device-local
// columns are not compared because they never leave the device.
func assertSyncedEqual(t *testing.T, a, b *device) {
	t.Helper()
	for _, entity := range syncedEntities {
		want := dumpTable(t, a.db, entity)
		got := dumpTable(t, b.db, entity)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s differs\nA: %s\nB: %s", entity, renderRows(want), renderRows(got))
		}
	}
}

// dumpTable reads every row of entity with its id and synced columns, oldest id first.
func dumpTable(t *testing.T, db *sql.DB, entity string) []map[string]any {
	t.Helper()
	columns := append([]string{"id"}, syncproto.Columns[entity]...)
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = `"` + c + `"`
	}
	rows, err := db.QueryContext(testCtx,
		"SELECT "+strings.Join(quoted, ", ")+" FROM "+`"`+entity+`"`+" ORDER BY id")
	if err != nil {
		t.Fatalf("read %s: %v", entity, err)
	}
	defer func() { _ = rows.Close() }()
	out := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			t.Fatalf("scan %s: %v", entity, err)
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			value := values[i]
			if raw, ok := value.([]byte); ok {
				value = string(raw)
			}
			row[column] = value
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read %s: %v", entity, err)
	}
	return out
}

func renderRows(rows []map[string]any) string {
	raw, err := json.Marshal(rows)
	if err != nil {
		return fmt.Sprintf("%v", rows)
	}
	return string(raw)
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(testCtx, query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func str(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var s string
	if err := db.QueryRowContext(testCtx, query, args...).Scan(&s); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return s
}

func int64Value(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRowContext(testCtx, query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(testCtx, query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

func must[T any](t *testing.T) func(T, error) T {
	t.Helper()
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func ptr[T any](v T) *T { return &v }

func newProject(t *testing.T, c *domain.Core, name string, parent *int64) *domain.Project {
	t.Helper()
	return must[*domain.Project](t)(c.CreateProject(testCtx, domain.CreateProjectInput{Name: name, ParentID: parent}))
}

func newTask(t *testing.T, c *domain.Core, in domain.CreateTaskInput) *domain.Task {
	t.Helper()
	return must[*domain.Task](t)(c.CreateTask(testCtx, in))
}

// taskInput is a plain task with a title, for tests that do not care about the rest.
func taskInput(title string) domain.CreateTaskInput {
	return domain.CreateTaskInput{Title: title}
}

// dotIcon is a 12 x 12 category icon with one pixel set.
var dotIcon = strings.Repeat(".", 143) + "#"

// post sends one authenticated request to the server for tests that drive the raw API.
func (h *harness) post(t *testing.T, path string, body, out any) int {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(testCtx, http.MethodPost, h.http.URL+path, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testToken)
	resp, err := h.http.Client().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read POST %s: %v", path, err)
	}
	if out != nil && resp.StatusCode/100 == 2 {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if err := dec.Decode(out); err != nil {
			t.Fatalf("decode POST %s: %v", path, err)
		}
	}
	return resp.StatusCode
}

// serverRows returns the server's stored rows of entity, including tombstones. Numbers are decoded
// with UseNumber and turned into int64, so they compare with what SQLite gives a device.
func (h *harness) serverRows(t *testing.T, entity string) map[int64]serverRow {
	t.Helper()
	rows, err := h.admin.Query(testCtx, `SELECT row_id, deleted, data::text FROM rows
		WHERE workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1) AND entity = $1`, entity)
	if err != nil {
		t.Fatalf("read server %s rows: %v", entity, err)
	}
	defer rows.Close()
	out := map[int64]serverRow{}
	for rows.Next() {
		var id int64
		var deleted bool
		var raw *string
		if err := rows.Scan(&id, &deleted, &raw); err != nil {
			t.Fatalf("scan server %s row: %v", entity, err)
		}
		row := serverRow{Deleted: deleted, Data: map[string]any{}}
		if raw != nil {
			dec := json.NewDecoder(strings.NewReader(*raw))
			dec.UseNumber()
			if err := dec.Decode(&row.Data); err != nil {
				t.Fatalf("decode server %s row %d: %v", entity, id, err)
			}
		}
		out[id] = row
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read server %s rows: %v", entity, err)
	}
	return out
}

// serverRow is one row of the server's rows table.
type serverRow struct {
	Deleted bool
	Data    map[string]any
}

// serverSeedingDevices counts the devices that are still seeding the test's workspace.
func (h *harness) serverSeedingDevices(t *testing.T) int {
	t.Helper()
	var n int
	if err := h.admin.QueryRow(testCtx, `SELECT COUNT(*) FROM devices
		WHERE workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1) AND seeding`).Scan(&n); err != nil {
		t.Fatalf("count seeding devices: %v", err)
	}
	return n
}

// serverIDCounter is the first id the workspace has not handed out for entity. A merge leases its
// fresh ids from here, so every renumbered local row is at or above it.
func (h *harness) serverIDCounter(t *testing.T, entity string) int64 {
	t.Helper()
	var next int64
	if err := h.admin.QueryRow(testCtx, `SELECT next FROM id_counters
		WHERE workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1) AND entity = $1`, entity).
		Scan(&next); err != nil {
		t.Fatalf("read %s id counter: %v", entity, err)
	}
	return next
}

// assertDeviceMatchesServer checks every live server row of every synced table against the device:
// the same ids, no missing row, no extra row, and the same value in every synced column.
func assertDeviceMatchesServer(t *testing.T, h *harness, d *device) {
	t.Helper()
	for _, entity := range syncedEntities {
		server := h.serverRows(t, entity)
		live := map[int64]map[string]any{}
		for id, row := range server {
			if !row.Deleted {
				live[id] = row.Data
			}
		}
		deviceRows := dumpTable(t, d.db, entity)
		if len(deviceRows) != len(live) {
			t.Errorf("%s: device has %d rows, server has %d live rows", entity, len(deviceRows), len(live))
			continue
		}
		for _, row := range deviceRows {
			id, ok := row["id"].(int64)
			if !ok {
				t.Fatalf("%s: device row id is %T", entity, row["id"])
			}
			data, ok := live[id]
			if !ok {
				t.Errorf("%s: device has row %d the server does not", entity, id)
				continue
			}
			for _, column := range syncproto.Columns[entity] {
				want, ok := data[column]
				if !ok {
					t.Errorf("%s %d: server has no column %s", entity, id, column)
					continue
				}
				want = serverValue(want)
				if got := row[column]; got != want {
					t.Errorf("%s %d %s = %v (%T), server has %v (%T)", entity, id, column, got, got, want, want)
				}
			}
		}
	}
}

// serverValue turns a JSON value from Postgres into the Go value SQLite returns.
func serverValue(value any) any {
	if n, ok := value.(json.Number); ok {
		if i, err := n.Int64(); err == nil {
			return i
		}
		if f, err := n.Float64(); err == nil {
			return f
		}
	}
	return value
}
