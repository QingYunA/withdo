package theme

import (
	"testing"
	"time"
)

func TestSpinnerLitBlockStepsClockwiseEvery100msAndTurnsIn800ms(t *testing.T) {
	for _, tc := range []struct {
		at   time.Duration
		want int
	}{
		{0, 0}, {99 * time.Millisecond, 0}, {100 * time.Millisecond, 1}, {750 * time.Millisecond, 7},
		{800 * time.Millisecond, 0}, {1700 * time.Millisecond, 1},
	} {
		if got := SpinnerFrame(time.UnixMilli(0).Add(tc.at), false); got != tc.want {
			t.Errorf("at %v: frame %d, want %d", tc.at, got, tc.want)
		}
	}
}

func TestSpinnerHoldsItsFirstFrameWithReduceMotion(t *testing.T) {
	for _, ms := range []int64{0, 100, 750, 123456} {
		if got := SpinnerFrame(time.UnixMilli(ms), true); got != 0 {
			t.Errorf("at %d ms: frame %d, want 0", ms, got)
		}
	}
}

func TestSpinnerLightsOneBlockAndTwoFadingBehindIt(t *testing.T) {
	// Frame 0 lights the top block; the two behind it, counter-clockwise, are at 60 % and 35 %.
	got := spinnerOpacities(0)
	want := [8]float32{1, 0.25, 0.25, 0.25, 0.25, 0.25, 0.35, 0.6}
	if got != want {
		t.Errorf("frame 0: %v, want %v", got, want)
	}
	if got := spinnerOpacities(3); got[3] != 1 || got[2] != 0.6 || got[1] != 0.35 || got[0] != 0.25 {
		t.Errorf("frame 3: %v", got)
	}
}

func TestSpinnerBlocksAreEightTwoByTwoSquaresInsideTheGrid(t *testing.T) {
	if len(spinnerBlocks) != 8 {
		t.Fatalf("%d blocks", len(spinnerBlocks))
	}
	seen := map[[2]int]bool{}
	for _, b := range spinnerBlocks {
		if b[0] < 0 || b[1] < 0 || b[0]+spinnerBlock > AvatarGrid || b[1]+spinnerBlock > AvatarGrid {
			t.Errorf("block %v leaves the %d x %d grid", b, AvatarGrid, AvatarGrid)
		}
		if seen[b] {
			t.Errorf("block %v twice", b)
		}
		seen[b] = true
	}
}
