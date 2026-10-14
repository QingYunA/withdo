package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const (
	noticeFor = 1500 * time.Millisecond
	askHint   = `Tell me what to remember — "remind me to review PR #12 at 4pm"`
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

// askField is the one large input at the top of Now. Until the agent arrives (M3), Enter adds
// what is typed to the Inbox. Enter on an empty field opens the highlighted row.
func (m *Main) askField(c *ui.Context, t *theme.Tokens) {
	focused := m.askRef.Focused(c)
	row := ui.Row(c.Key("ask")).Height(theme.AskHeight).Shrink(0).Padding(0, 14).Gap(10).AlignItems(ui.Center).
		Radius(theme.RadiusPanel).Background(t.BgSubtle).Border(1, ui.Transparent)
	if focused {
		row.Border(1, t.Accent.Alpha(0.5))
	}
	row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	row.Children(func() {
		ui.Icon(c, theme.IconSparkle).Size(16, 16).TextColor(t.Accent).Shrink(0)
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			in := theme.Input.Apply(ui.TextInputBase(c.Key("ask-input"), &m.ask.buf)).TextColor(t.Text).
				Label("Ask").Bind(&m.askRef)
			m.takeFocus(c)
			if in.Submitted() {
				m.submitAsk()
			}
			if m.ask.buf == "" {
				theme.Text(c, theme.Input, t.TextTertiary, askHint).Absolute().Left(0).Top(0).Right(0).Ellipsis("…").PassThrough()
			}
		})
		keyCap(c, t, "⌘K")
	})
	m.notice(c, t)
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

// submitAsk adds the typed text to the Inbox, or opens the highlighted row when nothing is typed.
func (m *Main) submitAsk() {
	title := strings.TrimSpace(m.ask.buf)
	if title == "" {
		m.openSelected()
		return
	}
	if err := m.app.AddToInbox(title); err != nil {
		return
	}
	m.ask.buf, m.ask.noticeAt = "", m.app.Now()
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
			theme.Text(c, theme.Meta, t.TextSecondary, "Added to Inbox")
		})
	})
}
