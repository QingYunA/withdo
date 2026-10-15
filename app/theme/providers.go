package theme

import "github.com/egoist/mygo/ui"

// The logo of a model provider is a colored rounded square with a monogram, in the same color in
// light and dark (docs/design/mockups/v4.html, frame 6). The colors are keyed by the preset's id.
var providerColors = map[string]ui.Color{
	"anthropic":   ui.Hex("#D97757"),
	"openai":      ui.Hex("#111111"),
	"deepseek":    ui.Hex("#4D6BFE"),
	"kimi":        ui.Hex("#1A1A1A"),
	"zhipu":       ui.Hex("#3859FF"),
	"qwen":        ui.Hex("#615CED"),
	"gemini":      ui.Hex("#6C7CF2"),
	"openrouter":  ui.Hex("#6466F1"),
	"siliconflow": ui.Hex("#7C3AED"),
	"ollama":      ui.Hex("#222222"),
	"magpie":      ui.Hex("#0F766E"),
	"custom":      ui.Hex("#A1A1A6"),
}

// ProviderColor is the color of a provider's logo tile; one the app does not know gets the grey
// of Custom.
func ProviderColor(id string) ui.Color {
	if c, ok := providerColors[id]; ok {
		return c
	}
	return providerColors["custom"]
}

// Sizes of a provider's logo tile in the Settings window.
const (
	LogoCard float32 = 38 // the provider in use
	LogoRow  float32 = 28 // a provider set up, in a list
	LogoTile float32 = 32 // a tile of the grid
)

// ProviderLogo draws a provider's logo tile, size points square.
func ProviderLogo(c *ui.Context, id, monogram string, size float32) ui.Element {
	s := Style{Size: size * 0.4, Line: size * 0.5, Weight: 700}
	return ui.Row(c).Size(size, size).Shrink(0).Radius(size * 0.27).Background(ProviderColor(id)).Center().Children(func() {
		Text(c, s, ui.Hex("#FFFFFF"), monogram)
	})
}
