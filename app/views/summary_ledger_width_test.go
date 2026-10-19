package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The width a ledger count is given must hold its widest digits at the weight of a title: "48" once
// ended in an ellipsis in the week's total.
func TestLedgerCountWidthHoldsAnyNumber(t *testing.T) {
	font := ui.Font{Size: ledgerCountStyle.Size, Weight: ledgerCountStyle.Weight}
	for n := 0; n < 1000; n++ {
		s := itoa(n)
		w, _ := ui.ShapeText(s, font, 0).Size()
		if box := ledgerCountWidth(n); w > box {
			t.Fatalf("%q is %.1f pt wide in a %.1f pt box", s, w, box)
		}
	}
}
