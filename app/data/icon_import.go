package data

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"path/filepath"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

// iconEdge is the long side, in pixels, a picture is scaled down to before it is read for ink.
const iconEdge = 256

// ErrIconCurves means an SVG is not made of right-angle paths, the only kind that has a 12 x 12
// icon of its own; the picture has to be exported as PNG first.
var ErrIconCurves = errors.New("this SVG has curves")

// IconImport is a file read to become a category's icon: a picture reduced to 10 x 10 cells that a
// threshold turns into an icon, or an SVG that is the icon already.
type IconImport struct {
	Name  string      // the file's name
	Thumb image.Image // the picture at most 256 pixels on a side; nil for an SVG
	cells pixmap.Reduced
	svg   pixmap.Picture
}

// IsVector reports whether the file was an SVG, which has no threshold.
func (i IconImport) IsVector() bool { return i.Thumb == nil }

// Threshold is the percent of ink the icon starts from: where Otsu's method splits the cells.
func (i IconImport) Threshold() int {
	if i.IsVector() {
		return 0
	}
	return i.cells.Otsu()
}

// Picture is the icon for a threshold in percent. Its colors come from the palette (the colors of
// the letters a to l).
func (i IconImport) Picture(percent int, palette [pixmap.Colors]color.RGBA) pixmap.Picture {
	if i.IsVector() {
		return i.svg
	}
	return i.cells.Picture(percent, palette)
}

// ReadIconFile reads a PNG, JPEG, GIF (its first frame), WebP or SVG file for an icon. A picture
// is scaled to 256 pixels and reduced; an SVG of right angles is rastered at 12 x 12. A file that is
// not a picture, or has nothing drawn on it, is ErrPictureFormat.
func ReadIconFile(path string) (IconImport, error) {
	raw, err := readSource(path)
	if err != nil {
		return IconImport{}, err
	}
	imp := IconImport{Name: filepath.Base(path)}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		b, _, err := pixmap.FromRectSVG(raw)
		if err != nil {
			return imp, ErrIconCurves
		}
		imp.svg = b.Picture()
		return imp, nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxSourcePixels {
		return imp, ErrPictureFormat
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return imp, ErrPictureFormat
	}
	imp.Thumb = downscale(src, iconEdge)
	var ok bool
	if imp.cells, ok = pixmap.Reduce(imp.Thumb); !ok {
		return imp, ErrPictureFormat
	}
	return imp, nil
}
