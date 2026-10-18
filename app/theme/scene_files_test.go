package theme

import (
	"image"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestEverySceneFileExistsAtItsSize(t *testing.T) {
	for file, size := range sceneFileSizes() {
		data, err := spriteFS.ReadFile("sprites/" + file)
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		cfg, _, err := image.DecodeConfig(strings.NewReader(string(data)))
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		if cfg.Width != size[0] || cfg.Height != size[1] {
			t.Errorf("%s is %d x %d pixels, want %d x %d", file, cfg.Width, cfg.Height, size[0], size[1])
		}
	}
}

func TestEveryActorHasAnIdlePose(t *testing.T) {
	for folder, poses := range actorPoses {
		if !hasPose(folder, "idle") {
			t.Errorf("actors/%s has no idle pose (%v)", folder, poses)
		}
	}
}

func TestAMissingPoseIsTheIdleOne(t *testing.T) {
	if actorBitmap("bot", "nowhere", ui.Color{}) != actorBitmap("bot", "idle", ui.Color{}) {
		t.Error("an unknown pose is not the idle bitmap")
	}
}

func TestTheSceneRobotFileHoldsTheThreeColorsThatTakeATint(t *testing.T) {
	img := loadNRGBA("actors/bot/idle.png")
	for i, shade := range botShades(ui.Hex(botTints[0])) {
		found := false
		for y := 0; y < ActorGrid && !found; y++ {
			for x := 0; x < ActorGrid; x++ {
				c := img.NRGBAAt(x, y)
				if c.A == 255 && c.R == shade.R && c.G == shade.G && c.B == shade.B {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("actors/bot/idle.png has no pixel of color %d of the gray robot (%v)", i, shade)
		}
	}
}

func TestTintingTheRobotChangesOnlyItsThreeColors(t *testing.T) {
	img := loadNRGBA("actors/bot/idle.png")
	tinted := tintNRGBA(img, ui.Hex("#D9488B"))
	changed, kept := 0, 0
	for y := 0; y < ActorGrid; y++ {
		for x := 0; x < ActorGrid; x++ {
			if img.NRGBAAt(x, y) != tinted.NRGBAAt(x, y) {
				changed++
			} else if img.NRGBAAt(x, y).A > 0 {
				kept++
			}
		}
	}
	if changed == 0 || kept == 0 {
		t.Errorf("%d pixels changed and %d kept: the white eyes and the outline stay, the body takes the tint", changed, kept)
	}
}

func TestAGoodTurnsFourQuarterTurnsBackToItself(t *testing.T) {
	img := cropNRGBA(loadNRGBA("goods/gem.png"), GoodGrid)
	turned := img
	for i := 0; i < 4; i++ {
		turned = turnNRGBA(turned)
	}
	for y := 0; y < GoodGrid; y++ {
		for x := 0; x < GoodGrid; x++ {
			if turned.NRGBAAt(x, y) != img.NRGBAAt(x, y) {
				t.Fatalf("pixel %d,%d changed", x, y)
			}
		}
	}
	once := turnNRGBA(img)
	if once.NRGBAAt(GoodGrid-1, 0) != img.NRGBAAt(0, 0) {
		t.Error("the top left corner does not go to the top right")
	}
}
