package theme

import "github.com/egoist/mygo/ui"

// The logo of a model provider is a pixel mark, like everything else that is a picture (see Pixel
// language in docs/design/ui-spec.md). The official marks of lobe-icons (MIT, logos/README.md) are
// sampled onto an 18 x 18 grid by docs/design/tools/sample_marks.py into provider_marks.go; the SVGs
// stay on disk as the sampler's source and are not part of the app. A mark with a color version is
// drawn in its colors; one that only has a single color (OpenAI, Kimi, Ollama) is drawn in the ink
// the caller gives, so it turns white in dark mode. Anthropic is the Claude Code crab, Magpie a
// tinted pixel robot and Custom a pixel plus.

// magpieSprite is the robot of Magpie, in its teal.
var magpieSprite = tintedBot("#0F766E")

// Sizes of a provider's logo in the Settings window and the composer: whole pixels, 1 pt a pixel
// at 18 and 2 pt at 36.
const (
	LogoCard float32 = 36 // the provider in use
	LogoRow  float32 = 18 // a provider set up, in a list
	LogoTile float32 = 36 // a tile of the grid
	LogoChip float32 = 18 // the model chip of the composer
)

// ProviderLogo draws the logo of the provider with preset id, size points square: its mark in its
// own colors, or in ink for a one-color mark and for Custom's plus. An id the app does not know
// gets the plus.
func ProviderLogo(c *ui.Context, id string, size float32, ink ui.Color) ui.Element {
	switch {
	case id == "anthropic":
		return SpriteIcon(c, spriteClaude, size)
	case id == "magpie":
		return SpriteIcon(c, magpieSprite, size)
	case oneColorMarks[id]:
		return SpriteIcon(c, inkedMark(providerMarks[id], ink), size)
	}
	if s, ok := providerMarks[id]; ok {
		return SpriteIcon(c, s, size)
	}
	return ui.Row(c).Size(size, size).Shrink(0).Center().Children(func() {
		ui.Icon(c, IconPlus).Size(size*2/3, size*2/3).TextColor(ink)
	})
}

// inkedMark is a one-color mark in ink: its key 'k' takes the color.
func inkedMark(s *Sprite, ink ui.Color) *Sprite {
	return &Sprite{Rows: s.Rows, Palette: map[rune]ui.Color{'k': ink}}
}
