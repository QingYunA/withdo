package theme

import "github.com/egoist/mygo/ui"

// Style is one row of the type scale.
type Style struct {
	Size, Line float32
	Weight     int
	Tracking   float32
	Upper      bool
	Mono       bool
}

// The type scale. A view uses at most three of these.
var (
	Title      = Style{Size: 20, Line: 26, Weight: 600}
	Input      = Style{Size: 15, Line: 20, Weight: 400}
	Body       = Style{Size: 13, Line: 18, Weight: 400}
	BodyStrong = Style{Size: 13, Line: 18, Weight: 500}
	Meta       = Style{Size: 12, Line: 16, Weight: 400}
	MetaStrong = Style{Size: 12, Line: 16, Weight: 500}
	Section    = Style{Size: 11, Line: 14, Weight: 600, Tracking: 0.4, Upper: true}
	Mono       = Style{Size: 11.5, Line: 16, Weight: 400, Mono: true}
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

// Text builds a single line of text in a style and color. Section styles are uppercased.
func Text(c *ui.Context, s Style, color ui.Color, text string) ui.Element {
	if s.Upper {
		text = upper(text)
	}
	return s.Apply(ui.Text(c, text)).TextColor(color).SingleLine()
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
