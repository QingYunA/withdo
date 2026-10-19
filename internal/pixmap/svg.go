package pixmap

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Path draws the lit pixels as one SVG path in a 24-unit view box, 2 units a pixel (the interface
// icons' grid). Neighbours in a row merge into one rectangle.
func (b Bitmap) Path() string {
	var sb strings.Builder
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; {
			if !b.At(x, y) {
				x++
				continue
			}
			w := 1
			for b.At(x+w, y) {
				w++
			}
			fmt.Fprintf(&sb, "M%d %dh%dv2h-%dz", 2*x, 2*y, 2*w, 2*w)
			x += w
		}
	}
	return sb.String()
}

// SVG draws the picture as a standalone SVG with one path per color. The Own pixels are filled
// with currentColor, like the files in pixel-icons, so the surrounding text color tints them; the
// pixels of palette letter FirstColor+i are filled with fills[i], a "#RRGGBB" string. A picture
// with no pixel yields an SVG that draws nothing.
func (p Picture) SVG(fills [Colors]string) []byte {
	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 24 24">`)
	path := func(c byte, fill string) {
		var b Bitmap
		for y, row := range p {
			for x, v := range row {
				b = b.Set(x, y, v == c)
			}
		}
		if d := b.Path(); d != "" {
			fmt.Fprintf(&sb, `<path%s d="%s"/>`, fill, d)
		}
	}
	path(Own, "")
	for i, hex := range fills {
		path(FirstColor+byte(i), ` fill="`+hex+`"`)
	}
	if p.Empty() {
		sb.WriteString(`<path d="M0 0z"/>`)
	}
	sb.WriteString(`</svg>`)
	return []byte(sb.String())
}

const gridUnits = 2 * Size // the 24-unit view box

var (
	viewBoxRe = regexp.MustCompile(`viewBox\s*=\s*"\s*0[\s,]+0[\s,]+24[\s,]+24\s*"`)
	pathRe    = regexp.MustCompile(`<path\b[^>]*?\sd\s*=\s*"([^"]*)"`)
	tokenRe   = regexp.MustCompile(`[A-Za-z]|[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`)
)

// edge is a vertical side of a polygon; the horizontal ones decide nothing under a ray cast.
type edge struct {
	x, y0, y1 float64
	dir       int
}

// FromRectSVG reads an SVG made of right-angle paths (M H V L Z, in either case) on a 24-unit view
// box, fills it with the nonzero rule at 24 x 24 and reduces it to 12 x 12: a pixel is lit when at
// least 2 of its 4 sub-cells are. The drawing may sit one unit off the 12 x 12 grid, so all four
// offsets of 0 or 1 unit are tried and the closest wins. errCells counts the 24 x 24 cells where
// the result still differs from the drawing: 0 is lossless. Curves, arcs and diagonals are refused.
func FromRectSVG(data []byte) (b Bitmap, errCells int, err error) {
	src := string(data)
	if !viewBoxRe.MatchString(src) {
		return b, 0, fmt.Errorf(`the SVG needs viewBox="0 0 24 24"`)
	}
	paths := pathRe.FindAllStringSubmatch(src, -1)
	if len(paths) == 0 {
		return b, 0, fmt.Errorf("the SVG has no path")
	}
	var grid [gridUnits][gridUnits]bool
	for _, p := range paths {
		edges, err := pathEdges(p[1])
		if err != nil {
			return b, 0, err
		}
		for y := 0; y < gridUnits; y++ {
			for x := 0; x < gridUnits; x++ {
				if windingAt(edges, float64(x)+0.5, float64(y)+0.5) != 0 {
					grid[y][x] = true
				}
			}
		}
	}
	best := -1
	for _, off := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		cand, diff := reduce(&grid, off[0], off[1])
		if best < 0 || diff < best {
			b, best = cand, diff
		}
	}
	return b, best, nil
}

// reduce moves the drawing by (dx, dy) units, halves it and counts the cells that went wrong.
func reduce(grid *[gridUnits][gridUnits]bool, dx, dy int) (Bitmap, int) {
	at := func(x, y int) bool {
		x, y = x-dx, y-dy
		return x >= 0 && x < gridUnits && y >= 0 && y < gridUnits && grid[y][x]
	}
	var b Bitmap
	diff := 0
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			lit := 0
			for _, c := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				if at(2*x+c[0], 2*y+c[1]) {
					lit++
				}
			}
			on := lit >= 2
			b = b.Set(x, y, on)
			if on {
				diff += 4 - lit
			} else {
				diff += lit
			}
		}
	}
	return b, diff
}

// windingAt is the nonzero winding number of the polygons at (px, py): a ray to the right crosses
// their vertical sides.
func windingAt(edges []edge, px, py float64) int {
	w := 0
	for _, e := range edges {
		if e.x > px && py >= e.y0 && py < e.y1 {
			w += e.dir
		}
	}
	return w
}

// pathEdges turns path data into the vertical sides of its polygons, closing every subpath the way
// a fill does. A side that is neither horizontal nor vertical is an error.
func pathEdges(d string) ([]edge, error) {
	toks := tokenRe.FindAllString(d, -1)
	var (
		edges          []edge
		cx, cy, sx, sy float64
		cmd            byte
		hasOpen        bool
	)
	line := func(nx, ny float64) error {
		if nx != cx && ny != cy {
			return fmt.Errorf("the SVG has a diagonal side; only right-angle paths can be turned into pixels")
		}
		if nx == cx && ny != cy {
			e := edge{x: cx, y0: cy, y1: ny, dir: 1}
			if ny < cy {
				e = edge{x: cx, y0: ny, y1: cy, dir: -1}
			}
			edges = append(edges, e)
		}
		cx, cy = nx, ny
		return nil
	}
	closeSub := func() error {
		if !hasOpen {
			return nil
		}
		hasOpen = false
		return line(sx, sy)
	}
	i := 0
	number := func() (float64, error) {
		if i >= len(toks) {
			return 0, fmt.Errorf("the path data ends in the middle of a command")
		}
		v, err := strconv.ParseFloat(toks[i], 64)
		if err != nil {
			return 0, fmt.Errorf("the path data has %q where a number belongs", toks[i])
		}
		i++
		return v, nil
	}
	for i < len(toks) {
		if t := toks[i]; len(t) == 1 && (t[0] >= 'A' && t[0] <= 'Z' || t[0] >= 'a' && t[0] <= 'z') {
			cmd = t[0]
			i++
			if cmd == 'Z' || cmd == 'z' {
				if err := closeSub(); err != nil {
					return nil, err
				}
				cx, cy = sx, sy
				continue
			}
		} else if cmd == 0 || cmd == 'Z' || cmd == 'z' {
			return nil, fmt.Errorf("the path data has a number before any command")
		}
		rel := cmd >= 'a'
		var err error
		switch cmd {
		case 'M', 'm', 'L', 'l':
			var x, y float64
			if x, err = number(); err != nil {
				return nil, err
			}
			if y, err = number(); err != nil {
				return nil, err
			}
			if rel {
				x, y = cx+x, cy+y
			}
			if cmd == 'M' || cmd == 'm' {
				if err := closeSub(); err != nil {
					return nil, err
				}
				cx, cy, sx, sy = x, y, x, y
				hasOpen = true
				// Further pairs after a move are line-tos.
				if cmd == 'M' {
					cmd = 'L'
				} else {
					cmd = 'l'
				}
			} else {
				hasOpen = true
				if err := line(x, y); err != nil {
					return nil, err
				}
			}
		case 'H', 'h', 'V', 'v':
			v, err := number()
			if err != nil {
				return nil, err
			}
			nx, ny := cx, cy
			switch {
			case cmd == 'H':
				nx = v
			case cmd == 'h':
				nx = cx + v
			case cmd == 'V':
				ny = v
			default:
				ny = cy + v
			}
			hasOpen = true
			if err := line(nx, ny); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("the SVG uses the %q command; only M, H, V, L and Z can be turned into pixels", string(cmd))
		}
	}
	if err := closeSub(); err != nil {
		return nil, err
	}
	return edges, nil
}
