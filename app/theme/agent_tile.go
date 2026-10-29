package theme

import (
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// The agents of Settings > Agents are pixel tiles: a connected agent's avatar bobs one sprite
// pixel (4 pt at 48 pt) and back on the 100 ms loop beat, three beats up and three down, each
// tile two beats behind the one before it. Like the spinner and the z's of the sleeping robot it
// is a pure function of the time, and with Reduce Motion it stands still.
const (
	// AgentBobStep is one beat of the bob: a loop is six beats.
	AgentBobStep = 100 * time.Millisecond
	// AgentBobBeats is the length of a loop: three beats up, three down.
	AgentBobBeats = 6
	// AgentBobLift is how far the avatar rises: one pixel of a 48 pt sprite.
	AgentBobLift float32 = 4
	// AgentAvatarSize and AgentAvatarStageH are the 48 pt avatar on its 56 pt stage.
	AgentAvatarSize   float32 = 48
	AgentAvatarStageH float32 = 56
	// agentGray is the opacity of an avatar of an agent that is not connected.
	agentGray float32 = 0.45
)

// AgentBobOffset is how far up the avatar of the tile at index is at now: AgentBobLift while the
// loop is on its first three beats, 0 on the last three. Neighbouring tiles are two beats apart.
// With Reduce Motion it is always 0.
func AgentBobOffset(now time.Time, index int, reduceMotion bool) float32 {
	if reduceMotion {
		return 0
	}
	beat := (int(now.UnixMilli()/AgentBobStep.Milliseconds()) + index*2) % AgentBobBeats
	if beat < AgentBobBeats/2 {
		return AgentBobLift
	}
	return 0
}

// AgentBobNext is how long until the bob changes.
func AgentBobNext(now time.Time) time.Duration {
	return AgentBobStep - time.Duration(now.UnixNano())%AgentBobStep
}

// AgentAvatarOnStage draws an agent's avatar at the foot of a stageH-high stage, lift points above
// its foot. The tile asks for the lift with AgentBobOffset; 0 stands.
func AgentAvatarOnStage(c *ui.Context, w Who, size, stageH, lift float32) ui.Element {
	return drawAvatarOnStage(c, w.Sprite(), size, stageH, lift)
}

// GrayAgentAvatarOnStage draws an avatar in gray at 45 % opacity, for an agent that is not
// connected. It stands still.
func GrayAgentAvatarOnStage(c *ui.Context, w Who, size, stageH float32) ui.Element {
	s := w.Sprite()
	if Resolve(c).Dark {
		s = s.inDark()
	}
	return drawAvatarOnStage(c, graySprite(s), size, stageH, 0)
}

func drawAvatarOnStage(c *ui.Context, s *Sprite, size, stageH, lift float32) ui.Element {
	if Resolve(c).Dark {
		s = s.inDark()
	}
	return ui.Box(c).Size(size, stageH).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		DrawSprite(p, ui.Rect{X: r.X, Y: r.Y + stageH - size - lift, W: size, H: size}, s)
	})
}

// graySprite is the sprite in shades of gray at agentGray opacity. The drawings are made once per
// sprite: a tile redraws every 100 ms while its agent bobs.
func graySprite(s *Sprite) *Sprite {
	graySpriteMu.Lock()
	defer graySpriteMu.Unlock()
	if g, ok := graySprites[s]; ok {
		return g
	}
	pal := make(map[rune]ui.Color, len(s.Palette))
	for k, c := range s.Palette {
		l := uint8(0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B) + 0.5)
		pal[k] = ui.RGBA(l, l, l, float32(c.A)/255*agentGray)
	}
	g := &Sprite{Rows: s.Rows, Palette: pal}
	graySprites[s] = g
	return g
}

var (
	graySpriteMu sync.Mutex
	graySprites  = map[*Sprite]*Sprite{}
)
