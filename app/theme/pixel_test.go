package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestAccentBotIsTheRobotInTheAccentColorAndIsMadeOncePerColor(t *testing.T) {
	accent := ui.Hex("#5B5BD6")
	a, b := AccentBot(accent), AccentBot(accent)
	if a.Sprite() != b.Sprite() {
		t.Error("the same color made two sprites")
	}
	if a.Sprite().Palette['m'] != accent || len(a.Sprite().Rows) != AvatarGrid {
		t.Errorf("palette %v", a.Sprite().Palette)
	}
	if AccentBot(ui.Hex("#7C7CF0")).Sprite() == a.Sprite() {
		t.Error("another color shares the sprite")
	}
}

func TestSpritesAreTwelveSquareAndUseTheirPalette(t *testing.T) {
	all := map[string]*Sprite{"you": spriteYou, "assistant": spriteAssistant, "claude": spriteClaude,
		"codex": spriteCodex, "cursor": spriteCursor, "gemini": spriteGemini, "opencode": spriteOpencode, "copilot": spriteCopilot,
		"qwen": spriteQwen, "kimi": spriteKimi, "windsurf": spriteWindsurf, "zed": spriteZed, "cline": spriteCline,
		"magpie": magpieSprite, "bot": bot("someone")}
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
		"cursor-vscode": AvatarCursor, "gemini-cli-mcp-client": AvatarGemini, "agent": AvatarOther, "unknown-thing": AvatarOther,
		"opencode": AvatarOpencode, "Visual Studio Code": AvatarCopilot, "Visual Studio Code - Insiders": AvatarCopilot,
		"github-copilot": AvatarCopilot, "qwen-code-mcp-client": AvatarQwen, "qwen-cli-mcp-client-adhd-todo": AvatarQwen,
		"kimi-cli": AvatarKimi, "Zed": AvatarZed, "zed": AvatarZed, "windsurf-client": AvatarWindsurf, "cline": AvatarCline,
		"optimized-agent": AvatarOther,
	} {
		if got := AgentAvatar(name).Kind; got != want {
			t.Errorf("%q: kind %d, want %d", name, got, want)
		}
	}
}

func TestUnknownAgentsGetAColorByName(t *testing.T) {
	a, b := bot("someone").Palette['m'], bot("someone").Palette['m']
	if a != b {
		t.Error("the same name gave two colors")
	}
	if bot("agent").Palette['m'] != bot("").Palette['m'] {
		t.Error("an agent with no name is not the plain gray robot")
	}
	if bot("someone").Palette['m'] == bot("").Palette['m'] {
		t.Error("a named agent kept the gray")
	}
}

func TestADarkSpriteOnlyRecolorsWhatItDrawsAndKeepsTheLightOneIntact(t *testing.T) {
	for name, s := range map[string]*Sprite{"opencode": spriteOpencode, "copilot": spriteCopilot} {
		d := s.inDark()
		if d == s || len(d.Palette) != len(s.Palette) {
			t.Fatalf("%s: no separate dark sprite", name)
		}
		for k := range s.Dark {
			if _, ok := s.Palette[k]; !ok {
				t.Errorf("%s: dark key %q is not in the palette", name, k)
			}
		}
		if d.Palette['k'] == s.Palette['k'] {
			t.Errorf("%s: the dark color is the light one", name)
		}
	}
	if spriteClaude.inDark() != spriteClaude {
		t.Error("a sprite with no dark colors was copied")
	}
}
