package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The composer of Home (docs/design/ui-spec.md, Mine (home), v34): the page's main surface. It spans
// both columns and has two rows: the assistant's avatar and the text, then the controls (a picture,
// the microphone, the key that opens the panel, and send).

// Metrics of the Home composer, in points.
const (
	homeAvatar  float32 = 24
	homeTop     float32 = 16 // above the text
	homeRowsGap float32 = 20 // between the text and the controls
	homeBottom  float32 = 8  // under the controls
	homeSide    float32 = 16
	homeControl float32 = 32 // the picture, the microphone and send
	// homeIcon is the picture in each: pixelarticons are drawn on a 24 grid, so 24 pt keeps every
	// pixel whole (18 pt blurred them), and it matches the 24 pt microphone.
	homeIcon     float32 = 24
	homeLineGap  float32 = 12 // between the box and what it says back
	homeLineLeft float32 = 14 // the left edge of the page (title, section headers, glyphs)
)

// homeComposer is the big input of Home.
func (m *Main) homeComposer(c *ui.Context, t *theme.Tokens) {
	s := surfacePage
	cs := &m.cmp[s]
	ready := m.assistReady()
	focused, typed := cs.ref.Focused(c), cs.buf != ""
	holds := typed || (ready && len(cs.pics) > 0)
	working := ready && m.convo.owner == s && (m.convo.thinking || m.replyIn(s, cs.task))
	box := ui.Column(c.Key("composer")).FillWidth().Shrink(0).Padding(homeTop, homeSide, homeBottom).Gap(homeRowsGap).
		Radius(theme.HomeComposerRadius)
	cs.dragOver = ready && box.FileDragOver()
	shadow := ui.RGBA(0, 0, 0, 0.18)
	if t.Dark {
		shadow = ui.RGBA(0, 0, 0, 0.5)
	}
	box.Shadow(0, 12, 32, -12, shadow)
	switch {
	case cs.dragOver:
		box.Background(t.AccentWell).Border(theme.Hairline, t.AccentEdge)
	case focused || holds || working:
		box.Background(t.Bg).Border(theme.Hairline, t.AccentEdge).Shadow(0, 0, 0, 4, t.AccentSoft)
	default:
		box.Background(t.Bg).Border(theme.Hairline, t.BorderControl)
	}
	box.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	if ready {
		if files := box.DroppedFiles(); len(files) > 0 {
			m.attachFiles(s, files)
			m.refocus(s)
		}
	}
	box.Children(func() {
		if ready {
			m.thumbnails(c, t, s)
		}
		ui.Row(c.Key("composer-line")).FillWidth().Shrink(0).Gap(10).AlignItems(ui.Start).Children(func() {
			m.homeAvatar(c, s)
			m.composerField(c, t, s, theme.Input)
		})
		m.homeControls(c, t, s, ready, working)
	})
	if box.Clicked() {
		cs.ref.Focus(c) // a click on the box, not on a control in it, is a click in the field
	}
}

// homeAvatar is the assistant's 24 pt avatar at the left of the text; it pulses while the assistant
// works on a request from the box.
func (m *Main) homeAvatar(c *ui.Context, s surface) {
	icon := theme.AvatarIcon(c.Key("assistant"), theme.Assistant, homeAvatar).Shrink(0)
	m.pulse(c, icon, s)
}

// homeControls is the row under the text: the picture and the microphone, aligned under the text,
// and at the right the key and send (stop while the assistant works).
func (m *Main) homeControls(c *ui.Context, t *theme.Tokens, s surface, ready, working bool) {
	cs := &m.cmp[s]
	// The text starts one avatar and a gap in; the 24 pt pictures sit in 32 pt buttons.
	inset := homeAvatar + 10 - (homeControl-homeIcon)/2
	ui.Row(c.Key("composer-controls")).FillWidth().Shrink(0).Height(homeControl).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(inset - 4).Shrink(0)
		if ready {
			m.homeAttach(c, t, s)
		}
		m.voiceButton(c, t, s)
		ui.Spacer(c)
		key := theme.Text(c, theme.Meta, t.TextSecondary, panelKeyHint()).Shrink(0).Margin(0, 8, 0, 0)
		if all, _, ok := m.modelsReady(); ok {
			m.modelPopover(c, t, s, key, all) // only open when a model could not read the pictures
		} else {
			cs.modelOpen = false
		}
		if working && m.convo.thinking {
			m.homeStop(c, t)
			return
		}
		m.homeSend(c, t, s, strings.TrimSpace(cs.buf) != "" || (ready && len(cs.pics) > 0))
	})
}

// homeAttach is the picture button: it chooses pictures for the assistant.
func (m *Main) homeAttach(c *ui.Context, t *theme.Tokens, s surface) {
	full := len(m.cmp[s].pics)+m.cmp[s].pending >= data.MaxPictures
	label := m.say.ComposerAttach()
	btn := ui.ButtonBase(c.Key("attach-"+itoa(int(s)))).Size(homeControl, homeControl).Shrink(0).Center().
		Radius(theme.RadiusRow).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	ink := t.TextSecondary
	if full {
		ink = t.TextTertiary
	} else if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.Text
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { ui.Icon(c, theme.IconImage).Size(homeIcon, homeIcon).TextColor(ink) })
	if btn.Clicked() && !full {
		m.choosePictures(s)
	}
}

// homeSend is the page's only solid button: slate 3 while there is nothing to send, iris 9 with a
// white arrow once there is.
func (m *Main) homeSend(c *ui.Context, t *theme.Tokens, s surface, holds bool) {
	cs := &m.cmp[s]
	can := holds && !m.convo.thinking && cs.pending == 0
	btn := ui.ButtonBase(c.Key("send-"+itoa(int(s)))).Size(homeControl, homeControl).Shrink(0).Center().
		Radius(theme.RadiusRow).Label(m.say.ComposerSend())
	ink := t.TextTertiary
	switch {
	case can && btn.Hovered():
		btn.Background(t.Accent.Mix(t.Text, 0.12)).Cursor(ui.CursorPointer)
		ink = t.AccentText
	case can:
		btn.Background(t.Accent).Cursor(ui.CursorPointer)
		ink = t.AccentText
	default:
		btn.Background(t.BgSubtle)
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { ui.Icon(c, theme.IconSend).Size(homeIcon, homeIcon).TextColor(ink) })
	if btn.Clicked() && can {
		m.submit(s)
	}
}

// homeStop takes the send button's place while the assistant works on a request from the box.
func (m *Main) homeStop(c *ui.Context, t *theme.Tokens) {
	btn := ui.ButtonBase(c.Key("stop")).Size(homeControl, homeControl).Shrink(0).Center().
		Radius(theme.RadiusRow).Cursor(ui.CursorPointer).Label(m.say.ComposerStop())
	if btn.Hovered() {
		btn.Background(t.BgSelected)
	} else {
		btn.Background(t.BgSubtle)
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { theme.PixelStop(c, theme.IconRow, t.TextSecondary) })
	if btn.Clicked() {
		m.cancelAsk()
	}
}

// homeLines is what the composer says back, under it: the answer, the voice line, the receipt with
// Undo, a picture problem. Each is 12 pt below the box, in 13 pt slate 11, on the page's left edge.
func (m *Main) homeLines(c *ui.Context, t *theme.Tokens) {
	m.homeLine = true
	defer func() { m.homeLine = false }()
	if m.replyIn(surfacePage, 0) {
		m.answerView(c, t, false)
	}
	m.voiceLine(c, t)
	m.notice(c, t)
	m.pictureNotice(c, t, surfacePage)
}

// lineEdge is where a line under a composer starts: its distance from the box and from the left.
func (m *Main) lineEdge() (top, left float32) {
	if m.homeLine {
		return homeLineGap, homeLineLeft
	}
	return 8, 16
}
