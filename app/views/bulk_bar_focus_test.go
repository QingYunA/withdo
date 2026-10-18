package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestTabReachesTheBulkBarAndTheArrowsWalkItsButtons(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	reached := false
	for range 40 {
		tt.Key(0, ui.KeyTab)
		if tt.Focused("Done") {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatal("Tab never reached the bar's first button")
	}
	tt.Key(0, ui.KeyRight)
	if !tt.Focused("Move to…") {
		t.Error("→ did not move the focus to the next button of the bar")
	}
}
