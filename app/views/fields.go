package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// field is the state of one text field that saves on blur or Enter. While the field is not being
// edited it mirrors the stored value, so changes made elsewhere show up at once.
type field struct {
	buf     string
	editing bool
	enter   bool
}

// fields keeps a field per name, for the task in the inspector.
type fields map[string]*field

func (f fields) get(name string) *field {
	if f[name] == nil {
		f[name] = &field{}
	}
	return f[name]
}

// fieldSpec describes a text field.
type fieldSpec struct {
	name        string
	value       string // what the store holds
	style       theme.Style
	color       ui.Color
	placeholder string
	multiline   bool // a text area; Enter saves a single-line title but breaks a line in notes
	enterSaves  bool // Enter saves instead of breaking the line
	secret      bool // a password field: the text is hidden
	minLines    int
	maxLines    int
}

// textField draws an editable text and reports the new text when the user saved it. The field is
// borderless: it reads as text until it is clicked.
func textField(c *ui.Context, t *theme.Tokens, fs fields, s fieldSpec) (saved string, ok bool) {
	f := fs.get(s.name)
	if !f.editing {
		f.buf = s.value
	}
	box := ui.Box(c.Key("box-" + s.name)).MinWidth(0)
	var el ui.Element
	box.Children(func() {
		if s.multiline {
			el = ui.TextAreaBase(c.Key(s.name), &f.buf).Lines(max(s.minLines, 1), max(s.maxLines, 1))
			if s.enterSaves && strings.Contains(f.buf, "\n") {
				// This mygo has no OnEnter: Enter arrives as a newline, which a title never holds.
				f.buf, f.enter = strings.ReplaceAll(f.buf, "\n", ""), true
			}
		} else {
			el = ui.TextInputBase(c.Key(s.name), &f.buf)
			if el.Submitted() {
				f.enter = true
			}
		}
		s.style.Apply(el).TextColor(s.color).Label(s.name)
		if s.secret && !s.multiline {
			el.Password()
		}
		if f.buf == "" && s.placeholder != "" {
			theme.Text(c, s.style, t.TextTertiary, s.placeholder).Absolute().Left(0).Top(0).PassThrough()
		}
	})
	focused := el.Focused()
	blurred := f.editing && !focused
	if focused {
		f.editing = true
	}
	if (f.enter || blurred) && f.buf != s.value {
		saved, ok = f.buf, true
	}
	if blurred {
		f.editing = false
	}
	f.enter = false
	return saved, ok
}
