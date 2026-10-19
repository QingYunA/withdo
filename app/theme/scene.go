package theme

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The Summary scene (ui-spec, Summary; reference docs/design/mockups/v18.html). It is a grid of
// SceneW x SceneH cells drawn at SceneCell points a cell. Every frame is a pure function of the plan
// and the time: Plan.Frame says what is drawn, SceneFrame.Draw draws it, and nothing here keeps
// state, so a frame can be tested, and screenshot at a fixed time.

// SceneCell is the side of a cell, in points; SceneHeight the scene's height in points.
const (
	SceneCell   float32 = 3
	SceneHeight         = SceneH * SceneCell
)

// Rows of the scene, in cells from its top.
const (
	sceneFeet = 71 // where the feet stand: the middle of the path
	actorTop  = sceneFeet - 22
	binX      = 112 // the bin's left edge: centred on the scene
	binRim    = sceneFeet - 16
)

// SkyKind is the sky behind the scene.
type SkyKind int

const (
	SkyDay SkyKind = iota
	SkyDusk
	SkyNight
)

// file is the backdrop's file name under sprites/scenes/.
func (s SkyKind) file() string {
	switch s {
	case SkyDay:
		return "scenes/day"
	case SkyDusk:
		return "scenes/dusk"
	}
	return "scenes/night"
}

// SkyAt is the sky of a scene opened at now: day before 17:00, dusk from 17:00, night from 20:00.
func SkyAt(now time.Time) SkyKind {
	switch h := now.Hour(); {
	case h < 17:
		return SkyDay
	case h < 20:
		return SkyDusk
	}
	return SkyNight
}

// Actor is how a person of the scene is drawn: the folder of its pictures under sprites/actors/, and,
// for the robot, the color it is tinted.
type Actor struct {
	Folder string
	Tint   ui.Color
}

// YouActor is the user.
var YouActor = Actor{Folder: "you"}

// actorFolders are the agents that have a figure of their own, by the avatar the name matches (so an
// agent's avatar and its actor always agree), and the folder of each under sprites/actors/.
var actorFolders = map[AvatarKind]string{
	AvatarClaude:   "claude",
	AvatarCodex:    "codex",
	AvatarGemini:   "gemini",
	AvatarCursor:   "cursor",
	AvatarCopilot:  "copilot",
	AvatarOpencode: "opencode",
}

// ActorOf is how an agent is drawn, by the name it reported under: Claude Code, Codex, Gemini CLI,
// Cursor, Copilot and opencode have their own, any other agent is the robot, tinted by its avatar's
// color so the two always agree.
func ActorOf(agent string) Actor {
	if folder, ok := actorFolders[AgentAvatar(agent).Kind]; ok {
		return Actor{Folder: folder}
	}
	return Actor{Folder: ActorBot, Tint: botTint(agent)}
}

// neverTurned reports whether an actor's picture is a mark with a set direction, so it is not turned
// round when it stands right of the bin (turning Codex round would turn its `>_` into `<_`).
func (a Actor) neverTurned() bool {
	switch a.Folder {
	case "codex", "cursor", "gemini", "opencode":
		return true
	}
	return false
}

// OpKind says what a SceneOp draws.
type OpKind int

const (
	OpActor    OpKind = iota // an actor in a pose
	OpGood                   // a good as it rests, with its shadow
	OpGoodSpin               // a good in the air, turned Spin quarter turns
	OpProp                   // a prop: the bin, a lid, the star
	OpRect                   // a rectangle of cells in one color: dust, shadows, digits, fireflies
)

// SceneOp is one thing of a frame, in cells from the scene's top left.
type SceneOp struct {
	Kind OpKind
	// Name is the good's kind ("gem"), the prop's file ("props/bin") or the actor's pose ("lift").
	Name  string
	Actor Actor
	Spin  int
	Flip  bool // draw the picture turned left to right
	X, Y  int
	W, H  int      // OpRect
	Color ui.Color // OpRect
}

// SceneFrame is one picture of the scene: its sky and what is drawn over it, bottom to top.
type SceneFrame struct {
	Sky SkyKind
	Ops []SceneOp
}

// Hit is a good under the pointer: its index in the scene's goods.
type Hit struct {
	Good int
	// Day is the day of the week the good is in (always 0 for a day).
	Day int
	// X, Y are the good's top left cell.
	X, Y int
}

// SceneOffset is how many cells the scene's grid is shifted right of a column w points wide: a wider
// column gets some, and a narrower one a negative number, which crops both sides alike.
func SceneOffset(w float32) int {
	cells := int(w / SceneCell)
	d := cells - SceneW
	if d < 0 {
		return -((-d + 1) / 2)
	}
	return d / 2
}

// CellAt is the cell of the scene under a point of a column w points wide, given relative to its top
// left corner.
func CellAt(w, px, py float32) (cx, cy int) {
	x := px/SceneCell - float32(SceneOffset(w))
	return floorInt(x), floorInt(py / SceneCell)
}

func floorInt(f float32) int {
	n := int(f)
	if float32(n) > f {
		n--
	}
	return n
}

// millis is a duration in whole milliseconds.
func millis(d time.Duration) int { return int(d / time.Millisecond) }

// rect is a SceneOp that fills w x h cells with a color.
func rect(x, y, w, h int, c ui.Color) SceneOp {
	return SceneOp{Kind: OpRect, X: x, Y: y, W: w, H: h, Color: c}
}
