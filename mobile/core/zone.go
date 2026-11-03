package core

import (
	"sync"
	"time"
)

var zoneMu sync.Mutex

// SetZone makes name, the phone's IANA time zone ("Asia/Shanghai"), the core's local zone. Go on
// Android has no local zone of its own (time.Local is UTC there), so without it "today", "tomorrow
// at 9", the reminder presets and the calendar would follow UTC. The app calls it before Open and
// whenever it comes back to the front. offsetSeconds is the zone's offset from UTC now, used when
// the zone database does not know name. It changes nothing when the zone is already that one.
func SetZone(name string, offsetSeconds int64) {
	loc, err := time.LoadLocation(name)
	if err != nil || name == "" || name == "Local" {
		loc = time.FixedZone(name, int(offsetSeconds))
	}
	zoneMu.Lock()
	defer zoneMu.Unlock()
	if time.Local.String() == loc.String() {
		_, now := time.Now().In(time.Local).Zone()
		if _, then := time.Now().In(loc).Zone(); now == then {
			return
		}
	}
	time.Local = loc
}
