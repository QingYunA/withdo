package pixmap

import (
	"cmp"
	"fmt"
)

// Own is a pixel in the category's own color: it follows the category when its color changes. The
// zero byte is an empty pixel, so the zero Picture is empty.
const Own byte = '#'

// The pixels that are not Own are the letters a to l, the colors of a palette the app defines
// (theme.PaletteColors): fixed colors, the same in light and dark mode.
const (
	FirstColor byte = 'a'
	Colors          = 12
)

// Picture is a 12 x 12 icon of several colors, row 0 first: each pixel is 0 (empty), Own or one of
// the palette letters. A Picture is a value: the methods that change it return a new one.
type Picture [Size][Size]byte

// IsPixel reports whether c may fill a pixel: Own or a palette letter.
func IsPixel(c byte) bool { return c == Own || c >= FirstColor && c < FirstColor+Colors }

// Picture is the bitmap with every lit pixel in the category's own color.
func (b Bitmap) Picture() Picture {
	var p Picture
	for y := range p {
		for x := range p[y] {
			if b.At(x, y) {
				p[y][x] = Own
			}
		}
	}
	return p
}

// Grid is the form the database keeps: the 12 rows one after the other, row 0 first, 12 characters
// each and no separators; "." is an empty pixel, "#" an Own one and a letter a palette color.
func (p Picture) Grid() string {
	out := make([]byte, 0, GridLen)
	for _, row := range p {
		for _, c := range row {
			if c == 0 {
				c = '.'
			}
			out = append(out, c)
		}
	}
	return string(out)
}

// ParseGrid reads what Grid wrote. Only ".", "#" and the letters a to l are pixels.
func ParseGrid(s string) (Picture, error) {
	var p Picture
	if len(s) != GridLen {
		return p, fmt.Errorf("a picture is %d characters of . # and the palette letters a to l", GridLen)
	}
	for i := 0; i < GridLen; i++ {
		switch c := s[i]; {
		case c == '.':
		case IsPixel(c):
			p[i/Size][i%Size] = c
		default:
			return Picture{}, fmt.Errorf("a picture is %d characters of . # and the palette letters a to l", GridLen)
		}
	}
	return p, nil
}

// At is the pixel at (x, y); anything off the grid is empty.
func (p Picture) At(x, y int) byte {
	if x < 0 || x >= Size || y < 0 || y >= Size {
		return 0
	}
	return p[y][x]
}

// Set returns p with pixel (x, y) set to c (0 clears it); a pixel off the grid changes nothing.
func (p Picture) Set(x, y int, c byte) Picture {
	if x >= 0 && x < Size && y >= 0 && y < Size {
		p[y][x] = c
	}
	return p
}

// Empty reports whether no pixel is set.
func (p Picture) Empty() bool { return p == Picture{} }

// Count is the number of pixels that are set.
func (p Picture) Count() int {
	n := 0
	for _, row := range p {
		for _, c := range row {
			if c != 0 {
				n++
			}
		}
	}
	return n
}

// Invert swaps empty and Own pixels. A palette pixel has no opposite, so it turns empty.
func (p Picture) Invert() Picture {
	for y := range p {
		for x, c := range p[y] {
			if c == 0 {
				p[y][x] = Own
			} else {
				p[y][x] = 0
			}
		}
	}
	return p
}

// Shift moves the picture by (dx, dy) pixels; what leaves the grid is dropped, nothing wraps.
func (p Picture) Shift(dx, dy int) Picture {
	var out Picture
	for y := range p {
		for x, c := range p[y] {
			out = out.Set(x+dx, y+dy, c)
		}
	}
	return out
}

// FlipH mirrors the picture left to right.
func (p Picture) FlipH() Picture {
	for y := range p {
		for x := 0; x < Size/2; x++ {
			p[y][x], p[y][Size-1-x] = p[y][Size-1-x], p[y][x]
		}
	}
	return p
}

// Line sets the cells from (x0, y0) to (x1, y1) to c, Bresenham's way, so a fast drag leaves no
// gap. The cells off the grid are dropped.
func (p Picture) Line(x0, y0, x1, y1 int, c byte) Picture {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := cmp.Compare(x1, x0), cmp.Compare(y1, y0)
	err := dx + dy
	for {
		p = p.Set(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return p
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// Outline keeps the pixels that touch an empty one or the edge of the grid (the four neighbours),
// as Own: the inside of a shape is hollowed out and the palette is dropped.
func (p Picture) Outline() Picture {
	var out Picture
	for y := range p {
		for x, c := range p[y] {
			inner := p.At(x-1, y) != 0 && p.At(x+1, y) != 0 && p.At(x, y-1) != 0 && p.At(x, y+1) != 0
			if c != 0 && !inner {
				out[y][x] = Own
			}
		}
	}
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
