package pixmap

import (
	"image"
	"image/color"
	"math"
	"slices"
)

// An imported picture becomes an icon in two steps. Reduce does the work that is done once: it
// measures how much of every pixel is the drawing (its ink), crops to the drawing, and averages
// it into 10 x 10 cells. Reduced.Picture then makes a 12 x 12 icon of those cells for a threshold,
// as often as the stepper asks, without reading the picture again.

// grid is the side of the averaged grid: the icon keeps a one-cell margin around it.
const grid = Size - 2

// cell is the average ink of an area of the source, and the average color of its inked pixels.
type cell struct{ ink, r, g, b float64 }

// Reduced is a picture reduced to 10 x 10 cells.
type Reduced struct{ cells [grid][grid]cell }

// Reduce measures the ink of img: its alpha when the picture is transparent around its edge, else
// the distance of each pixel from the background, the median color of the edge, as a share of the
// farthest color there could be. It crops to the pixels with more than a tenth of ink, pads that to
// a centered square and averages the square into 10 x 10 cells. It reports false when the picture
// has no ink at all.
func Reduce(img image.Image) (Reduced, bool) {
	b := img.Bounds()
	px := make([]color.NRGBA, 0, b.Dx()*b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px = append(px, color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA))
		}
	}
	w, h := b.Dx(), b.Dy()
	inkOf := inkFunc(px, w, h)
	ink := make([]float64, len(px))
	x0, y0, x1, y1 := w, h, -1, -1
	for i, p := range px {
		ink[i] = inkOf(p)
		if ink[i] > 0.1 {
			x, y := i%w, i/w
			x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
		}
	}
	if x1 < 0 {
		return Reduced{}, false
	}
	side := max(x1-x0+1, y1-y0+1)
	sx, sy := x0-(side-(x1-x0+1))/2, y0-(side-(y1-y0+1))/2
	var r Reduced
	var count [grid][grid]int
	var sums [grid][grid]cell
	for y := sy; y < sy+side; y++ {
		for x := sx; x < sx+side; x++ {
			i, j := (x-sx)*grid/side, (y-sy)*grid/side
			count[j][i]++
			if x < 0 || y < 0 || x >= w || y >= h {
				continue // the padding of the square has no ink
			}
			k := y*w + x
			s := &sums[j][i]
			s.ink += ink[k]
			s.r += ink[k] * float64(px[k].R)
			s.g += ink[k] * float64(px[k].G)
			s.b += ink[k] * float64(px[k].B)
		}
	}
	for j := range r.cells {
		for i, s := range sums[j] {
			if s.ink > 0 {
				r.cells[j][i] = cell{ink: s.ink / float64(count[j][i]), r: s.r / s.ink, g: s.g / s.ink, b: s.b / s.ink}
			}
		}
	}
	return r, true
}

// inkFunc is the ink of a pixel of the picture px, w x h: 0 to 1.
func inkFunc(px []color.NRGBA, w, h int) func(color.NRGBA) float64 {
	var edge []color.NRGBA
	for x := 0; x < w; x++ {
		edge = append(edge, px[x], px[(h-1)*w+x])
	}
	for y := 1; y < h-1; y++ {
		edge = append(edge, px[y*w], px[y*w+w-1])
	}
	transparent := 0
	for _, p := range edge {
		if p.A < 128 {
			transparent++
		}
	}
	if transparent*2 > len(edge) {
		return func(p color.NRGBA) float64 { return float64(p.A) / 255 }
	}
	bg := [3]float64{median(edge, func(p color.NRGBA) uint8 { return p.R }), median(edge, func(p color.NRGBA) uint8 { return p.G }), median(edge, func(p color.NRGBA) uint8 { return p.B })}
	far := math.Sqrt(math.Pow(math.Max(bg[0], 255-bg[0]), 2) + math.Pow(math.Max(bg[1], 255-bg[1]), 2) + math.Pow(math.Max(bg[2], 255-bg[2]), 2))
	return func(p color.NRGBA) float64 {
		d := math.Sqrt(math.Pow(float64(p.R)-bg[0], 2) + math.Pow(float64(p.G)-bg[1], 2) + math.Pow(float64(p.B)-bg[2], 2))
		return math.Min(1, d/far)
	}
}

func median(ps []color.NRGBA, channel func(color.NRGBA) uint8) float64 {
	v := make([]uint8, len(ps))
	for i, p := range ps {
		v[i] = channel(p)
	}
	slices.Sort(v)
	return float64(v[len(v)/2])
}

// minGap is how far apart, in ink, the classes of a split must be: a drawing that fills its square
// has cells that differ only by noise, which Otsu would split at random.
const minGap = 0.2

// Otsu is the threshold, in percent of ink, that best splits the 100 cells into a faint and an inked
// class (Otsu's method on a histogram of 64 bins); the stepper starts there. Cells that are all
// alike have nothing to split: 0, which keeps every cell that has any ink.
func (r Reduced) Otsu() int {
	var hist [64]float64
	for _, row := range r.cells {
		for _, c := range row {
			hist[min(63, int(c.ink*64))]++
		}
	}
	best, at := 0.0, 0
	for t := 1; t < 64; t++ {
		var w0, s0, w1, s1 float64
		for i, n := range hist {
			if i < t {
				w0, s0 = w0+n, s0+n*float64(i)
			} else {
				w1, s1 = w1+n, s1+n*float64(i)
			}
		}
		if w0 == 0 || w1 == 0 || (s1/w1-s0/w0)/64 < minGap {
			continue
		}
		if v := w0 * w1 * math.Pow(s0/w0-s1/w1, 2); v > best {
			best, at = v, t
		}
	}
	return int(math.Round(float64(at) * 100 / 64))
}

// Picture makes the icon: the cells with ink of at least percent (and any at all) are set, in the
// middle of the grid. Each takes the palette color nearest to its own in OKLab; when one color
// covers 80 percent of the cells those cells are Own instead, so the icon follows the category's
// color.
func (r Reduced) Picture(percent int, palette [Colors]color.RGBA) Picture {
	var p Picture
	var by [Colors]int
	total := 0
	for j, row := range r.cells {
		for i, c := range row {
			if c.ink > 0 && c.ink*100 >= float64(percent) {
				n := nearest(c, palette)
				p = p.Set(i+1, j+1, FirstColor+byte(n))
				by[n]++
				total++
			}
		}
	}
	for n, count := range by {
		if count > 0 && count*100 >= 80*total {
			for y := range p {
				for x, c := range p[y] {
					if c == FirstColor+byte(n) {
						p[y][x] = Own
					}
				}
			}
		}
	}
	return p
}

// nearest is the index of the palette color closest to the color of c in OKLab.
func nearest(c cell, palette [Colors]color.RGBA) int {
	l, a, b := oklab(c.r, c.g, c.b)
	best, at := math.Inf(1), 0
	for i, p := range palette {
		pl, pa, pb := oklab(float64(p.R), float64(p.G), float64(p.B))
		if d := (l-pl)*(l-pl) + (a-pa)*(a-pa) + (b-pb)*(b-pb); d < best {
			best, at = d, i
		}
	}
	return at
}

// oklab converts an sRGB color (0 to 255 a channel) to OKLab, where distance is close to what the
// eye sees.
func oklab(r, g, b float64) (float64, float64, float64) {
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	r, g, b = lin(r), lin(g), lin(b)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s
}
