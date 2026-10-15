package agent

import "time"

// Receipt is one change the agent made. The UI words it, in the user's language, from Detail.
type Receipt struct {
	ID     int    // for Undo
	TaskID int64  // the task it touched
	Title  string // the task's title after the change
	Detail Detail // what changed, e.g. a reminder tomorrow at 9:00
	Status string // the task's status after the change, for the glyph
	Bell   bool   // the change set or moved a reminder
	Undone bool
}

// DetailKind says what the detail line of a receipt is about.
type DetailKind int

const (
	DetailAdded    DetailKind = iota // a task without a date was added, to Project
	DetailReminder                   // a task was added with a reminder at At
	DetailDue                        // a task was added with a due time At
	DetailUpdated                    // a task was edited; Changes says how
	DetailDone                       // a task was marked done
)

// ChangeKind says which field of a task an update changed.
type ChangeKind int

const (
	ChangeReminder ChangeKind = iota // At is the new time, the zero time when it was cleared
	ChangeDue                        // At is the new time, the zero time when it was cleared
	ChangeStatus                     // Value is the new status
	ChangePriority                   // Value is the new priority
	ChangeProject                    // Value is the project's name, empty for the Inbox
	ChangeRenamed
	ChangeNotes
)

// Change is one field an update changed.
type Change struct {
	Kind  ChangeKind
	At    time.Time
	Value string
}

// Detail is what a receipt says about a change, as data: the UI formats the times and picks the
// words, so this package holds no wording of the user's language.
type Detail struct {
	Kind    DetailKind
	At      time.Time // DetailReminder, DetailDue
	Project string    // DetailAdded: the project's name, empty for the Inbox
	Changes []Change  // DetailUpdated
}

// Question is what ask_user asked: a short question and 2 to 4 answers to pick from.
type Question struct {
	Text    string
	Options []string
}

// Result is what one turn produced. When the turn ended with an error or was cancelled, Receipts
// still lists the changes that were made before it stopped, so they can be undone.
type Result struct {
	Reply    string
	Receipts []Receipt
	Question *Question
}
