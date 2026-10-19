// Package pixmap is the 12 x 12 picture behind a category's icon: the same grid as the interface
// icons and the avatars. The library's icons are one-bit Bitmaps (36 hex digits in the generated
// table); what a category keeps is a Picture, 144 characters in the database, whose pixels are in
// the category's color or in one of a palette's. It imports nothing from the app, so the store, the
// domain and the generator share it.
package pixmap

import (
	"fmt"
	"math/bits"
	"strconv"
)

// Size is the side of the grid, in pixels.
const Size = 12

// HexLen is the length of a bitmap's hex form: 12 rows of 3 digits.
const HexLen = Size * 3

// GridLen is the length of a bitmap's grid form: 12 rows of 12 characters.
const GridLen = Size * Size

// Bitmap is the grid, one entry per row, row 0 at the top. Bit 11 of a row is its leftmost pixel.
// A Bitmap is a value: the methods that change it return a new one.
type Bitmap [Size]uint16

// Hex returns the 36 lowercase hex digits the database keeps, 3 a row.
func (b Bitmap) Hex() string {
	out := make([]byte, 0, HexLen)
	for _, r := range b {
		out = append(out, fmt.Sprintf("%03x", r)...)
	}
	return string(out)
}

// ParseHex reads what Hex wrote. Uppercase digits are refused, so one bitmap has one spelling.
func ParseHex(s string) (Bitmap, error) {
	var b Bitmap
	if len(s) != HexLen {
		return b, fmt.Errorf("a bitmap is %d hex digits, 12 rows of 3", HexLen)
	}
	for i := range b {
		part := s[i*3 : i*3+3]
		for _, c := range part {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return Bitmap{}, fmt.Errorf("%q is not a lowercase hex digit", c)
			}
		}
		v, err := strconv.ParseUint(part, 16, 16)
		if err != nil {
			return Bitmap{}, err
		}
		b[i] = uint16(v)
	}
	return b, nil
}

// MustParseHex is ParseHex for a constant, such as the generated icon table.
func MustParseHex(s string) Bitmap {
	b, err := ParseHex(s)
	if err != nil {
		panic("pixmap: " + err.Error())
	}
	return b
}

// Grid is the 12 rows one after the other, row 0 first, 12 characters each and no separators; "#" is
// a lit pixel and "." a dark one. A Picture keeps the same form.
func (b Bitmap) Grid() string {
	out := make([]byte, 0, GridLen)
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			if b.At(x, y) {
				out = append(out, '#')
			} else {
				out = append(out, '.')
			}
		}
	}
	return string(out)
}

// ParseRows reads 12 strings of 12 characters, "#" for a lit pixel and "." for a dark one: the
// hand-drawn icons in docs/design/category-icons.txt.
func ParseRows(rows []string) (Bitmap, error) {
	var b Bitmap
	if len(rows) != Size {
		return b, fmt.Errorf("a bitmap is %d rows, got %d", Size, len(rows))
	}
	for y, row := range rows {
		if len(row) != Size {
			return Bitmap{}, fmt.Errorf("row %d is %d characters, want %d", y+1, len(row), Size)
		}
		for x, c := range row {
			switch c {
			case '#':
				b = b.Set(x, y, true)
			case '.':
			default:
				return Bitmap{}, fmt.Errorf("row %d has %q; use # and .", y+1, c)
			}
		}
	}
	return b, nil
}

// At reports whether pixel (x, y) is lit; anything off the grid is not.
func (b Bitmap) At(x, y int) bool {
	if x < 0 || x >= Size || y < 0 || y >= Size {
		return false
	}
	return b[y]>>(Size-1-x)&1 == 1
}

// Set returns b with pixel (x, y) lit or cleared; a pixel off the grid changes nothing.
func (b Bitmap) Set(x, y int, on bool) Bitmap {
	if x < 0 || x >= Size || y < 0 || y >= Size {
		return b
	}
	bit := uint16(1) << (Size - 1 - x)
	if on {
		b[y] |= bit
	} else {
		b[y] &^= bit
	}
	return b
}

// Empty reports whether no pixel is lit.
func (b Bitmap) Empty() bool { return b == Bitmap{} }

// Count is the number of lit pixels.
func (b Bitmap) Count() int {
	n := 0
	for _, r := range b {
		n += bits.OnesCount16(r)
	}
	return n
}
