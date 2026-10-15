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
	IconSend   = icon(`<path d="M8 13V3.2M3.6 7.4 8 3l4.4 4.4"/>`)
	IconAgents = icon(`<path d="m3 4.5 3.5 3.5L3 11.5M8 12h5"/>`)
	IconGitHub = icon(`<circle cx="8" cy="8" r="5.6"/><circle cx="8" cy="8" r="2.4" fill="currentColor" stroke="none"/>`)
	IconFolder = icon(`<path d="M2.5 4.8a1.3 1.3 0 0 1 1.3-1.3h2.3l1.4 1.6h4.7a1.3 1.3 0 0 1 1.3 1.3v5a1.3 1.3 0 0 1-1.3 1.3H3.8a1.3 1.3 0 0 1-1.3-1.3z"/>`)
)

// ComposerBorder is the outline of a settings control: stronger than a hairline.
func (t *Tokens) ComposerBorder() ui.Color { return t.Border.Alpha(0.2) }

// SendIdle is the round send button while there is nothing to send.
func (t *Tokens) SendIdle() ui.Color { return t.TextTertiary.Alpha(0.4) }
