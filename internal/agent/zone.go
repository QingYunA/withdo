package agent

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalZone is the user's time zone with its IANA name, which the prompt states. On macOS the
// Go runtime calls it "Local", so the name is read from the TZ variable or from where
// /etc/localtime points; failing both, time.Local is returned.
func LocalZone() *time.Location {
	return zoneFrom(os.Getenv("TZ"), func() string {
		p, err := filepath.EvalSymlinks("/etc/localtime")
		if err != nil {
			return ""
		}
		return p
	})
}

func zoneFrom(tz string, localtimeTarget func() string) *time.Location {
	tz = strings.TrimPrefix(tz, ":")
	if tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil && tz != "Local" {
			return loc
		}
	}
	if name := ianaFromPath(localtimeTarget()); name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return time.Local
}

// ianaFromPath takes "Asia/Shanghai" out of ".../zoneinfo/Asia/Shanghai".
func ianaFromPath(p string) string {
	_, name, ok := strings.Cut(filepath.ToSlash(p), "zoneinfo/")
	if !ok {
		return ""
	}
	return name
}
