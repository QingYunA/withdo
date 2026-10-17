package data

// EventKind is what happened that is worth a system notification.
type EventKind int

// The events a report can be: an agent handed a task back or got stuck, and the GitHub poller found
// a pull request's checks failing or the pull request merged.
const (
	EventHandedBack EventKind = iota + 1
	EventStuck
	EventChecksFailing
	EventMerged
)

// Event is a report the app first sees, with what a notification about it needs.
type Event struct {
	Kind        EventKind
	TaskID      int
	Title       string
	ProjectName string // empty for the Inbox
	// Agent is the id the agent reports under; empty for the GitHub events.
	Agent    string
	PRNumber int // 0 when the task has no pull request
	// Detail is what the notification body says when it is not fixed: the agent's last summary for
	// a handed-back task, what a stuck agent waits for.
	Detail string
}
