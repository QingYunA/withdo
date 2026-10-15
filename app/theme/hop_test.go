package theme

import (
	"math"
	"testing"
	"time"
)

func TestHopStartsAtTheAgentsRowAndLandsOnMine(t *testing.T) {
	const fx, fy, tx, ty = 20, 100, 180, 68
	start, ok := HopAt(0, fx, fy, tx, ty)
	if !ok || start.X != fx || start.Y != fy || start.Landed {
		t.Errorf("start %+v, %v", start, ok)
	}
	land, ok := HopAt(HopLanding, fx, fy, tx, ty)
	if !ok || !land.Landed || math.Abs(float64(land.X+land.W/2-(tx+HopSize/2))) > 0.01 || math.Abs(float64(land.Y+land.H-(ty+HopSize))) > 0.01 {
		t.Errorf("landing %+v: the feet are on the Mine row, in its middle", land)
	}
	if land.W <= HopSize || land.H >= HopSize {
		t.Errorf("landing %+v: squashed, wider and shorter", land)
	}
}

func TestHopArcPeaksJustAboveTheMineRow(t *testing.T) {
	const fx, fy, tx, ty = 166, 100, 180, 68
	top := float32(math.MaxFloat32)
	for e := time.Duration(0); e < HopDuration; e += time.Millisecond {
		if f, ok := HopAt(e, fx, fy, tx, ty); ok && !f.Landed && f.Y < top {
			top = f.Y
		}
	}
	if above := ty - top; above < HopPeak-1 || above > HopPeak+2 {
		t.Errorf("the arc peaks %.1f pt above the Mine row, want about %.0f", above, HopPeak)
	}
}

func TestHopIsOverAfterNineHundredMilliseconds(t *testing.T) {
	if _, ok := HopAt(-time.Millisecond, 0, 0, 0, 0); ok {
		t.Error("a hop that has not begun is drawn")
	}
	if _, ok := HopAt(HopDuration-time.Millisecond, 0, 0, 10, 10); !ok {
		t.Error("the last millisecond is not drawn")
	}
	if _, ok := HopAt(HopDuration, 0, 0, 10, 10); ok {
		t.Error("a finished hop is drawn")
	}
	end, _ := HopAt(HopDuration-time.Millisecond, 0, 0, 10, 10)
	if math.Abs(float64(end.W-HopSize)) > 1 || math.Abs(float64(end.H-HopSize)) > 1 {
		t.Errorf("settles at %vx%v, want the sprite's own size", end.W, end.H)
	}
}
