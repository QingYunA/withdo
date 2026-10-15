package agent

import (
	"testing"
	"time"
)

func TestZoneFrom(t *testing.T) {
	loadZone(t, "Asia/Shanghai") // skips without a tz database
	none := func() string { return "" }
	cases := []struct {
		name, tz string
		target   func() string
		want     string
	}{
		{"TZ wins", "Europe/Paris", func() string { return "/var/db/timezone/zoneinfo/Asia/Shanghai" }, "Europe/Paris"},
		{"TZ with a colon", ":Europe/Paris", none, "Europe/Paris"},
		{"symlink target", "", func() string { return "/var/db/timezone/zoneinfo/Asia/Shanghai" }, "Asia/Shanghai"},
		{"bad TZ falls back to the symlink", "Not/AZone", func() string { return "/usr/share/zoneinfo/America/New_York" }, "America/New_York"},
		{"nothing known", "", none, time.Local.String()},
		{"unrelated path", "", func() string { return "/etc/foo" }, time.Local.String()},
	}
	for _, tc := range cases {
		if got := zoneFrom(tc.tz, tc.target).String(); got != tc.want {
			t.Errorf("%s: zone = %q, want %q", tc.name, got, tc.want)
		}
	}
	if LocalZone() == nil {
		t.Error("LocalZone returned nil")
	}
}
