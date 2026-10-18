package theme

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// The day: everyone who finished something throws it into the shipping bin, in the order the tasks
// were finished, then everyone hops, and the page's ledger counts it up. The beats are v18's: a
// flight is a position that moves every refresh and snaps to whole cells, while poses, the spin of a
// good, the lid and the +1 change every 100 ms.

// DayThrow is one finished task: who threw it, the good it became, and whether its pull request
// merged (a good with a gold star and a trail of dust).
type DayThrow struct {
	Actor   int // index into DayScene.Actors
	Kind    string
	Starred bool
}

// DayScene is a day to draw: the sky, the people (you first, then the agents by how many they
// finished) and the tasks in the order they were finished.
type DayScene struct {
	Sky    SkyKind
	Actors []Actor
	Throws []DayThrow
}

const (
	heapCap     = 7   // goods drawn in the bin's mouth; the rest are only counted
	maxActors   = 4   // people who stand in the scene: two left of the bin, two right
	offEdge     = 8   // cells past the right edge a fifth actor's goods come from
	throwStart  = 700 // ms before the first throw
	throwGap    = 480 // ms between two throws
	throwBudget = 5760
	minGap      = 120
	liftLead    = 260 // ms an actor lifts its good before it throws
	throwHold   = 180 // ms of the throw pose
	sinkFor     = 110 // ms the bin dips as a good lands
	plusFor     = 700 // ms a +1 stays
	hopGap      = 90  // ms between the hops of two actors
	hopFor      = 420
	jumpAfter   = 350 // ms from the last landing to the hop
	jumpTotal   = 900 // ms from the hop to the end of the scene
	swayAfter   = 600 // ms from the hop to the sway
	swayBeat    = 450
	swayStagger = 170
	spinBeat    = 100
	lidShutTo   = 250 // ms the lid stays shut; it is half open to lidHalfTo, then open
	lidHalfTo   = 350
	flies       = 7
)

// actorSlots are the left cells of the people who stand in the scene, in the order they come: you
// and the next agent left of the bin, then two right of it.
var actorSlots = [maxActors]int{26, 66, 198, 160}

// heapSlots are the top left cells of the goods in the bin's mouth, the first layer of two, then a
// layer of two, then one each, going up to a point.
var heapSlots = [heapCap][2]int{{115, 49}, {133, 49}, {124, 44}, {116, 39}, {132, 39}, {124, 34}, {124, 29}}

type flight struct {
	start, dur, land int // ms
	fromX, fromY     float64
	toX, toY         float64
	arc              float64 // how high the good rises over the straight line, in cells
	dir              int     // 1 thrown to the right, -1 to the left
}

// DayPlan is the choreography of a day: when each good is thrown and where it lands.
type DayPlan struct {
	scene           DayScene
	flights         []flight
	last, jump, end int
	x               []int // each actor's left cell
	flip            []bool
	throwsOf        [][]int
	heapOf          []int // for each throw, its place in the heap, -1 when it is only counted
}

// NewDayPlan plans a day.
func NewDayPlan(s DayScene) *DayPlan {
	p := &DayPlan{scene: s, x: make([]int, len(s.Actors)), flip: make([]bool, len(s.Actors)), throwsOf: make([][]int, len(s.Actors))}
	for i, a := range s.Actors {
		p.x[i] = SceneW + offEdge
		if i < maxActors {
			p.x[i] = actorSlots[i]
		}
		p.flip[i] = p.x[i] > binX && !a.facesLeft()
	}
	n := len(s.Throws)
	gap := throwGap
	if n > 0 {
		gap = max(minGap, min(throwGap, throwBudget/n))
	}
	for i, th := range s.Throws {
		p.throwsOf[th.Actor] = append(p.throwsOf[th.Actor], i)
		p.flights = append(p.flights, p.plan(th, throwStart+i*gap, i))
		p.last = max(p.last, p.flights[i].land)
	}
	if n > 0 {
		p.jump = p.last + jumpAfter
		p.end = p.jump + jumpTotal
	}
	return p
}

// plan is the flight of one throw: from the actor's hands to its place in the heap.
func (p *DayPlan) plan(th DayThrow, start, i int) flight {
	a := p.scene.Actors[th.Actor]
	x0 := p.x[th.Actor]
	fx, fy := handsOf(a)
	switch {
	case th.Actor >= maxActors:
		fx = 0 // from the edge itself
		x0 = SceneW + offEdge
	case p.flip[th.Actor]:
		fx = ActorGrid - 1 - fx
	}
	slot := heapSlots[min(i, heapCap-1)]
	f := flight{start: start, fromX: float64(x0 + fx), fromY: float64(actorTop + fy), toX: float64(slot[0] + GoodGrid/2), toY: float64(slot[1] + GoodGrid/2)}
	f.dir = 1
	if f.toX < f.fromX {
		f.dir = -1
	}
	dx := math.Abs(f.toX - f.fromX)
	f.dur = int(560 + dx*2.2)
	f.arc = 16 + dx/5
	f.land = start + f.dur
	return f
}

// handsOf is where a good leaves an actor, in cells from its top left: held up, or shot from Codex's
// `>_`. The robot holds it higher, over its antenna.
func handsOf(a Actor) (x, y int) {
	switch a.Folder {
	case "codex":
		return 2, 15
	case "you":
		return 20, 4
	case ActorBot:
		return 12, -4
	}
	return 12, 2
}

// End is when the scene has played: the ledger starts to light up then. A day with nothing finished
// has no scene to play.
func (p *DayPlan) End() time.Duration { return time.Duration(p.end) * time.Millisecond }

// Calm is the time of the last frame, which a scene with Reduce Motion draws at once.
func (p *DayPlan) Calm() time.Duration { return p.End() + 2*time.Second }

// positionAt is where a flight has a good at ms: along the line, lifted by a parabola.
func (f flight) positionAt(t int) (float64, float64) {
	u := math.Min(math.Max(float64(t-f.start)/float64(f.dur), 0), 1)
	return f.fromX + (f.toX-f.fromX)*u, f.fromY + (f.toY-f.fromY)*u - f.arc*4*u*(1-u)
}

// Frame is the scene at t after it opened. With calm nothing sways or blinks: the frame is the one
// of Reduce Motion.
func (p *DayPlan) Frame(t time.Duration, calm bool) SceneFrame {
	now := millis(t)
	f := SceneFrame{Sky: p.scene.Sky}
	add := func(ops ...SceneOp) { f.Ops = append(f.Ops, ops...) }
	sink := p.sinkAt(now)
	lid := p.lidAt(now)
	add(p.groundShadows()...)
	if lid == lidOpen {
		add(SceneOp{Kind: OpProp, Name: "props/lid-open", X: binX, Y: binRim - 6 + sink})
	}
	for i, th := range p.scene.Throws {
		if slot := p.heapSlot(i); slot >= 0 && now >= p.flights[i].land {
			add(SceneOp{Kind: OpGood, Name: th.Kind, X: heapSlots[slot][0], Y: heapSlots[slot][1] + sink})
		}
	}
	add(p.flightOps(now)...)
	add(SceneOp{Kind: OpProp, Name: "props/bin", X: binX, Y: binRim + sink})
	switch lid {
	case lidHalf:
		add(SceneOp{Kind: OpProp, Name: "props/lid-half", X: binX, Y: binRim - 4})
	case lidShut:
		add(SceneOp{Kind: OpProp, Name: "props/lid-shut", X: binX - 1, Y: binRim - 3})
	}
	add(p.actorOps(now, calm)...)
	add(p.plusOps(now)...)
	if len(p.scene.Throws) > 0 && now > p.jump {
		add(fireflies(now)...)
	}
	return f
}

// groundShadows are the soft shadows under the people and the bin.
func (p *DayPlan) groundShadows() []SceneOp {
	var ops []SceneOp
	for i := range p.scene.Actors {
		if i < maxActors {
			ops = append(ops, rect(p.x[i]+5, sceneFeet+1, 15, 2, groundShadow))
		}
	}
	return append(ops, rect(binX-1, sceneFeet+1, 42, 2, groundShadow))
}

var groundShadow = ui.RGBA(30, 20, 40, 0.22)

type lidState int

const (
	lidShut lidState = iota
	lidHalf
	lidOpen
)

// lidAt: shut, a frame half open, then open for good. A day with nothing in it stays shut.
func (p *DayPlan) lidAt(now int) lidState {
	switch {
	case len(p.scene.Throws) == 0 || now < lidShutTo:
		return lidShut
	case now < lidHalfTo:
		return lidHalf
	}
	return lidOpen
}

// sinkAt is how many cells the bin dips: one for a moment after a good lands.
func (p *DayPlan) sinkAt(now int) int {
	for _, f := range p.flights {
		if now >= f.land && now < f.land+sinkFor {
			return 1
		}
	}
	return 0
}

// heapSlot is where throw i rests in the bin, or -1 when it is only counted: the heap draws seven.
func (p *DayPlan) heapSlot(i int) int {
	if i >= heapCap {
		return -1
	}
	return i
}

// flightOps are the goods in the air, with the gold dust behind a starred one.
func (p *DayPlan) flightOps(now int) []SceneOp {
	var ops []SceneOp
	for i, th := range p.scene.Throws {
		fl := p.flights[i]
		if now < fl.start || now >= fl.land {
			continue
		}
		if th.Starred {
			ops = append(ops, p.dust(fl, now)...)
		}
		x, y := fl.positionAt(now)
		turns := ((now-fl.start)/spinBeat*fl.dir%4 + 4) % 4
		ops = append(ops, SceneOp{Kind: OpGoodSpin, Name: th.Kind, Spin: turns, X: round(x) - GoodGrid/2, Y: round(y) - GoodGrid/2})
	}
	return ops
}

var dustColors = [3]ui.Color{ui.Hex("#fff6c0"), ui.Hex("#fee761"), ui.Hex("#e8a03a")}

// dust is the gold trail of a good whose pull request merged: six specks along where it has been.
func (p *DayPlan) dust(fl flight, now int) []SceneOp {
	var ops []SceneOp
	for k := 1; k <= 6; k++ {
		at := now - k*45
		if at < fl.start {
			break
		}
		x, y := fl.positionAt(at)
		c := dustColors[0]
		switch {
		case k >= 5:
			c = dustColors[2]
		case k >= 3:
			c = dustColors[1]
		}
		ops = append(ops, rect(round(x)+(k*7)%5-2, round(y)+(k*3)%5-2, 1, 1, c))
	}
	return ops
}

func round(f float64) int { return int(math.Floor(f + 0.5)) }

// actorOps are the people, in the pose each has at now, each holding the good it is about to throw.
func (p *DayPlan) actorOps(now int, calm bool) []SceneOp {
	var ops []SceneOp
	for i, a := range p.scene.Actors {
		if i >= maxActors {
			continue
		}
		pose, throw := p.poseAt(i, now, calm)
		y := actorTop + p.hopAt(i, now) + p.bobAt(i, now, calm)
		ops = append(ops, SceneOp{Kind: OpActor, Name: pose, Actor: a, Flip: p.flip[i], X: p.x[i], Y: y})
		if throw >= 0 && pose == "lift" && a.Folder != "codex" {
			fl, th := p.flights[throw], p.scene.Throws[throw]
			ops = append(ops, SceneOp{Kind: OpGood, Name: th.Kind, X: round(fl.fromX) - GoodGrid/2, Y: round(fl.fromY) - GoodGrid/2 + (y - actorTop)})
		}
	}
	return ops
}

// poseAt is an actor's pose at now and the throw it is lifting or throwing, -1 when it is neither:
// it lifts the good for liftLead before it leaves its hands, throws, hops once with the others, and
// sways.
func (p *DayPlan) poseAt(i, now int, calm bool) (string, int) {
	for _, n := range p.throwsOf[i] {
		start := p.flights[n].start
		switch {
		case now >= start-liftLead && now < start:
			return "lift", n
		case now >= start && now < start+throwHold:
			return "throw", n
		}
	}
	if len(p.scene.Throws) > 0 {
		if at := now - p.jump - i*hopGap; at >= 0 && at < hopFor {
			return "hop", -1
		}
		if !calm && now > p.jump+swayAfter && ((now+i*swayStagger)/swayBeat)%2 == 1 && hasPose(p.scene.Actors[i].Folder, "idle2") {
			return "idle2", -1
		}
	}
	return "idle", -1
}

// hopAt is how many cells an actor is lifted in the hop all of them make once the last good lands,
// one after the other.
func (p *DayPlan) hopAt(i, now int) int {
	if len(p.scene.Throws) == 0 {
		return 0
	}
	return hopLift(float64(now-p.jump-i*hopGap) / hopFor)
}

// hopLift is the height, in cells, of a hop at u (0 to 1) of the way through it.
func hopLift(u float64) int {
	if u <= 0 || u >= 1 {
		return 0
	}
	return -round(10 * 4 * u * (1 - u))
}

// bobAt is the sway: a cell down on every other beat while the page is open.
func (p *DayPlan) bobAt(i, now int, calm bool) int {
	if calm || len(p.scene.Throws) == 0 || now <= p.jump+swayAfter {
		return 0
	}
	return ((now + i*swayStagger) / swayBeat) % 2
}

// plusOps are the +1 that rises above the bin when a good lands, and a gold star beside it for one
// whose pull request merged.
func (p *DayPlan) plusOps(now int) []SceneOp {
	var ops []SceneOp
	for i, fl := range p.flights {
		age := now - fl.land
		if age < 0 || age > plusFor {
			continue
		}
		rise := int(math.Min(float64(age)/600, 1) * 10)
		ink := plusBright
		if age >= 450 {
			ink = plusDim
		}
		ops = append(ops, digits("+1", binX+15, binRim-14-rise, ink)...)
		if p.scene.Throws[i].Starred {
			ops = append(ops, SceneOp{Kind: OpProp, Name: "props/star", X: binX + 26, Y: binRim - 16 - rise})
		}
	}
	return ops
}

var (
	plusBright = ui.Hex("#ffffff")
	plusDim    = ui.Hex("#c8c4e8")
)

// fireflies come out at the right once the day is shipped: a speck of light with a glow to each side,
// each blinking off one beat in three.
func fireflies(now int) []SceneOp {
	var ops []SceneOp
	for i := 0; i < flies; i++ {
		if (now/300+i)%3 == 0 {
			continue
		}
		t := float64(now)
		x := round(196 + 26*math.Sin(t/1100+float64(i)*1.7) + float64(i)*6)
		y := round(30 + 12*math.Sin(t/800+float64(i)*2.3))
		ops = append(ops, rect(x-1, y, 1, 1, fireflyGlow), rect(x+1, y, 1, 1, fireflyGlow), rect(x, y, 1, 1, fireflyCore))
	}
	return ops
}

var (
	fireflyCore = ui.Hex("#fff6b0")
	fireflyGlow = ui.RGBA(254, 231, 97, 0.45)
)

// HitAt is the good of the heap under a cell at t, if any: the one drawn on top when two overlap.
func (p *DayPlan) HitAt(t time.Duration, cx, cy int) (Hit, bool) {
	now := millis(t)
	for i := min(len(p.scene.Throws), heapCap) - 1; i >= 0; i-- {
		x, y := heapSlots[i][0], heapSlots[i][1]
		if now >= p.flights[i].land && cx >= x && cx < x+GoodGrid && cy >= y && cy < y+GoodGrid {
			return Hit{Good: i, X: x, Y: y}, true
		}
	}
	return Hit{}, false
}

// ActorX is the left cell an actor stands on; an actor past the fourth is off the right edge.
func (p *DayPlan) ActorX(i int) int { return p.x[i] }

// Landed is how many goods are in the bin at t, drawn or not.
func (p *DayPlan) Landed(t time.Duration) int {
	n := 0
	for _, f := range p.flights {
		if millis(t) >= f.land {
			n++
		}
	}
	return n
}
