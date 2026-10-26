package views

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// Import image… of the category sheet: a picture reduced to 12 x 12, a threshold stepper and an
// outline switch. See docs/design/mockups/v24.html, frames 7c and 7d.

var (
	iconRed  = color.NRGBA{0xE5, 0x48, 0x4D, 255} // palette letter d
	iconBlue = color.NRGBA{0x3B, 0x82, 0xF6, 255} // letter e
)

// iconPNG writes an 100 x 100 transparent picture with an 80 x 80 square whose left half is left and
// whose right half is right; right may be translucent.
func iconPNG(t *testing.T, name string, left, right color.NRGBA) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	draw.Draw(img, image.Rect(10, 10, 50, 90), image.NewUniform(left), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(50, 10, 90, 90), image.NewUniform(right), image.Point{}, draw.Src)
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return writeIconFile(t, name, b.Bytes())
}

func writeIconFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// importRig is a window with the New category sheet open and the file panel answering path.
type importRig struct {
	tt  *ui.Tester
	m   *Main
	app *App
	rec *recorder
}

func newImportRig(t *testing.T, path string) *importRig {
	t.Helper()
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	old := pickIconFiles
	pickIconFiles = func(string) []string { return []string{path} }
	t.Cleanup(func() { pickIconFiles = old })
	openNewCategory(t, tt)
	return &importRig{tt: tt, m: m, app: app, rec: rec}
}

// importIt presses Import image… and lets the file be read.
func (r *importRig) importIt(t *testing.T) {
	t.Helper()
	click(t, r.tt, "Import image…")
	r.rec.drain(t) // the file, read
	r.tt.Frame()
}

func (r *importRig) icon() pixmap.Picture { return sheetIcon(r.m) }

func TestImportShowsTheImageAndItsTwelveByTwelveResult(t *testing.T) {
	r := newImportRig(t, iconPNG(t, "flag.png", iconRed, iconBlue))
	r.importIt(t)
	for _, want := range []string{"flag.png", "12 × 12", "Threshold", "Outline only", "Library", "Draw"} {
		if !r.tt.HasText(want) {
			t.Errorf("the import lacks %q in %q", want, r.tt.Texts())
		}
	}
	if r.tt.HasText("Search icons") {
		t.Error("the library stays under an import")
	}
	got := r.icon()
	if got.At(2, 5) != 'd' || got.At(9, 5) != 'e' || got.At(0, 0) != 0 || got.Count() != 100 {
		t.Errorf("red and blue halves map to two palette letters:\n%s", got.Grid())
	}
	if _, ok := r.tt.Find("coffee"); ok {
		t.Error("the library's cells are still there")
	}
}

func TestImportedColorsDrawInTheSidebarAfterCreate(t *testing.T) {
	r := newImportRig(t, iconPNG(t, "flag.png", iconRed, iconBlue))
	r.importIt(t)
	typeName(r.tt, "Flags")
	click(t, r.tt, "Create")
	cat := categoryNamed(t, r.app, "Flags")
	if cat.Icon != r.icon() {
		t.Errorf("stored %s, shown %s", cat.Icon.Grid(), r.icon().Grid())
	}
	// A row names its category in words; the icon shows where the category narrows the list.
	click(t, r.tt, "Show only Flags")
	row, _ := r.tt.Find("Showing Flags")
	slot := ui.Rect{X: row.X - 6 - theme.IconSmall, Y: row.Y, W: theme.IconSmall, H: row.H}
	if !paintedIn(r.tt.Image(), 1100, slot, theme.PaletteColor('e'), 40) {
		t.Error("the category icon is not blue")
	}
}

func TestTheThresholdStepsByFourAndTheIconFollows(t *testing.T) {
	faint := color.NRGBA{iconRed.R, iconRed.G, iconRed.B, 77} // 30 percent
	r := newImportRig(t, iconPNG(t, "faint.png", iconRed, faint))
	r.importIt(t)
	start := r.m.manage.category.imp.percent
	if !r.tt.HasText(strconv.Itoa(start)+"%") || r.icon().Count() != 50 {
		t.Fatalf("at Otsu's %d%% only the solid half stays: %d cells, %q", start, r.icon().Count(), r.tt.Texts())
	}
	click(t, r.tt, "Lower the threshold")
	if got := r.m.manage.category.imp.percent; got != start-4 || !r.tt.HasText(strconv.Itoa(start-4)+"%") {
		t.Errorf("threshold %d after Lower, want %d", got, start-4)
	}
	if got := r.icon().Count(); got != 100 {
		t.Errorf("below the faint half's ink it joins in: %d cells", got)
	}
	click(t, r.tt, "Raise the threshold")
	if r.icon().Count() != 50 {
		t.Errorf("back up it leaves again: %d cells", r.icon().Count())
	}
	for i := 0; i < 30; i++ {
		click(t, r.tt, "Raise the threshold")
	}
	if got := r.m.manage.category.imp.percent; got != 100 || !r.tt.HasText("100%") {
		t.Errorf("the stepper stops at 100, got %d", got)
	}
	for i := 0; i < 30; i++ {
		click(t, r.tt, "Lower the threshold")
	}
	if got := r.m.manage.category.imp.percent; got != 0 {
		t.Errorf("the stepper stops at 0, got %d", got)
	}
}

func TestOutlineOnlyHollowsTheResultAndDropsThePalette(t *testing.T) {
	r := newImportRig(t, iconPNG(t, "flag.png", iconRed, iconBlue))
	r.importIt(t)
	click(t, r.tt, "Outline only")
	got := r.icon()
	if got.Count() != 36 || got.At(5, 5) != 0 || got.At(1, 1) != pixmap.Own || got.At(10, 1) != pixmap.Own {
		t.Errorf("only the ring stays, in the category color:\n%s", got.Grid())
	}
	click(t, r.tt, "Outline only")
	if r.icon().Count() != 100 || r.icon().At(5, 5) != 'd' {
		t.Errorf("the switch off brings the inside and the colors back:\n%s", r.icon().Grid())
	}
}

func TestAnFileThatIsNotAPictureShowsTheErrorLine(t *testing.T) {
	r := newImportRig(t, writeIconFile(t, "notes.png", []byte("not a picture")))
	before := r.icon()
	r.importIt(t)
	if !r.tt.HasText("Couldn't read this image.") || r.icon() != before || !r.tt.HasText("Search icons") {
		t.Errorf("the line is missing or the sheet changed: %q", r.tt.Texts())
	}
}

func TestAnSVGWithCurvesSaysToExportItAsPNG(t *testing.T) {
	r := newImportRig(t, writeIconFile(t, "curvy.svg", []byte(`<svg viewBox="0 0 24 24"><path d="M2 2C4 4 6 6 8 8z"/></svg>`)))
	r.importIt(t)
	if !r.tt.HasText("This SVG has curves; export it as PNG first.") {
		t.Errorf("no word about curves: %q", r.tt.Texts())
	}
}

func TestAnSVGOfRightAnglesIsTheIconWithNoThreshold(t *testing.T) {
	r := newImportRig(t, writeIconFile(t, "block.svg", []byte(`<svg viewBox="0 0 24 24"><path d="M4 4h16v16H4z"/></svg>`)))
	r.importIt(t)
	if r.icon().Count() == 0 || r.icon().At(5, 5) != pixmap.Own || r.tt.HasText("Threshold") || !r.tt.HasText("Outline only") {
		t.Errorf("an SVG shows its raster and the outline switch only:\n%s", r.icon().Grid())
	}
	click(t, r.tt, "Outline only")
	if r.icon().At(5, 5) != 0 || r.icon().At(2, 2) != pixmap.Own {
		t.Errorf("outline of an SVG:\n%s", r.icon().Grid())
	}
}

func TestADroppedFileIsImportedToo(t *testing.T) {
	r := newImportRig(t, "")
	go r.m.readIcon(iconPNG(t, "dropped.png", iconRed, iconBlue)) // what the drop target starts
	r.rec.drain(t)
	r.tt.Frame()
	if !r.tt.HasText("dropped.png") || r.icon().At(2, 5) != 'd' {
		t.Errorf("the drop was not imported: %q", r.tt.Texts())
	}
}

func TestDrawFromAnImportOpensTheEditorOnTheResultAndLibraryGoesBack(t *testing.T) {
	r := newImportRig(t, iconPNG(t, "flag.png", iconRed, iconBlue))
	r.importIt(t)
	result := r.icon()
	click(t, r.tt, "Draw")
	if !r.tt.HasText("Icon drawing") || r.icon() != result || r.tt.HasText("Threshold") {
		t.Errorf("Draw opens the editor on the result: %q", r.tt.Texts())
	}
	click(t, r.tt, "Library")
	if !r.tt.HasText("Search icons") || r.tt.HasText("Icon drawing") {
		t.Errorf("Library returns to the grid: %q", r.tt.Texts())
	}
}

func TestCancelingTheFilePanelChangesNothing(t *testing.T) {
	r := newImportRig(t, "")
	asked := make(chan struct{})
	pickIconFiles = func(string) []string { close(asked); return nil } // cancelled
	before := r.icon()
	click(t, r.tt, "Import image…")
	<-asked
	r.tt.Frame()
	if r.icon() != before || !r.tt.HasText("Search icons") || r.tt.HasText("Couldn't read this image.") {
		t.Errorf("a cancelled panel changed the sheet: %q", r.tt.Texts())
	}
}

func TestTheImportReadsInChinese(t *testing.T) {
	r := newImportRig(t, iconPNG(t, "flag.png", iconRed, iconBlue))
	r.tt.SetLocale("zh-Hans")
	r.tt.Frame()
	click(t, r.tt, "导入图片…")
	r.rec.drain(t)
	r.tt.Frame()
	for _, want := range []string{"阈值", "只留轮廓", "图标库", "画"} {
		if !r.tt.HasText(want) {
			t.Errorf("no %q in %q", want, r.tt.Texts())
		}
	}
}
