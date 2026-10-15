package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// noticeFor is how long the "Added to Inbox" line of a plain task stays.
const noticeFor = 1500 * time.Millisecond

// surface is where a composer is drawn: at the top of a page, or at the bottom of the inspector.
type surface int

const (
	surfacePage surface = iota
	surfaceInspector
)

// composer is the state of one composer: what is typed, its keyboard focus, and the confirmation
// of a plain task it added.
type composer struct {
	buf    string
	ref    ui.Handle // the text area, for Cmd+K and for leaving it with Esc
	frames int       // how many frames it has been drawn since it appeared
	task   int       // inspector: the task it was typed for
	// drafts keep what was typed on the pages that were left, so it is there when I come back.
	drafts map[nav]string
	// noticeAt and noticeText are the "Added to …" line of a plain task, for 1.5 s.
	noticeAt   time.Time
	noticeText string
	// setUpHover keeps the "Set up the assistant" line while the pointer is on it.
	setUpHover bool
}

// pageHasComposer reports whether the screen shows a composer: Now, a project and the Inbox do;
// the Completed page and the search results do not.
func (m *Main) pageHasComposer() bool {
	if m.searching() {
		return false
	}
	switch m.nav.screen {
	case screenNow, screenInbox, screenProject:
		return true
	}
	return false
}

// placeOf is where a composer's requests come from.
func (m *Main) placeOf(s surface) data.Place {
	if s == surfaceInspector {
		return data.Place{Kind: data.PlaceTask, TaskID: m.inspector.lastOpen}
	}
	switch m.nav.screen {
	case screenInbox:
		return data.Place{Kind: data.PlaceInbox}
	case screenProject:
		return data.Place{Kind: data.PlaceProject, ProjectID: m.nav.projectID}
	}
	return data.Place{Kind: data.PlaceNow}
}

// focusPageComposer takes Cmd+K: it puts the cursor in the composer of the page, or goes to Now
// when the page has none. The composer takes the focus when it is drawn, in this frame or a few
// later.
func (m *Main) focusPageComposer(*ui.Context) {
	if !m.pageHasComposer() {
		m.goTo(nav{})
	}
	m.sel, m.focusComposer = 0, true
}

// takeFocus honors Cmd+K and "Ask about this task…". A field focused in the frame it appears, or in
// the one after, does not take text, so it waits until the field has been drawn three times.
func (m *Main) takeFocus(c *ui.Context, s surface) {
	cs, want := &m.cmp[s], &m.focusComposer
	if s == surfaceInspector {
		want = &m.focusInspectorComposer
	}
	cs.frames++
	switch {
	case !*want:
	case cs.frames > 2:
		cs.ref.Focus(c)
		*want = false
	default:
		c.After(16 * time.Millisecond)
	}
}

// composerKey is the keyboard of a composer: Enter sends, Shift+Enter breaks the line (the text
// area does it, for a key left alone), Esc cancels a running request, then closes the reply, then
// lets go of the keys, and Down walks from the page's composer into the rows.
func (m *Main) composerKey(s surface) func(ui.TextKey) bool {
	return func(k ui.TextKey) bool {
		if k.Mods == ui.Cmd && k.Key == ui.KeyZ && k.Text() == "" && m.convo.canUndo() {
			return m.undoLast() // nothing typed to undo: Cmd+Z is the assistant's
		}
		if k.Mods != 0 {
			return false
		}
		cs := &m.cmp[s]
		switch {
		case k.Key == ui.KeyEnter:
			cs.buf = k.Text()
			m.submit(s)
			if cs.buf != k.Text() {
				k.SetText(cs.buf) // what was sent leaves the field
			}
		case k.Key == ui.KeyEscape:
			m.composerEscape(s)
		case k.Key == ui.KeyDown && s == surfacePage:
			m.queue.move = 1
		default:
			return false
		}
		return true
	}
}

// composerEscape is Esc in a composer: it cancels a running request, else closes the reply, else
// lets go of the keys. In the page's composer that is also what Esc does anywhere: it closes the
// inspector, clears the selection and makes the assistant forget the conversation.
func (m *Main) composerEscape(s surface) {
	switch {
	case m.convo.thinking:
		m.cancelAsk()
	case m.convo.open && m.convo.owner == s:
		m.closeSheet()
	default:
		m.queue.leave = true
		if s == surfacePage {
			m.escape()
		}
	}
}

// keepDraft sets the page composer's text aside when the page changes, and brings back the draft
// of the page it changes to.
func (cs *composer) keepDraft(from, to nav) {
	if cs.drafts == nil {
		cs.drafts = map[nav]string{}
	}
	if cs.buf == "" {
		delete(cs.drafts, from)
	} else {
		cs.drafts[from] = cs.buf
	}
	cs.buf, cs.frames = cs.drafts[to], 0
}

// submit sends what is typed to the assistant, or adds a plain task when no model is connected,
// or opens the highlighted row when the page's composer is empty.
func (m *Main) submit(s surface) {
	cs := &m.cmp[s]
	text := strings.TrimSpace(cs.buf)
	if text == "" {
		if s == surfacePage {
			m.openSelected()
		}
		return
	}
	if m.assistReady() {
		if m.convo.thinking {
			return // one request at a time; what I typed stays in the field
		}
		cs.buf = ""
		m.sendToAssistant(text, m.placeOf(s), s)
		return
	}
	if s == surfacePage {
		m.addPlain(text)
	}
}

// addPlain makes the text a plain task in the place of the page: its project, else the Inbox.
func (m *Main) addPlain(text string) {
	cs, place := &m.cmp[surfacePage], m.placeOf(surfacePage)
	in := data.CreateTaskInput{Title: strings.Join(strings.Fields(text), " "), Source: data.SourceUser}
	project := ""
	if place.Kind == data.PlaceProject {
		in.ProjectID, project = data.IntPtr(place.ProjectID), m.app.projectName(&place.ProjectID)
	}
	if err := m.app.AddTask(in); err != nil {
		return
	}
	cs.buf, cs.noticeAt, cs.noticeText = "", m.app.Now(), m.say.DetailAdded(m.inboxOr(project))
}
