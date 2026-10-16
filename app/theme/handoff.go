package theme

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Two columns, Mine and Agents: the room between them, the tray the Agents column sits on, and
// the content width below which they stack.
const (
	ColumnsGap        float32 = 28 // between Mine and the Agents tray
	TrayRadius        float32 = 14
	TrayPad           float32 = 16 // inside the tray, left and right
	ColumnsStackBelow float32 = 640
	ColumnHead        float32 = 40 // a column's header, name and a line under it
	ColumnAvatar      float32 = 20
)

// The hand-off: a task of the user's dragged to the Agents tray. The row shrinks into a parcel, the
// user's sprite hops down from the Mine header to catch it and carries it to the gap between the
// columns, the agent's sprite hops from the Agents header to meet it, the parcel changes hands (the
// agent squashes as it catches), both hop home and the parcel opens into the new row, which keeps an
// accent wash for two seconds. Like the hop, the whole thing is a pure function of the time since
// the drop, so the view draws it from the clock it already has and nothing runs while it is still.
const (
	HandOffOpenAt   time.Duration = 1000 * time.Millisecond // the row begins to show
	HandOffOpenFor  time.Duration = 200 * time.Millisecond  // and is whole
	HandOffWash     time.Duration = 2 * time.Second         // the accent wash, from the row being whole
	HandOffWashMax  float32       = 0.08                    // its strength
	HandOffReduceIn time.Duration = MotionFast              // Reduce Motion: the row fades out and in
	handOffShrink   time.Duration = 250 * time.Millisecond  // the row folds into the parcel
	handOffParcelAt time.Duration = 150 * time.Millisecond  // the parcel shows
	handOffCatch    time.Duration = 400 * time.Millisecond  // the user's sprite has landed by the parcel
	handOffMeet     time.Duration = 820 * time.Millisecond  // both sprites have landed in the gap
	handOffPass     time.Duration = 110 * time.Millisecond  // the parcel crosses; the agent squashes as it catches
)

// The legs of the two sprites: when each takes off and how long it is in the air.
var (
	userDown  = leg{start: 100 * time.Millisecond, air: 300 * time.Millisecond}
	userCarry = leg{start: 520 * time.Millisecond, air: 300 * time.Millisecond}
	botMeet   = leg{start: 540 * time.Millisecond, air: 280 * time.Millisecond}
	homeLeg   = leg{start: 960 * time.Millisecond, air: 240 * time.Millisecond}
)

// Pt is a point of the window, in points.
type Pt struct{ X, Y float32 }

// HandOffPlan is where things are when the row is dropped, as the view measured them the frame
// before (the sprites' top left corners).
type HandOffPlan struct {
	Row     ui.Rect // the dragged row where it was
	You     Pt      // the user's sprite at its column header
	Bot     Pt      // the agent's, at its header
	Meet    Pt      // the middle of the gap between the columns, at the height of the row
	Reduced bool    // Reduce Motion
}

// HandOffFrame is one moment of a hand-off.
type HandOffFrame struct {
	You, Bot     HopFrame
	YouOn, BotOn bool    // the sprite is away from its header
	Parcel       ui.Rect // the parcel, when ParcelOn
	ParcelOn     bool
	Ghost        ui.Rect // the row folding into the parcel (or fading, with Reduce Motion)
	GhostOpacity float32 // 0 when it is gone
	RowOpacity   float32 // of the new row in the tray: 0 until the parcel opens
	Wash         float32 // 0 to HandOffWashMax: the accent wash of the new row
}

// HandOffEnd is when a hand-off has nothing left to draw, the wash included.
func HandOffEnd(reduced bool) time.Duration {
	if reduced {
		return HandOffReduceIn + HandOffWash
	}
	return HandOffOpenAt + HandOffOpenFor + HandOffWash
}

// leg is one hop of a sprite: when it takes off, how long it is in the air, and where from and to.
type leg struct {
	start, air time.Duration
	from, to   Pt
}

func (l leg) between(from, to Pt) leg { l.from, l.to = from, to; return l }

// squash is how long the sprite takes to settle after landing from the leg.
func (l leg) squash() time.Duration {
	return time.Duration(float64(HopDuration-HopLanding) * float64(l.air) / float64(HopLanding))
}

// frame is the sprite's frame in the leg elapsed after the hand-off began: HopAt with the air time
// and the squash stretched to the leg. Once it has settled it rests where it landed and ok is false.
func (l leg) frame(elapsed time.Duration) (f HopFrame, ok bool) {
	k := float64(HopLanding) / float64(l.air)
	f, ok = HopAt(time.Duration(float64(elapsed-l.start)*k), l.from.X, l.from.Y, l.to.X, l.to.Y)
	if !ok {
		f = HopFrame{X: l.to.X, Y: l.to.Y, W: HopSize, H: HopSize, Landed: true}
	}
	return f, ok
}

// onLegs is the frame of a sprite that hops through legs in order. It is not drawn before the first
// takes off, nor after the last has settled: it is home.
func onLegs(elapsed time.Duration, legs ...leg) (HopFrame, bool) {
	for i := len(legs) - 1; i >= 0; i-- {
		if elapsed >= legs[i].start {
			f, ok := legs[i].frame(elapsed)
			return f, ok || i < len(legs)-1
		}
	}
	return HopFrame{}, false
}

// HandOffAt is the frame of a hand-off elapsed after the drop.
func HandOffAt(elapsed time.Duration, p HandOffPlan) HandOffFrame {
	if p.Reduced {
		return handOffReduced(elapsed, p)
	}
	land := Pt{p.Row.X + 4, p.Row.Y + (p.Row.H-HopSize)/2} // the user's sprite lands by the parcel
	meetYou, meetBot := Pt{p.Meet.X - HopSize - 2, p.Meet.Y}, Pt{p.Meet.X + 6, p.Meet.Y}
	var f HandOffFrame
	f.You, f.YouOn = onLegs(elapsed, userDown.between(p.You, land), userCarry.between(land, meetYou), homeLeg.between(meetYou, p.You))
	f.Bot, f.BotOn = onLegs(elapsed, botMeet.between(p.Bot, meetBot), homeLeg.between(meetBot, p.Bot))
	f.Parcel, f.ParcelOn = parcelAt(elapsed, f, handPt(HopFrame{X: land.X, Y: land.Y, W: HopSize, H: HopSize}, false))
	f.Ghost, f.GhostOpacity = ghostAt(elapsed, p.Row)
	f.RowOpacity, f.Wash = rowAt(elapsed, HandOffOpenAt)
	return f
}

// handPt is where a sprite holds the parcel: the user on its right, the agent on its left, so the
// parcel passes between them.
func handPt(h HopFrame, agent bool) Pt {
	if agent {
		return Pt{h.X - ParcelSize + 4, h.Y + h.H - ParcelSize - 2}
	}
	return Pt{h.X + h.W - 4, h.Y + h.H - ParcelSize - 2}
}

// parcelAt is the parcel: folded out of the row, held by the user's sprite, passed across to the
// agent's, and shrinking into the row it opens. rest is where it lies before the user's sprite
// has landed by it.
func parcelAt(elapsed time.Duration, f HandOffFrame, rest Pt) (ui.Rect, bool) {
	var at Pt
	switch {
	case elapsed < handOffParcelAt || elapsed >= HandOffOpenAt+HandOffOpenFor:
		return ui.Rect{}, false
	case elapsed < handOffCatch:
		at = rest
	case elapsed < handOffMeet:
		at = handPt(f.You, false)
	case elapsed < handOffMeet+handOffPass:
		u := float32(elapsed-handOffMeet) / float32(handOffPass)
		from, to := handPt(f.You, false), handPt(f.Bot, true)
		at = Pt{from.X + (to.X-from.X)*u, from.Y + (to.Y-from.Y)*u - 24*u*(1-u)}
	default:
		at = handPt(f.Bot, true)
	}
	size := ParcelSize
	if elapsed >= HandOffOpenAt { // it opens: it shrinks as the row shows
		k := 1 - float32(elapsed-HandOffOpenAt)/float32(HandOffOpenFor)
		at.X, at.Y, size = at.X+ParcelSize*(1-k)/2, at.Y+ParcelSize*(1-k)/2, ParcelSize*k
	}
	return ui.Rect{X: at.X, Y: at.Y, W: size, H: size}, true
}

// ghostAt is the row folding into the parcel: its rectangle shrinks to the parcel's and fades.
func ghostAt(elapsed time.Duration, row ui.Rect) (ui.Rect, float32) {
	if elapsed >= handOffShrink {
		return ui.Rect{}, 0
	}
	u := float32(elapsed) / float32(handOffShrink)
	to := ui.Rect{X: row.X + HopSize, Y: row.Y + (row.H-ParcelSize)/2, W: ParcelSize, H: ParcelSize}
	lerp := func(a, b float32) float32 { return a + (b-a)*u }
	return ui.Rect{X: lerp(row.X, to.X), Y: lerp(row.Y, to.Y), W: lerp(row.W, to.W), H: lerp(row.H, to.H)}, 1 - u
}

// rowAt is the new row's opacity and the strength of its wash: the row shows from open, and the
// wash stays a moment, then fades over HandOffWash.
func rowAt(elapsed, open time.Duration) (opacity, wash float32) {
	switch {
	case elapsed < open:
		return 0, 0
	case elapsed < open+HandOffOpenFor:
		u := float32(elapsed-open) / float32(HandOffOpenFor)
		return u, HandOffWashMax * u
	}
	left := 1 - float32(elapsed-open-HandOffOpenFor)/float32(HandOffWash)
	return 1, HandOffWashMax * max(left, 0)
}

// handOffReduced is Reduce Motion: nothing moves; the row fades out where it was and in on the
// right.
func handOffReduced(elapsed time.Duration, p HandOffPlan) HandOffFrame {
	var f HandOffFrame
	if elapsed < HandOffReduceIn {
		f.Ghost, f.GhostOpacity = p.Row, 1-float32(elapsed)/float32(HandOffReduceIn)
	}
	if elapsed < HandOffReduceIn {
		f.RowOpacity = float32(elapsed) / float32(HandOffReduceIn)
		f.Wash = HandOffWashMax * f.RowOpacity
		return f
	}
	f.RowOpacity = 1
	f.Wash = HandOffWashMax * max(1-float32(elapsed-HandOffReduceIn)/float32(HandOffWash), 0)
	return f
}
