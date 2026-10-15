package theme

import (
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Style is one row of the type scale.
type Style struct {
	Size, Line float32
	Weight     int
	Tracking   float32
	Upper      bool
	Mono       bool
}

// The type scale (M4c: one step larger than the first version). A view uses at most three of
// these.
var (
	Title         = Style{Size: 26, Line: 32, Weight: 700}
	Heading       = Style{Size: 21, Line: 28, Weight: 700} // the title of the task in the inspector
	Input         = Style{Size: 17, Line: 25, Weight: 400}
	Body          = Style{Size: 15, Line: 22, Weight: 400}
	BodyStrong    = Style{Size: 15, Line: 22, Weight: 500}
	Compact       = Style{Size: 14, Line: 20, Weight: 400} // sidebar rows, menus, inspector values
	CompactStrong = Style{Size: 14, Line: 20, Weight: 500}
	Meta          = Style{Size: 13, Line: 18, Weight: 400}
	MetaStrong    = Style{Size: 13, Line: 18, Weight: 500}
	Section       = Style{Size: 13, Line: 18, Weight: 600}   // sentence case; callers color it text.secondary
	Badge         = Style{Size: 11.5, Line: 16, Weight: 600} // the P0 / P1 mark
	Mono          = Style{Size: 12.5, Line: 18, Weight: 400, Mono: true}
)

// MonoFamily is the font of branches and pull request numbers.
const MonoFamily = "SF Mono, Menlo, monospace"

// Apply sets a style on a text element or on a container whose texts inherit it.
func (s Style) Apply(e ui.Element) ui.Element {
	e = e.FontSize(s.Size).FixedLineHeight(s.Line).FontWeight(s.Weight)
	if s.Tracking != 0 {
		e = e.LetterSpacing(s.Tracking)
	}
	if s.Mono {
		e = e.Font(MonoFamily)
	}
	return e
}

// Text builds a single line of text in a style and color.
//
// A style with capitals or letter spacing (none of the scale has them now) suits Latin text, not
// Han characters: a text with any gets neither, as "Agent 进行中" stays as it is written.
func Text(c *ui.Context, s Style, color ui.Color, text string) ui.Element {
	if hasHan(text) {
		s.Tracking, s.Upper = 0, false
	}
	if s.Upper {
		text = upper(text)
	}
	return s.Apply(ui.Text(c, text)).TextColor(color).SingleLine()
}

// hasHan reports whether s holds a Han character, which is spaced by its design.
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func upper(s string) string {
	b := []rune(s)
	for i, r := range b {
		if r >= 'a' && r <= 'z' {
			b[i] = r - 32
		}
	}
	return string(b)
}

// Paragraph is Text that wraps, for titles that may take several lines; maxLines 0 means no limit.
func Paragraph(c *ui.Context, s Style, color ui.Color, text string, maxLines int) ui.Element {
	e := s.Apply(ui.Text(c, text)).TextColor(color)
	if maxLines > 0 {
		e = e.MaxLines(maxLines).Ellipsis("…")
	}
	return e
}
