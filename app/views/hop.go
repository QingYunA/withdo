package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The hop. When an agent hands something over and the user is not looking at Mine, the agent's
// sprite jumps from the Agents row of the sidebar to the Mine row, lands by its badge, squashes and
// settles, and the badge counts up as it lands (theme.HopAt is the motion). Up to three items hop,
// 120 ms apart; more count at once. With Reduce Motion nothing moves: the badge fades in.
//
// The hop is a function of the clock the app already has (App.Now), drawn in an overlay only while
// it plays. A frame is asked for every 16 ms only while a hop or the fade is under way: a still
// window asks for none.

// hopFrameEvery is how often a frame is drawn while a hop plays.
const hopFrameEvery = 16 * time.Millisecond

// badgeFade is how long the badge takes to fade in when Reduce Motion is on.
const badgeFade = theme.MotionFast

// hopFlight is one agent's sprite on its way: whose, and when it takes off.
type hopFlight struct {
	id    int
	who   theme.Who
	start time.Time
}

// hopper is the state of the hops and of the badge beside Mine.
type hopper struct {
	flights  []hopFlight
	launched map[int]time.Time // task id -> the hand-over time already hopped or counted
	// agents and mine are where the two sidebar rows were drawn last frame.
	agents, mine ui.Rect
	shown        int       // handed-over rows the badge counts
	shownAt      time.Time // when the badge went from nothing to something
	asked        int       // frames asked for, which the tests look at
}

// ready reports whether both rows have been drawn once, so the hop knows where to go.
func (h *hopper) ready() bool { return h.agents.W > 0 && h.mine.W > 0 }

// landed reports whether a flight has touched down at now.
func (f hopFlight) landed(now time.Time) bool { return now.Sub(f.start) >= theme.HopLanding }

// over reports whether a flight is finished at now.
func (f hopFlight) over(now time.Time) bool { return now.Sub(f.start) >= theme.HopDuration }

// mineShown reports whether the user is looking at the Mine screen.
func (m *Main) mineShown() bool { return m.nav.screen == screenMine && !m.searching() }

// hopTick runs once a frame before the window is built. Looking at Mine marks what is handed over
// as seen; elsewhere, what is new starts a hop (or, with Reduce Motion, just the badge), and the
// badge's count is the unseen rows whose hop has landed.
func (m *Main) hopTick(c *ui.Context) {
	h, now := &m.hop, m.app.Now()
	if m.mineShown() {
		m.app.markSeen(m.minePage.FromAgents)
		h.flights, h.launched, h.shown = nil, nil, 0
		return
	}
	reduce := c.Preferences().ReduceMotion
	unseen := data.Unseen(m.mine.FromAgents, m.app.seenSet())
	h.prune(now)
	for _, it := range unseen {
		if at, ok := h.launched[it.ID]; ok && !it.Since.After(at) {
			continue
		}
		if !reduce && !h.ready() {
			h.ask(c, time.Millisecond) // the rows are not laid out yet: try again next frame
			continue
		}
		h.launch(it, m.app.taskWho(it.Task), now, reduce)
	}
	n := 0
	for _, it := range unseen {
		if !h.inFlight(it.ID, now) && h.launchedFor(it) {
			n++
		}
	}
	if n > 0 && h.shown == 0 {
		h.shownAt = now
	}
	h.shown = n
	switch {
	case len(h.flights) > 0:
		h.ask(c, hopFrameEvery)
	case reduce && n > 0 && now.Sub(h.shownAt) < badgeFade:
		h.ask(c, hopFrameEvery)
	}
}

// launchedFor reports whether the row was taken in hand: it hopped, or is counted without a hop.
func (h *hopper) launchedFor(it data.Item) bool {
	at, ok := h.launched[it.ID]
	return ok && !it.Since.After(at)
}

// ask draws another frame after d, and counts the asking.
func (h *hopper) ask(c *ui.Context, d time.Duration) {
	h.asked++
	c.After(d)
}

// launch takes a new handed-over row in hand: it hops, 120 ms after the one before it, unless three
// hops are already on their way or Reduce Motion is on, in which case it is only counted.
func (h *hopper) launch(it data.Item, who theme.Who, now time.Time, reduce bool) {
	if h.launched == nil {
		h.launched = map[int]time.Time{}
	}
	h.launched[it.ID] = it.Since
	if reduce || len(h.flights) >= theme.HopMax {
		return
	}
	start := now
	for _, f := range h.flights {
		if next := f.start.Add(theme.HopStagger); next.After(start) {
			start = next
		}
	}
	h.flights = append(h.flights, hopFlight{id: it.ID, who: who, start: start})
}

// prune drops the hops that are over.
func (h *hopper) prune(now time.Time) {
	kept := h.flights[:0]
	for _, f := range h.flights {
		if !f.over(now) {
			kept = append(kept, f)
		}
	}
	h.flights = kept
}

// inFlight reports whether the row's hop has not touched down yet.
func (h *hopper) inFlight(id int, now time.Time) bool {
	for _, f := range h.flights {
		if f.id == id && !f.landed(now) {
			return true
		}
	}
	return false
}

// badgeOpacity is the badge's opacity: it fades in with Reduce Motion, and is simply there
// otherwise.
func (m *Main) badgeOpacity(c *ui.Context) float32 {
	return badgeFadeOpacity(c.Preferences().ReduceMotion, m.app.Now(), m.hop.shownAt)
}

// badgeFadeOpacity is the opacity of a badge that appeared at shownAt, at now.
func badgeFadeOpacity(reduceMotion bool, now, shownAt time.Time) float32 {
	if !reduceMotion {
		return 1
	}
	return min(max(float32(now.Sub(shownAt))/float32(badgeFade), 0), 1)
}

// hopOverlay draws the sprites on their way, above the whole window, and nothing at all when none is.
func (m *Main) hopOverlay(c *ui.Context) {
	h, now := &m.hop, m.app.Now()
	var frames []struct {
		who   theme.Who
		frame theme.HopFrame
	}
	fromX, fromY, toX, toY := h.ends()
	for _, f := range h.flights {
		if fr, ok := theme.HopAt(now.Sub(f.start), fromX, fromY, toX, toY); ok {
			frames = append(frames, struct {
				who   theme.Who
				frame theme.HopFrame
			}{f.who, fr})
		}
	}
	if len(frames) == 0 {
		return
	}
	ui.Overlay(c, func() {
		for i, f := range frames {
			sprite := f.who.Sprite()
			ui.Box(c.Key("hop-"+itoa(i))).Absolute().Left(f.frame.X).Top(f.frame.Y).Size(f.frame.W, f.frame.H).Shrink(0).
				Draw(func(p *ui.Painter, r ui.Rect) { theme.DrawSprite(p, r, sprite) })
		}
	})
}

// ends are where a sprite takes off and lands (their top left corners): by the counts at the right
// of the Agents row and of the Mine row, so it hops a short way up and a little right and never
// covers the rows' names.
func (h *hopper) ends() (fromX, fromY, toX, toY float32) {
	toX = h.mine.X + h.mine.W - sidebarRowPad - theme.BadgeSize - hopBadgeGap - theme.HopSize
	toY = h.mine.Y + (h.mine.H-theme.HopSize)/2
	return toX - hopDrift, h.agents.Y + (h.agents.H-theme.HopSize)/2, toX, toY
}

// hopDrift is how far right the sprite moves on its way up.
const hopDrift float32 = 14

// hopBadgeGap is the room between a landed sprite and the badge it lands by.
const hopBadgeGap float32 = 4
