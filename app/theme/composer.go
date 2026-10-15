package theme

import "github.com/egoist/mygo/ui"

// The composer's metrics (docs/design/ui-spec.md, Composer).
const (
	ComposerRadius  float32 = 12
	ComposerHeight  float32 = 50 // at rest, and with one line of text
	ComposerMinRows         = 1  // lines tall while empty
	ComposerMaxRows         = 6  // lines before the text scrolls
	SendSize        float32 = 28
)

// Icons of the composer and the Settings toolbar. Like the others they take the color of the text.
var (
	IconSend   = pixelIcon("arrow-up")
	IconAgents = pixelIcon("terminal")
	IconGitHub = pixelIcon("github")
	IconFolder = pixelIcon("folder")
)

// ComposerBorder is the outline of a settings control: stronger than a hairline.
func (t *Tokens) ComposerBorder() ui.Color { return t.Border.Alpha(0.2) }

// SendIdle is the round send button while there is nothing to send.
func (t *Tokens) SendIdle() ui.Color { return t.TextTertiary.Alpha(0.4) }
