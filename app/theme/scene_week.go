package theme

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The week: night, and one stack of goods a day, the week's goods falling from the sky one by one
// and bouncing once where they land. Last, whoever finished most hops beside the busiest day's stack.

const (
	weekDays     = 7
	stackWide    = 2  // goods side by side in a stack
	stackRows    = 4  // rows of goods drawn in a stack; the rest are only counted
	stackStep    = 15 // cells between two rows
	weekStart    = 400
	weekGap      = 110 // ms between two goods
	weekFall     = 520 // ms a good takes to fall
	weekBounce   = 0.35
	weekHopEvery = 1400
	weekHopFor   = 420
	weekAfter    = 900 // ms from the last landing to the hop
)

// dayX is the left cell of the stack of a day of the week (0 is the first).
func dayX(d int) int { return 6 + d*37 }

// DayLabelX and DayLabelW are where the label of a day sits under the scene, in cells.
func DayLabelX(d int) int { return dayX(d) }

// DayLabelW is how wide a day's label is, in cells.
const DayLabelW = 33

// WeekScene is a week to draw: the kind of every good finished each day, in the order they were
// finished, and who finished most.
type WeekScene struct {
	Days [weekDays][]string
	// Lead is who finished most, and hops beside the busiest day; nil when nothing was finished.
	Lead *Actor
}

type weekGood struct {
	day, index int // the day, and the place among its goods
	kind       string
	x, y       int // where it rests
	start      int // ms: when it begins to fall
}

// WeekPlan is the choreography of a week.
type WeekPlan struct {
	scene   WeekScene
	goods   []weekGood
	more    [weekDays]int // goods of a day that do not fit in its stack
	end     int
	busiest int
	// beside is the day whose slot the lead stands in.
	beside int
}

// NewWeekPlan plans a week.
func NewWeekPlan(s WeekScene) *WeekPlan {
	p := &WeekPlan{scene: s}
	best := -1
	for d, kinds := range s.Days {
		if len(kinds) > best {
			best, p.busiest = len(kinds), d
		}
		shown := min(len(kinds), stackRows*stackWide)
		p.more[d] = len(kinds) - shown
		for j := 0; j < shown; j++ {
			row := j / stackWide
			single := j == shown-1 && j%stackWide == 0
			x := dayX(d) + (j%stackWide)*17
			if single {
				x = dayX(d) + 8
			}
			p.goods = append(p.goods, weekGood{day: d, index: j, kind: kinds[j], x: x, y: sceneFeet - GoodGrid - row*stackStep})
		}
	}
	for n := range p.goods {
		p.goods[n].start = weekStart + n*weekGap
	}
	if len(p.goods) > 0 {
		p.end = p.goods[len(p.goods)-1].start + weekAfter
	}
	p.beside = p.slotBeside(p.busiest)
	return p
}

// slotBeside is the day the lead stands in: the empty one after the busiest day, else the empty one
// before it, else the busiest day itself.
func (p *WeekPlan) slotBeside(busiest int) int {
	for _, d := range []int{busiest + 1, busiest - 1} {
		if d >= 0 && d < weekDays && len(p.scene.Days[d]) == 0 {
			return d
		}
	}
	return busiest
}

// Busiest is the day with the most goods, the first when several tie.
func (p *WeekPlan) Busiest() int { return p.busiest }

// End is when the goods have all landed and the lead starts to hop: the ledger lights up then.
func (p *WeekPlan) End() time.Duration { return time.Duration(p.end) * time.Millisecond }

// Calm is the time of the last frame, which a scene with Reduce Motion draws at once.
func (p *WeekPlan) Calm() time.Duration { return p.End() + 2*time.Second }

// More is how many goods of a day do not fit in its stack.
func (p *WeekPlan) More(day int) int { return p.more[day] }

// fallY is where a good is at ms after it began to fall: it drops from above the scene, and bounces
// three cells up and down once.
func (g weekGood) fallY(now int) (int, bool) {
	u := float64(now-g.start) / weekFall
	switch {
	case u < 0:
		return 0, false
	case u < 1:
		return round(-20 + float64(g.y+20)*u*u), true
	case u < 1+weekBounce:
		v := (u - 1) / weekBounce
		return g.y - round(3*4*v*(1-v)), true
	}
	return g.y, true
}

// Frame is the scene at t after it opened.
func (p *WeekPlan) Frame(t time.Duration, calm bool) SceneFrame {
	now := millis(t)
	f := SceneFrame{Sky: SkyNight}
	add := func(ops ...SceneOp) { f.Ops = append(f.Ops, ops...) }
	for d, kinds := range p.scene.Days {
		if len(kinds) > 0 {
			add(rect(dayX(d)-1, sceneFeet+1, 35, 2, stackShadow))
		}
	}
	for _, g := range p.goods {
		if y, ok := g.fallY(now); ok {
			add(SceneOp{Kind: OpGood, Name: g.kind, X: g.x, Y: y})
		}
	}
	if p.scene.Lead != nil {
		add(p.leadOp(now, calm))
	}
	for d, n := range p.more {
		if n > 0 && now >= p.landedBy(d) {
			add(digits("+"+itoa(n), dayX(d)+DayLabelW/2-len("+"+itoa(n))*2, 2, plusBright)...)
		}
	}
	return f
}

var stackShadow = ui.RGBA(10, 10, 30, 0.3)

// landedBy is when the last good of a day has landed.
func (p *WeekPlan) landedBy(day int) int {
	at := 0
	for _, g := range p.goods {
		if g.day == day {
			at = g.start + weekFall
		}
	}
	return at
}

// leadOp is whoever finished most, beside the busiest day's stack: it hops once every 1.4 s once the
// goods have landed, and stands still with Reduce Motion.
func (p *WeekPlan) leadOp(now int, calm bool) SceneOp {
	pose, y := "idle", actorTop
	if !calm && len(p.goods) > 0 && now > p.end {
		u := float64((now-p.end)%weekHopEvery) / weekHopFor
		if u < 1 {
			pose, y = "hop", actorTop+hopLift(u)
		}
	}
	return SceneOp{Kind: OpActor, Name: pose, Actor: *p.scene.Lead, X: dayX(p.beside) + 2, Y: y}
}

// HitAt is the good of a stack under a cell at t, if any: the one drawn on top when two overlap.
func (p *WeekPlan) HitAt(t time.Duration, cx, cy int) (Hit, bool) {
	now := millis(t)
	for n := len(p.goods) - 1; n >= 0; n-- {
		g := p.goods[n]
		if now >= g.start+weekFall && cx >= g.x && cx < g.x+GoodGrid && cy >= g.y && cy < g.y+GoodGrid {
			return Hit{Day: g.day, Good: g.index, X: g.x, Y: g.y}, true
		}
	}
	return Hit{}, false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// Moving reports whether something travels at t (a good falling or bouncing, the lead hopping), so
// the scene wants a frame every refresh.
func (p *WeekPlan) Moving(t time.Duration) bool {
	now := millis(t)
	if len(p.goods) == 0 {
		return false
	}
	if now <= p.goods[len(p.goods)-1].start+weekFall*(1+weekBounce) {
		return true
	}
	return now > p.end && (now-p.end)%weekHopEvery < weekHopFor
}
