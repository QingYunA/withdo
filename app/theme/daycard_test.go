package theme

import (
	"testing"
	"time"
)

func TestDayCardEasesOutAndIsDoneAtItsDuration(t *testing.T) {
	var lastP, lastO float32 = -1, -1
	for ms := 0; ms < 140; ms++ {
		p, o, done := DayCardAt(time.Duration(ms) * time.Millisecond)
		if done {
			t.Fatalf("done at %d ms, before 140 ms", ms)
		}
		if p < lastP || o < lastO {
			t.Errorf("at %d ms progress %v, opacity %v fell from %v, %v", ms, p, o, lastP, lastO)
		}
		if p < 0 || p >= 1 || o < 0 || o > 1 {
			t.Errorf("at %d ms progress %v, opacity %v are out of range", ms, p, o)
		}
		lastP, lastO = p, o
	}
	if p, o, done := DayCardAt(DayCardDuration); p != 1 || o != 1 || !done {
		t.Errorf("at 140 ms: %v, %v, %v; want 1, 1, true", p, o, done)
	}
	if p, o, done := DayCardAt(time.Second); p != 1 || o != 1 || !done {
		t.Errorf("long after: %v, %v, %v; want 1, 1, true", p, o, done)
	}
}

func TestDayCardCurveIsEaseOutCubicAndOpacityIsFullFrom100ms(t *testing.T) {
	if p, o, _ := DayCardAt(0); p != 0 || o != 0 {
		t.Errorf("at 0 ms: %v, %v; want 0, 0", p, o)
	}
	// Ease-out is past the middle at half the time: 1-(1-0.5)^3 = 0.875.
	if p, _, _ := DayCardAt(70 * time.Millisecond); p < 0.874 || p > 0.876 {
		t.Errorf("progress at 70 ms is %v, want 0.875", p)
	}
	if _, o, done := DayCardAt(100 * time.Millisecond); o != 1 || done {
		t.Errorf("at 100 ms opacity %v, done %v; want 1, false", o, done)
	}
	if _, o, _ := DayCardAt(139 * time.Millisecond); o != 1 {
		t.Errorf("opacity falls back to %v at 139 ms", o)
	}
}

func TestDayCardBeforeItOpensIsItsFirstFrame(t *testing.T) {
	if p, o, done := DayCardAt(-time.Millisecond); p != 0 || o != 0 || done {
		t.Errorf("a negative time gives %v, %v, %v", p, o, done)
	}
}
