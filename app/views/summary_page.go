package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The scene refreshes at the screen's rate while something travels in it, and on the 100 ms beat of
// a loop (the sway, the fireflies) once it rests.
const (
	sceneFrame = 16 * time.Millisecond
	sceneLoop  = 100 * time.Millisecond
	tipGap     = 54 // points the hover tip sits right of the good it names
)

// summaryScreen is the Summary page: the header, the scene edge to edge of the content column, then
// the assistant's paragraph and the ledger in the page's own column. The whole page scrolls.
func (m *Main) summaryScreen(c *ui.Context, t *theme.Tokens) {
	now := m.app.Now()
	v := m.summaryFor(now)
	pad := m.sidePad(c)
	ui.Scroll(c.Key(m.pageKey())).Grow(1).MinHeight(0).TrackScroll(&m.scroll).Children(func() {
		col := ui.Column(c).FillWidth().Focusable().FocusRing(false)
		if !m.focusComposer && !m.searchRef.Focused(c) {
			col.AutoFocus()
		}
		col.Bind(&m.listRef).Children(func() {
			centered(c, "summary-head", pad, func() {
				m.summaryHeader(c, t, v, now)
				ui.Box(c).Height(14).Shrink(0)
			})
			m.summaryScene(c, t, v, now)
			centered(c, "summary-body", pad, func() {
				m.paragraphBlock(c, t, v, now)
				m.ledger(c, t, v, now)
				ui.Box(c).Height(48).Shrink(0)
			})
		})
	})
	m.exportSheetView(c, t, v)
}

// centered is a column of at most the width of a list, centered, with the page's side padding.
func centered(c *ui.Context, key string, pad float32, body func()) {
	ui.Column(c.Key(key)).FillWidth().AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(theme.ContentMax).Padding(0, pad).Children(body)
	})
}

// summaryHeader is the title, the Today | This week segment, ‹ date › paging back, and Export Markdown.
func (m *Main) summaryHeader(c *ui.Context, t *theme.Tokens, v *summaryView, now time.Time) {
	st := &m.sum
	ui.Row(c.Key("summary-header")).FillWidth().Height(36).Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Title, t.Text, m.say.NavSummary()).Shrink(0)
		sel := 0
		if st.week {
			sel = 1
		}
		if i := segmented(c, t, "summary-period", []string{m.say.SummarySegmentDay(), m.say.SummarySegmentWeek()}, sel); i >= 0 {
			st.week, st.back = i == 1, 0
		}
		m.summaryPaging(c, t, v, now)
		ui.Spacer(c)
		m.exportButton(c, t, v)
	})
}

// summaryPaging is ‹ date ›: a day or a week back, and forward as far as today.
func (m *Main) summaryPaging(c *ui.Context, t *theme.Tokens, v *summaryView, now time.Time) {
	ui.Row(c.Key("summary-paging")).Shrink(1).MinWidth(0).Gap(2).AlignItems(ui.Center).Children(func() {
		m.pagingButton(c, t, "back", theme.IconBack, m.say.SummaryPrevious(), true, func() { m.sum.back++ })
		theme.Text(c, theme.Compact, t.TextSecondary, m.periodTitle(v.period)).Shrink(1).MinWidth(0).Ellipsis("…").Margin(0, 4)
		m.pagingButton(c, t, "forward", theme.IconForward, m.say.SummaryNext(), m.sum.back > 0, func() { m.sum.back-- })
	})
}

func (m *Main) pagingButton(c *ui.Context, t *theme.Tokens, key string, icon *ui.SVG, label string, on bool, do func()) {
	btn := ui.ButtonBase(c.Key("summary-"+key)).Size(26, 26).Shrink(0).Center().Radius(6).Label(label).Tooltip(label)
	ink := t.TextTertiary
	if on {
		btn.Cursor(ui.CursorPointer)
		if btn.Hovered() {
			ink = t.TextSecondary
		}
	} else {
		ink = t.TextTertiary.Alpha(0.4)
	}
	btn.Children(func() { ui.Icon(c, icon).Size(theme.IconRow, theme.IconRow).TextColor(ink) })
	if on && btn.Clicked() {
		do()
	}
}

// periodTitle is the day or the week between the paging buttons.
func (m *Main) periodTitle(p data.Period) string {
	if p.Week {
		return m.weekRange(p)
	}
	return m.clock.DayTitle(p.Start)
}

// weekRange is "Oct 13 – 19", or "Oct 30 – Nov 5" when the week crosses a month.
func (m *Main) weekRange(p data.Period) string {
	last := p.Day(6)
	to := m.clock.MonthDay(last)
	if last.Month() == p.Start.Month() {
		to = m.say.SummaryDayOfMonth(itoa(last.Day()))
	}
	return m.say.SummaryWeekRange(m.clock.MonthDay(p.Start), to)
}

// summaryScene is the picture. It plays when the page opens or the period changes, then holds its
// last frame, swaying. A click on a good opens its task; a click anywhere else plays it again.
func (m *Main) summaryScene(c *ui.Context, t *theme.Tokens, v *summaryView, now time.Time) {
	st := &m.sum
	at, calm := m.sceneTime(c, v, now)
	frame := m.sceneFrame(v, at, calm)
	scene := ui.Box(c.Key("summary-scene")).FillWidth().Height(theme.SceneHeight).Shrink(0).ClipY().
		Cursor(ui.CursorPointer).Role(ui.RoleImage).Label(m.say.SummaryA11yScene())
	hit, hovered := m.sceneHit(scene, v, at)
	if hovered {
		scene.Cursor(ui.CursorPointer)
	}
	scene.Draw(func(p *ui.Painter, r ui.Rect) {
		st.sceneW, st.scene = r.W, r
		frame.Draw(p, r)
	})
	scene.Children(func() {
		if v.week != nil {
			m.dayLabels(c, v)
		}
		if hovered {
			m.sceneTip(c, v, hit)
		}
	})
	if scene.Clicked() {
		if hovered {
			m.openRow(hit.task.ID)
		} else {
			m.replayScene()
		}
	}
	m.sceneTick(c, v, at, calm)
}

// sceneFrame is the frame of the scene at a time.
func (m *Main) sceneFrame(v *summaryView, at time.Duration, calm bool) theme.SceneFrame {
	if v.week != nil {
		return v.week.Frame(at, calm)
	}
	return v.day.Frame(at, calm)
}

// sceneTick asks for the next frame: at the screen's rate while something moves, on the loop's beat
// while the scene rests, and never with Reduce Motion or a frozen frame.
func (m *Main) sceneTick(c *ui.Context, v *summaryView, at time.Duration, calm bool) {
	if calm || m.sum.freeze != nil {
		return
	}
	moving := false
	if v.week != nil {
		moving = v.week.Moving(at)
	} else {
		moving = v.day.Moving(at)
	}
	if moving {
		c.After(sceneFrame)
		return
	}
	c.After(sceneLoop)
}

// sceneGood is the good under the pointer: where it is and which task it is.
type sceneGood struct {
	theme.Hit
	task data.SummaryTask
}

// sceneHit is the good of the bin or of a stack under the pointer, if any.
func (m *Main) sceneHit(scene ui.Element, v *summaryView, at time.Duration) (sceneGood, bool) {
	px, py, over := scene.PointerPosition()
	if !over {
		return sceneGood{}, false
	}
	cx, cy := theme.CellAt(m.sum.sceneW, px, py)
	if v.week != nil {
		if h, ok := v.week.HitAt(at, cx, cy); ok && h.Good < len(v.byDay[h.Day]) {
			return sceneGood{Hit: h, task: v.byDay[h.Day][h.Good]}, true
		}
		return sceneGood{}, false
	}
	if h, ok := v.day.HitAt(at, cx, cy); ok && h.Good < len(v.tasks) {
		return sceneGood{Hit: h, task: v.tasks[h.Good]}, true
	}
	return sceneGood{}, false
}

// sceneTipFill is the dark of the hover tip: it is the same on a light and on a dark window.
var sceneTipFill = ui.RGBA(20, 20, 24, 0.88)

// sceneTip is the small dark tip beside a good: its title and its last report line.
func (m *Main) sceneTip(c *ui.Context, v *summaryView, g sceneGood) {
	off := float32(theme.SceneOffset(m.sum.sceneW))
	line := m.taskLine(g.task)
	if line == "" {
		line = g.task.Project
	}
	ui.Column(c.Key("summary-tip")).Absolute().Left((float32(g.X)+off)*theme.SceneCell+tipGap).Top(float32(g.Y)*theme.SceneCell-10).
		Padding(4, 8).Radius(6).Background(sceneTipFill).MaxWidth(360).PassThrough().Children(func() {
		white := ui.Hex("#FFFFFF")
		theme.Text(c, theme.Meta, white, g.task.Title).Ellipsis("…")
		if line != "" {
			theme.Text(c, theme.Meta, white.Alpha(0.7), line).Ellipsis("…")
		}
	})
}

// dayLabels are the names of the week's days under their stacks.
func (m *Main) dayLabels(c *ui.Context, v *summaryView) {
	off := float32(theme.SceneOffset(m.sum.sceneW))
	for d := 0; d < 7; d++ {
		ui.Row(c.Key("day-" + itoa(d))).Absolute().Left((float32(theme.DayLabelX(d)) + off) * theme.SceneCell).Bottom(4).
			Width(theme.DayLabelW * theme.SceneCell).Justify(ui.Center).PassThrough().Children(func() {
			theme.Text(c, theme.MetaStrong, ui.Hex("#FFFFFF").Alpha(0.8), m.clock.Weekday(v.period.Day(d)))
		})
	}
}
