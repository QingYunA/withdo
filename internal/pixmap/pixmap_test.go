package pixmap

import (
	"os"
	"strings"
	"testing"
)

const (
	briefcaseHex = "0000600903fc4024024024024024023fc000"
	homeHex      = "0000600901082044024624924924923fc000"
)

func readSVG(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".svg")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHexRoundTrip(t *testing.T) {
	b := Bitmap{0, 0x060, 0x090, 0x3fc, 0x402, 0x402, 0x402, 0x402, 0x402, 0x402, 0x3fc, 0}
	if got := b.Hex(); got != briefcaseHex {
		t.Fatalf("Hex = %s, want %s", got, briefcaseHex)
	}
	back, err := ParseHex(briefcaseHex)
	if err != nil || back != b {
		t.Fatalf("ParseHex = %v, %v", back, err)
	}
}

func TestParseHexRejectsMalformedInput(t *testing.T) {
	for _, s := range []string{"", "00", strings.Repeat("0", 35), strings.Repeat("0", 37), strings.Repeat("g", 36), strings.Repeat("F", 36), strings.Repeat("0", 35) + "-"} {
		if _, err := ParseHex(s); err == nil {
			t.Errorf("ParseHex(%q) accepted", s)
		}
	}
}

func TestAtSetEmptyInvert(t *testing.T) {
	var b Bitmap
	if !b.Empty() {
		t.Fatal("zero bitmap is not empty")
	}
	b2 := b.Set(0, 0, true).Set(11, 11, true)
	if !b.Empty() || b2.Empty() {
		t.Fatal("Set must return a copy and leave the receiver alone")
	}
	if !b2.At(0, 0) || !b2.At(11, 11) || b2.At(1, 0) || b2.At(0, 1) {
		t.Fatalf("At wrong: %v", b2)
	}
	if b2[0] != 0x800 || b2[11] != 0x001 {
		t.Fatalf("MSB is the leftmost pixel: %03x %03x", b2[0], b2[11])
	}
	if b2.Set(0, 0, false).At(0, 0) {
		t.Fatal("Set(false) did not clear")
	}
	if b2.At(-1, 0) || b2.At(0, 12) || b2.Set(12, 0, true) != b2 {
		t.Fatal("out of range reads off and writes nothing")
	}
}

func TestPathMergesRunsPerRow(t *testing.T) {
	b := Bitmap{}.Set(1, 0, true).Set(2, 0, true).Set(3, 0, true).Set(5, 0, true).Set(0, 1, true)
	want := "M2 0h6v2h-6zM10 0h2v2h-2zM0 2h2v2h-2z"
	if got := b.Path(); got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestPictureSVGHasOnePathPerColor(t *testing.T) {
	var fills [Colors]string
	for i := range fills {
		fills[i] = "#0000" + string(rune('a'+i)) + "0"
	}
	svg := string(Picture{}.Set(0, 0, Own).Set(1, 0, 'a').Set(2, 0, 'l').SVG(fills))
	for _, want := range []string{`viewBox="0 0 24 24"`, `fill="currentColor"`, `<path d="M0 0h2v2h-2z"/>`,
		`<path fill="#0000a0" d="M2 0h2v2h-2z"/>`, `<path fill="#0000l0" d="M4 0h2v2h-2z"/>`} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG lacks %s: %s", want, svg)
		}
	}
	if n := strings.Count(svg, "<path"); n != 3 {
		t.Errorf("%d paths, want 3: %s", n, svg)
	}
	if empty := string(Picture{}.SVG(fills)); !strings.Contains(empty, `<path d="M0 0z"/>`) {
		t.Errorf("an empty picture draws nothing: %s", empty)
	}
}

func TestPathRoundTripsThroughFromRectSVG(t *testing.T) {
	for _, hex := range []string{briefcaseHex, homeHex} {
		b, _ := ParseHex(hex)
		got, errCells, err := FromRectSVG(b.Picture().SVG([Colors]string{}))
		if err != nil || errCells != 0 || got != b {
			t.Errorf("%s: got %s, %d, %v", hex, got.Hex(), errCells, err)
		}
	}
}

func TestBriefcaseAndHomeAreTheOldIcons(t *testing.T) {
	for name, hex := range map[string]string{"briefcase": briefcaseHex, "home": homeHex} {
		got, errCells, err := FromRectSVG(readSVG(t, name))
		if err != nil || errCells != 0 {
			t.Fatalf("%s: %d, %v", name, errCells, err)
		}
		if got.Hex() != hex {
			t.Errorf("%s = %s, want %s", name, got.Hex(), hex)
		}
	}
}

func TestHeartIsLosslessOnceShifted(t *testing.T) {
	// The heart's x coordinates are odd: the 12 x 12 grid only fits after moving it by one unit.
	b, errCells, err := FromRectSVG(readSVG(t, "heart"))
	if err != nil || errCells != 0 {
		t.Fatalf("heart: %d, %v", errCells, err)
	}
	want, _ := ParseRows([]string{
		"............",
		"...##...##..",
		"..#..#.#..#.",
		".#....#....#",
		".#.........#",
		".#.........#",
		"..#.......#.",
		"...#.....#..",
		"....#...#...",
		".....#.#....",
		"......#.....",
		"............",
	})
	if b != want {
		t.Fatalf("heart = %s, want %s", b.Hex(), want.Hex())
	}
}

func TestAlarmClockIsApproximate(t *testing.T) {
	b, errCells, err := FromRectSVG(readSVG(t, "alarm-clock"))
	if err != nil {
		t.Fatal(err)
	}
	// Its drawing mixes 1-unit and 2-unit sizes, so a 12 x 12 grid can only approximate it.
	if errCells == 0 || b.Empty() {
		t.Fatalf("alarm-clock: %d error cells, %d pixels", errCells, b.Count())
	}
}

func TestFromRectSVGRefusesWhatItCannotRaster(t *testing.T) {
	for name, svg := range map[string]string{
		"curve":    `<svg viewBox="0 0 24 24"><path d="M2 2C4 4 6 6 8 8z"/></svg>`,
		"arc":      `<svg viewBox="0 0 24 24"><path d="M2 2a2 2 0 0 1 4 4z"/></svg>`,
		"diagonal": `<svg viewBox="0 0 24 24"><path d="M2 2L8 6L2 6z"/></svg>`,
		"no path":  `<svg viewBox="0 0 24 24"></svg>`,
		"viewbox":  `<svg viewBox="0 0 16 16"><path d="M2 2h2v2H2z"/></svg>`,
	} {
		if _, _, err := FromRectSVG([]byte(svg)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestFromRectSVGAcceptsRelativeAndAbsoluteRightAngles(t *testing.T) {
	a, _, errA := FromRectSVG([]byte(`<svg viewBox="0 0 24 24"><path d="M2 2h2v2H2zM6 2L8 2L8 4L6 4Z"/></svg>`))
	b, _, errB := FromRectSVG([]byte(`<svg viewBox="0 0 24 24"><path d="m2 2 h2 v2 h-2 z m4 0 h2 v2 h-2 z"/></svg>`))
	if errA != nil || errB != nil || a != b || a.Count() != 2 {
		t.Fatalf("%v %v %v %v", a, b, errA, errB)
	}
}

func TestFromRectSVGFillsNonzero(t *testing.T) {
	// Two overlapping squares in one path: the overlap stays lit under nonzero fill.
	b, _, err := FromRectSVG([]byte(`<svg viewBox="0 0 24 24"><path d="M0 0h4v4H0zM2 2h4v4H2z"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if !b.At(1, 1) || !b.At(2, 2) || b.At(0, 2) {
		t.Fatalf("%v", b)
	}
}

func TestParseRows(t *testing.T) {
	rows := make([]string, 12)
	for i := range rows {
		rows[i] = "............"
	}
	rows[0] = "#..........#"
	b, err := ParseRows(rows)
	if err != nil || b[0] != 0x801 || b.Count() != 2 {
		t.Fatalf("%v %v", b, err)
	}
	for _, bad := range [][]string{rows[:11], append(append([]string{}, rows[:11]...), "..........."), append(append([]string{}, rows[:11]...), "...........x")} {
		if _, err := ParseRows(bad); err == nil {
			t.Errorf("ParseRows(%q) accepted", bad)
		}
	}
}

func TestGridRoundTrip(t *testing.T) {
	b, _ := ParseHex(briefcaseHex)
	grid := b.Grid()
	if len(grid) != GridLen || GridLen != 144 {
		t.Fatalf("grid is %d characters", len(grid))
	}
	want := strings.Join([]string{
		"............",
		".....##.....",
		"....#..#....",
		"..########..",
		".#........#.",
		".#........#.",
		".#........#.",
		".#........#.",
		".#........#.",
		".#........#.",
		"..########..",
		"............",
	}, "")
	if grid != want {
		t.Fatalf("grid = %s\nwant   %s", grid, want)
	}
	back, err := ParseGrid(grid)
	if err != nil || back != b.Picture() {
		t.Fatalf("ParseGrid = %v, %v", back, err)
	}
}
