package theme

import (
	"testing"
	"time"
)

func TestDayCardGrowsInFourPixelSteps(t *testing.T) {
	for _, tc := range []struct {
		ms    int
		scale float32
		done  bool
	}{
		{0, 0.55, false},
		{49, 0.55, false},
		{50, 0.7, false},
		{99, 0.7, false},
		{120, 0.85, false},
		{150, 1, false},
		{199, 1, false},
		{200, 1, true},
		{900, 1, true},
	} {
		scale, done := DayCardAt(time.Duration(tc.ms) * time.Millisecond)
		if scale != tc.scale || done != tc.done {
			t.Errorf("DayCardAt(%d ms) = %v, %v; want %v, %v", tc.ms, scale, done, tc.scale, tc.done)
		}
	}
}

func TestDayCardBeforeItOpensIsTheFirstStep(t *testing.T) {
	if scale, done := DayCardAt(-time.Millisecond); scale != 0.55 || done {
		t.Errorf("a negative time gives %v, %v", scale, done)
	}
}
