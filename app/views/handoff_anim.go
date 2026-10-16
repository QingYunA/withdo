package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The hand-off's motion (theme.HandOffAt): the dropped row folds into a parcel, the user's sprite
// hops from the Mine header to catch it and carries it to the gap between the columns, the agent's
// sprite hops from the Agents header to meet it, the parcel changes hands, both hop home, and the
// parcel opens into the new row of Waiting for an agent, which keeps an accent wash for two seconds.
// About 1.3 s. With Reduce Motion nothing hops: the row fades out on the left and in on the right.
//
// Like the hop to Mine, it is a function of the app's clock, drawn in an overlay only while it
// plays; a frame is asked for every 16 ms meanwhile, and none when it is over.

// handOffState is the hand-off: how each waiting task was started, the row being dragged, the motion
// that is playing and the Undo that ⌘Z does.
type handOffState struct {
	how     map[int]bool // task id -> started in Terminal (false: the prompt was copied)
	dragged dragged
	anim    *handOffAnim
	frame   theme.HandOffFrame // this frame's moment of anim
	undo    func()
	undoSeq int
	asked   int // frames asked for, which the tests look at
}

// dragged is the row being dragged and where its slot is.
type dragged struct {
	id   int
	rect ui.Rect
}

// handOffAnim is one hand-off on its way.
type handOffAnim struct {
	id    int
	start time.Time
	plan  theme.HandOffPlan
	who   theme.Who // the agent
	title string    // of the dropped row, which folds into the parcel
}

// begin starts the motion for a task just handed to the agent in res.
func (h *handOffState) begin(plan theme.HandOffPlan, task data.Task, res data.HandOffResult, now time.Time) {
	h.anim = &handOffAnim{id: task.ID, start: now, plan: plan, who: theme.AgentAvatar(res.Agent), title: task.Title}
	h.frame = theme.HandOffAt(0, plan)
}

// youAway reports whether the user's sprite is out of its header.
func (h *handOffState) youAway() bool { return h.anim != nil && h.frame.YouOn }

// rowOpacity is the opacity of a row of the tray: the one the parcel is opening into shows as it
// opens.
func (h *handOffState) rowOpacity(id int) float32 {
	if h.anim != nil && h.anim.id == id {
		return h.frame.RowOpacity
	}
	return 1
}

// rowWash is the strength of the accent wash of the new row.
func (h *handOffState) rowWash(id int) float32 {
	if h.anim != nil && h.anim.id == id {
		return h.frame.Wash
	}
	return 0
}

// handOffPlan measures the hand-off from where the frame before drew the columns: the dragged
// row, the two headers' avatars, and the middle of the gap the sprites meet in.
func (m *Main) handOffPlan(c *ui.Context, task data.Task) theme.HandOffPlan {
	geo := m.cols.geo
	row := geo.mine
	row.Y += 60
	row.H = theme.RowOneLine
	if m.ho.dragged.id == task.ID {
		row = m.ho.dragged.rect
	}
	home := func(r ui.Rect) theme.Pt {
		return theme.Pt{X: r.X + (r.W-theme.HopSize)/2, Y: r.Y + (r.H-theme.HopSize)/2}
	}
	meet := theme.Pt{X: (geo.mine.X + geo.mine.W + geo.tray.X) / 2, Y: row.Y + (row.H-theme.HopSize)/2}
	if geo.stacked {
		meet = theme.Pt{X: geo.tray.X + geo.tray.W/2, Y: (geo.mine.Y+geo.mine.H+geo.tray.Y)/2 - theme.HopSize/2}
	}
	return theme.HandOffPlan{Row: row, You: home(geo.you), Bot: home(geo.bot), Meet: meet, Reduced: c.Preferences().ReduceMotion}
}

// handOffTick runs once a frame before the window is built: it works out this frame's moment of the
// hand-off, asks for the next frame while it plays, and drops it when it is over or its screen is gone.
func (m *Main) handOffTick(c *ui.Context) {
	h := &m.ho
	if h.anim == nil {
		return
	}
	elapsed := m.app.Now().Sub(h.anim.start)
	if elapsed >= theme.HandOffEnd(h.anim.plan.Reduced) || !m.columnsActive() {
		h.anim, h.frame = nil, theme.HandOffFrame{}
		return
	}
	h.frame = theme.HandOffAt(elapsed, h.anim.plan)
	h.asked++
	c.After(hopFrameEvery)
}

// handOffOverlay draws what moves, above the whole window, and nothing at all when nothing does.
func (m *Main) handOffOverlay(c *ui.Context, t *theme.Tokens) {
	a, f := m.ho.anim, m.ho.frame
	if a == nil || (f.GhostOpacity <= 0 && !f.YouOn && !f.BotOn && !f.ParcelOn) {
		return
	}
	ui.Overlay(c, func() {
		if f.GhostOpacity > 0 {
			m.ghostRow(c, t, a.title, f)
		}
		if f.YouOn {
			spriteAt(c, "ho-you", f.You, theme.You)
		}
		if f.BotOn {
			spriteAt(c, "ho-bot", f.Bot, a.who)
		}
		if f.ParcelOn {
			r := f.Parcel
			ui.Box(c.Key("ho-parcel")).Absolute().Left(r.X).Top(r.Y).Size(r.W, r.H).Shrink(0).
				Draw(func(p *ui.Painter, r ui.Rect) { theme.DrawParcel(p, r) })
		}
	})
}

// spriteAt draws an avatar in a frame of the hop (squashed as it lands).
func spriteAt(c *ui.Context, key string, f theme.HopFrame, who theme.Who) {
	sprite := who.Sprite()
	ui.Box(c.Key(key)).Absolute().Left(f.X).Top(f.Y).Size(f.W, f.H).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) { theme.DrawSprite(p, r, sprite) })
}

// ghostRow is the dropped row as it folds into the parcel, or as it fades out with Reduce Motion.
func (m *Main) ghostRow(c *ui.Context, t *theme.Tokens, title string, f theme.HandOffFrame) {
	r := f.Ghost
	ui.Row(c.Key("ho-ghost")).Absolute().Left(r.X).Top(r.Y).Size(r.W, r.H).Shrink(0).Padding(0, 0, 0, rowSlot+listIndent).
		AlignItems(ui.Center).Radius(theme.RadiusCard).Background(t.Bg).Border(1, t.Border).Clip().Opacity(f.GhostOpacity).
		Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, title).Shrink(1).MinWidth(0).Ellipsis("…")
		})
}
