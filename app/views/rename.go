package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// renaming is a project being renamed in its sidebar row: the row's label is an input until Enter
// saves, or Esc or a click elsewhere puts the old name back.
type renaming struct {
	id   int
	buf  string
	seen bool // the input has had the keyboard; losing it afterwards ends the edit
	page bool // the input is the page's title, not the sidebar row's label
}

func (m *Main) startRename(id int, name string) {
	m.manage.rename = renaming{id: id, buf: name}
}

// startPageRename is the same edit in the project page's title, which a double click starts.
func (m *Main) startPageRename(id int, name string) {
	m.manage.rename = renaming{id: id, buf: name, page: true}
}

// renameInput is the label of the project row, or the page's title, while it is renamed.
func (m *Main) renameInput(c *ui.Context, t *theme.Tokens, style theme.Style) {
	r := &m.manage.rename
	if focused := m.renameRef.Focused(c); focused {
		r.seen = true
	} else if r.seen {
		*r = renaming{} // a click elsewhere puts the old name back
		return
	}
	box := ui.Box(c).Grow(1).MinWidth(0)
	box.Children(func() {
		style.Apply(ui.TextInputBase(c.Key("rename-input"), &r.buf)).TextColor(t.Text).
			Label(m.say.ManageRenameLabel()).AutoFocus().Bind(&m.renameRef).OnTextKey(m.renameKey)
	})
}

// renameKey is the keyboard of the input: Enter saves, Esc puts the old name back.
func (m *Main) renameKey(k ui.TextKey) bool {
	if k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyEnter:
		r := m.manage.rename
		name := strings.TrimSpace(k.Text())
		if p, ok := m.app.Project(r.id); name != "" && ok && name != p.Name {
			_ = m.app.RenameProject(r.id, name)
		}
		m.manage.rename = renaming{}
	case ui.KeyEscape:
		m.manage.rename = renaming{}
	default:
		return false
	}
	m.queue.leave = true
	return true
}
