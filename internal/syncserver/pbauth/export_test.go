package pbauth

import "time"

// SetRealtimePingInterval changes how often realtime streams are pinged and returns a function that
// puts the old interval back.
func SetRealtimePingInterval(d time.Duration) (restore func()) {
	old := realtimePingInterval
	realtimePingInterval = d
	return func() { realtimePingInterval = old }
}
