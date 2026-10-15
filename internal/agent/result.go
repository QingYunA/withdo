package agent

// Receipt is one change the agent made, in the words the UI shows under the reply.
type Receipt struct {
	ID     int    // for Undo
	TaskID int64  // the task it touched
	Title  string // the task's title after the change
	Detail string // e.g. "Tomorrow 9:00 AM · reminder"
	Status string // the task's status after the change, for the glyph
	Bell   bool   // the change set or moved a reminder
	Undone bool
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
