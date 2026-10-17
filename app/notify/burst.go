package notify

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// Window is how long events count as one burst: the third event within Window of the first one still
// shown turns them into a summary, and a summary ends when Window passes with no event.
const Window = 10 * time.Second

// summaryAt is the number of events that make a burst.
const summaryAt = 3

// Counts say how many events of each kind a summary stands for.
type Counts struct {
	HandedBack, Stuck, ChecksFailing, Merged int
}

// Total is the number of events.
func (c Counts) Total() int { return c.HandedBack + c.Stuck + c.ChecksFailing + c.Merged }

func (c Counts) with(kind data.EventKind) Counts {
	switch kind {
	case data.EventHandedBack:
		c.HandedBack++
	case data.EventStuck:
		c.Stuck++
	case data.EventChecksFailing:
		c.ChecksFailing++
	case data.EventMerged:
		c.Merged++
	}
	return c
}

// shownEvent is an event whose own notification is on screen.
type shownEvent struct {
	id   string
	kind data.EventKind
	at   time.Time
}

// Burst decides, event by event, whether a notification shows the event or a summary of the burst
// it is part of. It reads no clock: the caller passes the time of each event. The zero value is
// ready; it is not safe for concurrent use.
type Burst struct {
	shown       []shownEvent
	summarising bool
	counts      Counts
	last        time.Time
}

// Step is what to do about an event: show it alone, or show the summary of Counts under the
// summary's ID after closing the notifications in Retire, which it replaces.
type Step struct {
	Summary bool
	Counts  Counts
	Retire  []string
}

// Add takes an event whose own notification would have the given id, at time now.
func (b *Burst) Add(kind data.EventKind, id string, now time.Time) Step {
	if b.summarising {
		if now.Sub(b.last) <= Window {
			b.counts, b.last = b.counts.with(kind), now
			return Step{Summary: true, Counts: b.counts}
		}
		*b = Burst{} // a new burst
	}
	recent := b.recent(now)
	if len(recent)+1 < summaryAt {
		b.shown = append(recent, shownEvent{id: id, kind: kind, at: now})
		return Step{}
	}
	counts := Counts{}.with(kind)
	retire := make([]string, 0, len(recent))
	for _, e := range recent {
		counts = counts.with(e.kind)
		retire = append(retire, e.id)
	}
	*b = Burst{summarising: true, counts: counts, last: now}
	return Step{Summary: true, Counts: counts, Retire: retire}
}

// recent are the shown events not more than Window old.
func (b *Burst) recent(now time.Time) []shownEvent {
	out := make([]shownEvent, 0, len(b.shown)+1)
	for _, e := range b.shown {
		if now.Sub(e.at) <= Window {
			out = append(out, e)
		}
	}
	return out
}
