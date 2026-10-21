package remind

import (
	"testing"
	"time"
)

func TestPresetsFollowTheDayOfTheWeekAndTheHour(t *testing.T) {
	at := func(day, hour, min int) time.Time { return time.Date(2026, 10, day, hour, min, 0, 0, time.Local) }
	// Oct 12 2026 is a Monday.
	for _, c := range []struct {
		name string
		now  time.Time
		want map[Kind]time.Time
	}{
		{"wednesday afternoon", at(14, 14, 30), map[Kind]time.Time{
			Later: at(14, 18, 0), Tomorrow: at(15, 9, 0), Monday: at(19, 9, 0), Weekend: at(17, 10, 0)}},
		{"wednesday 5:45 pm leaves out the evening, which is under half an hour away", at(14, 17, 45), map[Kind]time.Time{
			Tomorrow: at(15, 9, 0), Monday: at(19, 9, 0), Weekend: at(17, 10, 0)}},
		{"monday means the monday a week ahead", at(12, 8, 0), map[Kind]time.Time{
			Later: at(12, 18, 0), Tomorrow: at(13, 9, 0), Monday: at(19, 9, 0), Weekend: at(17, 10, 0)}},
		{"saturday early is this weekend, today", at(17, 8, 0), map[Kind]time.Time{
			Later: at(17, 18, 0), Tomorrow: at(18, 9, 0), Monday: at(19, 9, 0), Weekend: at(17, 10, 0)}},
		{"saturday noon has no weekend left to offer", at(17, 12, 0), map[Kind]time.Time{
			Later: at(17, 18, 0), Tomorrow: at(18, 9, 0), Monday: at(19, 9, 0)}},
		{"sunday: tomorrow is the monday", at(18, 10, 0), map[Kind]time.Time{
			Later: at(18, 18, 0), Tomorrow: at(19, 9, 0)}},
	} {
		got := map[Kind]time.Time{}
		last := Kind(-1)
		for _, p := range Presets(c.now) {
			got[p.Kind] = p.At
			if p.Kind <= last {
				t.Errorf("%s: presets are out of order", c.name)
			}
			last = p.Kind
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: %d presets, want %d (%v)", c.name, len(got), len(c.want), got)
		}
		for k, w := range c.want {
			if g, ok := got[k]; !ok || !g.Equal(w) {
				t.Errorf("%s: preset %d is %v, want %v", c.name, k, g, w)
			}
		}
	}
}
