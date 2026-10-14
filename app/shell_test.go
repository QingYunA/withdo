package main

import "testing"

func TestNotificationIDsRoundTrip(t *testing.T) {
	id := notificationID(42)
	if got, ok := taskFromNotification(id); !ok || got != 42 {
		t.Errorf("%q -> %d, %v", id, got, ok)
	}
	for _, bad := range []string{"", "task/", "task/0", "task/x", "other/5"} {
		if _, ok := taskFromNotification(bad); ok {
			t.Errorf("%q accepted", bad)
		}
	}
}
