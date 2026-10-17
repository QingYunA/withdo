package notify

import (
	"slices"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

var t0 = time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)

func at(s int) time.Time { return t0.Add(time.Duration(s) * time.Second) }

func TestTwoEventsAreShownOneByOne(t *testing.T) {
	var b Burst
	for i, kind := range []data.EventKind{data.EventHandedBack, data.EventStuck} {
		if step := b.Add(kind, "task/"+string(rune('1'+i)), at(i)); step.Summary || len(step.Retire) != 0 {
			t.Errorf("event %d: %+v, want it shown alone", i+1, step)
		}
	}
}

func TestTheThirdWithinTheWindowBecomesASummaryThatReplacesTheFirstTwo(t *testing.T) {
	var b Burst
	b.Add(data.EventHandedBack, "task/1", at(0))
	b.Add(data.EventHandedBack, "task/2", at(4))
	step := b.Add(data.EventStuck, "task/3", at(10)) // exactly Window after the first
	want := Step{Summary: true, Counts: Counts{HandedBack: 2, Stuck: 1}, Retire: []string{"task/1", "task/2"}}
	if !step.Summary || step.Counts != want.Counts || !slices.Equal(step.Retire, want.Retire) {
		t.Errorf("step = %+v, want %+v", step, want)
	}
}

func TestEventsAfterTheSummaryUpdateItInPlaceUntilTheWindowPassesWithNone(t *testing.T) {
	var b Burst
	b.Add(data.EventHandedBack, "task/1", at(0))
	b.Add(data.EventHandedBack, "task/2", at(1))
	b.Add(data.EventMerged, "task/3", at(2))
	// Each event extends the burst: 9 s after the last is still inside it.
	step := b.Add(data.EventChecksFailing, "task/4", at(11))
	if !step.Summary || step.Counts != (Counts{HandedBack: 2, Merged: 1, ChecksFailing: 1}) || len(step.Retire) != 0 {
		t.Errorf("fourth = %+v", step)
	}
	step = b.Add(data.EventStuck, "task/5", at(20))
	if !step.Summary || step.Counts.Total() != 5 {
		t.Errorf("fifth = %+v", step)
	}
}

func TestAfterTheWindowPassesWithNoEventTheNextOneStartsOver(t *testing.T) {
	var b Burst
	b.Add(data.EventHandedBack, "task/1", at(0))
	b.Add(data.EventHandedBack, "task/2", at(1))
	b.Add(data.EventHandedBack, "task/3", at(2)) // summary
	if step := b.Add(data.EventStuck, "task/4", at(13)); step.Summary {
		t.Errorf("11 s after the last: %+v, want a new burst that shows it alone", step)
	}
	b.Add(data.EventStuck, "task/5", at(14))
	step := b.Add(data.EventStuck, "task/6", at(15))
	if !step.Summary || step.Counts != (Counts{Stuck: 3}) {
		t.Errorf("a second burst = %+v", step)
	}
}

func TestEventsMoreThanTheWindowApartAreNotABurst(t *testing.T) {
	var b Burst
	b.Add(data.EventHandedBack, "task/1", at(0))
	b.Add(data.EventHandedBack, "task/2", at(8))
	// The first is 12 s old, so this is the second event of a burst, not the third.
	if step := b.Add(data.EventHandedBack, "task/3", at(12)); step.Summary {
		t.Errorf("step = %+v, want it shown alone", step)
	}
	// Now the second (at 8 s) is within 10 s of the first one still shown, and the third is 6 s on.
	if step := b.Add(data.EventHandedBack, "task/4", at(14)); !step.Summary || step.Counts.Total() != 3 || !slices.Equal(step.Retire, []string{"task/2", "task/3"}) {
		t.Errorf("step = %+v", step)
	}
}
