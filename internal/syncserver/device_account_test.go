package syncserver_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// TestForeignDeviceCode: a device id of another account is a 403 with a machine-readable code on
// every route that takes a device, and the old "error" text is still there for older clients.
// A 403 that means something else (the device is not seeding) has no such code.
func TestForeignDeviceCode(t *testing.T) {
	h := newHarness(t)
	alice := h.device(t, aliceToken, "alice-laptop")
	bob := h.device(t, bobToken, "bob-phone")

	check := func(what, method, path string, body any) {
		t.Helper()
		resp, data := h.do(t, method, path, bobToken, body)
		requireStatus(t, what, resp.StatusCode, http.StatusForbidden)
		var out map[string]string
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", what, data, err)
		}
		if out["code"] != "device_not_in_account" || out["error"] != "device does not belong to this user" {
			t.Fatalf("%s: body = %v, want code device_not_in_account and the old error text", what, out)
		}
	}
	check("push", http.MethodPost, "/v1/push", syncproto.PushRequest{DeviceID: alice.DeviceID})
	check("pull", http.MethodGet, fmt.Sprintf("/v1/pull?device_id=%s&cookie=0&limit=10", alice.DeviceID), nil)
	check("events", http.MethodGet, "/v1/events?device_id="+alice.DeviceID, nil)
	check("ids", http.MethodPost, "/v1/ids", syncproto.IDsRequest{DeviceID: alice.DeviceID, Entity: syncproto.EntityTasks, Count: 1})
	check("seeded", http.MethodPost, "/v1/devices/seeded", syncproto.SeededRequest{DeviceID: alice.DeviceID})
	check("unknown device", http.MethodGet, "/v1/events?device_id=00000000-0000-4000-8000-000000000000", nil)

	// bob's own device is no longer seeding after the first call: a 403 with no code.
	h.seeded(t, bobToken, bob.DeviceID)
	resp, data := h.do(t, http.MethodPost, "/v1/devices/seeded", bobToken, syncproto.SeededRequest{DeviceID: bob.DeviceID})
	requireStatus(t, "not seeding", resp.StatusCode, http.StatusForbidden)
	var out map[string]string
	_ = json.Unmarshal(data, &out)
	if out["code"] != "" {
		t.Fatalf("not-seeding 403 carries code %q", out["code"])
	}
}
