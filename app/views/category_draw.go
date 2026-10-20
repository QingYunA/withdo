package views

import (
	"image"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// The icon editor of the category sheet (v24 frame 7b): a 12 x 12 grid of 18 pt cells with the
// palette under it and the tools beside it. It edits the sheet's icon in place.

const (
	drawCell      float32 = 18
	drawSide              = pixmap.Size*drawCell + 1 // a 1 pt line before each cell and after the last
	paletteSwatch float32 = 22
	toolButton    float32 = 26
	toolsGap      float32 = 16
)

// drawing is the editor's state: the picked pen (pixmap.Own or a palette letter), what ⌘Z goes
// back through, the cell the keyboard is on and the stroke the pointer is making.
type drawing struct {
	pen    byte
	undo   []pixmap.Picture // the icon before each step, the last step last
	at     image.Point
	stroke stroke
}

// stroke is a press that has not been released. It paints with pen, or erases when pen is 0, and
// remembers the cell it last reached so a fast drag leaves no gap.
type stroke struct {
	on  bool
	pen byte
	at  image.Point
}

// startDraw opens the editor on the icon as it is.
func (s *categorySheet) startDraw() {
	s.mode, s.draw, s.iconErr = modeDraw, drawing{pen: pixmap.Own}, nil
}

// edit makes next the icon, as one step ⌘Z takes back.
func (s *categorySheet) edit(next pixmap.Picture) {
	if next == s.icon {
		return
	}
	s.draw.undo = append(s.draw.undo, s.icon)
	s.icon = next
}

// undoStep takes back the last step, if there is one.
func (s *categorySheet) undoStep() {
	if n := len(s.draw.undo); n > 0 {
		s.icon, s.draw.undo = s.draw.undo[n-1], s.draw.undo[:n-1]
	}
}

// penAt is what a press on cell p does: it paints with the picked pen, unless the cell has that
// color already, which erases it.
func (s *categorySheet) penAt(p image.Point) byte {
	if s.icon.At(p.X, p.Y) == s.draw.pen {
		return 0
	}
	return s.draw.pen
}

// categoryDraw is the editor: grid and tools side by side, the palette, a line of help.
func (m *Main) categoryDraw(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	editor := ui.Column(c.Key("draw-editor")).Gap(8)
	if editor.Shortcut(ui.Cmd, ui.KeyZ) {
		s.undoStep()
	}
	editor.Children(func() {
		ui.Row(c).Gap(toolsGap).AlignItems(ui.Start).Children(func() {
			m.drawGrid(c, t, s)
			m.drawTools(c, t, s)
		})
		m.drawPalette(c, t, s)
		theme.Paragraph(c, theme.Meta, t.TextSecondary, m.say.CategoryDrawHelp(keyHint("⌘Z")), 0)
	})
}

// drawGrid is the 12 x 12 cells. The pointer paints, erases and drags; with the keyboard on it the
// arrows move a cell and Space toggles it.
func (m *Main) drawGrid(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	grid := ui.Box(c.Key("draw-grid")).Size(drawSide, drawSide).Shrink(0).Focusable().FocusRing(false).
		Label(m.say.CategoryDrawing()).Cursor(ui.CursorCrosshair)
	m.drawPointer(grid, s)
	m.drawKeys(grid, s)
	icon, own, at, focus := s.icon, theme.CategoryColor(s.color), s.draw.at, grid.FocusVisible()
	grid.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, t.Border, 0)
		for y := 0; y < pixmap.Size; y++ {
			for x := 0; x < pixmap.Size; x++ {
				ink := t.Bg
				switch v := icon.At(x, y); v {
				case 0:
				case pixmap.Own:
					ink = own
				default:
					ink = theme.PaletteColor(v)
				}
				p.Fill(cellRect(r, x, y), ink, 0)
			}
		}
		if focus {
			p.FocusRing(cellRect(r, at.X, at.Y), [4]float32{})
		}
	})
}

// cellRect is the box of cell (x, y) in a grid whose box is r.
func cellRect(r ui.Rect, x, y int) ui.Rect {
	return ui.Rect{X: r.X + float32(x)*drawCell + 1, Y: r.Y + float32(y)*drawCell + 1, W: drawCell - 1, H: drawCell - 1}
}

// cellAt is the cell under a point of the grid, the nearest one for a point on its edge.
func cellAt(x, y float32) image.Point {
	at := func(v float32) int { return max(0, min(pixmap.Size-1, int(v/drawCell))) }
	return image.Pt(at(x), at(y))
}

// drawPointer carries out the press that is on the grid: it starts a stroke on the cell it began
// on, which decides whether the stroke paints or erases, and goes on from cell to cell.
func (m *Main) drawPointer(grid ui.Element, s *categorySheet) {
	d := &s.draw
	if !grid.Pressed() {
		d.stroke = stroke{}
		return
	}
	x, y, over := grid.PointerPosition()
	if !over {
		return
	}
	at := cellAt(x, y)
	if !d.stroke.on {
		d.stroke = stroke{on: true, pen: s.penAt(at), at: at}
		d.undo = append(d.undo, s.icon)
	}
	s.icon = s.icon.Line(d.stroke.at.X, d.stroke.at.Y, at.X, at.Y, d.stroke.pen)
	d.stroke.at, d.at = at, at
}

// drawKeys is the keyboard of the grid.
func (m *Main) drawKeys(grid ui.Element, s *categorySheet) {
	if !grid.Focused() {
		return
	}
	d := &s.draw
	step := func(dx, dy int) {
		d.at = image.Pt(max(0, min(pixmap.Size-1, d.at.X+dx)), max(0, min(pixmap.Size-1, d.at.Y+dy)))
	}
	switch {
	case grid.Shortcut(0, ui.KeyLeft):
		step(-1, 0)
	case grid.Shortcut(0, ui.KeyRight):
		step(1, 0)
	case grid.Shortcut(0, ui.KeyUp):
		step(0, -1)
	case grid.Shortcut(0, ui.KeyDown):
		step(0, 1)
	case grid.Shortcut(0, ui.KeySpace):
		s.edit(s.icon.Set(d.at.X, d.at.Y, s.penAt(d.at)))
	}
}

// drawTools is the column beside the grid: the live preview, Clear and Invert, the four arrows
// that move the drawing a cell, and Flip.
func (m *Main) drawTools(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
		m.categoryPreview(c, t, s, theme.IconBar, theme.IconRow, theme.IconSmall)
		ui.Row(c).Gap(6).Children(func() {
			if m.textTool(c, t, "draw-clear", m.say.CategoryClear(), nil) {
				s.edit(pixmap.Picture{})
			}
			if m.textTool(c, t, "draw-invert", m.say.CategoryInvert(), nil) {
				s.edit(s.icon.Invert())
			}
		})
		ui.Row(c).Gap(4).Children(func() {
			for _, a := range []struct {
				key, label string
				icon       *ui.SVG
				dx, dy     int
			}{
				{"left", m.say.CategoryShiftLeft(), theme.IconArrowLeft, -1, 0},
				{"up", m.say.CategoryShiftUp(), theme.IconSend, 0, -1},
				{"down", m.say.CategoryShiftDown(), theme.IconArrowDown, 0, 1},
				{"right", m.say.CategoryShiftRight(), theme.IconMove, 1, 0},
			} {
				if m.toolButton(c, t, "draw-shift-"+a.key, a.label, a.icon) {
					s.edit(s.icon.Shift(a.dx, a.dy))
				}
			}
		})
		if m.textTool(c, t, "draw-flip", m.say.CategoryFlip(), theme.IconFlip) {
			s.edit(s.icon.FlipH())
		}
	})
}

// textTool is a 26 pt button with a label, and a 12 pt icon before it when icon is not nil; it
// reports a click.
func (m *Main) textTool(c *ui.Context, t *theme.Tokens, key, label string, icon *ui.SVG) bool {
	btn := ui.ButtonBase(c.Key(key)).Height(toolButton).Padding(0, 8).Gap(6).Radius(theme.RadiusControl).Border(1, t.Border).
		AlignItems(ui.Center).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Text)
		}
		theme.Text(c, theme.Compact, t.Text, label)
	})
	return btn.Clicked()
}

// toolButton is a square button with a 12 pt icon, labeled for assistive technology; it reports a
// click.
func (m *Main) toolButton(c *ui.Context, t *theme.Tokens, key, label string, icon *ui.SVG) bool {
	btn := ui.ButtonBase(c.Key(key)).Size(toolButton, toolButton).Shrink(0).Center().Radius(theme.RadiusControl).Border(1, t.Border).
		Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() { ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Text) })
	return btn.Clicked()
}

// drawPalette is the pens: the category's own color first, set off by a rule since it may match a
// palette color but follows the category, then the twelve of the palette; the picked one has a
// heavier border.
func (m *Main) drawPalette(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	ui.Row(c).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
		m.penSwatch(c, t, s, pixmap.Own, theme.CategoryColor(s.color), m.say.CategoryPenOwn())
		ui.Box(c).Width(1).Height(paletteSwatch).Shrink(0).Margin(0, 4).Background(t.Border)
		for i := 0; i < pixmap.Colors; i++ {
			letter := pixmap.FirstColor + byte(i)
			m.penSwatch(c, t, s, letter, theme.PaletteColor(letter), m.say.A11yCategoryPen(m.paletteName(i)))
		}
	})
}

func (m *Main) penSwatch(c *ui.Context, t *theme.Tokens, s *categorySheet, pen byte, fill ui.Color, label string) {
	btn := ui.ButtonBase(c.Key("pen-"+string(pen))).Size(paletteSwatch, paletteSwatch).Shrink(0).Background(fill).Label(label).Tooltip(label).
		Cursor(ui.CursorPointer)
	if s.draw.pen == pen {
		btn.Border(2, t.Text)
	} else {
		btn.Border(1, t.Border)
	}
	if btn.Clicked() {
		s.draw.pen = pen
	}
}

// paletteName is the name of the i-th palette color: the project colors have their own names; the
// four after them are white, gray, black and brown.
func (m *Main) paletteName(i int) string {
	if i < len(data.CategoryColors) {
		return m.categoryColorName(data.CategoryColors[i])
	}
	return [...]string{m.say.CategoryColorWhite(), m.say.CategoryColorGray(), m.say.CategoryColorBlack(), m.say.CategoryColorBrown()}[i-len(data.CategoryColors)]
}
