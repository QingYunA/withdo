package notify

import (
	"fmt"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/egoist/mygo"
)

// rig is a Notifier on a fake clock and a fake system that records what was shown and closed.
type rig struct {
	*Notifier
	now      time.Time
	on       bool
	watching bool
	showErr  error
	shown    []mygo.NotificationOptions
	closed   []string
	denials  int
}

func newRig() *rig {
	r := &rig{now: t0, on: true}
	r.Notifier = &Notifier{
		Now:      func() time.Time { return r.now },
		Enabled:  func() bool { return r.on },
		Watching: func() bool { return r.watching },
		Say:      func() locales.Messages { return en },
		Show: func(o mygo.NotificationOptions) error {
			r.shown = append(r.shown, o)
			return r.showErr
		},
		Close:         func(id string) { r.closed = append(r.closed, id) },
		DeniedChanged: func() { r.denials++ },
	}
	return r
}

func (r *rig) event(kind data.EventKind, id int) {
	r.Event(data.Event{Kind: kind, TaskID: id, Title: fmt.Sprintf("Task %d", id), Agent: "claude-code", PRNumber: id})
}

func TestTwoEventsAreTwoNotifications(t *testing.T) {
	r := newRig()
	r.event(data.EventHandedBack, 1)
	r.event(data.EventStuck, 2)
	if len(r.shown) != 2 || r.shown[0].ID != "task/1" || r.shown[1].ID != "task/2" || len(r.closed) != 0 {
		t.Errorf("shown %+v, closed %v", r.shown, r.closed)
	}
}

func TestTheThirdEventReplacesThemWithOneSummaryThatLaterEventsUpdate(t *testing.T) {
	r := newRig()
	r.event(data.EventHandedBack, 1)
	r.now = r.now.Add(2 * time.Second)
	r.event(data.EventHandedBack, 2)
	r.now = r.now.Add(2 * time.Second)
	r.event(data.EventStuck, 3)
	last := r.shown[len(r.shown)-1]
	if last.ID != SummaryID || last.Title != "3 things need you" || last.Body != "2 finished, 1 stuck · open Mine" {
		t.Errorf("summary = %+v", last)
	}
	if len(r.closed) != 2 || r.closed[0] != "task/1" || r.closed[1] != "task/2" {
		t.Errorf("closed = %v, want the first two", r.closed)
	}
	r.now = r.now.Add(5 * time.Second)
	r.event(data.EventMerged, 4)
	if last = r.shown[len(r.shown)-1]; last.ID != SummaryID || last.Title != "4 things need you" || len(r.closed) != 2 {
		t.Errorf("update = %+v, closed %v", last, r.closed)
	}
	// 11 s of quiet end the burst: the next event is shown alone again.
	r.now = r.now.Add(11 * time.Second)
	r.event(data.EventHandedBack, 5)
	if last = r.shown[len(r.shown)-1]; last.ID != "task/5" {
		t.Errorf("after the quiet = %+v", last)
	}
}

func TestNothingIsShownWhileTheUserWatchesMine(t *testing.T) {
	r := newRig()
	r.watching = true
	for i, kind := range []data.EventKind{data.EventHandedBack, data.EventStuck, data.EventChecksFailing, data.EventMerged} {
		r.event(kind, i+1)
	}
	if len(r.shown) != 0 {
		t.Errorf("shown %+v while Mine is in front", r.shown)
	}
	// What was not shown does not count towards a burst.
	r.watching = false
	r.event(data.EventHandedBack, 5)
	r.event(data.EventHandedBack, 6)
	if len(r.shown) != 2 || r.shown[1].ID != "task/6" {
		t.Errorf("shown %+v", r.shown)
	}
	// A reminder always shows.
	r.watching = true
	r.Reminder(data.Reminder{TaskID: 9, Title: "Call back"})
	if got := r.shown[len(r.shown)-1]; got.ID != "task/9" || got.Title != "Reminder: Call back" {
		t.Errorf("reminder = %+v", got)
	}
}

func TestTheSwitchOffSilencesEverythingRemindersIncluded(t *testing.T) {
	r := newRig()
	r.on = false
	r.event(data.EventHandedBack, 1)
	r.Reminder(data.Reminder{TaskID: 2, Title: "x"})
	if len(r.shown) != 0 {
		t.Errorf("shown %+v with the switch off", r.shown)
	}
	r.on = true
	r.event(data.EventHandedBack, 3)
	if len(r.shown) != 1 {
		t.Errorf("shown %+v after switching on", r.shown)
	}
}

func TestDeniedIsKnownAfterARefusedNotificationAndForgottenWhenOneShows(t *testing.T) {
	r := newRig()
	r.showErr = mygo.ErrNotificationsDenied
	r.event(data.EventHandedBack, 1)
	if !r.Denied() || r.denials != 1 {
		t.Errorf("denied %v, told %d times", r.Denied(), r.denials)
	}
	r.event(data.EventHandedBack, 2) // refused again: nothing new to tell
	if r.denials != 1 {
		t.Errorf("told %d times", r.denials)
	}
	r.showErr = nil
	r.Reminder(data.Reminder{TaskID: 3, Title: "x"})
	if r.Denied() || r.denials != 2 {
		t.Errorf("denied %v, told %d times", r.Denied(), r.denials)
	}
}
