package theme

import (
	"embed"
	"path"

	"github.com/egoist/mygo/ui"
)

// The logo of a model provider is its official mark from lobe-icons (MIT, logos/README.md), drawn
// from the SVG that ships inside the app. A mark with a color version is drawn in its colors; one
// that only has a single color (OpenAI, Kimi, Ollama) follows the text color, so it turns white in
// dark mode. Magpie is a tinted pixel robot and Custom a plus.

//go:embed logos/*.svg
var logoFiles embed.FS

// logoFile is the file of each provider's mark, keyed by the preset's id, and whether it is drawn
// in its own colors.
var logoFile = map[string]struct {
	name  string
	color bool
}{
	"anthropic":   {"claude-color", true},
	"openai":      {"openai", false},
	"deepseek":    {"deepseek-color", true},
	"kimi":        {"kimi", false},
	"zhipu":       {"zhipu-color", true},
	"qwen":        {"qwen-color", true},
	"gemini":      {"gemini-color", true},
	"openrouter":  {"openrouter-color", true},
	"siliconflow": {"siliconcloud-color", true},
	"ollama":      {"ollama", false},
}

// logos are the parsed marks, by preset id; parsed once, not in a view.
var logos = parseLogos()

func parseLogos() map[string]*ui.SVG {
	out := make(map[string]*ui.SVG, len(logoFile))
	for id, f := range logoFile {
		b, err := logoFiles.ReadFile(path.Join("logos", f.name+".svg"))
		if err != nil {
			panic("theme: " + err.Error())
		}
		out[id] = ui.MustParseSVG(b)
	}
	return out
}

// magpieSprite is the robot of Magpie, in its teal.
var magpieSprite = tintedBot("#0F766E")

// Sizes of a provider's logo in the Settings window and the composer.
const (
	LogoCard float32 = 38 // the provider in use
	LogoRow  float32 = 28 // a provider set up, in a list
	LogoTile float32 = 32 // a tile of the grid
	LogoChip float32 = 16 // the model chip of the composer
)

// ProviderLogo draws the logo of the provider with preset id, size points square: its mark in its
// own colors, or in the text color for a one-color mark and for Custom's plus (set it with
// TextColor on the result; it is what the text around it has otherwise). An id the app does not
// know gets the plus.
func ProviderLogo(c *ui.Context, id string, size float32) ui.Element {
	if id == "magpie" {
		return SpriteIcon(c, magpieSprite, size)
	}
	f, ok := logoFile[id]
	if !ok {
		return ui.Row(c).Size(size, size).Shrink(0).Center().Children(func() {
			ui.Icon(c, IconPlus).Size(size*0.62, size*0.62)
		})
	}
	if f.color {
		return ui.Image(c, logos[id]).Size(size, size).Shrink(0)
	}
	return ui.Icon(c, logos[id]).Size(size, size).Shrink(0)
}
