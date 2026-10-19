package theme

import "github.com/egoist/mygo/ui"

// GoodIcon draws a good of the Summary scene, as the Recent changes page marks a merged pull request
// with it: the file's cells at a whole number of points each (one, for a 17 cell file in an 18 pt
// slot), centered in a box size points square, so that every cell stays a crisp square. An unknown
// kind is a crate.
func GoodIcon(c *ui.Context, kind string, size float32) ui.Element {
	known := false
	for _, k := range goodKinds {
		known = known || k == kind
	}
	if !known {
		kind = "crate"
	}
	bmp := fileBitmap("goods/" + kind)
	return ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		w, h := bmp.Size()
		scale := float32(max(1, int(size)/w))
		dw, dh := float32(w)*scale, float32(h)*scale
		p.ImageRegion(bmp, ui.Rect{}, ui.Rect{X: r.X + (r.W-dw)/2, Y: r.Y + (r.H-dh)/2, W: dw, H: dh}, ui.ImageOptions{})
	})
}
