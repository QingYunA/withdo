package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// inspector is the state of the right-hand pane: the text fields that save on blur, and the
// date popovers.
type inspector struct {
	fields   fields
	lastOpen int // kept while the pane slides shut
	dueOpen  bool
	remOpen  bool
	when     time.Time // the date being picked
}

// editing reports whether a text field of the inspector has the keyboard.
func (i *inspector) editing() bool {
	for _, f := range i.fields {
		if f.editing {
			return true
		}
	}
	return false
}

// reverted reports, once, that Esc sent a field back to its stored text: the keys then go to the
// page, so the next Esc closes the pane.
func (i *inspector) reverted() bool {
	any := false
	for _, f := range i.fields {
		any, f.revert = any || f.revert, false
	}
	return any
}

func (i *inspector) reset() {
	i.fields = fields{}
	i.dueOpen, i.remOpen = false, false
}

// inspectorPane slides open at 360 pt when a task is open, and shuts again on Esc.
func (m *Main) inspectorPane(c *ui.Context, t *theme.Tokens) {
	target := float32(0)
	if m.openID != 0 {
		target = theme.InspectorWidth + 1
		m.inspector.lastOpen = m.openID
	}
	pane := ui.Row(c.Key("inspector")).AlignItems(ui.Stretch).Shrink(0).ClipX()
	w := pane.Animate("width", target, theme.MotionFast)
	pane.Width(w)
	if w < 1 {
		return
	}
	pane.Children(func() {
		vline(c, t)
		task, ok := m.app.Task(m.inspector.lastOpen)
		if !ok {
			return
		}
		ui.Column(c).Width(theme.InspectorWidth).Shrink(0).Background(t.Bg).Children(func() {
			m.inspectorHeader(c, t, task)
			hairline(c, t)
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Padding(20, 24, 32).Gap(20).Children(func() {
					m.titleField(c, t, task)
					m.properties(c, t, task)
					m.notesField(c, t, task)
					m.activity(c, t, task)
					m.deleteTaskButton(c, t, task)
				})
			})
			m.inspectorComposer(c, t, task)
		})
	})
}

func (m *Main) inspectorHeader(c *ui.Context, t *theme.Tokens, task data.Task) {
	ui.Row(c).Height(titleBarHeight).Shrink(0).Padding(0, 12, 0, 24).AlignItems(ui.Center).DragWindow().Children(func() {
		name := m.app.projectName(task.ProjectID)
		if name == "" {
			name = m.say.NavInbox()
		}
		theme.Text(c, theme.Meta, t.TextSecondary, name+"  ·  #"+itoa(task.ID))
		ui.Spacer(c)
		btn := ui.ButtonBase(c.Key("close")).Size(theme.HitTarget, theme.HitTarget).Center().Radius(theme.RadiusControl).
			Label(m.say.InspectorClose()).Tooltip(m.say.InspectorCloseTip()).TextColor(t.TextSecondary)
		if btn.Hovered() {
			btn.Background(t.BgHover)
		}
		btn.Children(func() { ui.Icon(c, theme.IconClose).Size(14, 14) })
		if btn.Clicked() {
			m.openID = 0
		}
	})
}

func (m *Main) titleField(c *ui.Context, t *theme.Tokens, task data.Task) {
	title, ok := textField(c, t, m.inspector.fields, fieldSpec{
		name: "Title", label: m.say.FieldTitle(), value: task.Title, style: theme.Heading, color: t.Text,
		multiline: true, enterSaves: true, escReverts: true, minLines: 1, maxLines: 4,
	})
	if ok && title != "" {
		_ = m.app.Update(task.ID, data.Patch{Title: &title})
	}
}

func (m *Main) notesField(c *ui.Context, t *theme.Tokens, task data.Task) {
	notes, ok := textField(c, t, m.inspector.fields, fieldSpec{
		name: "Notes", label: m.say.FieldNotes(), value: task.Notes, style: theme.Body, color: t.Text, placeholder: m.say.NotesPlaceholder(),
		multiline: true, escReverts: true, minLines: 3, maxLines: 12,
	})
	if ok {
		_ = m.app.Update(task.ID, data.Patch{Notes: &notes})
	}
}
