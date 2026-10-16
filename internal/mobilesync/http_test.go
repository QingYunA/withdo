package mobilesync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testToken = "test-private-owner-secret"

func request(t *testing.T, handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("non-JSON response: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("sync data can be cached")
	}
	return w
}

func handler(t *testing.T, s *Store) http.Handler {
	t.Helper()
	h, err := NewHandler(s, testToken)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestHTTPRequiresExplicitSecretAndAuthenticatesAllSyncRoutes(t *testing.T) {
	s := openTestStore(t)
	for _, token := range []string{"", "  ", " spaced ", "line\nbreak"} {
		if _, err := NewHandler(s, token); err == nil {
			t.Fatalf("accepted bad token %q", token)
		}
	}
	h := handler(t, s)
	if w := request(t, h, "GET", "/healthz", "", ""); w.Code != 200 || strings.Contains(w.Body.String(), testToken) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, route := range []struct{ method, path string }{{"POST", "/v1/sync/push"}, {"GET", "/v1/sync/pull?cursor=0"}, {"GET", "/v1/sync/snapshot"}} {
		for _, token := range []string{"", "wrong"} {
			w := request(t, h, route.method, route.path, "{}", token)
			if w.Code != 401 || !strings.Contains(w.Body.String(), "unauthorized") || strings.Contains(w.Body.String(), testToken) {
				t.Fatal(w.Code, w.Body.String())
			}
		}
	}
}

func TestHTTPRejectMalformedOversizeAndUnboundedRequests(t *testing.T) {
	h := handler(t, openTestStore(t))
	for _, body := range []string{
		"{", "null", `{}`, `{"deviceId":"42","mutations":[]}`,
		`{"deviceId":"` + uid(1) + `","mutations":null}`,
		`{"deviceId":"` + uid(1) + `","mutations":[],"unknown":true}`,
		`{"deviceId":"` + uid(1) + `","mutations":[]} {}`,
	} {
		if w := request(t, h, "POST", "/v1/sync/push", body, testToken); w.Code != 400 {
			t.Fatalf("body %s: %d %s", body, w.Code, w.Body.String())
		}
	}
	w := request(t, h, "POST", "/v1/sync/push", strings.Repeat(" ", MaxRequestBytes+1), testToken)
	if w.Code != 413 || !strings.Contains(w.Body.String(), "request_too_large") {
		t.Fatal(w.Code, w.Body.String())
	}
	batch := PushRequest{DeviceID: uid(1), Mutations: make([]Mutation, MaxMutations+1)}
	raw, _ := json.Marshal(batch)
	if w := request(t, h, "POST", "/v1/sync/push", string(raw), testToken); w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, query := range []string{"", "?cursor=-1", "?cursor=no", "?cursor=0&limit=0", "?cursor=0&limit=201", "?cursor=9007199254740992", "?cursor=0&limit=oops"} {
		if w := request(t, h, "GET", "/v1/sync/pull"+query, "", testToken); w.Code != 400 {
			t.Fatalf("query %s: %d %s", query, w.Code, w.Body.String())
		}
	}
}

func TestHTTPJSONContractRoundTrip(t *testing.T) {
	h := handler(t, openTestStore(t))
	m := change("task", uid(1), 0, sampleTask(1))
	body, _ := json.Marshal(PushRequest{DeviceID: uid(500), Mutations: []Mutation{m}})
	first := request(t, h, "POST", "/v1/sync/push", string(body), testToken)
	if first.Code != 200 {
		t.Fatal(first.Code, first.Body.String())
	}
	var pushed PushResponse
	if err := json.Unmarshal(first.Body.Bytes(), &pushed); err != nil || len(pushed.Results) != 1 {
		t.Fatalf("%s %v", first.Body.String(), err)
	}
	wantStatus(t, pushed.Results[0], "accepted", "")
	retried := request(t, h, "POST", "/v1/sync/push", string(body), testToken)
	if retried.Body.String() != first.Body.String() {
		t.Fatal("HTTP retry changed receipt")
	}
	snapshot := request(t, h, "GET", "/v1/sync/snapshot", "", testToken)
	var snap SnapshotResponse
	if snapshot.Code != 200 || json.Unmarshal(snapshot.Body.Bytes(), &snap) != nil || snap.Cursor != 1 || len(snap.Records) != 1 || snap.Records[0].ID != m.EntityID || snap.Epoch == "" {
		t.Fatal(snapshot.Code, snapshot.Body.String())
	}
	delta := request(t, h, "GET", "/v1/sync/pull?cursor=0&limit=1&epoch="+snap.Epoch, "", testToken)
	var pulled PullResponse
	if delta.Code != 200 || json.Unmarshal(delta.Body.Bytes(), &pulled) != nil || pulled.Cursor != snap.Cursor || len(pulled.Changes) != 1 || pulled.HasMore {
		t.Fatal(delta.Code, delta.Body.String())
	}
	for _, query := range []string{"?cursor=0&epoch=old", "?cursor=2"} {
		w := request(t, h, "GET", "/v1/sync/pull"+query, "", testToken)
		if w.Code != 409 || w.Body.String() != "{\"error\":\"resnapshot_required\"}\n" {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w := request(t, h, "GET", "/unknown", "", testToken); w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestHTTPInternalErrorsDoNotLeakDetails(t *testing.T) {
	s := openTestStore(t)
	h := handler(t, s)
	_ = s.Close()
	w := request(t, h, "GET", "/v1/sync/snapshot", "", testToken)
	if w.Code != 500 || w.Body.String() != "{\"error\":\"internal_error\"}\n" {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/healthz", "", "")
	if w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
}
