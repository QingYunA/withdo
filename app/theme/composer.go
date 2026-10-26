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

// The big input on Home (docs/design/ui-spec.md, Mine (home)) and the Agents panel beside the lists.
const (
	HomeComposerRadius float32 = 16 // the composer of Home
	AgentsPanelWidth   float32 = 300
	MineGap            float32 = 24  // between the lists and the Agents panel
	MinePad            float32 = 24  // the page's side padding when the panel is beside the lists and room is short
	MineListsMin       float32 = 508 // the narrowest the lists get beside the panel
	// MineSplitBelow is the content width under which the Agents panel is not drawn and its section
	// shows inline at the top of the lists: 880 = 2 x 24 + 508 + 24 + 300. The default 1100 pt window
	// has 883 pt of content, so it is just wide enough.
	MineSplitBelow = 2*MinePad + MineListsMin + MineGap + AgentsPanelWidth
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

// StopWell is the round stop button that takes the send button's place while the assistant works.
func (t *Tokens) StopWell() ui.Color { return t.Text.Alpha(0.08) }
