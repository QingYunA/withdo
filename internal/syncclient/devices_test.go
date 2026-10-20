package syncclient_test

import (
	"testing"
	"time"
)

// TestS3UIa1Devices is S3-UI-a acceptance criterion 1: two devices push and pull, GET /v1/devices
// lists both with a last_seen_at that moves on each sync, newest first, and never another user's
// device.
func TestS3UIa1Devices(t *testing.T) {
	h := newHarness(t)
	a := h.newDevice(t, nil)
	newTask(t, a.core, taskInput("A's task"))
	a.link(t, "A")
	aID, err := a.client.DeviceID(testCtx)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := a.client.Devices(testCtx)
	if err != nil {
		t.Fatalf("devices after A linked: %v", err)
	}
	if len(devices) != 1 || devices[0].ID != aID || devices[0].Name != "A" {
		t.Fatalf("devices = %+v, want A only", devices)
	}
	if devices[0].LastSeenAt == nil {
		t.Fatal("A has no last_seen_at after linking")
	}
	first := *devices[0].LastSeenAt

	// A second sync moves A's last_seen_at. The sleep only makes the Postgres clock tick visible.
	time.Sleep(2 * time.Millisecond)
	a.sync(t)
	devices, err = a.client.Devices(testCtx)
	if err != nil {
		t.Fatalf("devices after A synced: %v", err)
	}
	if len(devices) != 1 || devices[0].ID != aID {
		t.Fatalf("devices = %+v, want A only", devices)
	}
	if devices[0].LastSeenAt == nil || !devices[0].LastSeenAt.After(first) {
		t.Fatalf("A's last_seen_at = %v, want later than %v", devices[0].LastSeenAt, first)
	}
	second := *devices[0].LastSeenAt

	b := h.newDevice(t, nil)
	b.link(t, "B")
	bID, err := b.client.DeviceID(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	b.sync(t)
	devices, err = a.client.Devices(testCtx)
	if err != nil {
		t.Fatalf("devices after B synced: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("devices = %+v, want A and B", devices)
	}
	// B synced last, so it comes first.
	if devices[0].ID != bID || devices[0].Name != "B" {
		t.Errorf("first device = %+v, want B (newest first)", devices[0])
	}
	if devices[0].LastSeenAt == nil {
		t.Error("B has no last_seen_at")
	}
	for _, device := range devices {
		if device.ID == aID && (device.LastSeenAt == nil || !device.LastSeenAt.Equal(second)) {
			t.Errorf("A's last_seen_at = %v, want unchanged at %v", device.LastSeenAt, second)
		}
	}

	// A's next sync moves its own timestamp again and puts it first.
	time.Sleep(2 * time.Millisecond)
	a.sync(t)
	devices, err = a.client.Devices(testCtx)
	if err != nil {
		t.Fatalf("devices after A synced again: %v", err)
	}
	if len(devices) != 2 || devices[0].ID != aID {
		t.Fatalf("devices = %+v, want A first", devices)
	}
	if devices[0].LastSeenAt == nil || !devices[0].LastSeenAt.After(second) {
		t.Errorf("A's last_seen_at = %v, want later than %v", devices[0].LastSeenAt, second)
	}

	// A device of another user is not in A's list.
	c := h.newDeviceWithToken(t, nil, otherToken)
	c.link(t, "C")
	c.sync(t)
	cID, err := c.client.DeviceID(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	devices, err = a.client.Devices(testCtx)
	if err != nil {
		t.Fatalf("devices with a second user: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("devices = %+v, want only A and B", devices)
	}
	for _, device := range devices {
		if device.ID == cID {
			t.Fatalf("A's list has another user's device %s", cID)
		}
	}
}
