package theme

import (
	"math"
	"testing"
	"time"
)

func count(cues []Cue, s Sound) int {
	n := 0
	for _, c := range cues {
		if c.Sound == s {
			n++
		}
	}
	return n
}

func only(cues []Cue, s Sound) []Cue {
	var out []Cue
	for _, c := range cues {
		if c.Sound == s {
			out = append(out, c)
		}
	}
	return out
}

func TestADayOpensTheLidLandsEachGoodAndEndsOnDone(t *testing.T) {
	p := day(2, 0, 1, 0)
	cues := p.SoundCues(false)
	if n := len(cues); n != 5 {
		t.Fatalf("%d cues, want lid + 3 lands + done: %+v", n, cues)
	}
	if cues[0] != (Cue{At: at(lidShutTo), Sound: SoundLid}) {
		t.Errorf("first %+v: the lid opens when it leaves the shut frame", cues[0])
	}
	for i, c := range only(cues, SoundLand) {
		if c.At != at(p.flights[i].land) || c.Step != i {
			t.Errorf("land %d is %+v, flight lands at %v", i, c, p.flights[i].land)
		}
	}
	if last := cues[len(cues)-1]; last != (Cue{At: at(p.jump), Sound: SoundDone}) {
		t.Errorf("last %+v, want the done at the hop, %d ms", last, p.jump)
	}
}

func TestAStarSoundsWithTheLandOfAMergedGood(t *testing.T) {
	s := DayScene{Actors: []Actor{YouActor}, Throws: []DayThrow{{Kind: "gem"}, {Kind: "gem", Starred: true}, {Kind: "gem"}}}
	p := NewDayPlan(s)
	cues := p.SoundCues(false)
	stars := only(cues, SoundStar)
	if len(stars) != 1 || stars[0].At != at(p.flights[1].land) {
		t.Fatalf("stars %+v, want one at the second land (%d ms)", stars, p.flights[1].land)
	}
	for i, c := range cues {
		if c.Sound == SoundStar && cues[i-1].Sound != SoundLand {
			t.Errorf("the star follows %+v, not a land", cues[i-1])
		}
	}
}

func TestADenseDaySoundsEightLandsAtMostAndStarsOnlyOnThose(t *testing.T) {
	var lands []landing
	for i := 0; i < 18; i++ {
		lands = append(lands, landing{at: at(1000 + i*120), star: i%2 == 1})
	}
	cues := landCues(lands)
	got := only(cues, SoundLand)
	// 0, 240, 480, ...: every other one of the 120 ms apart, since each needs 150 ms after the last.
	if len(got) != 8 {
		t.Fatalf("%d lands, want 8: %+v", len(got), got)
	}
	for i, c := range got {
		if c.At != at(1000+i*240) || c.Step != i {
			t.Errorf("land %d is %+v", i, c)
		}
	}
	// The goods that sound are the 1st, 3rd, ...: the even ones, which are not starred.
	if n := count(cues, SoundStar); n != 0 {
		t.Errorf("%d stars: a star belongs to a land that sounds", n)
	}
	for i := range lands {
		lands[i].star = true
	}
	if n := count(landCues(lands), SoundStar); n != 8 {
		t.Errorf("%d stars for 8 lands that sound", n)
	}
}

func TestADayOfFourteenGoodsSoundsEightLandsAndTheirStars(t *testing.T) {
	s := DayScene{Actors: []Actor{YouActor, ActorOf("claude-code")}}
	for i := 0; i < 14; i++ {
		s.Throws = append(s.Throws, DayThrow{Actor: i % 2, Kind: "gem", Starred: i == 12})
	}
	cues := NewDayPlan(s).SoundCues(false)
	if n := count(cues, SoundLand); n != 8 {
		t.Errorf("%d lands, want 8", n)
	}
	if n := count(cues, SoundStar); n != 0 {
		t.Errorf("%d stars: the 13th good does not sound, so neither does its star", n)
	}
}

func TestLandsSoundAtLeast150MsApartWhateverTheOrderOfTheirFlights(t *testing.T) {
	// A long day squeezes the throws to 120 ms, and a far throw lands after a nearer one that left later.
	var who []int
	for i := 0; i < 60; i++ {
		who = append(who, i%4)
	}
	cues := only(day(4, who...).SoundCues(false), SoundLand)
	if len(cues) == 0 || len(cues) > landMax {
		t.Fatalf("%d lands", len(cues))
	}
	for i := 1; i < len(cues); i++ {
		if cues[i].At-cues[i-1].At < landGap {
			t.Errorf("lands %d and %d are %v apart", i-1, i, cues[i].At-cues[i-1].At)
		}
	}
}

func TestLandsAreTakenInTheOrderTheyLandNotThrown(t *testing.T) {
	cues := landCues([]landing{{at: at(900)}, {at: at(500)}, {at: at(700)}})
	if len(cues) != 3 || cues[0].At != at(500) || cues[1].At != at(700) || cues[2].At != at(900) {
		t.Errorf("%+v", cues)
	}
}

func TestALandJust150MsAfterTheLastSounds(t *testing.T) {
	cues := landCues([]landing{{at: at(0)}, {at: at(149)}, {at: at(150)}, {at: at(299)}})
	if len(cues) != 2 || cues[1].At != at(150) {
		t.Errorf("%+v: 149 ms is too soon, 150 is not", cues)
	}
}

func TestLandsClimbThePentatonicAndTheStepIsTheSpeed(t *testing.T) {
	want := []int{0, 2, 4, 7, 9, 12, 14, 16}
	for i, semi := range want {
		got := Cue{Sound: SoundLand, Step: i}.Speed()
		if math.Abs(got-math.Pow(2, float64(semi)/12)) > 1e-9 {
			t.Errorf("land %d plays at %v, want %d semitones up", i, got, semi)
		}
	}
	if s := (Cue{Sound: SoundLid}).Speed(); s != 1 {
		t.Errorf("the lid plays at %v", s)
	}
	if s := (Cue{Sound: SoundLand, Step: 99}).Speed(); s != math.Pow(2, 19.0/12) {
		t.Errorf("a step past the table plays at %v: it stays on the top note", s)
	}
}

func TestADayWithNothingIsSilentAndSoIsItsCalmFrame(t *testing.T) {
	p := day(1)
	if c := p.SoundCues(false); c != nil {
		t.Errorf("%+v", c)
	}
	if c := p.SoundCues(true); c != nil {
		t.Errorf("%+v", c)
	}
}

func TestWithReduceMotionADaySoundsOnlyTheDoneAtOnce(t *testing.T) {
	cues := day(2, 0, 1, 0).SoundCues(true)
	if len(cues) != 1 || cues[0] != (Cue{Sound: SoundDone}) {
		t.Errorf("%+v", cues)
	}
}

func weekOf(goods ...int) *WeekPlan {
	var s WeekScene
	for d, n := range goods {
		for i := 0; i < n; i++ {
			s.Days[d] = append(s.Days[d], "gem")
		}
	}
	return NewWeekPlan(s)
}

func TestAWeekTicksEachLandingAndEndsOnDone180MsAfterTheLast(t *testing.T) {
	// 110 ms between goods: 220 ms is the least between ticks, so every second good ticks.
	p := weekOf(5)
	cues := p.SoundCues(false)
	ticks := only(cues, SoundTick)
	if len(ticks) != 3 {
		t.Fatalf("%d ticks for 5 goods 110 ms apart, want 3: %+v", len(ticks), ticks)
	}
	for i, c := range ticks {
		if c.At != at(weekStart+i*2*weekGap+weekFall) || c.Step != i {
			t.Errorf("tick %d is %+v", i, c)
		}
	}
	last := p.goods[len(p.goods)-1].start + weekFall
	if end := cues[len(cues)-1]; end != (Cue{At: at(last + 180), Sound: SoundDone}) {
		t.Errorf("last %+v, want the done 180 ms after the last landing at %d", end, last)
	}
}

func TestWeekTicksAreAtLeast220MsApartAndTheDoneFollowsTheLastGoodNotTheLastTick(t *testing.T) {
	p := weekOf(9, 9, 6)
	cues := only(p.SoundCues(false), SoundTick)
	for i := 1; i < len(cues); i++ {
		if cues[i].At-cues[i-1].At < tickGap {
			t.Errorf("ticks %d and %d are %v apart", i-1, i, cues[i].At-cues[i-1].At)
		}
	}
	all := p.SoundCues(false)
	// The last good lands where a tick would not sound, and the done still counts from it.
	if want := at(p.goods[len(p.goods)-1].start + weekFall + 180); all[len(all)-1].At != want {
		t.Errorf("done at %v, want %v", all[len(all)-1].At, want)
	}
}

func TestTheTicksPitchVariesALittleInAFiveStepCycle(t *testing.T) {
	var speeds []float64
	for i := 0; i < 6; i++ {
		speeds = append(speeds, Cue{Sound: SoundTick, Step: i}.Speed())
	}
	if speeds[0] != 1 || speeds[5] != 1 {
		t.Errorf("%v: the cycle returns to 1 after five", speeds)
	}
	for i := 1; i < 5; i++ {
		if math.Abs(speeds[i]-(1+0.06*float64(i))) > 1e-9 {
			t.Errorf("tick %d at %v", i, speeds[i])
		}
	}
}

func TestWithReduceMotionAWeekSoundsOnlyTheDoneAtOnce(t *testing.T) {
	cues := weekOf(3, 2).SoundCues(true)
	if len(cues) != 1 || cues[0] != (Cue{Sound: SoundDone}) {
		t.Errorf("%+v", cues)
	}
}

func TestAWeekWithNothingIsSilent(t *testing.T) {
	if c := weekOf().SoundCues(false); c != nil {
		t.Errorf("%+v", c)
	}
}

func TestEveryCueSoundsBeforeTheSceneEnds(t *testing.T) {
	d, w := day(3, 0, 1, 2, 0, 1), weekOf(4, 0, 3)
	for name, c := range map[string]struct {
		cues []Cue
		end  time.Duration
	}{"day": {d.SoundCues(false), d.End()}, "week": {w.SoundCues(false), w.End()}} {
		for _, cue := range c.cues {
			if cue.At < 0 || cue.At > c.end {
				t.Errorf("%s: %+v outside 0..%v", name, cue, c.end)
			}
		}
	}
}
