package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// field is the state of one text field that saves on blur or Enter. While the field is not being
// edited it mirrors the stored value, so changes made elsewhere show up at once.
type field struct {
	buf     string
	editing bool
	enter   bool // Enter was pressed in the field
	revert  bool // Esc was pressed in the field: its text went back and the page takes the keys
	wake    bool // an ellipsized field was clicked and its input takes the keyboard as it appears
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
	name        string // the key of the field, whatever the language
	label       string // what assistive technology calls it
	value       string // what the store holds
	style       theme.Style
	color       ui.Color
	placeholder string
	multiline   bool // a text area; Enter saves a single-line title but breaks a line in notes
	enterSaves  bool // Enter saves instead of breaking the line
	escReverts  bool // Esc puts the stored text back and leaves the field
	secret      bool // a password field: the text is hidden
	focus       bool // the field takes the keyboard as it appears
	ellipsis    bool // a single line that ends in … while it is not edited; a click edits it
	minLines    int
	maxLines    int
}

// key is the keyboard of the field: Enter saves, except where it breaks a line, and Esc reverts
// where the field asks for it. Anything else is typing.
func (f *field) key(s fieldSpec) func(ui.TextKey) bool {
	return func(k ui.TextKey) bool {
		if k.Mods != 0 {
			return false
		}
		switch {
		case k.Key == ui.KeyEnter && (!s.multiline || s.enterSaves):
			f.buf, f.enter = k.Text(), true
		case k.Key == ui.KeyEscape && s.escReverts:
			k.SetText(s.value)
			f.buf, f.revert = s.value, true
		default:
			return false
		}
		return true
	}
}

// textField draws an editable text and reports the new text when the user saved it. The field is
// borderless: it reads as text until it is clicked.
func textField(c *ui.Context, t *theme.Tokens, fs fields, s fieldSpec) (saved string, ok bool) {
	f := fs.get(s.name)
	if !f.editing {
		f.buf = s.value
	}
	if s.ellipsis && !s.multiline && !f.editing && !f.wake {
		ellipsized(c, t, f, s)
		return "", false
	}
	box := ui.Box(c.Key("box-" + s.name)).MinWidth(0)
	var el ui.Element
	box.Children(func() {
		if s.multiline {
			el = ui.TextAreaBase(c.Key(s.name), &f.buf).Lines(max(s.minLines, 1), max(s.maxLines, 1))
		} else {
			el = ui.TextInputBase(c.Key(s.name), &f.buf)
		}
		s.style.Apply(el).TextColor(s.color).Label(s.label).OnTextKey(f.key(s))
		if s.secret && !s.multiline {
			el.Password()
		}
		if s.focus || f.wake {
			el.AutoFocus()
		}
		if f.buf == "" && s.placeholder != "" {
			theme.Text(c, s.style, t.TextSecondary, s.placeholder).Absolute().Left(0).Top(0).PassThrough()
		}
	})
	focused := el.Focused()
	blurred := f.editing && !focused
	if focused {
		f.editing, f.wake = true, false
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

// ellipsized shows a single-line field's text the way every long value reads: cut to the room it
// has and ended with an ellipsis. A text input cannot do that, so the field is this text until it
// is clicked (or reached with Tab and Enter), and then the input takes over.
func ellipsized(c *ui.Context, t *theme.Tokens, f *field, s fieldSpec) {
	btn := ui.ButtonBase(c.Key("show-" + s.name)).Label(s.label).Shrink(1).MinWidth(0).Justify(ui.Start).AlignItems(ui.Center).Cursor(ui.CursorText)
	btn.Children(func() {
		text, color := s.value, s.color
		if text == "" {
			text, color = s.placeholder, t.TextTertiary
		}
		theme.Text(c, s.style, color, text).Shrink(1).MinWidth(0).Ellipsis("…")
	})
	if btn.Clicked() {
		f.wake = true
	}
}
