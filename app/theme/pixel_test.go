package theme

import "testing"

func TestSpritesAreTwelveSquareAndUseTheirPalette(t *testing.T) {
	all := map[string]*Sprite{"you": spriteYou, "assistant": spriteAssistant, "claude": spriteClaude,
		"codex": spriteCodex, "cursor": spriteCursor, "gemini": spriteGemini, "bot": bot("opencode")}
	for name, s := range all {
		if s.Height() != AvatarGrid {
			t.Errorf("%s: %d rows", name, s.Height())
		}
		for y, row := range s.Rows {
			if len([]rune(row)) != AvatarGrid {
				t.Errorf("%s row %d: %d pixels", name, y, len([]rune(row)))
			}
			for _, ch := range row {
				if _, ok := s.Palette[ch]; !ok && ch != '.' {
					t.Errorf("%s row %d: %q is not in the palette", name, y, ch)
				}
			}
		}
	}
}

func TestAgentAvatarRecognizesTheAgentsByTheirClientName(t *testing.T) {
	for name, want := range map[string]AvatarKind{
		"claude-code": AvatarClaude, "Claude Code": AvatarClaude, "codex-mcp-client": AvatarCodex,
		"cursor-vscode": AvatarCursor, "gemini-cli-mcp-client": AvatarGemini, "opencode": AvatarOther, "agent": AvatarOther,
	} {
		if got := AgentAvatar(name).Kind; got != want {
			t.Errorf("%q: kind %d, want %d", name, got, want)
		}
	}
}

func TestUnknownAgentsGetAColorByName(t *testing.T) {
	a, b := bot("opencode").Palette['m'], bot("opencode").Palette['m']
	if a != b {
		t.Error("the same name gave two colors")
	}
	if bot("agent").Palette['m'] != bot("").Palette['m'] {
		t.Error("an agent with no name is not the plain gray robot")
	}
	if bot("opencode").Palette['m'] == bot("").Palette['m'] {
		t.Error("a named agent kept the gray")
	}
}
