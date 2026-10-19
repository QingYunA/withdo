package views

import (
	"image"
	"image/color"
	"sort"
)

// weighted is one distinct color of a screenshot and how many pixels use it.
type weighted struct {
	c [3]uint8
	n int
}

// widest returns the channel along which the colors differ most, and by how much.
func widest(box []weighted) (axis, span int) {
	lo, hi := [3]int{255, 255, 255}, [3]int{}
	for _, e := range box {
		for i := range lo {
			v := int(e.c[i])
			lo[i], hi[i] = min(lo[i], v), max(hi[i], v)
		}
	}
	for i := range lo {
		if hi[i]-lo[i] > span {
			axis, span = i, hi[i]-lo[i]
		}
	}
	return axis, span
}

// quantizeScreen turns an opaque screenshot into a 256-color image by median cut over its distinct
// colors, weighted by pixel count, with no dithering. A screenshot is flat fills and antialiased
// text with a few thousand distinct colors, so the palette is near-exact and the PNG is a third of
// the size; dithering would only add noise and bytes. Alpha is ignored.
func quantizeScreen(src *image.RGBA) *image.Paletted {
	b := src.Bounds()
	counts := map[[3]uint8]int{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			o := src.PixOffset(x, y)
			counts[[3]uint8{src.Pix[o], src.Pix[o+1], src.Pix[o+2]}]++
		}
	}
	all := make([]weighted, 0, len(counts))
	for c, n := range counts {
		all = append(all, weighted{c, n})
	}
	// map order is random; a fixed start keeps the palette, and so the PNG, reproducible
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i].c, all[j].c
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		if a[1] != b[1] {
			return a[1] < b[1]
		}
		return a[2] < b[2]
	})

	boxes := [][]weighted{all}
	for len(boxes) < 256 {
		// split the box where a wrong color would cost the most: widest span times pixels
		best, bestScore := -1, 0
		for i, box := range boxes {
			if len(box) < 2 {
				continue
			}
			total := 0
			for _, e := range box {
				total += e.n
			}
			_, span := widest(box)
			if s := span * total; s > bestScore {
				best, bestScore = i, s
			}
		}
		if best < 0 {
			break // every box is one color
		}
		box := boxes[best]
		axis, _ := widest(box)
		sort.SliceStable(box, func(i, j int) bool { return box[i].c[axis] < box[j].c[axis] })
		total := 0
		for _, e := range box {
			total += e.n
		}
		seen, cut := 0, len(box)-1
		for i, e := range box {
			seen += e.n
			if seen*2 >= total {
				cut = min(i+1, len(box)-1)
				break
			}
		}
		boxes[best] = box[:cut]
		boxes = append(boxes, box[cut:])
	}

	pal := make(color.Palette, len(boxes))
	for i, box := range boxes {
		var sum [3]int
		total := 0
		for _, e := range box {
			for k := range sum {
				sum[k] += int(e.c[k]) * e.n
			}
			total += e.n
		}
		pal[i] = color.RGBA{uint8((sum[0] + total/2) / total), uint8((sum[1] + total/2) / total), uint8((sum[2] + total/2) / total), 255}
	}

	out := image.NewPaletted(b, pal)
	index := make(map[[3]uint8]uint8, len(counts))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			o := src.PixOffset(x, y)
			c := [3]uint8{src.Pix[o], src.Pix[o+1], src.Pix[o+2]}
			i, ok := index[c]
			if !ok {
				i = uint8(pal.Index(color.RGBA{c[0], c[1], c[2], 255}))
				index[c] = i
			}
			out.SetColorIndex(x, y, i)
		}
	}
	return out
}
