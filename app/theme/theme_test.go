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
	if light.Accent != ui.Hex("#5B5BD6") || dark.Accent != ui.Hex("#5B5BD6") {
		t.Error("default accents differ from the spec")
	}
	if light.Blocked() != ui.Hex("#E5484D") || dark.Blocked() != ui.Hex("#E5484D") {
		t.Error("blocked red differs from the spec")
	}
	// Selection is neutral (slate 4) and does not follow the accent.
	chosen := For(false, ui.RGB(255, 149, 0))
	if light.BgSelected != ui.Hex("#E8E8EC") || dark.BgSelected != ui.Hex("#272A2D") || chosen.BgSelected != light.BgSelected {
		t.Error("selection is not neutral slate 4")
	}
	if light.AccentSoft != ui.Hex("#F0F1FE") || light.AccentInk != ui.Hex("#5753C6") || dark.BorderControl != ui.Hex("#43484E") {
		t.Error("accent.soft, accent.ink or border.control differ from the spec")
	}
	if chosen.Accent != ui.RGB(255, 149, 0) {
		t.Error("a chosen system accent is ignored")
	}
	for _, s := range data.Statuses {
		if light.Status(s) == (ui.Color{}) {
			t.Errorf("no color for %s", s)
		}
	}
}

// The M4c type scale: one step larger than the first version, section headers in sentence case.
func TestTypeScaleIsTheM4cOne(t *testing.T) {
	for name, c := range map[string]struct {
		got        Style
		size, line float32
		weight     int
	}{
		"title": {Title, 24, 32, 600}, "input": {Input, 17, 24, 400},
		"body": {Body, 15, 22, 400}, "body.strong": {BodyStrong, 15, 22, 500},
		"meta": {Meta, 13, 18, 400}, "section": {Section, 13, 18, 600}, "mono": {Mono, 12.5, 18, 400},
	} {
		if c.got.Size != c.size || c.got.Line != c.line || c.got.Weight != c.weight {
			t.Errorf("%s is %v/%v %d, want %v/%v %d", name, c.got.Size, c.got.Line, c.got.Weight, c.size, c.line, c.weight)
		}
	}
	if Section.Upper || Section.Tracking != 0 {
		t.Error("section headers are sentence case without tracking")
	}
	if GlyphSize != 18 || ContentMax != 780 || ContentPad != 48 || RadiusCard != 9 || SidebarRow != 32 {
		t.Error("metrics differ from the spec")
	}
}
