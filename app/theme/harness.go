package theme

import "github.com/egoist/mygo/ui"

// HarnessMark draws the official mark of a coding harness at size points: 24 in the rows of On
// this Mac, 18 in the compact rows of More (see the Pixel language in docs/design/ui-spec.md).
// The ids are the agentsetup ones; Claude Code is the app's own 12 x 12 crab, not a sampled mark,
// and keeps it at either size.
//
// The marks are two grids, not one: the official drawings do not survive 12 x 12, so they are
// sampled at 24 and at 18, each with the dark variant the mockup draws (v14). A size other than
// those two stretches the nearest grid; callers should not ask for one.
func HarnessMark(c *ui.Context, id string, size float32) ui.Element {
	if id == "claude" {
		return SpriteIcon(c, spriteClaude, size)
	}
	small := size <= 18
	dark := Resolve(c).Dark
	var s *Sprite
	switch {
	case small && dark:
		s = harnessMarks18Dark[id]
	case small:
		s = harnessMarks18[id]
	case dark:
		s = harnessMarks24Dark[id]
	default:
		s = harnessMarks24[id]
	}
	if s == nil {
		return ui.Box(c).Size(size, size).Shrink(0)
	}
	return SpriteIcon(c, s, size)
}

// harnessSprite is the 18 x 18 mark of a harness as a sprite that knows its dark variant, for
// the places that draw a harness as an avatar (the robots' row) instead of in Settings.
func harnessSprite(id string) *Sprite {
	s := *harnessMarks18[id]
	s.dark = harnessMarks18Dark[id]
	return &s
}
