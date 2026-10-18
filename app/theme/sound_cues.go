package theme

import (
	"math"
	"slices"
	"time"
)

// The sounds of the Summary scene (ui-spec, Summary > Sound) and when they play. The cues are worked
// out from a plan, not from frames, and apply the spec's restraint: a busy day does not turn into a
// rattle.

// Sound is one of the scene's sounds; each is a file under sounds/.
type Sound string

const (
	SoundLid  Sound = "lid"  // the lid opens
	SoundLand Sound = "land" // a good lands in the bin
	SoundStar Sound = "star" // a starred good lands, over the land
	SoundDone Sound = "done" // everyone hops, or the week ends
	SoundTick Sound = "tick" // a good lands on its stack in the week
)

const (
	landGap    = 150 * time.Millisecond // least between two lands that sound
	landMax    = 8                      // lands that sound in a scene
	tickGap    = 220 * time.Millisecond // least between two ticks
	weekDoneAt = 180 * time.Millisecond // from the last landing of the week to its done
)

// landSemitones are the steps of the major pentatonic that the lands climb, one per land.
var landSemitones = [...]int{0, 2, 4, 7, 9, 12, 14, 16, 19}

// Cue is one sound to make: when after the scene began, which, and its place in the climb.
type Cue struct {
	At    time.Duration
	Sound Sound
	// Step is how many lands (or ticks) sounded before this one.
	Step int
}

// Speed is how fast the player plays the file: a land goes up its step of the pentatonic, a tick
// varies a little, the rest play as they are.
func (c Cue) Speed() float64 {
	switch c.Sound {
	case SoundLand:
		return math.Pow(2, float64(landSemitones[min(c.Step, len(landSemitones)-1)])/12)
	case SoundTick:
		return 1 + float64(c.Step%5)*0.06
	}
	return 1
}

// landing is a good touching down at a time; star is one whose pull request merged.
type landing struct {
	at   time.Duration
	star bool
}

// SoundCues are the cues of a day: the lid, a land (and a star) for the goods that land, and the
// done when everyone hops. With calm (Reduce Motion, the last frame at once) only the done, at once.
func (p *DayPlan) SoundCues(calm bool) []Cue {
	if len(p.flights) == 0 {
		return nil
	}
	done := Cue{At: msec(p.jump), Sound: SoundDone}
	if calm {
		return []Cue{{Sound: SoundDone}}
	}
	lands := make([]landing, len(p.flights))
	for i, f := range p.flights {
		lands[i] = landing{msec(f.land), p.scene.Throws[i].Starred}
	}
	return append(append([]Cue{{At: msec(lidShutTo), Sound: SoundLid}}, landCues(lands)...), done)
}

// SoundCues are the cues of a week: a tick for the goods that land on their stacks, and the done
// shortly after the last one. With calm only the done, at once.
func (p *WeekPlan) SoundCues(calm bool) []Cue {
	if len(p.goods) == 0 {
		return nil
	}
	if calm {
		return []Cue{{Sound: SoundDone}}
	}
	var out []Cue
	var last, sounded time.Duration
	for i, g := range p.goods {
		at := msec(g.start + weekFall)
		last = max(last, at)
		if i == 0 || at-sounded >= tickGap {
			out = append(out, Cue{At: at, Sound: SoundTick, Step: len(out)})
			sounded = at
		}
	}
	return append(out, Cue{At: last + weekDoneAt, Sound: SoundDone})
}

// landCues applies the restraint to the landings, in the order they touch down: a land sounds when
// the last one that sounded was landGap ago, up to landMax, each a step higher; a star sounds only
// with a land that does.
func landCues(lands []landing) []Cue {
	slices.SortStableFunc(lands, func(a, b landing) int { return int(a.at - b.at) })
	var out []Cue
	var sounded time.Duration
	n := 0
	for _, l := range lands {
		if n >= landMax || (n > 0 && l.at-sounded < landGap) {
			continue
		}
		out = append(out, Cue{At: l.at, Sound: SoundLand, Step: n})
		if l.star {
			out = append(out, Cue{At: l.at, Sound: SoundStar})
		}
		sounded = l.at
		n++
	}
	return out
}

func msec(n int) time.Duration { return time.Duration(n) * time.Millisecond }
