package theme

import (
	"strings"
	"testing"
	"time"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// stampCellCount is how many pixels of the check frame f has lit.
func stampCellCount(f int) int {
	n := 0
	for _, row := range stampSprites[f].Rows {
		n += strings.Count(row, "c")
	}
	return n
}

func TestStampDrawsSevenFramesOfOneRefreshEach(t *testing.T) {
	for _, c := range []struct{ at, frame int }{{0, 0}, {16, 0}, {17, 1}, {50, 2}, {101, 5}, {102, 6}, {119, 6}, {400, 6}} {
		if got := StampFrameAt(ms(c.at), false); got != c.frame {
			t.Errorf("at %d ms frame %d, want %d", c.at, got, c.frame)
		}
	}
	if got := StampFrameAt(0, true); got != StampFrames-1 {
		t.Errorf("Reduce Motion starts on frame %d, want the last", got)
	}
}

func TestStampAddsTwoPixelsAFrameLeftToRightUntilTheWholeCheck(t *testing.T) {
	prev := 0
	for f := 0; f < StampFrames; f++ {
		n := stampCellCount(f)
		if n != 2*(f+1) || n <= prev {
			t.Errorf("frame %d lights %d pixels, want %d", f, n, 2*(f+1))
		}
		prev = n
	}
	if prev != len(glyphCheck) {
		t.Errorf("the last frame lights %d pixels, the check has %d", prev, len(glyphCheck))
	}
	// Left to right: no pixel of frame 0 is right of a pixel of the last frame's later cells.
	for i := 1; i < len(stampCells); i++ {
		if stampCells[i][0] < stampCells[i-1][0] {
			t.Fatalf("cells %v then %v run right to left", stampCells[i-1], stampCells[i])
		}
	}
}

func TestAcceptRunsAbout420Milliseconds(t *testing.T) {
	if AcceptFoldAt != ms(199) || AcceptDuration != ms(419) {
		t.Errorf("the fold begins at %v and the accept ends at %v, want 199 ms and 419 ms", AcceptFoldAt, AcceptDuration)
	}
}

func TestFoldClosesTheHeightEasedOutAndFadesFirst(t *testing.T) {
	if h, o := FoldAt(0, false); h != 1 || o != 1 {
		t.Errorf("at the start: %v, %v; want 1, 1", h, o)
	}
	lastH, lastO := float32(1), float32(1)
	for e := 1; e <= 220; e++ {
		h, o := FoldAt(ms(e), false)
		if h > lastH || o > lastO || h < 0 || o < 0 {
			t.Fatalf("at %d ms height %v, opacity %v after %v, %v", e, h, o, lastH, lastO)
		}
		lastH, lastO = h, o
	}
	if h, o := FoldAt(FoldFade, false); o != 0 || h <= 0 {
		t.Errorf("at 180 ms: %v, %v; the fade is over, the height is not", h, o)
	}
	if h, o := FoldAt(FoldDuration, false); h != 0 || o != 0 {
		t.Errorf("at 220 ms: %v, %v; want 0, 0", h, o)
	}
	// Ease-out cubic: 110 ms in, the height has 1/8 left.
	if h, _ := FoldAt(ms(110), false); h < 0.124 || h > 0.126 {
		t.Errorf("at 110 ms the height is %v, want 0.125", h)
	}
}

func TestFoldWithReducedMotionOnlyFadesThenIsGone(t *testing.T) {
	if h, o := FoldAt(ms(90), true); h != 1 || o <= 0 || o >= 1 {
		t.Errorf("halfway: %v, %v; want a full height fading", h, o)
	}
	if h, o := FoldAt(FoldFade, true); h != 0 || o != 0 {
		t.Errorf("after the fade: %v, %v; want gone", h, o)
	}
}

func TestOpenGrowsEasedOutAndAppearsAtOnceWithReducedMotion(t *testing.T) {
	if h, o := OpenAt(0, false); h != 0 || o != 0 {
		t.Errorf("at the start: %v, %v; want 0, 0", h, o)
	}
	if h, o := OpenAt(FoldDuration, false); h != 1 || o != 1 {
		t.Errorf("at the end: %v, %v; want 1, 1", h, o)
	}
	if h, o := OpenAt(0, true); h != 1 || o != 1 {
		t.Errorf("Reduce Motion: %v, %v; want 1, 1", h, o)
	}
}
