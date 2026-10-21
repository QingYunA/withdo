package theme

import "testing"

func TestTheImportantStarIsATwelveSquarePixelDrawing(t *testing.T) {
	for name, rows := range map[string][]string{"filled": starFilled, "outline": starOutline} {
		if len(rows) != 12 {
			t.Fatalf("%s: %d rows", name, len(rows))
		}
		for y, r := range rows {
			if len(r) != 12 {
				t.Errorf("%s row %d: %d pixels", name, y, len(r))
			}
		}
	}
	if IconImportant == nil || IconImportantOff == nil {
		t.Fatal("the star icons are not set")
	}
}
