package views

import (
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// hintShownAbove is the content width under which the card drops its keyboard hint.
const hintShownAbove float32 = 560

// pageComposer is the composer at the top of Now, a project or the Inbox, with the "Added to …"
// confirmation under it and the reply of the request it sent.
func (m *Main) pageComposer(c *ui.Context, t *theme.Tokens) {
	if !m.pageHasComposer() {
		return
	}
	m.composerCard(c, t)
	m.notice(c, t)
	if m.replyIn(surfacePage, 0) {
		m.convoSheet(c, t, false)
	}
}

// composerCard is the card: the text area, 2 lines tall empty and growing to 6, and under it the
// bar with the place's chip, the keyboard hint and the round send button.
func (m *Main) composerCard(c *ui.Context, t *theme.Tokens) {
	cs := &m.cmp[surfacePage]
	focused := cs.ref.Focused(c)
	card := ui.Column(c.Key("composer")).Shrink(0).Padding(14, 16, 10).Radius(theme.ComposerRadius).Background(t.Bg)
	if focused {
		card.Border(1, t.Accent.Alpha(0.55)).Shadow(0, 0, 0, 3, t.Accent.Alpha(0.12))
	} else {
		card.Border(1, t.ComposerBorder()).Shadow(0, 4, 16, 0, t.Shadow.Alpha(0.06))
	}
	card.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	card.Children(func() {
		m.composerField(c, t, surfacePage, theme.Input)
		ui.Row(c).Height(theme.SendSize).Shrink(0).Margin(6, 0, 0).Gap(8).AlignItems(ui.Center).Children(func() {
			m.composerChip(c, t)
			ui.Spacer(c)
			if m.width >= hintShownAbove && !(m.convo.thinking && m.convo.owner == surfacePage) {
				m.composerHint(c, t, focused, cs.buf != "")
			}
			m.sendButton(c, t, surfacePage)
		})
	})
	if card.Clicked() {
		cs.ref.Focus(c) // a click on the card, not on a control in it, is a click in the field
	}
}

// composerHint is the keyboard hint at the right of the bar, as little as the moment needs: only
// the ⌘K key cap while the field is not focused, "↵ to send" once it is, and the line break as well
// when there is text to break.
func (m *Main) composerHint(c *ui.Context, t *theme.Tokens, focused, typed bool) {
	switch {
	case !focused:
		keyCap(c, t, m.say.ComposerHintIdle())
	case typed:
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.ComposerHintTyped()).Shrink(1).MinWidth(0).Ellipsis("…")
	default:
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.ComposerHintSend()).Shrink(1).MinWidth(0).Ellipsis("…")
	}
}

// keyCap is a shortcut in a small outlined cap.
func keyCap(c *ui.Context, t *theme.Tokens, text string) {
	ui.Row(c).Shrink(0).Padding(1, 6).Radius(5).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Meta, t.TextTertiary, text)
	})
}

// composerField is the text area of a composer with its placeholder over it.
func (m *Main) composerField(c *ui.Context, t *theme.Tokens, s surface, style theme.Style) {
	cs := &m.cmp[s]
	label, key := m.say.ComposerLabel(), "composer-input"
	if s == surfaceInspector {
		label, key = m.say.ComposerLabelTask(), "composer-input-task"
	}
	box := ui.Box(c.Key(key + "-box")).Shrink(0).MinWidth(0).Grow(1)
	box.Children(func() {
		style.Apply(ui.TextAreaBase(c.Key(key), &cs.buf).Lines(theme.ComposerMinRows, theme.ComposerMaxRows)).
			TextColor(t.Text).Label(label).Bind(&cs.ref).OnTextKey(m.composerKey(s))
		m.takeFocus(c, s)
		if cs.buf == "" {
			theme.Paragraph(c, style, t.TextTertiary, m.placeholder(s), 2).Absolute().Left(0).Top(0).Right(0).PassThrough()
		}
	})
}

// placeholder is what the empty field says: where it is, or that the assistant is working, or that
// it waits for an answer.
func (m *Main) placeholder(s surface) string {
	cv := &m.convo
	switch {
	case cv.thinking && cv.owner == s:
		return m.say.AskThinking()
	case cv.open && cv.owner == s && len(cv.turns) > 0 && cv.turns[len(cv.turns)-1].question != nil:
		return m.say.AskAnswerHint()
	}
	switch place := m.placeOf(s); place.Kind {
	case data.PlaceProject:
		return m.say.ComposerPlaceholderProject(m.app.projectName(&place.ProjectID))
	case data.PlaceTask:
		return m.say.ComposerPlaceholderTask()
	}
	return m.say.ComposerPlaceholder()
}

// composerChip names the place new tasks go to: a project, the Inbox, or on Now the assistant and
// its model. While no model is connected it offers to set one up.
func (m *Main) composerChip(c *ui.Context, t *theme.Tokens) {
	place := m.placeOf(surfacePage)
	setUp := m.app.Assistant != nil && !m.assistReady()
	switch {
	case place.Kind == data.PlaceProject:
		chip(c, t, "chip", func() { theme.Text(c, theme.Meta, t.TextSecondary, "# "+m.app.projectName(&place.ProjectID)) })
	case place.Kind == data.PlaceInbox:
		chip(c, t, "chip", func() { theme.Text(c, theme.Meta, t.TextSecondary, m.say.NavInbox()) })
	case m.assistReady():
		chip(c, t, "chip", func() {
			m.sparkle(c, t, surfacePage, 14)
			theme.Text(c, theme.Meta, t.TextSecondary, m.assistantChip())
		})
	case !setUp:
		chip(c, t, "chip", func() { theme.Text(c, theme.Meta, t.TextSecondary, m.say.NavInbox()) })
	}
	if setUp {
		if link(c, t, m.say.ComposerChipSetUp()).Shrink(0).Clicked() {
			m.openSettings(PaneAssistant)
		}
	}
}

// chip is a small label on a quiet well.
func chip(c *ui.Context, t *theme.Tokens, key string, body func()) ui.Element {
	return ui.Row(c.Key(key)).Shrink(0).Padding(2, 8).Gap(5).AlignItems(ui.Center).Radius(theme.ChipRadius).
		Background(t.BgSubtle).Children(body)
}

// assistantChip is "Assistant · Haiku 5.5", or "Assistant" when the model's name is not known.
func (m *Main) assistantChip() string {
	if m.app.Models != nil {
		for _, p := range m.app.Models.Providers() {
			if p.Active && p.Model != "" {
				return m.say.ComposerChipAssistant(shortModel(p.Model))
			}
		}
	}
	return m.say.ComposerChipAssistantBare()
}

// shortModel is a model's id as a person says it: "claude-haiku-5-5" is "Haiku 5.5". Any other
// id is left as it is.
func shortModel(id string) string {
	rest, ok := strings.CutPrefix(id, "claude-")
	if !ok {
		return id
	}
	var words, nums []string
	for _, part := range strings.Split(rest, "-") {
		if part != "" && unicode.IsDigit(rune(part[0])) {
			nums = append(nums, part)
			continue
		}
		words = append(words, strings.ToUpper(part[:1])+part[1:])
	}
	if len(nums) == 0 || len(words) == 0 {
		return id
	}
	return strings.Join(words, " ") + " " + strings.Join(nums, ".")
}

// sendButton is the round button at the right of the bar: accent when there is something to send,
// neutral when not.
func (m *Main) sendButton(c *ui.Context, t *theme.Tokens, s surface) {
	can := strings.TrimSpace(m.cmp[s].buf) != "" && !m.convo.thinking
	btn := ui.ButtonBase(c.Key("send-"+itoa(int(s)))).Size(theme.SendSize, theme.SendSize).Shrink(0).Center().
		Radius(theme.SendSize / 2).Label(m.say.ComposerSend())
	switch {
	case can && btn.Hovered():
		btn.Background(t.Accent.Mix(t.Text, 0.12)).Cursor(ui.CursorPointer)
	case can:
		btn.Background(t.Accent).Cursor(ui.CursorPointer)
	default:
		btn.Background(t.SendIdle())
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { ui.Icon(c, theme.IconSend).Size(16, 16).TextColor(t.AccentText) })
	if btn.Clicked() && can {
		m.submit(s)
	}
}

// sparkle is the accent mark of the assistant. While it works on a request from s it pulses,
// unless Reduce Motion is on.
func (m *Main) sparkle(c *ui.Context, t *theme.Tokens, s surface, size float32) {
	icon := ui.Icon(c, theme.IconSparkle).Size(size, size).TextColor(t.Accent).Shrink(0)
	if !m.convo.thinking || m.convo.owner != s || c.Preferences().ReduceMotion {
		return
	}
	phase := float64(m.app.Now().UnixMilli()%pulsePeriod) / pulsePeriod
	icon.Opacity(float32(0.4 + 0.6*(0.5+0.5*math.Cos(2*math.Pi*phase))))
	c.After(40 * time.Millisecond)
}

// pulsePeriod is one breath of the sparkle, in milliseconds.
const pulsePeriod = 1400

// notice is the inline "Added to …" confirmation of a plain task: 1.5 s, fading at the end.
func (m *Main) notice(c *ui.Context, t *theme.Tokens) {
	cs := &m.cmp[surfacePage]
	room := ui.Box(c).Height(noticeRoom).Shrink(0)
	age := m.app.Now().Sub(cs.noticeAt)
	if cs.noticeAt.IsZero() || age >= noticeFor {
		room.Children(func() {})
		return
	}
	c.After(noticeFor - age)
	opacity := float32(1)
	if age > noticeFor-300*time.Millisecond {
		opacity = float32(noticeFor-age) / float32(300*time.Millisecond)
	}
	room.Children(func() {
		ui.Row(c).Absolute().Left(4).Top(10).Gap(6).Opacity(opacity).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.TextSecondary)
			theme.Text(c, theme.Meta, t.TextSecondary, cs.noticeText)
		})
	})
}

// inspectorComposer is the compact composer at the bottom of the inspector: it edits the open
// task in plain words. The reply of its request shows above it. It is not drawn while no model is
// connected.
func (m *Main) inspectorComposer(c *ui.Context, t *theme.Tokens, task data.Task) {
	if !m.assistReady() {
		return
	}
	cs := &m.cmp[surfaceInspector]
	if cs.task != task.ID {
		cs.task, cs.buf, cs.frames = task.ID, "", 0
	}
	focused := cs.ref.Focused(c)
	hairline(c, t) // the properties scroll above it; this stack stays where it is
	ui.Column(c.Key("inspector-composer")).Shrink(0).Padding(12, 14, 14).Children(func() {
		if m.replyIn(surfaceInspector, task.ID) {
			// The words line up with the properties above (24 pt); a receipt's well reaches out to the card's edge.
			ui.Scroll(c.Key("inspector-reply")).MaxHeight(inspectorReplyMax).Margin(0, 0, 10).Children(func() {
				ui.Column(c).Padding(0, 10).Children(func() { m.convoSheet(c, t, true) })
			})
		}
		row := ui.Row(c.Key("inspector-card")).Shrink(0).Padding(10, 10, 10, 14).Gap(10).AlignItems(ui.Center).
			Radius(theme.ComposerRadius).Background(t.Bg)
		if focused {
			row.Border(1, t.Accent.Alpha(0.55)).Shadow(0, 0, 0, 3, t.Accent.Alpha(0.12))
		} else {
			row.Border(1, t.ComposerBorder())
		}
		row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
		row.Children(func() {
			m.sparkle(c, t, surfaceInspector, 16)
			m.composerField(c, t, surfaceInspector, theme.Body)
			m.sendButton(c, t, surfaceInspector)
		})
		if row.Clicked() {
			cs.ref.Focus(c)
		}
	})
}

// inspectorReplyMax keeps a long reply from pushing the properties out of the inspector.
const inspectorReplyMax float32 = 240
