package core

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// TestDevicesGolden pins the device list's JSON shape: syncproto.Device's fields, a null last seen
// time before the first sync, and "this" on this phone.
func TestDevicesGolden(t *testing.T) {
	seen := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	raw, err := encodeDevices([]syncclient.Device{
		{ID: "d1", Name: "Pixel", LastSeenAt: &seen},
		{ID: "d2", Name: "Mac"},
	}, "d1")
	if err != nil {
		t.Fatalf("encodeDevices: %v", err)
	}
	checkGolden(t, "devices.golden.json", raw)
}
