package views

import (
	"math"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const (
	noticeFor = 1500 * time.Millisecond
	// noticeRoom is the gap under the field. The confirmation sits in it and the section header
	// margin below, so showing it never moves the page.
	noticeRoom float32 = 6
)

// askState is the ask field: what is typed and when the last "Added to Inbox" appeared.
type askState struct {
	buf      string
	noticeAt time.Time
}

// focusAsk takes Cmd+K: it goes to Now and puts the cursor in the ask field, which takes the
// focus when it is drawn, in this frame.
func (m *Main) focusAsk(*ui.Context) {
	m.goTo(nav{})
	m.sel, m.wantAsk = 0, true
}

// askField is the one large input at the top of Now. Enter sends what is typed to the assistant;
// with no model connected it adds the text to the Inbox. Enter on an empty field opens the
// highlighted row.
func (m *Main) askField(c *ui.Context, t *theme.Tokens) {
	focused := m.askRef.Focused(c)
	row := ui.Row(c.Key("ask")).Height(theme.AskHeight).Shrink(0).Padding(0, 14).Gap(10).AlignItems(ui.Center).
		Radius(theme.RadiusPanel).Background(t.BgSubtle).Border(1, ui.Transparent)
	if focused {
		row.Border(1, t.Accent.Alpha(0.5))
	}
	row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	row.Children(func() {
		m.sparkle(c, t)
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			theme.Input.Apply(ui.TextInputBase(c.Key("ask-input"), &m.ask.buf)).TextColor(t.Text).
				Label(m.say.AskLabel()).Bind(&m.askRef).OnTextKey(m.askKey)
			m.takeFocus(c)
			if m.ask.buf == "" {
				theme.Text(c, theme.Input, t.TextTertiary, m.placeholder()).Absolute().Left(0).Top(0).Right(0).Ellipsis("…").PassThrough()
			}
		})
		keyCap(c, t, "⌘K")
	})
	m.notice(c, t)
	m.noModelHint(c, t, focused)
}

// askKey is the keyboard of the ask field: Enter sends, Esc cancels a running request or closes
// the sheet (what Esc does anywhere), Down walks into the rows.
func (m *Main) askKey(k ui.TextKey) bool {
	if k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyEnter:
		m.ask.buf = k.Text()
		m.submitAsk()
		if m.ask.buf != k.Text() {
			k.SetText(m.ask.buf) // what was sent leaves the field
		}
	case ui.KeyEscape:
		m.escape()
	case ui.KeyDown:
		m.queue.move = 1
	default:
		return false
	}
	return true
}

// placeholder is what the empty field says: the invitation, or that the assistant is working, or
// that it waits for an answer.
func (m *Main) placeholder() string {
	switch {
	case m.convo.thinking:
		return m.say.AskThinking()
	case m.convo.open && len(m.convo.turns) > 0 && m.convo.turns[len(m.convo.turns)-1].question != nil:
		return m.say.AskAnswerHint()
	}
	return m.say.AskHint()
}

// sparkle is the accent mark in the field. While the assistant works it pulses, unless Reduce
// Motion is on.
func (m *Main) sparkle(c *ui.Context, t *theme.Tokens) {
	icon := ui.Icon(c, theme.IconSparkle).Size(16, 16).TextColor(t.Accent).Shrink(0)
	if !m.convo.thinking || c.Preferences().ReduceMotion {
		return
	}
	phase := float64(m.app.Now().UnixMilli()%pulsePeriod) / pulsePeriod
	icon.Opacity(float32(0.4 + 0.6*(0.5+0.5*math.Cos(2*math.Pi*phase))))
	c.After(40 * time.Millisecond)
}

// pulsePeriod is one breath of the sparkle, in milliseconds.
const pulsePeriod = 1400

// noModelHint is the one line under the field while no model is connected. It stays once shown
// until I leave the screen, so a click on Set up still finds it after the field loses focus.
func (m *Main) noModelHint(c *ui.Context, t *theme.Tokens, focused bool) {
	if m.app.Assistant == nil {
		return
	}
	if focused && !m.assistReady() {
		m.hintOn = true
	}
	if !m.hintOn || m.assistReady() || m.noticeShowing() {
		return // "Added to Inbox" has the line under the field for 1.5 s
	}
	ui.Row(c.Key("no-model")).Shrink(0).Margin(8, 0, 0, 4).Gap(5).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, m.say.AskNoModel())
		if link(c, t, m.say.AskSetUp()).Clicked() {
			m.openSettings(PaneAssistant)
		}
	})
}

// takeFocus honors Cmd+K. A field focused in the frame it appears, or in the one after, does not
// take text, so after a change of screen it waits until the field has been drawn three times.
func (m *Main) takeFocus(c *ui.Context) {
	m.askFrames++
	switch {
	case !m.wantAsk:
	case m.askFrames > 2:
		m.askRef.Focus(c)
		m.wantAsk = false
	default:
		c.After(16 * time.Millisecond)
	}
}

// keyCap is a small keyboard hint on a quiet outline.
func keyCap(c *ui.Context, t *theme.Tokens, text string) {
	ui.Row(c).Shrink(0).Padding(1, 6).Radius(5).Background(t.Bg).Border(1, t.Border).Children(func() {
		theme.Text(c, theme.Meta, t.TextTertiary, text)
	})
}

// submitAsk sends the typed text to the assistant, or adds it to the Inbox when no model is
// connected, or opens the highlighted row when nothing is typed.
func (m *Main) submitAsk() {
	title := strings.TrimSpace(m.ask.buf)
	if title == "" {
		m.openSelected()
		return
	}
	if m.assistReady() {
		if m.convo.thinking {
			return // one request at a time; what I typed stays in the field
		}
		m.ask.buf = ""
		m.sendToAssistant(title)
		return
	}
	if err := m.app.AddToInbox(title); err != nil {
		return
	}
	m.ask.buf, m.ask.noticeAt = "", m.app.Now()
}

// noticeShowing is true while "Added to Inbox" is on screen.
func (m *Main) noticeShowing() bool {
	return !m.ask.noticeAt.IsZero() && m.app.Now().Sub(m.ask.noticeAt) < noticeFor
}

// notice is the inline "Added to Inbox" confirmation: 1.5 s, fading at the end.
func (m *Main) notice(c *ui.Context, t *theme.Tokens) {
	room := ui.Box(c).Height(noticeRoom).Shrink(0)
	age := m.app.Now().Sub(m.ask.noticeAt)
	if m.ask.noticeAt.IsZero() || age >= noticeFor {
		room.Children(func() {})
		return
	}
	c.After(noticeFor - age)
	opacity := float32(1)
	if age > noticeFor-300*time.Millisecond {
		opacity = float32(noticeFor-age) / float32(300*time.Millisecond)
	}
	room.Children(func() {
		ui.Row(c).Absolute().Left(4).Top(8).Gap(6).Opacity(opacity).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.TextSecondary)
			theme.Text(c, theme.Meta, t.TextSecondary, m.say.AskAdded())
		})
	})
}
