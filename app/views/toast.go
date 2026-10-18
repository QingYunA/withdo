package views

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The window's toasts are drawn here, in the look mygo gives them (the inverse of the theme, a 6 pt
// shadow, the accent-tinted button), so that the toast of a hand-off can have a second button:
// Copy for the agent, beside Undo. A toast whose Data is a handOffCopy has it.

// handOffCopy is the Data of the toast of a hand-off: the sentence its Copy button puts on the
// clipboard.
type handOffCopy struct{ text string }

// toastFade is how long a toast takes to fade out at the end of its time.
const toastFade = 200 * time.Millisecond

// copyForAgent is what to say to an agent to have it do the tasks: "Do adhd-todo task #12." or "Do
// adhd-todo tasks #31, #32 and #33.". It is English on purpose: it is a prompt for an agent, not
// the window's own words.
func copyForAgent(ids []int) string {
	sorted := slices.Sorted(slices.Values(ids))
	marks := make([]string, len(sorted))
	for i, id := range sorted {
		marks[i] = "#" + strconv.Itoa(id)
	}
	switch len(marks) {
	case 0:
		return ""
	case 1:
		return "Do adhd-todo task " + marks[0] + "."
	}
	return "Do adhd-todo tasks " + strings.Join(marks[:len(marks)-1], ", ") + " and " + marks[len(marks)-1] + "."
}

// toastsView builds the toasts of the window.
func (m *Main) toastsView(c *ui.Context, t *theme.Tokens) {
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Bottom(24).AlignItems(ui.Center).Gap(8)
		for _, ts := range toasts {
			m.toast(c, t, viewport, ts)
		}
	})
}

// toast is one toast: its title and its button or buttons, on the inverse of the theme.
func (m *Main) toast(c *ui.Context, t *theme.Tokens, viewport ui.Element, ts ui.Toast) {
	fill, ink := t.Text, t.Bg
	w, _ := c.Size()
	parts := ui.ToastBase(c, ts)
	box := parts.Root.Row().AlignItems(ui.Center).Gap(16).Padding(10, 16).Radius(8).MaxWidth(w-48).
		Background(fill).TextColor(ink)
	box.Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.25))
	if left := parts.Left(); left > 0 && left < toastFade && !viewport.Hovered() {
		box.Opacity(float32(left) / float32(toastFade)) // it fades out at the end of its time
		c.AnimationFrame()
	} else if left > toastFade {
		c.After(left - toastFade)
	}
	button := func(btn ui.Element) ui.Element {
		btn.Padding(4, 10).Radius(6).TextColor(t.Accent.Mix(ink, 0.35)).FontWeight(600)
		if btn.Hovered() {
			btn.Background(ink.Alpha(0.12))
		}
		return btn
	}
	box.Children(func() {
		if ts.Description == "" {
			ui.Text(c, ts.Title)
		} else {
			ui.Column(c).Shrink(1).Gap(2).Children(func() {
				ui.Text(c, ts.Title).FontWeight(600)
				ui.Text(c, ts.Description).TextColor(ink.Alpha(0.75))
			})
		}
		if ts.Action == "" {
			return
		}
		undo := func() {
			button(parts.ActionButton()).Children(func() { ui.Text(c, ts.Action) })
		}
		cp, ok := ts.Data.(handOffCopy)
		if !ok {
			undo()
			return
		}
		ui.Row(c).Shrink(0).Gap(2).AlignItems(ui.Center).Children(func() {
			m.copyButton(c, button, cp)
			undo()
		})
	})
}

// copyButton is Copy for the agent: it puts the sentence on the clipboard, which leaves the toast
// as it is, and reads Copied for a moment.
func (m *Main) copyButton(c *ui.Context, style func(ui.Element) ui.Element, cp handOffCopy) {
	now := m.app.Now()
	copied := !m.ho.copiedAt.IsZero() && now.Sub(m.ho.copiedAt) < copiedFor
	label := m.say.HandOffCopy()
	if copied {
		label = m.say.HandOffCopied()
		c.After(copiedFor - now.Sub(m.ho.copiedAt))
	}
	btn := style(ui.ButtonBase(c.Key("toast-copy")).Label(label).Cursor(ui.CursorPointer))
	btn.Children(func() { ui.Text(c, label) })
	if btn.Clicked() {
		c.WriteClipboard(cp.text)
		m.ho.copiedAt = now
	}
}
