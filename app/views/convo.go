package views

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The one-line answer of a turn that asked nothing stays 6 s, the last 400 ms of them a fade.
const (
	answerFor     = 6 * time.Second
	answerFadeFor = 400 * time.Millisecond
)

// convoTurn is one exchange: what I typed (kept for Retry, never shown) and what came back.
type convoTurn struct {
	user     string
	pics     []picture  // what went with it, for Retry and for the field when I stop the request
	place    data.Place // where it was typed: a retry and an answer go out from the same place
	reply    string
	receipts []data.Receipt
	question *data.Question
	err      string // in plain words, with a Retry
	stopped  bool   // I pressed Esc after some changes were already made
}

// convo is the answer under the composer that sent the request: this conversation only, never
// stored. It shows under that composer, on the page or the task it was sent from: the last turn's
// one line, question or error.
type convo struct {
	owner    surface    // the composer that sent the last request
	nav      nav        // the screen it was sent from, for a page composer
	place    data.Place // and the place, for the inspector's
	turns    []convoTurn
	open     bool
	thinking bool
	stopping bool // Esc was pressed; the answer that comes back is only read for receipts
	cancel   context.CancelFunc
	sentAt   time.Time    // when the request in flight was sent, for the seconds of the working line
	doneAt   time.Time    // when the last turn came back, for the 6 s of its line
	typed    string       // what the composer held then: typing anything else ends the line
	undone   map[int]bool // receipts that were undone
	recent   []data.Receipt
	live     bool // the assistant remembers a conversation, so a second Esc forgets it
}

// assistReady reports whether a model is connected. Without one the composer adds plain tasks.
func (m *Main) assistReady() bool {
	return m.app.Assistant != nil && m.app.Assistant.Configured()
}

// sendToAssistant sends text, typed in place into the composer on s, to the assistant, off the UI
// thread. The answer comes back through Actions.Post.
func (m *Main) sendToAssistant(text string, place data.Place, s surface, pics ...picture) {
	cv := &m.convo
	if cv.thinking {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cv.cancel, cv.thinking, cv.stopping, cv.open = cancel, true, false, true
	cv.sentAt = m.app.Now()
	if cv.owner != s || cv.nav != m.nav || cv.place != place {
		cv.turns = nil // another composer, page or task: a new sheet
	}
	cv.owner, cv.nav, cv.place = s, m.nav, place
	cv.turns = append(cv.turns, convoTurn{user: text, pics: pics, place: place})
	cv.turns = cv.turns[max(0, len(cv.turns)-maxTurnsShown):]
	asst, post := m.app.Assistant, m.app.Act.Post
	go func() {
		reply, err := asst.Ask(ctx, text, place, attachments(pics)...)
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
	cv.typed = m.cmp[cv.owner].buf
	cv.recent = append(cv.recent, reply.Receipts...)
	m.markReceipts(reply.Receipts)
	switch {
	case errors.Is(err, context.Canceled):
		m.stopped(text, i)
	case data.ProblemOf(err) == data.ProblemNoImages:
		turn.err = m.say.ProblemNoImages(m.modelName())
		m.cmp[cv.owner].modelOpen = true // to pick a model that can read them; Retry sends them again
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
	m.cmp[cv.owner].pics = cv.turns[i].pics
	cv.turns = cv.turns[:i]
	m.cmp[cv.owner].buf = text
	m.closeSheet()
}

// closeSheet hides the answer and forgets what it showed. The assistant still remembers the
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
	m.convo.live, m.convo.recent = false, nil
	if m.app.Assistant != nil {
		m.app.Assistant.Reset()
	}
}

// pick sends an option of a question as my answer.
func (m *Main) pick(option string) {
	cv := &m.convo
	if len(cv.turns) == 0 {
		return
	}
	m.sendToAssistant(option, cv.turns[len(cv.turns)-1].place, cv.owner)
}

// retry sends the failed turn again.
func (m *Main) retry() {
	cv := &m.convo
	if len(cv.turns) == 0 || cv.thinking {
		return
	}
	last := cv.turns[len(cv.turns)-1]
	cv.turns = cv.turns[:len(cv.turns)-1]
	m.sendToAssistant(last.user, last.place, cv.owner, last.pics...)
}

// undo reverts one receipt. Its line stays, struck through, so it is clear what happened.
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
	m.unmark(r.TaskID)
	cv.doneAt = m.app.Now() // the line stays a little longer so the result can be seen
	m.app.Refresh()
}

// canUndo reports whether the assistant has a change left that Cmd+Z can undo.
func (cv *convo) canUndo() bool {
	for _, r := range cv.recent {
		if !cv.undone[r.ID] {
			return true
		}
	}
	return false
}

// undoLast is Cmd+Z: it undoes the assistant's last change that is not undone, while its line is
// shown or the conversation is live. With the line gone the result shows as "Undone" for a moment.
func (m *Main) undoLast() bool {
	cv := &m.convo
	if m.app.Assistant == nil || cv.thinking {
		return false
	}
	for i := len(cv.recent) - 1; i >= 0; i-- {
		if r := cv.recent[i]; !cv.undone[r.ID] {
			m.undo(r)
			m.showLastTurn()
			return true
		}
	}
	return false
}

// showLastTurn brings the last turn's line back after Cmd+Z, if it is the turn that was undone and
// its composer is still on screen.
func (m *Main) showLastTurn() {
	cv := &m.convo
	if n := len(cv.turns); n > 0 && !cv.open && len(cv.turns[n-1].receipts) > 0 && m.replyFor(cv.owner, cv.place.TaskID) {
		cv.open, cv.typed = true, m.cmp[cv.owner].buf
	}
}

// sheetDone is true when the last turn is over and asked me nothing: its line will go by itself.
func (cv *convo) sheetDone() bool {
	if !cv.open || cv.thinking || len(cv.turns) == 0 {
		return false
	}
	last := cv.turns[len(cv.turns)-1]
	return last.question == nil && last.err == "" && !last.stopped
}

// replyIn reports whether the answer shows under the composer on s: it belongs to the composer
// that sent the request, on the page or the task it was sent from.
func (m *Main) replyIn(s surface, task int) bool {
	return m.convo.open && m.replyFor(s, task)
}

// replyFor is replyIn for an answer that may be hidden already.
func (m *Main) replyFor(s surface, task int) bool {
	cv := &m.convo
	if len(cv.turns) == 0 || cv.owner != s {
		return false
	}
	if s == surfaceInspector {
		return cv.place.TaskID == task
	}
	return cv.nav == m.nav
}

// dropStaleReply closes a reply whose composer is gone from the screen: the page changed, or the
// inspector closed or shows another task. A request still running keeps its sheet until it ends.
func (m *Main) dropStaleReply() {
	cv := &m.convo
	if !cv.open || cv.thinking {
		return
	}
	shown := cv.nav == m.nav && m.pageHasComposer()
	if cv.owner == surfaceInspector {
		shown = m.openID != 0 && cv.place.TaskID == m.openID && m.assistReady()
	}
	if !shown {
		m.closeSheet()
	}
}
