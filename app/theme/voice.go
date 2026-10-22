package theme

import "github.com/egoist/mygo/ui"

// micRows is the pixel microphone on the 12 x 12 grid, drawn at 24 pt (2 pt a pixel).
var micRows = [AvatarGrid]string{
	"....####....",
	"...######...",
	"...######...",
	"...######...",
	"...######...",
	".#.######.#.",
	".#.######.#.",
	".#..####..#.",
	"..##....##..",
	"....####....",
	".....##.....",
	"...######...",
}

// VoiceMic is the microphone button's picture: the pixel microphone, size points square in color.
// Draw it at a whole multiple of the grid (24 pt) so no pixel is split.
func VoiceMic(c *ui.Context, size float32, color ui.Color) ui.Element {
	u := size / AvatarGrid
	return ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		for y, row := range micRows {
			for x, ch := range row {
				if ch == '#' {
					p.Fill(ui.Rect{X: r.X + float32(x)*u, Y: r.Y + float32(y)*u, W: u, H: u}, color, 0)
				}
			}
		}
	})
}

// The waveform shown while the microphone records: WaveBars bars of WaveCell-point cells, each bar
// two cells wide with one cell between, centred on a midline and 2, 4, 6 or 8 cells tall.
const (
	WaveBars = 12
	WaveCell = 2
	// WaveHeight and WaveWidth are the size of the waveform in points.
	WaveHeight float32 = 8 * WaveCell
	WaveWidth  float32 = WaveBars*3*WaveCell - WaveCell
)

// WaveCells is how many cells tall a bar of loudness level (0 to 1) is: 2, 4, 6 or 8.
func WaveCells(level float32) int {
	switch {
	case level >= 0.75:
		return 8
	case level >= 0.5:
		return 6
	case level >= 0.2:
		return 4
	}
	return 2
}

// PixelWave is the waveform of the last WaveBars loudness levels, oldest at the left. It is pixel
// art in frames: the caller pushes a level every 100 ms, the bars never ease.
func PixelWave(c *ui.Context, levels []float32, color ui.Color) ui.Element {
	return ui.Box(c).Size(WaveWidth, WaveHeight).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		for i := 0; i < WaveBars; i++ {
			var level float32
			if i < len(levels) {
				level = levels[i]
			}
			h := float32(WaveCells(level)) * WaveCell
			p.Fill(ui.Rect{X: r.X + float32(i)*3*WaveCell, Y: r.Y + (WaveHeight-h)/2, W: 2 * WaveCell, H: h}, color, 0)
		}
	})
}
