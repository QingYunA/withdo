package theme

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestDefaultMacAccentIsIgnored(t *testing.T) {
	for _, c := range []ui.Color{ui.Transparent, ui.RGB(0, 122, 255), ui.RGB(10, 132, 255)} {
		if !isDefaultSystemAccent(c) {
			t.Errorf("%v should count as the default accent", c)
		}
	}
	if isDefaultSystemAccent(ui.RGB(255, 149, 0)) {
		t.Error("orange is a chosen accent")
	}
}

func TestTokensFollowTheSpec(t *testing.T) {
	light, dark := For(false, ui.Transparent), For(true, ui.Transparent)
	if light.Accent != ui.Hex("#5B5BD6") || dark.Accent != ui.Hex("#7C7CF0") {
		t.Error("default accents differ from the spec")
	}
	if light.Blocked() != ui.Hex("#E5484D") || dark.Blocked() != ui.Hex("#FF6369") {
		t.Error("blocked red differs from the spec")
	}
	if light.BgSelected.A != 31 || dark.BgSelected.A != 56 { // 12% and 22% of 255
		t.Errorf("selection alphas %d and %d", light.BgSelected.A, dark.BgSelected.A)
	}
	chosen := For(false, ui.RGB(255, 149, 0))
	if chosen.Accent != ui.RGB(255, 149, 0) {
		t.Error("a chosen system accent is ignored")
	}
	for _, s := range data.Statuses {
		if light.Status(s) == (ui.Color{}) {
			t.Errorf("no color for %s", s)
		}
	}
}
