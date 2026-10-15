package views

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// sheetClosesAfter is how long the conversation sheet stays after a turn that asked nothing.
const sheetClosesAfter = 4 * time.Second

// convoTurn is one exchange in the sheet: what I typed and what came back.
type convoTurn struct {
	user     string
	reply    string
	receipts []data.Receipt
	question *data.Question
	err      string // in plain words, with a Retry
	stopped  bool   // I pressed Esc after some changes were already made
}

// convo is the conversation sheet under the ask field: this conversation only, never stored.
type convo struct {
	turns    []convoTurn
	open     bool
	thinking bool
	stopping bool // Esc was pressed; the answer that comes back is only read for receipts
	cancel   context.CancelFunc
	doneAt   time.Time    // when the last turn came back, for the 4 s close and Undo's lifetime
	undone   map[int]bool // receipts that were undone
	live     bool         // the assistant remembers a conversation, so a second Esc forgets it
}

// assistReady reports whether a model is connected. Without one the ask field adds Inbox tasks.
func (m *Main) assistReady() bool {
	return m.app.Assistant != nil && m.app.Assistant.Configured()
}

// sendToAssistant sends text to the assistant, off the UI thread. The answer comes back through Actions.Post.
func (m *Main) sendToAssistant(text string) {
	cv := &m.convo
	if cv.thinking {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cv.cancel, cv.thinking, cv.stopping, cv.open = cancel, true, false, true
	cv.turns = append(cv.turns, convoTurn{user: text})
	cv.turns = cv.turns[max(0, len(cv.turns)-maxTurnsShown):]
	asst, post := m.app.Assistant, m.app.Act.Post
	go func() {
		reply, err := asst.Ask(ctx, text, data.Place{})
		post(func() { m.answered(text, reply, err) })
	}()
}

// maxTurnsShown keeps the sheet short: a long exchange scrolls the older turns away.
const maxTurnsShown = 3

// answered runs on the UI thread when a turn came back.
func (m *Main) answered(text string, reply data.Reply, err error) {
	cv := &m.convo
	cv.thinking, cv.cancel = false, nil
	cv.doneAt = m.app.Now()
	i := len(cv.turns) - 1
	if i < 0 {
		return
	}
	turn := &cv.turns[i]
	turn.reply, turn.receipts, turn.question = reply.Text, reply.Receipts, reply.Question
	switch {
	case errors.Is(err, context.Canceled):
		m.stopped(text, i)
	case err != nil:
		turn.err = problemText(m.say, err)
	default:
		cv.live = true
	}
	m.app.Refresh()
}

// stopped finishes a turn I cancelled: with nothing changed it is as if I never asked and my words
// go back into the field; with changes made, the sheet stays to show them and offer Undo.
func (m *Main) stopped(text string, i int) {
	cv := &m.convo
	if len(cv.turns[i].receipts) > 0 {
		cv.turns[i].stopped = true
		return
	}
	cv.turns = cv.turns[:i]
	m.ask.buf = text
	m.closeSheet()
}

// closeSheet hides the sheet and forgets what it showed. The assistant still remembers the
// conversation until it idles out or I press Esc again.
func (m *Main) closeSheet() {
	m.convo.open = false
	m.convo.turns = nil
}

// cancelAsk is Esc while the assistant works: the request is cancelled through its context.
func (m *Main) cancelAsk() {
	cv := &m.convo
	if cv.cancel != nil && !cv.stopping {
		cv.stopping = true
		cv.cancel()
	}
}

// forgetConversation is the second Esc: the assistant starts fresh.
func (m *Main) forgetConversation() {
	m.convo.live = false
	if m.app.Assistant != nil {
		m.app.Assistant.Reset()
	}
}

// pick sends an option of a question as my answer.
func (m *Main) pick(option string) { m.sendToAssistant(option) }

// retry sends the failed turn again.
func (m *Main) retry() {
	cv := &m.convo
	if len(cv.turns) == 0 || cv.thinking {
		return
	}
	last := cv.turns[len(cv.turns)-1]
	cv.turns = cv.turns[:len(cv.turns)-1]
	m.sendToAssistant(last.user)
}

// undo reverts one receipt. The row stays, struck through, so it is clear what happened.
func (m *Main) undo(r data.Receipt) {
	cv := &m.convo
	if err := m.app.Assistant.Undo(r.ID); err != nil {
		if n := len(cv.turns); n > 0 {
			cv.turns[n-1].err = m.say.ConvoUndoFailed(strings.TrimSpace(problemText(m.say, err)))
		}
		return
	}
	if cv.undone == nil {
		cv.undone = map[int]bool{}
	}
	cv.undone[r.ID] = true
	cv.doneAt = m.app.Now() // the sheet stays a little longer so the result can be seen
	m.app.Refresh()
}

// sheetDone is true when the last turn is over and asked me nothing: the sheet will close itself.
func (cv *convo) sheetDone() bool {
	if !cv.open || cv.thinking || len(cv.turns) == 0 {
		return false
	}
	last := cv.turns[len(cv.turns)-1]
	return last.question == nil && last.err == "" && !last.stopped
}
