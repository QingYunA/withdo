package data

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

var iconPalette = func() (p [pixmap.Colors]color.RGBA) {
	for i := range p {
		p[i] = color.RGBA{A: 255}
	}
	p[3] = color.RGBA{R: 0xE5, G: 0x48, B: 0x4D, A: 255} // d, red
	return p
}()

func writeIconFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// redBlock is a transparent w x h picture with a red rectangle.
func redBlock(w, h int, r image.Rectangle) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, r, image.NewUniform(color.NRGBA{R: 0xE5, G: 0x48, B: 0x4D, A: 255}), image.Point{}, draw.Src)
	return img
}

func TestReadIconFileTurnsAPictureIntoAnIcon(t *testing.T) {
	// A big picture is scaled down first: 600 x 300 with a red block in the middle.
	path := writeIconFile(t, "block.png", pngOf(t, redBlock(600, 300, image.Rect(100, 50, 500, 250))))
	imp, err := ReadIconFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if imp.Name != "block.png" || imp.IsVector() || imp.Thumb == nil || max(imp.Thumb.Bounds().Dx(), imp.Thumb.Bounds().Dy()) > 256 {
		t.Fatalf("import %q vector %v thumb %v", imp.Name, imp.IsVector(), imp.Thumb)
	}
	p := imp.Picture(imp.Threshold(), iconPalette)
	// The block is twice as wide as high: the square around it has empty bands above and below.
	if p.At(6, 6) != pixmap.Own || p.At(1, 6) != pixmap.Own || p.At(10, 6) != pixmap.Own || p.At(6, 1) != 0 || p.At(6, 0) != 0 {
		t.Errorf("a red block is Own in the middle rows, with its margin: %s", p.Grid())
	}
}

func TestReadIconFileTakesAnSVGOfRightAnglesAsItIs(t *testing.T) {
	svg := `<svg viewBox="0 0 24 24"><path d="M4 4h8v16H4zM14 8h6v4h-6z"/></svg>`
	imp, err := ReadIconFile(writeIconFile(t, "shape.svg", []byte(svg)))
	if err != nil {
		t.Fatal(err)
	}
	want, _, _ := pixmap.FromRectSVG([]byte(svg))
	if !imp.IsVector() || imp.Thumb != nil || imp.Picture(50, iconPalette) != want.Picture() || imp.Threshold() != 0 {
		t.Errorf("an SVG is its own picture, whatever the threshold: %s", imp.Picture(50, iconPalette).Grid())
	}
}

func TestReadIconFileRefusesCurvesAndWhatIsNotAPicture(t *testing.T) {
	curve := `<svg viewBox="0 0 24 24"><path d="M2 2C4 4 6 6 8 8z"/></svg>`
	if _, err := ReadIconFile(writeIconFile(t, "curve.svg", []byte(curve))); !errors.Is(err, ErrIconCurves) {
		t.Errorf("curves: %v", err)
	}
	if _, err := ReadIconFile(writeIconFile(t, "notes.png", []byte("not a picture"))); !errors.Is(err, ErrPictureFormat) {
		t.Errorf("garbage: %v", err)
	}
	if _, err := ReadIconFile(writeIconFile(t, "blank.png", pngOf(t, image.NewNRGBA(image.Rect(0, 0, 50, 50))))); !errors.Is(err, ErrPictureFormat) {
		t.Errorf("a picture with nothing in it: %v", err)
	}
	if _, err := ReadIconFile(filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Error("a missing file read")
	}
}
