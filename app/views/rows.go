package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// rowStyle is the padding of a row. Rows bleed their highlight past the column edge by the same
// amount, so the text stays on the column's left edge.
type rowStyle struct{ vert, horz float32 }

// rowGap is the space between the columns of a row: glyph or avatar, body, right end.
const rowGap float32 = 12

var (
	plainRow = rowStyle{9, 10}
	needRow  = rowStyle{10, 8} // inside the Needs you container
)

// rowShell is a row of the Now screen or a list: hover, the keyboard highlight, and the fade and
// fold of a task that was just marked done, or the opening of one that just joined Done. The
// caller fills it and reports clicks.
func (m *Main) rowShell(c *ui.Context, t *theme.Tokens, id int, st rowStyle, ph phase) ui.Element {
	box := ui.Column(c.Key(foldKey(id, ph.done))).Shrink(0)
	var row ui.Element
	box.Children(func() { row = m.plainShell(c, t, id, st, ph.opacity) })
	closeTo(box, row, ph.height)
	return row
}

// plainShell is the row inside its box.
func (m *Main) plainShell(c *ui.Context, t *theme.Tokens, id int, st rowStyle, opacity float32) ui.Element {
	row := ui.Row(c.Key(id)).Shrink(0).Margin(0, -st.horz).Padding(st.vert, st.horz).Gap(rowGap).
		Radius(theme.RadiusCard).AlignItems(ui.Stretch).Cursor(ui.CursorPointer)
	picked := m.multi.Has(id)
	switch {
	case picked || m.sel == id || m.openID == id || (m.pop.open && m.pop.task == id):
		row.Background(t.BgSelected)
	case row.Hovered():
		row.Background(t.BgHover)
	}
	if picked {
		m.pickedRow(t, row, id)
	}
	if m.sel == id {
		m.selRect = row.Bounds()
		if m.moved {
			row.ScrollIntoView()
			m.moved = false
		}
	}
	if opacity < 1 {
		row.Opacity(opacity)
	}
	if row.RightClicked() {
		m.rowMenu(row, id)
	}
	return row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
}

// pickedRow gives a row of the selection its look: rows next to each other in it join into one block,
// with no rounded corners between them, the keyboard row has a ring, and assistive technology reads
// it as selected.
func (m *Main) pickedRow(t *theme.Tokens, row ui.Element, id int) {
	r := row.Bounds()
	m.rects[id] = r
	above, below := m.joinedNeighbours(id, r)
	top, bottom := theme.RadiusCard, theme.RadiusCard
	if above {
		top = 0
	}
	if below {
		bottom = 0
	}
	row.Radius(top, top, bottom, bottom).Checked(true)
	if m.sel == id {
		row.DrawOver(func(p *ui.Painter, box ui.Rect) {
			p.Stroke(box, t.Accent.Alpha(0.55), theme.RadiusCard, 1.5)
		})
	}
}

// joinedNeighbours says whether a selected row sits right above and right below this one, by where
// the rows were drawn the frame before.
func (m *Main) joinedNeighbours(id int, r ui.Rect) (above, below bool) {
	for other, o := range m.prevRects {
		if other == id || !m.multi.Has(other) || o.X+o.W <= r.X || o.X >= r.X+r.W {
			continue
		}
		if gap := r.Y - (o.Y + o.H); gap > -1.5 && gap < 1.5 {
			above = true
		}
		if gap := o.Y - (r.Y + r.H); gap > -1.5 && gap < 1.5 {
			below = true
		}
	}
	return above, below
}

// openRow opens a row's task in the inspector and highlights it.
func (m *Main) openRow(id int) {
	m.openID, m.sel = id, id
	m.multi.Open(id)
	m.openFail = openFailure{}
	m.inspector.reset()
}

// body is the column that grows between a row's glyph and its right end.
func body(c *ui.Context, fn func()) {
	ui.Column(c).Grow(1).MinWidth(0).Children(fn)
}

// Content widths below which a row drops its companions: first the project name, then the branch
// and pull request, so the title keeps its room beside the inspector.
const (
	dropProjectBelow float32 = 600
	dropPillsBelow   float32 = 460
)

// titleLine is a title with its quiet companions: a branch, a pull request, a project name. The
// Needs you rows leave the pills out: their summary line already says what is waiting.
func (m *Main) titleLine(c *ui.Context, t *theme.Tokens, task data.Task, project string, pills bool) {
	ui.Row(c).Gap(8).MinWidth(0).AlignItems(ui.Center).Children(func() {
		// In a tight row the project name gives way first, then the branch, and the title last.
		theme.Text(c, theme.BodyStrong, t.Text, task.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		if task.Branch != "" && pills && m.width >= dropPillsBelow {
			pill(c, t, task.Branch).MaxWidth(180).Shrink(4)
		}
		if task.PRNumber > 0 && pills && m.width >= dropPillsBelow {
			pill(c, t, "#"+itoa(task.PRNumber)).Shrink(0)
		}
		if project != "" && m.width >= dropProjectBelow {
			theme.Text(c, theme.Meta, t.TextTertiary, project).Shrink(8).MinWidth(0).Ellipsis("…")
		}
	})
}

// summaryLine is the second line of a row.
func summaryLine(c *ui.Context, t *theme.Tokens, text string) {
	if text != "" {
		theme.Text(c, theme.Meta, t.TextSecondary, text).Margin(2, 0, 0).Ellipsis("…")
	}
}

// right is the cluster at a row's right end, centered against the row's height.
func right(c *ui.Context, fn func()) {
	ui.Row(c).Shrink(0).Gap(10).AlignItems(ui.Center).Children(fn)
}

// glyphSlot puts the status glyph button where the mockup draws the bare glyph: its 24 pt hit
// target reaches past the 18 pt glyph, so it takes negative margins.
func (m *Main) glyphSlot(c *ui.Context, t *theme.Tokens, task data.Task, fill float32) {
	pad := (theme.HitTarget - theme.GlyphSize) / 2
	if m.multi.Picks(task.ID) {
		m.checkSlot(c, t, task.ID).Margin(-pad)
		return
	}
	m.glyphButton(c, t, task, fill).Margin(-pad)
}

// glyphSlotTop is glyphSlot for a row whose lines start at its top (a title with a summary under
// it): the glyph sits 2 pt lower, level with the middle of the first line.
func (m *Main) glyphSlotTop(c *ui.Context, t *theme.Tokens, task data.Task, fill float32) {
	pad := (theme.HitTarget - theme.GlyphSize) / 2
	if m.multi.Picks(task.ID) {
		m.checkSlot(c, t, task.ID).Margin(2-pad, -pad, -pad-2, -pad)
		return
	}
	m.glyphButton(c, t, task, fill).Margin(2-pad, -pad, -pad-2, -pad)
}

// groupGlyph is the status glyph button of a row in a status group. The group says the status, so
// at rest the row has none; while the pointer is over the row or in the glyph column beside it, its
// task menu is open or its completion plays, the button shows under the header's glyph, left of the
// row's avatar, so a task can be completed from the list. It takes no room (its margins cancel its
// width and the row's gap), so the avatar and the title stay where they are. top is for a row whose
// lines start at its top.
//
// The button lies partly left of the row, so an empty zone of its size and the row's height stands
// there at rest, and the pointer in it counts as over the row. The zone has no handler, label or
// focus: a click on it is the row's. It asks where the pointer is rather than for its hover, as a
// press on the button ends the hover of the row (the pointer is not over the row) but not the fact
// that the pointer is in the zone.
func (m *Main) groupGlyph(c *ui.Context, t *theme.Tokens, row ui.Element, task data.Task, ph phase, top bool) {
	if m.multi.Picks(task.ID) || row.Dragging() {
		return
	}
	pad := (theme.HitTarget - theme.GlyphSize) / 2
	left := -(listIndent + pad) // centered under the header's glyph
	right := -(theme.HitTarget + rowGap) - left
	zone := ui.Box(c.Key("glyph-zone")).Width(theme.HitTarget).AlignSelf(ui.Stretch).Shrink(0).Margin(0, right, 0, left)
	_, _, inZone := zone.PointerPosition()
	menu := m.pop.open && m.pop.task == task.ID
	if !(row.Hovered() || inZone || ph.moving || menu) {
		return
	}
	btn := m.glyphButton(c, t, task, ph.fill)
	if top {
		btn.Margin(2-pad, right, -pad-2, left)
	} else {
		btn.Margin(-pad, right, -pad, left)
	}
}

// ageLabel is how long ago, in the stale color when the agent went quiet.
func (m *Main) ageLabel(c *ui.Context, t *theme.Tokens, it data.Item) {
	color := t.TextTertiary
	if it.Stale {
		color = t.Stale
	}
	theme.Text(c, theme.Meta, color, m.clock.Age(it.Since, m.app.Now())).Shrink(0)
}

// liveDot is the 6 pt doing dot with a soft ring, drawn while an agent reported in the last ten
// minutes.
func liveDot(c *ui.Context, t *theme.Tokens) {
	ui.Box(c).Size(12, 12).Shrink(0).Radius(6).Background(t.Status(data.Doing).Alpha(0.18)).Center().Children(func() {
		theme.Dot(c, t.Status(data.Doing), 6)
	})
}

// actionButton is the small tinted button at the right of a handed-over row.
func actionButton(c *ui.Context, t *theme.Tokens, key int, text, label string) ui.Element {
	return actionButtonKeyed(c, t, "action-"+itoa(key), text, label)
}

// actionButtonKeyed is actionButton for a row with several: each takes its own key.
func actionButtonKeyed(c *ui.Context, t *theme.Tokens, key, text, label string) ui.Element {
	return stampButton(c, t, key, text, label, -1, nil)
}

// stampButton is actionButtonKeyed that can be stamped: from frame 0 on it turns status.done, its
// tint at 16 %, and draws the pixel check (12 pt) before the word, a frame at a time. A frame below 0
// is the plain button. The change of color is at once: the stamp is a stroke, not a fade. A button with an
// icon draws it (12 pt, in the color of the word) before the word.
func stampButton(c *ui.Context, t *theme.Tokens, key, text, label string, frame int, icon *ui.SVG) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Padding(3, 9).Radius(theme.RadiusControl).Shrink(0).
		Background(t.BgSelected).Label(label).Cursor(ui.CursorPointer)
	if frame >= 0 {
		ink := t.Status(data.Done)
		btn.Background(ink.Alpha(0.16)).Children(func() {
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				theme.Stamp(c, ink, frame)
				theme.Text(c, theme.MetaStrong, ink, text)
			})
		})
		return btn
	}
	if btn.Hovered() {
		btn.Background(t.Accent.Alpha(0.2))
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() {
		if icon == nil {
			theme.Text(c, theme.MetaStrong, t.Accent, text)
			return
		}
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Accent)
			theme.Text(c, theme.MetaStrong, t.Accent, text)
		})
	})
	return btn
}
