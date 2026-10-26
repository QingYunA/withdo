package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// The icon editor of the category sheet: a 12 x 12 grid of 18 pt cells, a palette, and the tools
// beside it. See docs/design/mockups/v24.html, frame 7b.

// openDraw opens the New category sheet on Draw and returns the grid's box.
func openDraw(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	openNewCategory(t, tt)
	click(t, tt, "Draw")
	return gridBox(t, tt)
}

func gridBox(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	g, ok := tt.Find("Icon drawing")
	if !ok {
		t.Fatalf("no editor grid in %q", tt.Texts())
	}
	return g
}

// cellPoint is the middle of cell (x, y) of a grid whose box is g: 18 pt cells, a 1 pt line first.
func cellPoint(g ui.Rect, x, y int) (float32, float32) {
	return g.X + 18*float32(x) + 10, g.Y + 18*float32(y) + 10
}

func clickCell(tt *ui.Tester, g ui.Rect, x, y int) {
	px, py := cellPoint(g, x, y)
	tt.ClickAt(px, py)
}

// dragCells presses cell (x0, y0), moves to each of the others and releases on the last.
func dragCells(tt *ui.Tester, g ui.Rect, cells ...[2]int) {
	px, py := cellPoint(g, cells[0][0], cells[0][1])
	tt.Press(px, py)
	for _, c := range cells[1:] {
		px, py = cellPoint(g, c[0], c[1])
		tt.Move(px, py)
	}
	tt.Release(px, py)
}

// typeName clicks into the Name field, which a click on a button took the focus from, and types.
func typeName(tt *ui.Tester, name string) {
	field, _ := tt.Find("Name")
	tt.ClickAt(field.X+40, field.Y+field.H+18)
	tt.Type(name)
}

func sheetIcon(m *Main) pixmap.Picture { return m.manage.category.icon }

func TestDrawOpensTheEditorOnTheCurrentIconAndLibraryGoesBack(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	openNewCategory(t, tt)
	start := sheetIcon(m)
	click(t, tt, "Draw")
	g := gridBox(t, tt)
	if tt.HasText("Search icons") || tt.HasText("Study") {
		t.Errorf("the editor replaces the library: %q", tt.Texts())
	}
	for _, want := range []string{"Clear", "Invert", "Library", "Click a cell to draw, drag to keep going. ⌘Z undoes."} {
		if !tt.HasText(want) {
			t.Errorf("the editor lacks %q in %q", want, tt.Texts())
		}
	}
	if g.W < 216 || g.H < 216 || sheetIcon(m) != start {
		t.Errorf("grid %+v, icon changed %v", g, sheetIcon(m) != start)
	}
	// The cells show the icon: a set cell is in the category's color, an empty one is not.
	ink := theme.CategoryColor(m.manage.category.color)
	var set, empty [2]int
	for y := 0; y < pixmap.Size; y++ {
		for x := 0; x < pixmap.Size; x++ {
			if start.At(x, y) != 0 {
				set = [2]int{x, y}
			} else {
				empty = [2]int{x, y}
			}
		}
	}
	cell := func(c [2]int) ui.Rect {
		return ui.Rect{X: g.X + 18*float32(c[0]) + 4, Y: g.Y + 18*float32(c[1]) + 4, W: 10, H: 10}
	}
	if !paintedIn(tt.Image(), 1100, cell(set), ink, 6) || paintedIn(tt.Image(), 1100, cell(empty), ink, 6) {
		t.Error("the grid does not show the icon in the category's color")
	}
	click(t, tt, "Library")
	if !tt.HasText("Search icons") || tt.HasText("Icon drawing") {
		t.Errorf("Library did not bring the picker back: %q", tt.Texts())
	}
	click(t, tt, "Draw")
	if !tt.HasText("Icon drawing") {
		t.Error("Draw again opens the editor")
	}
}

func TestClickingACellPaintsItAndClickingItAgainErasesIt(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	g := openDraw(t, tt)
	if sheetIcon(m).At(0, 11) != 0 {
		t.Fatal("the cell is not empty to begin with")
	}
	clickCell(tt, g, 0, 11)
	if got := sheetIcon(m).At(0, 11); got != pixmap.Own {
		t.Errorf("a click paints with the category color, got %q", got)
	}
	clickCell(tt, g, 0, 11)
	if got := sheetIcon(m).At(0, 11); got != 0 {
		t.Errorf("a click on a cell of the picked color erases it, got %q", got)
	}
}

func TestDraggingKeepsPaintingAndFillsTheCellsBetween(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	g := openDraw(t, tt)
	dragCells(tt, g, [2]int{0, 11}, [2]int{5, 11}, [2]int{5, 11}, [2]int{11, 11}) // a long jump: the cells between fill
	for x := 0; x < pixmap.Size; x++ {
		if sheetIcon(m).At(x, 11) != pixmap.Own {
			t.Errorf("cell %d of the row is %q after the drag", x, sheetIcon(m).At(x, 11))
		}
	}
	// A drag that starts on a painted cell erases as it goes.
	dragCells(tt, g, [2]int{2, 11}, [2]int{8, 11})
	for x, want := range []byte{pixmap.Own, pixmap.Own, 0, 0, 0, 0, 0, 0, 0, pixmap.Own, pixmap.Own, pixmap.Own} {
		if got := sheetIcon(m).At(x, 11); got != want {
			t.Errorf("cell %d after the erasing drag is %q, want %q", x, got, want)
		}
	}
}

func TestThePaletteChoosesTheColorACellTakes(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	g := openDraw(t, tt)
	for _, want := range []string{"Category color", "Draw with Indigo", "Draw with Red", "Draw with White", "Draw with Gray", "Draw with Black", "Draw with Brown"} {
		if _, ok := tt.Find(want); !ok {
			t.Errorf("no swatch %q in %q", want, tt.Texts())
		}
	}
	click(t, tt, "Draw with Red")
	clickCell(tt, g, 0, 11)
	if got := sheetIcon(m).At(0, 11); got != 'd' {
		t.Errorf("red is letter d, got %q", got)
	}
	click(t, tt, "Draw with Blue")
	clickCell(tt, g, 0, 11) // another color than the picked one: it paints over
	if got := sheetIcon(m).At(0, 11); got != 'e' {
		t.Errorf("blue is letter e, got %q", got)
	}
	clickCell(tt, g, 0, 11)
	if got := sheetIcon(m).At(0, 11); got != 0 {
		t.Errorf("the picked color again erases, got %q", got)
	}
	click(t, tt, "Category color")
	clickCell(tt, g, 0, 11)
	if got := sheetIcon(m).At(0, 11); got != pixmap.Own {
		t.Errorf("the first swatch is the category color, got %q", got)
	}
	// The cell is drawn in the palette color, whatever the category's color is.
	click(t, tt, "Draw with Teal")
	clickCell(tt, g, 1, 11)
	cell := ui.Rect{X: g.X + 18 + 4, Y: g.Y + 18*11 + 4, W: 10, H: 10}
	if !paintedIn(tt.Image(), 1100, cell, theme.PaletteColor('g'), 6) {
		t.Error("a teal cell is not drawn teal")
	}
}

func TestClearInvertShiftAndFlip(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	openDraw(t, tt)
	start := sheetIcon(m)
	click(t, tt, "Invert")
	if sheetIcon(m) != start.Invert() {
		t.Error("Invert swaps the empty cells and the cells of the category color")
	}
	click(t, tt, "Invert")
	click(t, tt, "Move the drawing right")
	if sheetIcon(m) != start.Shift(1, 0) {
		t.Error("the right arrow moves the drawing one cell right")
	}
	click(t, tt, "Move the drawing left")
	click(t, tt, "Move the drawing down")
	if sheetIcon(m) != start.Shift(0, 1) {
		t.Error("the down arrow moves the drawing one cell down")
	}
	click(t, tt, "Move the drawing up")
	if sheetIcon(m) != start {
		t.Error("up and down cancel")
	}
	click(t, tt, "Flip")
	if sheetIcon(m) != start.FlipH() {
		t.Error("Flip mirrors the drawing")
	}
	click(t, tt, "Clear")
	if !sheetIcon(m).Empty() {
		t.Error("Clear empties the drawing")
	}
	typeName(tt, "Empty")
	click(t, tt, "Create") // the button is off: an empty icon cannot be saved
	if len(m.app.snap.categories) != 2 || !tt.HasText("Create") {
		t.Errorf("a category was made with an empty icon, or the sheet closed: %v", categoryNames(m.app))
	}
}

func TestCmdZUndoesTheEditorsStepsOneByOne(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	g := openDraw(t, tt)
	start := sheetIcon(m)
	dragCells(tt, g, [2]int{0, 11}, [2]int{4, 11}) // one stroke
	click(t, tt, "Flip")
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := sheetIcon(m); got.At(4, 11) != pixmap.Own || got.At(7, 11) != 0 {
		t.Errorf("the first ⌘Z undoes the flip, leaving the stroke:\n%s", got.Grid())
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if sheetIcon(m) != start {
		t.Errorf("the second ⌘Z undoes the whole stroke:\n%s", sheetIcon(m).Grid())
	}
	tt.Key(ui.Cmd, ui.KeyZ) // nothing left: nothing happens
	if sheetIcon(m) != start || !tt.HasText("Icon drawing") {
		t.Error("⌘Z with nothing to undo changed the sheet")
	}
}

func TestTheArrowsMoveTheKeyboardCellAndSpaceTogglesIt(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	g := openDraw(t, tt)
	clickCell(tt, g, 0, 11) // paints (0, 11) and puts the keyboard there
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeySpace)
	if got := sheetIcon(m).At(2, 11); got != pixmap.Own {
		t.Errorf("Space after two arrows paints (2, 11), got %q", got)
	}
	tt.Key(0, ui.KeySpace)
	if got := sheetIcon(m).At(2, 11); got != 0 {
		t.Errorf("Space again erases it, got %q", got)
	}
	was := sheetIcon(m).At(2, 10)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeySpace)
	if got := sheetIcon(m).At(2, 10); got == was {
		t.Errorf("Up moves to (2, 10), which Space should have toggled from %q", was)
	}
}

func TestCreateStoresTheDrawingAndTheSidebarShowsItsColors(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	g := openDraw(t, tt)
	click(t, tt, "Clear")
	click(t, tt, "Draw with Red")
	dragCells(tt, g, [2]int{2, 2}, [2]int{9, 2})
	click(t, tt, "Draw with Teal")
	dragCells(tt, g, [2]int{2, 9}, [2]int{9, 9})
	click(t, tt, "Category color")
	dragCells(tt, g, [2]int{5, 4}, [2]int{5, 7})
	typeName(tt, "Paint")
	click(t, tt, "Create")
	cat := categoryNamed(t, app, "Paint")
	if cat.Icon.At(2, 2) != 'd' || cat.Icon.At(9, 9) != 'g' || cat.Icon.At(5, 5) != pixmap.Own || cat.Icon.Count() != 8+8+4 {
		t.Errorf("the stored drawing:\n%s", cat.Icon.Grid())
	}
	click(t, tt, "Show only Paint") // a row names its category in words; the icon shows where it narrows the list
	row, ok := tt.Find("Showing Paint")
	if !ok {
		t.Fatalf("no filter line in %q", tt.Texts())
	}
	slot := ui.Rect{X: row.X - 6 - theme.IconSmall, Y: row.Y, W: theme.IconSmall, H: row.H}
	img := tt.Image()
	for _, want := range []ui.Color{theme.PaletteColor('d'), theme.PaletteColor('g'), theme.CategoryColor(cat.Color)} {
		if !paintedIn(img, 1100, slot, want, 40) {
			t.Errorf("the sidebar icon lacks %v", want)
		}
	}
}

func TestTheEditorReadsInChinese(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	openNewCategory(t, tt)
	tt.SetLocale("zh-Hans")
	tt.Frame()
	click(t, tt, "画")
	for _, want := range []string{"清空", "反相", "翻转", "图标库", "导入图片…", "点一格来画，拖动可以连着画。⌘Z 撤销。"} {
		if !tt.HasText(want) {
			t.Errorf("no %q in %q", want, tt.Texts())
		}
	}
	if _, ok := tt.Find("类别色"); !ok {
		t.Error("no category color swatch")
	}
}
