package theme

import (
	"math"
	"testing"
	"time"
)

func TestTheCarrierBobsOneToTwoPointsFourStepsASecond(t *testing.T) {
	top, steps, prev := float32(0), 0, float32(0)
	for e := time.Duration(0); e < time.Second; e += time.Millisecond {
		b := CarryBob(e)
		if b < 0 || b > CarryBobHeight+0.001 {
			t.Fatalf("at %v the bob is %v", e, b)
		}
		if prev > 0.1 && b <= 0.1 {
			steps++
		}
		top, prev = max(top, b), b
	}
	if top < 1 || steps < 3 || steps > 4 {
		t.Errorf("tallest %v, %d steps in a second", top, steps)
	}
	if CarryBob(-time.Second) != 0 {
		t.Error("bobbing before it started")
	}
}

func TestTheAgentReachesAndHopsAgainEveryEightHundredMilliseconds(t *testing.T) {
	const x, y = 100, 50
	rest := Reach(0, x, y)
	if rest.Y != y || rest.W != HopSize {
		t.Errorf("at rest %+v", rest)
	}
	up := Reach(300*time.Millisecond, x, y)
	if y-up.Y < HopPeak-3 || up.X != x {
		t.Errorf("mid hop %+v: should be about %v up", up, HopPeak)
	}
	if again := Reach(ReachEvery+300*time.Millisecond, x, y); again != up {
		t.Errorf("the second hop %+v differs from the first %+v", again, up)
	}
	for e := time.Duration(0); e < 3*time.Second; e += 7 * time.Millisecond {
		if f := Reach(e, x, y); math.IsNaN(float64(f.Y)) || f.Y > y+HopSize*0.31 {
			t.Fatalf("at %v the sprite is at %v", e, f.Y)
		}
	}
}

func TestTheAgentSquashesOnceAsItCatchesTheRow(t *testing.T) {
	f, ok := Catch(0, 100, 50)
	if !ok || !f.Landed || f.W <= HopSize || f.H >= HopSize {
		t.Errorf("at the drop %+v %v: wider and shorter", f, ok)
	}
	if got := f.Y + f.H; math.Abs(float64(got-(50+HopSize))) > 0.01 {
		t.Errorf("the feet are at %v, want %v", got, 50+HopSize)
	}
	if _, ok := Catch(CatchFor, 100, 50); ok {
		t.Error("still squashing when it should have settled")
	}
}

func TestTheWashStartsAtEightPercentAndFadesOverTwoSeconds(t *testing.T) {
	if Wash(0) != WashMax || Wash(WashFor) != 0 || Wash(-1) != 0 {
		t.Errorf("wash %v, %v, %v", Wash(0), Wash(WashFor), Wash(-1))
	}
	if h := Wash(WashFor / 2); math.Abs(float64(h-WashMax/2)) > 0.001 {
		t.Errorf("half way %v", h)
	}
}
