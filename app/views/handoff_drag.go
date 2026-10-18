package views

import (
	"slices"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The sprites of the hand-off play during the drag, not after it. mygo paints the dragged row itself
// under the pointer (a copy of its element tree), so the user's sprite is a child of the row that
// only exists while it is dragged: it stands under the left end of the card, the card held over its
// head, and goes wherever the pointer takes the row. While the pointer moves it walks (a small bob,
// four steps a second). While the row is over the Agents tray the robot in the tray's header hops up
// and reaches, again every 0.8 s; on the drop it squashes once as it catches the row, and the row
// keeps an accent wash for two seconds. With Reduce Motion the sprite still carries the row but
// neither walks, hops nor squashes.
//
// All of it is a function of the app's clock, and a frame is asked for every 16 ms only while the
// sprite walks, the agent hops or squashes, or the wash fades: a still window asks for none.

// carrierSize is the user's sprite under the lifted row.
const carrierSize = theme.HopSize

// carrierTuck is how far the sprite's head goes under the card, so the card rests on it.
const carrierTuck float32 = 3

// handOffState is the hand-off's memory: the pointer of the row being carried, whether the row is
// over the tray, the row that just landed, and the Undo that ⌘Z does.
type handOffState struct {
	carry   carry
	over    bool      // a row of Mine is over the Agents tray
	overAt  time.Time // since when
	landed  landing
	undo    func()
	undoSeq int
	asked   int // frames asked for, which the tests look at
}

// carry follows the pointer of the row in flight, to tell whether it walks.
type carry struct {
	x, y     float32 // the pointer, in the row's own coordinates, the frame before
	has      bool
	seen     bool      // a carried row was built this frame
	since    time.Time // the sprite began walking
	lastMove time.Time // the pointer last moved
}

// landing is the rows that were just dropped on the tray, in the order they land, and when. The
// agent catches them one after the other, catchEvery apart.
type landing struct {
	ids []int
	at  time.Time
}

// catchEvery is how long after one task the robot of the tray squashes for the next of a stack.
const catchEvery = 120 * time.Millisecond

// has reports whether the row is one of those that just landed.
func (l landing) has(id int) bool { return id != 0 && slices.Contains(l.ids, id) }

// walk notes where the pointer of the carried row is and reports how far the sprite is lifted: it
// bobs while the pointer moves and stands still a moment after it stopped.
func (h *handOffState) walk(c *ui.Context, row ui.Element, now time.Time) float32 {
	px, py, _ := row.PointerPosition()
	k := &h.carry
	k.seen = true
	moved := !k.has || px != k.x || py != k.y
	k.x, k.y, k.has = px, py, true
	if moved {
		if now.Sub(k.lastMove) > theme.CarryStopsAfter {
			k.since = now // it had stopped: a new walk, from the first step
		}
		k.lastMove = now
	}
	if c.Preferences().ReduceMotion || now.Sub(k.lastMove) > theme.CarryStopsAfter {
		return 0
	}
	h.ask(c)
	return theme.CarryBob(now.Sub(k.since))
}

// ask draws another frame in 16 ms, and counts the asking.
func (h *handOffState) ask(c *ui.Context) {
	h.asked++
	c.After(hopFrameEvery)
}

// carrier is the user's sprite under the left end of the carried row: bob lifts it while it walks.
func (m *Main) carrier(c *ui.Context, bob float32) {
	sprite := theme.You.Sprite()
	ui.Box(c.Key("carrier")).Absolute().Left(rowSlot).Bottom(carrierTuck-carrierSize-bob).Size(carrierSize, carrierSize).Shrink(0).
		Draw(func(p *ui.Painter, r ui.Rect) { theme.DrawSprite(p, r, sprite) })
}

// handOffTick runs once a frame before the window is built: it forgets the pointer of a row that is
// no longer carried and asks for frames while the agent hops, squashes or the wash fades.
func (m *Main) handOffTick(c *ui.Context) {
	h, now := &m.ho, m.app.Now()
	if !h.carry.seen {
		h.carry = carry{}
	}
	h.carry.seen = false
	if len(h.landed.ids) > 0 && now.Sub(h.landed.at) >= theme.WashFor {
		h.landed = landing{}
	}
	if len(h.landed.ids) > 0 || (h.over && !c.Preferences().ReduceMotion) {
		h.ask(c)
	}
}

// setOver notes whether a row of Mine is over the tray this frame.
func (h *handOffState) setOver(over bool, now time.Time) {
	if over && !h.over {
		h.overAt = now
	}
	h.over = over
}

// agentFrame is the robot of the Agents header in its box r this frame: at rest, hopping while a row
// is over the tray, or squashing as it catches the row just dropped. The frame is HopAt's, which is
// drawn for a 24 pt sprite; it is scaled to the header's 20 pt with its feet on the same line.
func (m *Main) agentFrame(reduce bool, r ui.Rect) ui.Rect {
	h, now := &m.ho, m.app.Now()
	f := theme.HopFrame{W: theme.HopSize, H: theme.HopSize}
	if !reduce {
		if catch, ok := h.catching(now); ok {
			f = catch
		} else if h.over {
			f = theme.Reach(now.Sub(h.overAt), 0, 0)
		}
	}
	k := r.W / theme.HopSize
	w, hh := f.W*k, f.H*k
	bottom := r.Y + r.H + (f.Y+f.H-theme.HopSize)*k
	return ui.Rect{X: r.X + (r.W-w)/2, Y: bottom - hh, W: w, H: hh}
}

// catching is the squash of the robot at now: that of the task it caught last, one every catchEvery.
func (h *handOffState) catching(now time.Time) (theme.HopFrame, bool) {
	for i := len(h.landed.ids) - 1; i >= 0; i-- {
		if f, ok := theme.Catch(now.Sub(h.landed.at)-time.Duration(i)*catchEvery, 0, 0); ok {
			return f, true
		}
	}
	return theme.HopFrame{}, false
}

// rowWash is the strength of the accent wash of the row that just landed in Waiting for an agent.
func (h *handOffState) rowWash(id int, now time.Time) float32 {
	if !h.landed.has(id) {
		return 0
	}
	return theme.Wash(now.Sub(h.landed.at))
}
