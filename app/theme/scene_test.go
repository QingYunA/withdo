package theme

import (
	"image/color"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func at(msec int) time.Duration { return time.Duration(msec) * time.Millisecond }

// day is a scene of n people and the throws given as the actor of each.
func day(people int, throws ...int) *DayPlan {
	s := DayScene{Sky: SkyDusk}
	for i := 0; i < people; i++ {
		s.Actors = append(s.Actors, []Actor{YouActor, ActorOf("claude-code"), ActorOf("codex"), ActorOf("kimi-cli"), ActorOf("zed"), ActorOf("mystery")}[i])
	}
	for _, a := range throws {
		s.Throws = append(s.Throws, DayThrow{Actor: a, Kind: "gem"})
	}
	return NewDayPlan(s)
}

func ops(f SceneFrame, kind OpKind) []SceneOp {
	var out []SceneOp
	for _, op := range f.Ops {
		if op.Kind == kind {
			out = append(out, op)
		}
	}
	return out
}

func named(f SceneFrame, name string) []SceneOp {
	var out []SceneOp
	for _, op := range f.Ops {
		if op.Name == name {
			out = append(out, op)
		}
	}
	return out
}

func TestTheGoodKindsOfTheSceneAreTheOnesTheAssistantPicks(t *testing.T) {
	if !reflect.DeepEqual(GoodKinds(), data.GoodKinds) {
		t.Errorf("scene %v, data %v", GoodKinds(), data.GoodKinds)
	}
}

func TestSkyFollowsTheHourTheSceneOpens(t *testing.T) {
	for hour, want := range map[int]SkyKind{0: SkyDay, 9: SkyDay, 16: SkyDay, 17: SkyDusk, 19: SkyDusk, 20: SkyNight, 23: SkyNight} {
		if got := SkyAt(time.Date(2026, 10, 17, hour, 59, 0, 0, time.Local)); got != want {
			t.Errorf("%d:59 is %v, want %v", hour, got, want)
		}
	}
}

func TestTheGridIsCentredCroppedAndExtended(t *testing.T) {
	for w, want := range map[float32]int{792: 0, 798: 1, 822: 5, 780: -2, 700: -16, 740: -9, 860: 11} {
		if got := SceneOffset(w); got != want {
			t.Errorf("width %v: offset %d, want %d", w, got, want)
		}
	}
	// A point of a column 822 wide: five cells of extended sky, then the grid.
	if cx, cy := CellAt(822, 15, 6); cx != 0 || cy != 2 {
		t.Errorf("cell %d,%d", cx, cy)
	}
	if cx, _ := CellAt(822, 14, 0); cx != -1 {
		t.Errorf("cell %d", cx)
	}
}

func TestTheDayIsAPureFunctionOfItsPlanAndTime(t *testing.T) {
	p := day(3, 0, 1, 2, 0, 1)
	for _, at := range []int{0, 300, 1450, 3000, 9000} {
		if a, b := p.Frame(time.Duration(at)*time.Millisecond, false), p.Frame(time.Duration(at)*time.Millisecond, false); !reflect.DeepEqual(a, b) {
			t.Errorf("frames at %d ms differ", at)
		}
	}
}

func TestPeopleStandTwoLeftOfTheBinAndTwoRight(t *testing.T) {
	p := day(4, 0)
	if got := []int{p.ActorX(0), p.ActorX(1), p.ActorX(2), p.ActorX(3)}; !slices.Equal(got, []int{26, 66, 198, 160}) {
		t.Errorf("slots %v", got)
	}
	for i := 0; i < 4; i++ {
		x := p.ActorX(i)
		if i < 2 && x+ActorGrid > binX || i >= 2 && x < binX+binW {
			t.Errorf("person %d at %d is on the wrong side of the bin", i, x)
		}
	}
}

func TestAFifthPersonThrowsFromOffTheRightEdgeAndIsNotDrawn(t *testing.T) {
	p := day(6, 4, 5)
	if p.ActorX(4) < SceneW || p.ActorX(5) < SceneW {
		t.Errorf("fifth and sixth stand at %d and %d", p.ActorX(4), p.ActorX(5))
	}
	f := p.Frame(p.Calm(), false)
	if got := len(ops(f, OpActor)); got != 4 {
		t.Errorf("%d actors drawn, want the first four", got)
	}
	if fl := p.flights[0]; fl.fromX < SceneW {
		t.Errorf("a goods comes from x=%v, inside the scene", fl.fromX)
	}
	if got := len(named(p.Frame(at(p.flights[0].start+100), false), "gem")); got == 0 {
		t.Error("the good thrown from the edge is not in the air")
	}
}

func TestOnlySevenGoodsAreDrawnInTheBinButEveryOneIsCounted(t *testing.T) {
	throws := make([]int, 12)
	p := day(1, throws...)
	f := p.Frame(p.Calm(), false)
	if heap := len(ops(f, OpGood)); heap != 7 {
		t.Errorf("%d goods in the bin's mouth, want 7", heap)
	}
	if got := p.Landed(p.Calm()); got != 12 {
		t.Errorf("%d landed, want all 12", got)
	}
	if _, ok := p.HitAt(p.Calm(), heapSlots[6][0]+1, heapSlots[6][1]+1); !ok {
		t.Error("the seventh good cannot be hovered")
	}
}

func TestManyThrowsAreSqueezedToFitTheScene(t *testing.T) {
	slow, fast := day(1, make([]int, 6)...), day(1, make([]int, 60)...)
	if slow.End() > 8*time.Second {
		t.Errorf("six throws take %v", slow.End())
	}
	if fast.End() > 11*time.Second {
		t.Errorf("sixty throws take %v, want about eight seconds of throwing", fast.End())
	}
	if gap := fast.flights[1].start - fast.flights[0].start; gap != minGap {
		t.Errorf("gap %d, want the minimum %d", gap, minGap)
	}
	if gap := slow.flights[1].start - slow.flights[0].start; gap != throwGap {
		t.Errorf("gap %d, want %d", gap, throwGap)
	}
}

func TestTheLidOpensInThreeFramesAndStaysOpen(t *testing.T) {
	p := day(1, 0)
	want := map[int]string{0: "props/lid-shut", 200: "props/lid-shut", 300: "props/lid-half", 400: "", 5000: ""}
	for ms, lid := range want {
		f := p.Frame(at(ms), false)
		got := ""
		for _, name := range []string{"props/lid-shut", "props/lid-half"} {
			if len(named(f, name)) > 0 {
				got = name
			}
		}
		if got != lid {
			t.Errorf("at %d ms the lid is %q, want %q", ms, got, lid)
		}
		if open := len(named(f, "props/lid-open")) > 0; open != (ms >= 350) {
			t.Errorf("at %d ms open = %v", ms, open)
		}
	}
}

func TestADayWithNothingShowsNoPeopleAndAShutLid(t *testing.T) {
	p := day(0)
	if p.End() != 0 {
		t.Errorf("end %v", p.End())
	}
	for _, at := range []time.Duration{0, at(2000), p.Calm()} {
		f := p.Frame(at, false)
		if len(ops(f, OpActor)) != 0 || len(ops(f, OpGood)) != 0 || len(named(f, "props/lid-shut")) != 1 || len(named(f, "props/lid-open")) != 0 {
			t.Errorf("frame at %v: %+v", at, f.Ops)
		}
	}
}

func TestAGoodLandsTheBinDipsAndAPlusOneRises(t *testing.T) {
	p := day(1, 0)
	land := time.Duration(p.flights[0].land) * time.Millisecond
	rest := binRim
	bin := func(f SceneFrame) int { return named(f, "props/bin")[0].Y }
	if bin(p.Frame(land-at(10), false)) != rest || bin(p.Frame(land+at(50), false)) != rest+1 || bin(p.Frame(land+at(150), false)) != rest {
		t.Error("the bin dips one cell for 110 ms as a good lands")
	}
	early, late := p.Frame(land+at(50), false), p.Frame(land+at(500), false)
	count := func(f SceneFrame, c ui.Color) int {
		n := 0
		for _, op := range ops(f, OpRect) {
			if op.Color == c && op.W == 1 {
				n++
			}
		}
		return n
	}
	if count(early, plusBright) == 0 || count(early, plusDim) != 0 || count(late, plusBright) != 0 || count(late, plusDim) == 0 {
		t.Error("the +1 is bright for 450 ms and then dim")
	}
	if len(p.Frame(land+at(800), false).Ops) >= len(late.Ops) {
		t.Error("the +1 does not go away")
	}
}

func TestAStarredGoodTrailsDustAndPutsUpAStar(t *testing.T) {
	s := DayScene{Actors: []Actor{ActorOf("claude")}, Throws: []DayThrow{{Kind: "gem", Starred: true}}}
	p := NewDayPlan(s)
	inFlight := p.Frame(at(p.flights[0].start+400), false)
	dust := 0
	for _, op := range ops(inFlight, OpRect) {
		if op.Color == dustColors[0] || op.Color == dustColors[1] || op.Color == dustColors[2] {
			dust++
		}
	}
	if dust == 0 {
		t.Error("no gold dust behind a starred good")
	}
	landed := p.Frame(at(p.flights[0].land+100), false)
	if len(named(landed, "props/star")) != 1 {
		t.Error("no gold star when it lands")
	}
	plainPlan := NewDayPlan(DayScene{Actors: []Actor{ActorOf("claude")}, Throws: []DayThrow{{Kind: "jar"}}})
	plain := plainPlan.Frame(at(plainPlan.flights[0].land+100), false)
	if len(named(plain, "props/star")) != 0 {
		t.Error("a good without a merged pull request has a star")
	}
}

func TestAGoodSpinsEvery100msInTheAirAndFlightMovesInWholeCells(t *testing.T) {
	p := day(1, 0)
	fl := p.flights[0]
	spins := map[int]bool{}
	var xs []int
	for ms := fl.start; ms < fl.land; ms += 17 {
		f := p.Frame(at(ms), false)
		air := ops(f, OpGoodSpin)
		if len(air) != 1 {
			t.Fatalf("at %d ms %d goods in the air", ms, len(air))
		}
		spins[air[0].Spin] = true
		xs = append(xs, air[0].X)
	}
	if len(spins) != 4 {
		t.Errorf("turns seen %v, want all four quarter turns", spins)
	}
	moved := 0
	for i := 1; i < len(xs); i++ {
		if xs[i] != xs[i-1] {
			moved++
		}
	}
	if moved < len(xs)/3 {
		t.Errorf("the good moved on %d of %d 17 ms refreshes: a flight steps with the refresh", moved, len(xs))
	}
	if a, b := p.Frame(at(fl.start+99), false), p.Frame(at(fl.start+100), false); ops(a, OpGoodSpin)[0].Spin == ops(b, OpGoodSpin)[0].Spin {
		t.Error("the spin did not step at 100 ms")
	}
}

func TestAnActorLiftsHoldingItsGoodThenThrows(t *testing.T) {
	p := day(1, 0)
	start := p.flights[0].start
	pose := func(ms int) (string, int) {
		f := p.Frame(at(ms), false)
		a := ops(f, OpActor)[0]
		return a.Name, len(ops(f, OpGood))
	}
	if n, _ := pose(start - liftLead - 10); n != "idle" {
		t.Errorf("before: %s", n)
	}
	if n, goods := pose(start - 100); n != "lift" || goods != 1 {
		t.Errorf("lifting: %s with %d goods in hand", n, goods)
	}
	if n, _ := pose(start + 50); n != "throw" {
		t.Errorf("throwing: %s", n)
	}
	if n, _ := pose(start + throwHold + 10); n != "idle" {
		t.Errorf("after: %s", n)
	}
}

func TestCodexShootsItsGoodWithoutHoldingItUp(t *testing.T) {
	p := NewDayPlan(DayScene{Actors: []Actor{ActorOf("codex")}, Throws: []DayThrow{{Kind: "jar"}}})
	f := p.Frame(at(p.flights[0].start-100), false)
	if a := ops(f, OpActor)[0]; a.Name != "lift" || len(ops(f, OpGood)) != 0 || a.Flip {
		t.Errorf("%+v with %d goods in hand", a, len(ops(f, OpGood)))
	}
}

func TestActorsRightOfTheBinFaceItExceptCodex(t *testing.T) {
	p := day(3, 0)
	f := p.Frame(p.Calm(), false)
	flip := map[string]bool{}
	for _, a := range ops(f, OpActor) {
		flip[a.Actor.Folder] = a.Flip
	}
	if flip["you"] || flip["claude"] || flip["codex"] {
		t.Errorf("%v: you and claude stand left, and codex is never turned round", flip)
	}
	robot := day(3, 0)
	robot.flip[2] = true
	if !robot.flip[2] {
		t.Fatal("setup")
	}
	right := NewDayPlan(DayScene{Actors: []Actor{YouActor, YouActor, YouActor}})
	if !right.flip[2] || right.flip[0] {
		t.Errorf("flips %v: the third person stands right of the bin and faces left", right.flip)
	}
}

func TestEveryoneHopsOnceTheLastGoodLandsThenSways(t *testing.T) {
	p := day(3, 0, 1, 2)
	jump := p.jump
	for i := 0; i < 3; i++ {
		start := jump + i*hopGap
		peak := start + hopFor/2
		f := p.Frame(at(peak), false)
		a := ops(f, OpActor)[i]
		if a.Name != "hop" || a.Y >= actorTop {
			t.Errorf("actor %d at its peak: %s y=%d, want the hop pose, lifted", i, a.Name, a.Y)
		}
	}
	// A hop is lifted at most ten cells.
	for ms := jump; ms < jump+swayAfter; ms += 17 {
		if y := ops(p.Frame(at(ms), false), OpActor)[0].Y; y < actorTop-10 || y > actorTop {
			t.Fatalf("at %d ms the actor is at %d", ms, y)
		}
	}
	// Then, after 600 ms, a cell down on every other 450 ms beat.
	downs := map[int]bool{}
	for ms := jump + swayAfter + 50; ms < jump+swayAfter+2000; ms += 50 {
		downs[ops(p.Frame(at(ms), false), OpActor)[0].Y] = true
	}
	if !downs[actorTop] || !downs[actorTop+1] || len(downs) != 2 {
		t.Errorf("sway heights %v", downs)
	}
}

func TestReduceMotionIsTheLastFrameAtOnceAndNothingSways(t *testing.T) {
	p := day(3, 0, 1, 2, 0)
	for _, ms := range []int{0, 5000, 50000} {
		f := p.Frame(p.Calm()+at(ms), true)
		for _, a := range ops(f, OpActor) {
			if a.Name != "idle" || a.Y != actorTop {
				t.Errorf("a calm actor is %s at %d", a.Name, a.Y)
			}
		}
		if len(ops(f, OpGood)) != 4 || len(ops(f, OpGoodSpin)) != 0 {
			t.Errorf("calm frame holds %d goods at rest, %d in the air", len(ops(f, OpGood)), len(ops(f, OpGoodSpin)))
		}
		if len(named(f, "props/lid-open")) != 1 {
			t.Error("the lid is shut")
		}
	}
}

func TestHoveringTheHeapFindsTheGoodOnTop(t *testing.T) {
	p := day(1, 0, 0, 0)
	if _, ok := p.HitAt(at(100), heapSlots[0][0]+2, heapSlots[0][1]+2); ok {
		t.Error("a good that has not landed was hit")
	}
	end := p.Calm()
	if h, ok := p.HitAt(end, heapSlots[1][0]+13, heapSlots[1][1]+13); !ok || h.Good != 1 || h.X != heapSlots[1][0] {
		t.Errorf("%+v %v", h, ok)
	}
	// The third good sits over the first two where they meet.
	if h, ok := p.HitAt(end, 124, 52); !ok || h.Good != 2 {
		t.Errorf("%+v %v", h, ok)
	}
	if _, ok := p.HitAt(end, 5, 5); ok {
		t.Error("the sky was hit")
	}
}

func TestFirefliesComeOutOnlyOnceTheDayIsShipped(t *testing.T) {
	p := day(1, 0, 0)
	glow := func(ms int) int {
		n := 0
		for _, op := range ops(p.Frame(at(ms), false), OpRect) {
			if op.Color == fireflyCore {
				n++
			}
		}
		return n
	}
	if glow(p.jump-50) != 0 || glow(p.jump+1000) == 0 {
		t.Error("fireflies before the end or none after it")
	}
}

func TestDigitsHaveAnOutlineAndALitCore(t *testing.T) {
	got := digits("+1", 10, 10, plusBright)
	lit, dark := 0, 0
	for _, op := range got {
		switch op.Color {
		case plusBright:
			lit++
		case digitOutline:
			dark++
		}
	}
	if lit != 13 || dark < lit {
		t.Errorf("%d lit cells and %d outline cells for +1", lit, dark)
	}
	for _, r := range "0123456789+" {
		if _, ok := digitGlyphs[r]; !ok {
			t.Errorf("no glyph for %q", r)
		}
	}
}

func TestRobotsOfUnknownAgentsAreTintedLikeTheirAvatar(t *testing.T) {
	if a := ActorOf("Qwen Code"); a.Folder != ActorBot || a.Tint != botTint("Qwen Code") || a.Tint != ui.Hex("#615CED") {
		t.Errorf("%+v", a)
	}
	if a, b := ActorOf("mystery-1"), ActorOf("mystery-1"); a != b || a.Tint == ui.Hex(botTints[0]) {
		t.Errorf("%+v %+v", a, b)
	}
	if ActorOf("").Tint != ui.Hex(botTints[0]) || ActorOf("agent").Folder != ActorBot {
		t.Error("an agent with no name is the gray robot")
	}
	if ActorOf("claude-code").Folder != "claude" || ActorOf("Codex CLI").Folder != "codex" {
		t.Error("claude and codex have folders")
	}
}

// --- the week ---

func week(days ...int) *WeekPlan {
	var s WeekScene
	for d, n := range days {
		for i := 0; i < n; i++ {
			s.Days[d] = append(s.Days[d], "gem")
		}
	}
	lead := ActorOf("claude")
	s.Lead = &lead
	return NewWeekPlan(s)
}

func TestAWeekMakesOneStackADayTwoGoodsWide(t *testing.T) {
	p := week(4, 3, 6, 5, 7, 0, 0)
	f := p.Frame(p.Calm(), false)
	byDay := map[int][]SceneOp{}
	for _, op := range ops(f, OpGood) {
		d := (op.X - 6) / 37
		byDay[d] = append(byDay[d], op)
	}
	for d, want := range []int{4, 3, 6, 5, 7, 0, 0} {
		if len(byDay[d]) != want {
			t.Errorf("day %d has %d goods, want %d", d, len(byDay[d]), want)
		}
		xs := map[int]bool{}
		for _, op := range byDay[d] {
			xs[op.X] = true
		}
		if len(xs) > 3 { // two columns, and a single one over the middle
			t.Errorf("day %d is %d columns wide", d, len(xs))
		}
	}
	// Goods stand on the path and go up a row every 15 cells.
	if got := ops(f, OpGood)[0].Y; got != sceneFeet-GoodGrid {
		t.Errorf("first good at y=%d", got)
	}
}

func TestAStackWithAnOddGoodCentresTheLastOne(t *testing.T) {
	p := week(3)
	f := p.Frame(p.Calm(), false)
	goods := ops(f, OpGood)
	if len(goods) != 3 || goods[0].X != 6 || goods[1].X != 23 || goods[2].X != 14 || goods[2].Y != sceneFeet-GoodGrid-stackStep {
		t.Errorf("%+v", goods)
	}
}

func TestAStackIsCappedAtFourRowsAndSaysHowManyMoreThereAre(t *testing.T) {
	p := week(0, 12)
	f := p.Frame(p.Calm(), false)
	if got := len(ops(f, OpGood)); got != 8 {
		t.Errorf("%d goods drawn, want 8", got)
	}
	if p.More(1) != 4 || p.More(0) != 0 {
		t.Errorf("more %d %d", p.More(1), p.More(0))
	}
	lit := 0
	for _, op := range ops(f, OpRect) {
		if op.Color == plusBright && op.Y < 10 {
			lit++
		}
	}
	if lit == 0 {
		t.Error("no +4 over the full stack")
	}
	if top := ops(f, OpGood)[7].Y; top < 0 {
		t.Errorf("the top row is off the scene: y=%d", top)
	}
}

func TestWeekGoodsFallInOrder110msApartAndBounceOnce(t *testing.T) {
	p := week(2, 2)
	for n, g := range p.goods {
		if g.start != weekStart+n*weekGap {
			t.Errorf("good %d starts at %d", n, g.start)
		}
	}
	g := p.goods[0]
	if _, ok := g.fallY(g.start - 1); ok {
		t.Error("drawn before it falls")
	}
	if y, _ := g.fallY(g.start + 10); y >= 0 {
		t.Errorf("starts above the scene, y=%d", y)
	}
	if y, _ := g.fallY(g.start + weekFall); y != g.y {
		t.Errorf("lands at %d, want %d", y, g.y)
	}
	bounce := g.y
	for ms := g.start + weekFall; ms < g.start+weekFall+200; ms += 10 {
		if y, _ := g.fallY(ms); y < bounce {
			bounce = y
		}
	}
	if g.y-bounce != 3 {
		t.Errorf("bounces %d cells, want 3", g.y-bounce)
	}
	if y, _ := g.fallY(g.start + 2000); y != g.y {
		t.Errorf("rests at %d", y)
	}
}

func TestTheLeadHopsBesideTheBusiestDayEvery1400ms(t *testing.T) {
	p := week(4, 3, 6, 5, 7, 0, 0)
	if p.Busiest() != 4 || p.beside != 5 {
		t.Fatalf("busiest %d beside %d", p.Busiest(), p.beside)
	}
	lead := func(ms int) SceneOp { return ops(p.Frame(at(ms), false), OpActor)[0] }
	if a := lead(p.end + 10); a.X != dayX(5)+2 {
		t.Errorf("lead at x=%d", a.X)
	}
	hops, rests := 0, 0
	for ms := p.end; ms < p.end+2800; ms += 17 {
		if lead(ms).Name == "hop" {
			hops++
		} else {
			rests++
		}
	}
	if hops == 0 || rests < hops {
		t.Errorf("%d hopping frames, %d resting", hops, rests)
	}
	if a := lead(p.end + 1400 + 10); a.Name != "hop" {
		t.Errorf("a second hop 1.4 s on: %s", a.Name)
	}
	if a := ops(p.Frame(p.Calm(), true), OpActor)[0]; a.Name != "idle" || a.Y != actorTop {
		t.Errorf("a calm lead is %s at %d", a.Name, a.Y)
	}
}

func TestTheLeadStandsInTheNeighbourWhichIsEmptyOrElseBesideTheStack(t *testing.T) {
	for _, c := range []struct {
		days    []int
		busiest int
		beside  int
	}{
		{[]int{1, 5, 0, 0, 0, 0, 0}, 1, 2},
		{[]int{1, 1, 1, 1, 1, 1, 5}, 6, 6}, // no empty neighbour: on it
		{[]int{0, 0, 0, 0, 0, 0, 5}, 6, 5}, // Sunday: the day before
		{[]int{0, 0, 0, 0, 0, 0, 0}, 0, 1},
	} {
		if p := week(c.days...); p.Busiest() != c.busiest || p.beside != c.beside {
			t.Errorf("%v: busiest %d beside %d, want %d, %d", c.days, p.Busiest(), p.beside, c.busiest, c.beside)
		}
	}
}

func TestAWeekWithNothingHasNoGoodsAndNoLead(t *testing.T) {
	p := NewWeekPlan(WeekScene{})
	f := p.Frame(p.Calm(), false)
	if p.End() != 0 || len(ops(f, OpGood)) != 0 || len(ops(f, OpActor)) != 0 || f.Sky != SkyNight {
		t.Errorf("%v %+v", p.End(), f)
	}
}

func TestHoveringAWeekStackFindsTheGoodAndItsDay(t *testing.T) {
	p := week(1, 3)
	end := p.Calm()
	if h, ok := p.HitAt(end, dayX(1)+17+3, sceneFeet-GoodGrid+3); !ok || h.Day != 1 || h.Good != 1 {
		t.Errorf("%+v %v", h, ok)
	}
	if _, ok := p.HitAt(at(100), dayX(0)+3, sceneFeet-GoodGrid+3); ok {
		t.Error("a good still in the air was hit")
	}
	if _, ok := p.HitAt(end, dayX(3)+3, sceneFeet-GoodGrid+3); ok {
		t.Error("a bare day was hit")
	}
}

// --- drawing ---

func TestADayDrawsInsideItsColumn(t *testing.T) {
	p := day(3, 0, 1, 2)
	frame := p.Frame(p.Calm(), true)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(792, SceneHeight).Draw(func(painter *ui.Painter, r ui.Rect) { frame.Draw(painter, r) })
	}, 792, 240)
	tt.Frame()
	img := tt.Image()
	if b := img.Bounds(); b.Dx() != 792 || b.Dy() != 240 {
		t.Fatalf("image %v", b)
	}
	// The sky is dusk: the top left is the dark end of the ramp, and the bin's wood is drawn.
	if c := color.RGBAModel.Convert(img.At(2, 2)).(color.RGBA); c.R > 90 || c.B < 60 {
		t.Errorf("top left %v", c)
	}
	wood := 0
	for x := binX * 3; x < (binX+40)*3; x += 3 {
		if c := img.RGBAAt(x+1, (binRim+8)*3+1); c.R > 100 && c.G > 50 && c.B < 90 {
			wood++
		}
	}
	if wood < 10 {
		t.Errorf("%d wooden columns where the bin is", wood)
	}
}

func TestAWiderColumnRepeatsTheEdgesAndANarrowerOneCrops(t *testing.T) {
	frame := day(0).Frame(0, true)
	for _, w := range []float32{1000, 600} {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Box(c).Size(w, SceneHeight).Draw(func(painter *ui.Painter, r ui.Rect) { frame.Draw(painter, r) })
		}, int(w), 240)
		tt.Frame()
		img := tt.Image()
		for _, x := range []int{0, int(w) - 1} {
			if c := img.RGBAAt(x, 100); c.A != 255 {
				t.Errorf("width %v: column %d is not painted (%v)", w, x, c)
			}
		}
	}
}
