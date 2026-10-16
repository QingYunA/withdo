package theme

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

var testPlan = HandOffPlan{
	Row:  ui.Rect{X: 260, Y: 400, W: 400, H: 36},
	You:  Pt{X: 258, Y: 296},
	Bot:  Pt{X: 710, Y: 296},
	Meet: Pt{X: 690, Y: 405},
}

func TestAHopDownToALowerRowArcsAboveItsStartAndNeverGoesNaN(t *testing.T) {
	const fx, fy, tx, ty = 20, 100, 180, 300
	top := float32(math.MaxFloat32)
	for e := time.Duration(0); e < HopLanding; e += time.Millisecond {
		f, ok := HopAt(e, fx, fy, tx, ty)
		if !ok || math.IsNaN(float64(f.Y)) || math.IsNaN(float64(f.X)) {
			t.Fatalf("at %v: %+v %v", e, f, ok)
		}
		top = min(top, f.Y)
	}
	if above := fy - top; above < HopPeak-1 || above > HopPeak+2 {
		t.Errorf("the arc peaks %.1f pt above the start, want about %.0f", above, HopPeak)
	}
	land, _ := HopAt(HopLanding, fx, fy, tx, ty)
	if math.Abs(float64(land.Y+land.H-(ty+HopSize))) > 0.01 {
		t.Errorf("lands at %+v, the feet should be at %v", land, ty+HopSize)
	}
}

func TestTheHandOffRunsInOrderAndTheSpritesEndHome(t *testing.T) {
	at := func(ms int) HandOffFrame { return HandOffAt(time.Duration(ms)*time.Millisecond, testPlan) }
	if f := at(0); f.YouOn || f.BotOn || f.ParcelOn || f.GhostOpacity != 1 || f.RowOpacity != 0 {
		t.Errorf("at 0 only the row is there: %+v", f)
	}
	if f := at(200); !f.ParcelOn || !f.YouOn || f.BotOn {
		t.Errorf("at 200 the parcel is out and the user's sprite on its way: %+v", f)
	}
	if f := at(700); !f.YouOn || !f.BotOn || !f.ParcelOn {
		t.Errorf("at 700 both sprites carry it to the middle: %+v", f)
	}
	// The agent catches: at the meeting it is squashed (wider and shorter), and the parcel is near.
	f := at(840)
	if !f.Bot.Landed || f.Bot.W <= HopSize || f.Bot.H >= HopSize {
		t.Errorf("at 840 the agent should squash as it catches: %+v", f.Bot)
	}
	if f.Parcel.X < testPlan.Meet.X-30 || f.Parcel.X > testPlan.Meet.X+30 {
		t.Errorf("at 840 the parcel is at %v, by the middle %v", f.Parcel.X, testPlan.Meet.X)
	}
	if f := at(1100); f.RowOpacity <= 0 || f.RowOpacity >= 1 || f.Wash <= 0 {
		t.Errorf("at 1100 the row is opening: %+v", f)
	}
	end := at(int((HandOffOpenAt + HandOffOpenFor + HopDuration) / time.Millisecond))
	if end.YouOn || end.BotOn || end.ParcelOn || end.RowOpacity != 1 {
		t.Errorf("when it is done the sprites are home and the row is whole: %+v", end)
	}
}

func TestTheSpritesNeverLeaveTheWindowAndNothingIsNaN(t *testing.T) {
	for e := time.Duration(0); e < HandOffEnd(false); e += 5 * time.Millisecond {
		f := HandOffAt(e, testPlan)
		for _, v := range []float32{f.You.X, f.You.Y, f.Bot.X, f.Bot.Y, f.Parcel.X, f.Parcel.Y, f.Ghost.W, f.Wash} {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				t.Fatalf("at %v: %+v", e, f)
			}
		}
		if f.YouOn && (f.You.X < 200 || f.You.X > 800 || f.You.Y < 250 || f.You.Y > 520) {
			t.Fatalf("at %v the user's sprite is at %v, %v", e, f.You.X, f.You.Y)
		}
	}
}

func TestTheWashStartsAtEightPercentAndFadesOverTwoSeconds(t *testing.T) {
	whole := HandOffAt(HandOffOpenAt+HandOffOpenFor, testPlan)
	if whole.Wash != HandOffWashMax || whole.RowOpacity != 1 {
		t.Errorf("when the row is whole: %+v", whole)
	}
	half := HandOffAt(HandOffOpenAt+HandOffOpenFor+HandOffWash/2, testPlan)
	if math.Abs(float64(half.Wash-HandOffWashMax/2)) > 0.001 {
		t.Errorf("half way %v, want %v", half.Wash, HandOffWashMax/2)
	}
	if gone := HandOffAt(HandOffEnd(false), testPlan); gone.Wash != 0 {
		t.Errorf("at the end %v", gone.Wash)
	}
}

func TestReduceMotionFadesTheRowOutAndInWithoutMoving(t *testing.T) {
	p := testPlan
	p.Reduced = true
	for e := time.Duration(0); e < HandOffEnd(true); e += 10 * time.Millisecond {
		f := HandOffAt(e, p)
		if f.YouOn || f.BotOn || f.ParcelOn {
			t.Fatalf("at %v something hops", e)
		}
	}
	if f := HandOffAt(0, p); f.GhostOpacity != 1 || f.RowOpacity != 0 || f.Ghost != p.Row {
		t.Errorf("at 0 %+v", f)
	}
	if f := HandOffAt(HandOffReduceIn, p); f.GhostOpacity != 0 || f.RowOpacity != 1 || f.Wash != HandOffWashMax {
		t.Errorf("after the fade %+v", f)
	}
	if HandOffEnd(true) >= HandOffEnd(false) {
		t.Error("the Reduce Motion hand-off should end sooner")
	}
}
