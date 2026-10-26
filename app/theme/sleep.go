package theme

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The sleeping robot of the idle Agents panel is the gray robot with its eyes shut
// (sprites/avatars/bot-asleep.png), drawn at 3 pt a pixel. Two pixel "z" letters rise from beside
// its head in steps, never eased: a loop is four steps of SleepStep, the second z half a loop behind
// the first. Like the spinner it is a pure function of the time.
const (
	// SleepStep is how long a step of the z's lasts: a loop is four steps, 2 s.
	SleepStep = 500 * time.Millisecond
	// SleepSize is the robot's side in points (3 pt a pixel).
	SleepSize float32 = 36
	// SleepBoxW and SleepBoxH are the box the robot and its z's are drawn in. The robot stands at
	// its foot, so the z's have room above and to the right of its head.
	SleepBoxW, SleepBoxH float32 = 54, 52
	// SleepZRoom is how far the box reaches above the 40 pt the layout gives it.
	SleepZRoom float32 = 12
	// sleepZUnit is the side of a cell of a z in points.
	sleepZUnit float32 = 2
)

var spriteAsleep = loadSprite("avatars/bot-asleep")

// zGlyph is the letter z on a 5 x 5 grid.
var zGlyph = [5]string{"#####", "...#.", "..#..", ".#...", "#####"}

// sleepZs are where each z starts in the box: its top left corner at the lowest point of its rise.
var sleepZs = [2][2]float32{{34, 14}, {44, 4}}

// SleepZ is one z in a step: whether it is drawn and how far it has risen from its lowest point
// (0, 2 or 4 pt, counted up from the start of the loop).
type SleepZ struct {
	Shown bool
	Rise  float32
}

// sleepRises are the four steps of a loop: hidden, then shown on the way up, shown at the top,
// hidden again.
var sleepRises = [4]SleepZ{{false, 0}, {true, 2}, {true, 4}, {false, 4}}

// SleepFrame is the two z's at now. With Reduce Motion both stand still at the top.
func SleepFrame(now time.Time, reduceMotion bool) [2]SleepZ {
	if reduceMotion {
		return [2]SleepZ{{true, 4}, {true, 4}}
	}
	step := int(now.UnixMilli()/SleepStep.Milliseconds()) % len(sleepRises)
	return [2]SleepZ{sleepRises[step], sleepRises[(step+2)%len(sleepRises)]}
}

// SleepNext is how long until the z's change.
func SleepNext(now time.Time) time.Duration {
	return SleepStep - time.Duration(now.UnixNano())%SleepStep
}

// SleepingBot is the sleeping robot and its z's in a frame, in a SleepBoxW x SleepBoxH box; the
// robot is its lower left. The caller asks for the next frame with c.After(SleepNext(now)) while it
// is shown, unless Reduce Motion is on.
func SleepingBot(c *ui.Context, t *Tokens, frame [2]SleepZ) ui.Element {
	return ui.Box(c).Size(SleepBoxW, SleepBoxH).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		DrawSprite(p, ui.Rect{X: r.X, Y: r.Y + SleepBoxH - SleepSize, W: SleepSize, H: SleepSize}, spriteAsleep)
		for i, z := range frame {
			if z.Shown {
				drawZ(p, r.X+sleepZs[i][0], r.Y+sleepZs[i][1]-z.Rise, t.TextTertiary)
			}
		}
	})
}

// drawZ draws the letter with its top left corner at x, y.
func drawZ(p *ui.Painter, x, y float32, color ui.Color) {
	for row, cells := range zGlyph {
		for col, cell := range cells {
			if cell == '#' {
				p.Fill(ui.Rect{X: x + float32(col)*sleepZUnit, Y: y + float32(row)*sleepZUnit, W: sleepZUnit, H: sleepZUnit}, color, 0)
			}
		}
	}
}
