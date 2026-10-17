package theme

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The busy indicator is eight 2 x 2 blocks round the 12 x 12 grid, clockwise from the top. One block
// is lit and steps clockwise every SpinnerStep; the two behind it trail at 60 % and 35 %, the rest
// at 25 %. It is pixel art in frames, never rotated or eased, and a pure function of the time,
// like the day card.
const (
	// SpinnerStep is how long a block stays lit: a turn is eight steps, 800 ms.
	SpinnerStep = 100 * time.Millisecond
	// spinnerBlock is the side of a block in grid pixels.
	spinnerBlock = 2
	// spinnerRest is the opacity of a block that is neither lit nor in the tail.
	spinnerRest float32 = 0.25
)

// spinnerBlocks are the top left corners of the blocks on the grid, clockwise from the top.
var spinnerBlocks = [8][2]int{{5, 1}, {8, 2}, {9, 5}, {8, 8}, {5, 9}, {2, 8}, {1, 5}, {2, 2}}

// spinnerTail is the opacity of the lit block and of the two behind it.
var spinnerTail = [3]float32{1, 0.6, 0.35}

// SpinnerFrame is the block that is lit at now, 0 to 7 from the top: it steps every SpinnerStep. With
// Reduce Motion it is always the first.
func SpinnerFrame(now time.Time, reduceMotion bool) int {
	if reduceMotion {
		return 0
	}
	return int(now.UnixMilli()/SpinnerStep.Milliseconds()) % len(spinnerBlocks)
}

// spinnerOpacities are the opacities of the eight blocks in a frame.
func spinnerOpacities(frame int) [8]float32 {
	var out [8]float32
	for k := range out {
		out[k] = spinnerRest
		if age := (frame - k + len(out)) % len(out); age < len(spinnerTail) {
			out[k] = spinnerTail[age]
		}
	}
	return out
}

// PixelSpinner is the busy indicator in a frame, size points square in color. Draw it at a whole
// multiple of the grid (12, 18, 24 pt) so no pixel is split; the caller asks for the next frame
// with c.After(SpinnerStep) while it is shown, unless Reduce Motion is on.
func PixelSpinner(c *ui.Context, size float32, color ui.Color, frame int) ui.Element {
	op, u := spinnerOpacities(frame), size/AvatarGrid
	return ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		for k, b := range spinnerBlocks {
			p.Fill(ui.Rect{X: r.X + float32(b[0])*u, Y: r.Y + float32(b[1])*u, W: spinnerBlock * u, H: spinnerBlock * u},
				color.Alpha(op[k]), 0)
		}
	})
}

// stopMark is the stop mark's side in grid pixels, centred on the grid.
const stopMark = 6

// PixelStop is the stop mark, a solid square 6 x 6 pixels in the middle of the 12 x 12 grid, size
// points square in color (9 pt at 18 pt).
func PixelStop(c *ui.Context, size float32, color ui.Color) ui.Element {
	u := size / AvatarGrid
	return ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		off := (AvatarGrid - stopMark) / 2 * u
		p.Fill(ui.Rect{X: r.X + off, Y: r.Y + off, W: stopMark * u, H: stopMark * u}, color, 0)
	})
}
