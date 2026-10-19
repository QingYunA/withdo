package syncserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

const (
	aliceToken   = "alice-token"
	aliceSubject = "alice-subject"
	bobToken     = "bob-token"
	bobSubject   = "bob-subject"
)

var testTokens = syncserver.StaticTokens{
	aliceToken: aliceSubject,
	bobToken:   bobSubject,
}

var schemaCounter atomic.Uint64

// harness is one sync server on a fresh Postgres schema, so tests can run in parallel.
type harness struct {
	server *syncserver.Server
	http   *httptest.Server
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	databaseURL := os.Getenv("ADHD_SYNC_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ADHD_SYNC_TEST_DATABASE_URL is not set")
	}
	t.Parallel()

	ctx := context.Background()
	schema := fmt.Sprintf("s2a_%d_%d", os.Getpid(), schemaCounter.Add(1))
	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Postgres: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create schema: %v", err)
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
	h := &harness{server: server, http: httptest.NewServer(server.Handler())}
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

func (h *harness) do(t *testing.T, method, path, token string, body any) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, h.http.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := h.http.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	data, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, path, err)
	}
	return resp, data
}

func (h *harness) post(t *testing.T, path, token string, body, out any) int {
	t.Helper()
	resp, data := h.do(t, http.MethodPost, path, token, body)
	if out != nil && resp.StatusCode/100 == 2 {
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("decode %s response %q: %v", path, data, err)
		}
	}
	return resp.StatusCode
}

func (h *harness) get(t *testing.T, path, token string, out any) int {
	t.Helper()
	resp, data := h.do(t, http.MethodGet, path, token, nil)
	if out != nil && resp.StatusCode/100 == 2 {
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("decode %s response %q: %v", path, data, err)
		}
	}
	return resp.StatusCode
}

func (h *harness) device(t *testing.T, token, name string) syncproto.CreateDeviceResponse {
	t.Helper()
	var out syncproto.CreateDeviceResponse
	if code := h.post(t, "/v1/devices", token,
		syncproto.CreateDeviceRequest{Name: name}, &out); code != http.StatusOK {
		t.Fatalf("POST /v1/devices = %d, want 200", code)
	}
	if out.DeviceID == "" {
		t.Fatal("device id is empty")
	}
	return out
}

func (h *harness) seeded(t *testing.T, token, deviceID string) int {
	t.Helper()
	var out syncproto.SeededResponse
	return h.post(t, "/v1/devices/seeded", token, syncproto.SeededRequest{DeviceID: deviceID}, &out)
}

func (h *harness) push(t *testing.T, token, deviceID string, mutations ...syncproto.Mutation) (int, syncproto.PushResponse) {
	t.Helper()
	var out syncproto.PushResponse
	code := h.post(t, "/v1/push", token,
		syncproto.PushRequest{DeviceID: deviceID, Mutations: mutations}, &out)
	return code, out
}

func (h *harness) ids(t *testing.T, token, deviceID, entity string, count int) (int, syncproto.IDsResponse) {
	t.Helper()
	var out syncproto.IDsResponse
	code := h.post(t, "/v1/ids", token,
		syncproto.IDsRequest{DeviceID: deviceID, Entity: entity, Count: count}, &out)
	return code, out
}

func (h *harness) pull(t *testing.T, token, deviceID string, cookie int64, limit int) (int, syncproto.PullResponse) {
	t.Helper()
	var out syncproto.PullResponse
	code := h.get(t, fmt.Sprintf("/v1/pull?device_id=%s&cookie=%d&limit=%d", deviceID, cookie, limit),
		token, &out)
	return code, out
}

// idsE is ids without t.Fatalf, so concurrent tests can call it from goroutines.
func (h *harness) idsE(token, deviceID, entity string, count int) (syncproto.IDsResponse, error) {
	var out syncproto.IDsResponse
	raw, err := json.Marshal(syncproto.IDsRequest{DeviceID: deviceID, Entity: entity, Count: count})
	if err != nil {
		return out, err
	}
	req, err := http.NewRequest(http.MethodPost, h.http.URL+"/v1/ids", bytes.NewReader(raw))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := h.http.Client().Do(req)
	if err != nil {
		return out, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return out, err
	}
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("POST /v1/ids = %d: %s", resp.StatusCode, data)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

func requireStatus(t *testing.T, what string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", what, got, want)
	}
}

func findChange(t *testing.T, changes []syncproto.Change, entity string, rowID int64) syncproto.Change {
	t.Helper()
	for _, c := range changes {
		if c.Entity == entity && c.RowID == rowID {
			return c
		}
	}
	t.Fatalf("no %s row %d in changes %+v", entity, rowID, changes)
	return syncproto.Change{}
}
