package pixmap

import (
	"strings"
	"testing"
)

func TestPictureOfABitmapIsAllOwnColor(t *testing.T) {
	b, _ := ParseHex(briefcaseHex)
	p := b.Picture()
	if p.Grid() != b.Grid() {
		t.Fatalf("picture %s, bitmap %s", p.Grid(), b.Grid())
	}
	if p.Count() != b.Count() || p.At(5, 1) != Own || p.At(0, 0) != 0 {
		t.Errorf("count %d, At(5,1) %q", p.Count(), p.At(5, 1))
	}
}

func TestPictureGridRoundTripWithPaletteLetters(t *testing.T) {
	p := Picture{}.Set(0, 0, Own).Set(1, 0, 'a').Set(11, 11, 'l')
	grid := p.Grid()
	if len(grid) != GridLen || !strings.HasPrefix(grid, "#a.") || !strings.HasSuffix(grid, ".l") {
		t.Fatalf("grid = %s", grid)
	}
	back, err := ParseGrid(grid)
	if err != nil || back != p {
		t.Fatalf("ParseGrid = %v, %v", back, err)
	}
}

func TestParseGridRefusesAnythingButDotsHashesAndAToL(t *testing.T) {
	ok := strings.Repeat(".", GridLen)
	for name, s := range map[string]string{
		"short":     ok[:143],
		"long":      ok + ".",
		"empty":     "",
		"past l":    ok[:143] + "m",
		"upper":     ok[:143] + "A",
		"a space":   ok[:143] + " ",
		"zero":      ok[:143] + "0",
		"multibyte": ok[:141] + "é",
	} {
		if _, err := ParseGrid(s); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	for _, c := range "#abcdefghijkl" {
		if _, err := ParseGrid(ok[:143] + string(c)); err != nil {
			t.Errorf("%q refused: %v", c, err)
		}
	}
}

func TestSetReturnsACopyAndIgnoresWhatIsOffTheGrid(t *testing.T) {
	var p Picture
	q := p.Set(3, 4, 'c')
	if !p.Empty() || q.At(3, 4) != 'c' || q.Count() != 1 {
		t.Fatalf("Set must leave the receiver alone: %v %v", p, q)
	}
	if q.Set(12, 0, Own) != q || q.Set(0, -1, Own) != q || q.At(-1, 0) != 0 {
		t.Error("a pixel off the grid changes and reads nothing")
	}
	if !q.Set(3, 4, 0).Empty() {
		t.Error("Set with 0 clears")
	}
}

func TestInvertSwapsEmptyAndOwnColorAndDropsPalettePixels(t *testing.T) {
	p := Picture{}.Set(0, 0, Own).Set(1, 0, 'd')
	inv := p.Invert()
	if inv.At(0, 0) != 0 || inv.At(1, 0) != 0 || inv.At(2, 0) != Own || inv.Count() != 142 {
		t.Fatalf("Invert = %s", inv.Grid())
	}
	if (Picture{}).Invert().Count() != GridLen {
		t.Error("Invert of empty is full")
	}
}

func TestShiftMovesByOneCellWithoutWrapping(t *testing.T) {
	p := Picture{}.Set(0, 0, 'a').Set(11, 5, 'b')
	r := p.Shift(1, 1)
	if r.At(1, 1) != 'a' || r.At(0, 0) != 0 || r.Count() != 1 {
		t.Fatalf("Shift(1,1) = %s", r.Grid())
	}
	l := p.Shift(-1, -1)
	if l.At(10, 4) != 'b' || l.Count() != 1 {
		t.Fatalf("Shift(-1,-1) = %s", l.Grid())
	}
}

func TestFlipHMirrorsEachRow(t *testing.T) {
	p := Picture{}.Set(0, 2, 'a').Set(3, 2, Own)
	f := p.FlipH()
	if f.At(11, 2) != 'a' || f.At(8, 2) != Own || f.Count() != 2 || f.FlipH() != p {
		t.Fatalf("FlipH = %s", f.Grid())
	}
}

func TestLineFillsTheCellsBetweenTwoSamples(t *testing.T) {
	p := Picture{}.Line(1, 1, 6, 3, 'e')
	for _, c := range [][2]int{{1, 1}, {6, 3}} {
		if p.At(c[0], c[1]) != 'e' {
			t.Errorf("the end %v is not drawn", c)
		}
	}
	for x := 1; x < 6; x++ { // a shallow line has one cell in every column
		n := 0
		for y := 0; y < Size; y++ {
			if p.At(x, y) != 0 {
				n++
			}
		}
		if n != 1 {
			t.Errorf("column %d has %d cells", x, n)
		}
	}
	if (Picture{}).Line(5, 5, 5, 5, Own).Count() != 1 {
		t.Error("a line from a cell to itself is that cell")
	}
	if got := (Picture{}).Line(9, 9, 20, 20, Own); got.At(11, 11) != Own || got.Count() != 3 {
		t.Errorf("what is off the grid is dropped: %s", got.Grid())
	}
}

func TestOutlineKeepsTheCellsThatTouchAnEmptyOneOrTheEdge(t *testing.T) {
	p := Picture{}
	for y := 2; y <= 6; y++ {
		for x := 2; x <= 6; x++ {
			p = p.Set(x, y, 'k')
		}
	}
	o := p.Outline()
	if o.At(4, 4) != 0 || o.At(2, 2) != Own || o.At(4, 2) != Own || o.Count() != 16 {
		t.Fatalf("Outline = %s", o.Grid())
	}
	edge := Picture{}.Set(0, 0, 'a').Set(1, 0, 'a').Set(0, 1, 'a').Set(1, 1, 'a').Outline()
	if edge.Count() != 4 {
		t.Errorf("a 2 x 2 block has no inside: %s", edge.Grid())
	}
	full := (Picture{}).Invert().Outline()
	if full.Count() != 44 { // the ring of the grid
		t.Errorf("a full grid keeps its ring, got %d", full.Count())
	}
}
