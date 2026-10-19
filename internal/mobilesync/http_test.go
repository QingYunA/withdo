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

func TestHTTPRejectsAmbiguousAndCaseVariantIdentityFields(t *testing.T) {
	for _, kind := range []string{"project", "task", "report"} {
		t.Run(kind, func(t *testing.T) {
			s := openTestStore(t)
			h := handler(t, s)
			// A report needs a real task reference; keep it separate from the
			// entity whose inconsistent identity the request attempts to create.
			push(t, s, change("task", uid(100), 0, sampleTask(100)))
			var data any
			switch kind {
			case "project":
				data = sampleProject(3)
			case "task":
				data = sampleTask(3)
			case "report":
				data = report{ID: uid(3), TaskID: uid(100), Agent: "codex", Summary: "Progress", CreatedAt: stamp}
			}
			original := change(kind, uid(3), 0, data)
			for _, extraKey := range []string{"ID", "id"} {
				m := original
				m.ID = uid(100000 + mutationSequence.Add(1))
				// encoding/json used to validate ID=003 while persisting raw
				// data.id=004 when the later uppercase ID field was present.
				m.Data = []byte(strings.Replace(string(original.Data), `"id":"`+uid(3)+`"`, `"id":"`+uid(4)+`","`+extraKey+`":"`+uid(3)+`"`, 1))
				body, _ := json.Marshal(PushRequest{DeviceID: uid(500), Mutations: []Mutation{m}})
				w := request(t, h, "POST", "/v1/sync/push", string(body), testToken)
				if extraKey == "id" {
					if w.Code != 400 {
						t.Fatalf("duplicate identity accepted: %d %s", w.Code, w.Body.String())
					}
				} else {
					var result PushResponse
					if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || len(result.Results) != 1 {
						t.Fatal(w.Code, w.Body.String())
					}
					wantStatus(t, result.Results[0], "rejected", "invalid_"+kind)
				}
			}
			w := request(t, h, "GET", "/v1/sync/snapshot", "", testToken)
			var snapshot SnapshotResponse
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &snapshot) != nil || snapshot.Cursor != 1 || len(snapshot.Records) != 1 || snapshot.Records[0].ID != uid(100) {
				t.Fatalf("invalid identity persisted: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHTTPRequiresExactEnvelopeAndMutationKeys(t *testing.T) {
	h := handler(t, openTestStore(t))
	m := change("task", uid(1), 0, sampleTask(1))
	body, _ := json.Marshal(PushRequest{DeviceID: uid(500), Mutations: []Mutation{m}})
	for _, invalid := range []string{
		strings.Replace(string(body), `"deviceId":`, `"DeviceId":`, 1),
		strings.Replace(string(body), `"baseVersion":`, `"BaseVersion":`, 1),
		strings.Replace(string(body), `"entityId":`, `"entityID":`, 1),
		strings.Replace(string(body), `"deviceId":"`+uid(500)+`"`, `"deviceId":"`+uid(500)+`","deviceId":"`+uid(500)+`"`, 1),
	} {
		if w := request(t, h, "POST", "/v1/sync/push", invalid, testToken); w.Code != 400 {
			t.Fatalf("inexact fields accepted: %d %s", w.Code, w.Body.String())
		}
	}
	// Ordinary camelCase records still pass the same decoder.
	w := request(t, h, "POST", "/v1/sync/push", string(body), testToken)
	var result PushResponse
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || len(result.Results) != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	wantStatus(t, result.Results[0], "accepted", "")
}
