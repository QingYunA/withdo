package theme

import (
	"sort"
	"time"

	"github.com/egoist/mygo/ui"
)

// Accepting a task (v10 B, docs/design/mockups/v10.html): its button stamps a pixel check, the row
// holds for a beat, then folds out of its list while the task opens at the top of Done. Like the
// day card it is a pure function of the time since it began, so a view draws it from the clock it
// already has and nothing runs once it is over.
const (
	// StampFrames is how many frames the check is drawn in: two pixels more a frame.
	StampFrames = 7
	// StampBeat is one frame of the check: a stroke being drawn steps once a 60 Hz refresh.
	StampBeat time.Duration = 17 * time.Millisecond
	// StampHold is how long the stamped button rests before the row folds, so the eye sees it.
	StampHold time.Duration = 80 * time.Millisecond
	// AcceptFoldAt is when the fold begins, counted from the click.
	AcceptFoldAt = StampFrames*StampBeat + StampHold
	// FoldDuration is how long a row takes to fold away, or a task to open in Done.
	FoldDuration time.Duration = 220 * time.Millisecond
	// FoldFade is how long the fade lasts within the fold.
	FoldFade time.Duration = 180 * time.Millisecond
	// AcceptDuration is from the click to the end of the fold.
	AcceptDuration = AcceptFoldAt + FoldDuration
)

// StampFrameAt is the frame of the check shown elapsed after the click, from 0 to StampFrames-1; it
// holds the last frame from then on. With reduce it is the last frame at once.
func StampFrameAt(elapsed time.Duration, reduce bool) int {
	if reduce || elapsed >= StampFrames*StampBeat {
		return StampFrames - 1
	}
	return int(max(elapsed, 0) / StampBeat)
}

// FoldAt is how much of a row's height is left and how opaque it is, from 1 to 0, elapsed after the
// fold began: the height closes over FoldDuration and the fade over FoldFade, both along an
// ease-out cubic. With reduce the row only fades, keeping its height until it is gone.
func FoldAt(elapsed time.Duration, reduce bool) (height, opacity float32) {
	opacity = 1 - easeOut(clamp01(float32(elapsed)/float32(FoldFade)))
	if reduce {
		if elapsed >= FoldFade {
			return 0, 0
		}
		return 1, opacity
	}
	return 1 - easeOut(clamp01(float32(elapsed)/float32(FoldDuration))), opacity
}

// OpenAt is how much of its height a row has and how opaque it is, from 0 to 1, elapsed after the
// task began to open in Done: both along an ease-out cubic over FoldDuration. With reduce it is
// there at once.
func OpenAt(elapsed time.Duration, reduce bool) (height, opacity float32) {
	if reduce {
		return 1, 1
	}
	k := easeOut(clamp01(float32(elapsed) / float32(FoldDuration)))
	return k, k
}

func clamp01(v float32) float32 { return max(0, min(v, 1)) }

// stampCells are the cells of the done glyph's check, left to right: the stamp draws them in that
// order, so it reads as one stroke.
var stampCells = func() [][2]int {
	cells := append([][2]int(nil), glyphCheck...)
	sort.SliceStable(cells, func(i, j int) bool { return cells[i][0] < cells[j][0] })
	return cells
}()

// stampSprites are the frames of the check, built once; a pixel is 'c'.
var stampSprites = func() [StampFrames]*Sprite {
	var frames [StampFrames]*Sprite
	for f := range frames {
		n := (len(stampCells)*(f+1) + StampFrames/2) / StampFrames // round(14 (f+1) / 7)
		rows := make([][]byte, glyphGrid)
		for y := range rows {
			rows[y] = []byte("............")
		}
		for _, xy := range stampCells[:n] {
			rows[xy[1]][xy[0]] = 'c'
		}
		out := make([]string, glyphGrid)
		for y, r := range rows {
			out[y] = string(r)
		}
		frames[f] = &Sprite{Rows: out}
	}
	return frames
}()

// glyphGrid is the side of the glyph grids, in pixels.
const glyphGrid = 12

// StampSize is the side of the stamp in points: one pixel of the grid is a point.
const StampSize float32 = 12

// Stamp draws frame f (0 to StampFrames-1) of the check in col, StampSize points square.
func Stamp(c *ui.Context, col ui.Color, f int) ui.Element {
	sprite := *stampSprites[max(0, min(f, StampFrames-1))]
	sprite.Palette = map[rune]ui.Color{'c': col}
	return ui.Box(c.Key("stamp")).Size(StampSize, StampSize).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		DrawSprite(p, r, &sprite)
	})
}
