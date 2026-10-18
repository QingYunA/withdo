package notify

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/egoist/mygo"
)

// Notifier decides what to show of the events and reminders it is given, and shows it. Everything
// that touches the system is a field, so that a test can stand in for it.
type Notifier struct {
	// Now is the clock the burst is measured with.
	Now func() time.Time
	// Enabled says whether the user has the notifications switched on.
	Enabled func() bool
	// Watching says whether the user is looking at Mine right now: the app is in front and its
	// window shows Mine. What happens there is already on screen.
	Watching func() bool
	// Say gives the messages in the language the app shows.
	Say func() locales.Messages
	// Show shows a notification and returns once the system has it; the first one waits for the
	// user's answer. Close removes the one with an ID.
	Show  func(mygo.NotificationOptions) error
	Close func(id string)
	// DeniedChanged is called when macOS starts or stops refusing the notifications.
	DeniedChanged func()

	// mu keeps events in order and the burst whole while a notification waits for its answer.
	mu     sync.Mutex
	burst  Burst
	denied atomic.Bool
}

// Denied says whether macOS refused the last notification shown.
func (n *Notifier) Denied() bool { return n.denied.Load() }

// Event shows the notification of an event, or the summary of the burst it is part of; or nothing,
// when the switch is off or the user is looking at Mine. It blocks while the system answers.
func (n *Notifier) Event(e data.Event) {
	if !n.Enabled() || n.Watching() {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	opts := Event(n.Say(), e)
	step := n.burst.Add(e.Kind, opts.ID, n.Now())
	if step.Summary {
		for _, id := range step.Retire {
			n.Close(id)
		}
		opts = Summary(n.Say(), step.Counts)
	}
	n.show(opts)
}

// Reminder shows the notification of a due reminder, whatever the user is looking at.
func (n *Notifier) Reminder(r data.Reminder) {
	if !n.Enabled() {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.show(Reminder(n.Say(), r))
}

func (n *Notifier) show(o mygo.NotificationOptions) {
	err := n.Show(o)
	if err != nil {
		log.Printf("notification: %v", err)
	}
	if denied := errors.Is(err, mygo.ErrNotificationsDenied); n.denied.Swap(denied) != denied && n.DeniedChanged != nil {
		n.DeniedChanged()
	}
}

// Announce shows a notification that was decided elsewhere, such as the end-of-day reminder, and
// keeps track of macOS refusing it like any other. It blocks while the system answers.
func (n *Notifier) Announce(o mygo.NotificationOptions) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.show(o)
}
