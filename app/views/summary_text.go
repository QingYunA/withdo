package views

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The assistant's paragraph (ui-spec, Summary): written once, the first time a period is opened, by the
// model of the composer; kept; shown whole on the next visit, typed in when it is new.

const (
	typeEvery   = 30 * time.Millisecond  // a character
	caretBeat   = 500 * time.Millisecond // the caret is on, then off
	writeLimit  = 90 * time.Second
	bubbleEdge  = 3  // pt, the frame of the speech bubble
	tailW       = 12 // the stepped tail, in pt
	tailH       = 15
	bubbleGapY  = 16
	bubbleAvGap = 0
)

// paragraph is the paragraph of one period.
type paragraph struct {
	loaded  bool
	stored  *data.StoredSummary // nil until one is written
	writing bool
	failed  bool
	// typedAt is when new text began to type in; zero for a stored paragraph, which shows whole.
	typedAt time.Time
}

// paragraphOf is the paragraph of a period, read from the store the first time it is asked for.
func (m *Main) paragraphOf(p data.Period) *paragraph {
	st := &m.sum
	if st.paras == nil {
		st.paras = map[string]*paragraph{}
	}
	key := summaryKey(p)
	para := st.paras[key]
	if para == nil {
		para = &paragraph{}
		st.paras[key] = para
	}
	if !para.loaded {
		para.loaded = true
		if s, err := m.app.Store.GetSummary(p); err != nil {
			log.Printf("summary: %v", err)
		} else {
			para.stored = s
		}
	}
	return para
}

// hasModel reports whether the assistant has a model to write with.
func (m *Main) hasModel() bool { return m.app.Assistant != nil && m.app.Assistant.Configured() }

// writeIfDue writes the paragraph the first time a period with something in it is opened.
func (m *Main) writeIfDue(c *ui.Context, v *summaryView, para *paragraph) {
	if m.hasModel() && v.sum.Total() > 0 && para.stored == nil && !para.writing && !para.failed {
		m.writeParagraph(c, v, para)
	}
}

// writeParagraph asks the assistant for the paragraph of the period and the goods of its tasks, off
// the UI thread, and keeps the answer. The paragraph being replaced shows until the new one arrives.
func (m *Main) writeParagraph(c *ui.Context, v *summaryView, para *paragraph) {
	if para.writing || !m.hasModel() {
		return
	}
	para.writing, para.failed = true, false
	req := v.sum.Request(m.periodLabel(v.period), languageName(c.Locale()), agentName)
	asst, store, post, now, period := m.app.Assistant, m.app.Store, m.app.Act.Post, m.app.Now, v.period
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), writeLimit)
		defer cancel()
		text, err := asst.Summarize(ctx, req)
		var kept data.StoredSummary
		if err == nil {
			kept, err = store.PutSummary(period, data.StoredSummary{Paragraph: text.Paragraph, Model: text.Model, Kinds: text.Kinds})
		}
		post(func() {
			para.writing = false
			if err != nil {
				log.Printf("summary: %v", err)
				para.failed = true
				return
			}
			para.stored, para.typedAt = &kept, now()
		})
	}()
}

// languageName is the language the paragraph is written in, as the model is told.
func languageName(tag string) string {
	if strings.HasPrefix(tag, "zh") {
		return "Simplified Chinese"
	}
	return "English"
}

// periodLabel says which period it is, for the model: "Friday 2026-10-17", "2026-10-12 to 2026-10-18".
func (m *Main) periodLabel(p data.Period) string {
	if p.Week {
		return p.Start.Format("2006-01-02") + " to " + p.Day(6).Format("2006-01-02")
	}
	return p.Start.Format("Monday 2006-01-02")
}

// newSince reports whether more was finished since the paragraph was written, while the period is
// not over.
func (m *Main) newSince(v *summaryView, para *paragraph, now time.Time) bool {
	return para.stored != nil && !v.period.Over(now) && v.sum.FinishedSince(para.stored.CreatedAt) > 0
}

// shownText is the paragraph as far as it has typed in at now, and whether it is still typing. A
// stored paragraph, and any with Reduce Motion, shows whole.
func (p *paragraph) shownText(now time.Time, reduce bool) (string, bool) {
	text := p.stored.Paragraph
	if reduce || p.typedAt.IsZero() {
		return text, false
	}
	runes := []rune(text)
	n := int(max(0, now.Sub(p.typedAt)) / typeEvery)
	if n >= len(runes) {
		return text, false
	}
	return string(runes[:n]), true
}

// paragraphBlock is the place under the scene: the assistant's bubble, or, for a period with nothing
// finished in it, the quiet line that says so. With no model nothing but that line shows.
func (m *Main) paragraphBlock(c *ui.Context, t *theme.Tokens, v *summaryView, now time.Time) {
	if v.sum.Total() == 0 {
		theme.Paragraph(c, theme.Body, t.TextSecondary, m.emptyLine(v.period, now), 0).Margin(bubbleGapY+4, 0, 0)
		return
	}
	if !m.hasModel() {
		return
	}
	para := m.paragraphOf(v.period)
	m.writeIfDue(c, v, para)
	m.bubble(c, t, v, para, now)
}

// emptyLine says nothing was finished in a period.
func (m *Main) emptyLine(p data.Period, now time.Time) string {
	over := p.Over(now)
	switch {
	case p.Week && over:
		return m.say.SummaryEmptyPastWeek()
	case p.Week:
		return m.say.SummaryEmptyWeek()
	case over:
		return m.say.SummaryEmptyDay()
	}
	return m.say.SummaryEmptyToday()
}

// bubble is the assistant's avatar and its pixel speech bubble: a 3 pt square-cornered frame with a
// stepped tail, a header, Rewrite, and the paragraph.
func (m *Main) bubble(c *ui.Context, t *theme.Tokens, v *summaryView, para *paragraph, now time.Time) {
	ui.Row(c.Key("summary-ai")).FillWidth().Shrink(0).Margin(bubbleGapY, 0, 0).AlignItems(ui.Start).Children(func() {
		theme.AvatarIcon(c.Key("summary-ai-face"), theme.Assistant, 36).Shrink(0)
		ui.Box(c.Key("summary-ai-tail")).Size(tailW, tailH).Shrink(0).Margin(14, -bubbleEdge, 0, 12).
			Draw(func(p *ui.Painter, r ui.Rect) { drawTail(p, r, t.Text, t.Bg) })
		m.bubbleBody(c, t, v, para, now)
	})
}

// bubbleBody is the framed box. Its corners are cut a cell: four squares of the page's color.
func (m *Main) bubbleBody(c *ui.Context, t *theme.Tokens, v *summaryView, para *paragraph, now time.Time) {
	box := ui.Column(c.Key("summary-bubble")).Grow(1).MinWidth(0).Padding(10+bubbleEdge, 14+bubbleEdge, 12+bubbleEdge).
		Background(t.Bg).Border(bubbleEdge, t.Text)
	box.Children(func() {
		for i, at := range [4][2]bool{{true, true}, {false, true}, {true, false}, {false, false}} {
			corner := ui.Box(c.Key("cut-"+itoa(i))).Absolute().Size(bubbleEdge, bubbleEdge).Background(t.Bg).PassThrough()
			if at[0] {
				corner.Left(-bubbleEdge)
			} else {
				corner.Right(-bubbleEdge)
			}
			if at[1] {
				corner.Top(-bubbleEdge)
			} else {
				corner.Bottom(-bubbleEdge)
			}
		}
		m.bubbleHeader(c, t, v, para, now)
		m.bubbleText(c, t, para, now)
	})
}

// bubbleHeader is "Today · written by the assistant" and, at the right, Rewrite with a dot when more
// was finished since.
func (m *Main) bubbleHeader(c *ui.Context, t *theme.Tokens, v *summaryView, para *paragraph, now time.Time) {
	ui.Row(c.Key("summary-ai-head")).FillWidth().Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.SummaryWritten(m.bubblePeriod(v.period, now))).Shrink(1).MinWidth(0).Ellipsis("…")
		ui.Spacer(c)
		if m.newSince(v, para, now) && !para.writing {
			theme.Dot(c, t.Accent, 7).Shrink(0).Tooltip(m.say.SummaryRewriteNew())
		}
		if para.writing {
			theme.PixelSpinner(c.Key("summary-writing"), 12, t.Accent, theme.SpinnerFrame(now, c.Preferences().ReduceMotion)).Shrink(0)
			c.After(theme.SpinnerStep)
			return
		}
		if link(c, t, m.say.SummaryRewrite()).Clicked() {
			m.writeParagraph(c, v, para)
		}
	})
}

// bubblePeriod is the period in the bubble's header: Today, This week, or the day or the week it was.
func (m *Main) bubblePeriod(p data.Period, now time.Time) string {
	switch {
	case p.Week && !p.Over(now):
		return m.say.SummarySegmentWeek()
	case p.Week:
		return m.weekRange(p)
	}
	return m.clock.Day(p.Start, now)
}

// bubbleText is the paragraph, typing in behind a blinking block caret, or the line that says writing
// it failed, or just the caret while the first one is on its way.
func (m *Main) bubbleText(c *ui.Context, t *theme.Tokens, para *paragraph, now time.Time) {
	reduce := c.Preferences().ReduceMotion
	body := func(text string, caret bool) {
		e := theme.Paragraph(c.Key("summary-ai-text"), theme.Body, t.Text, text, 0).FillWidth().MinHeight(44)
		if caret {
			e.Children(func() {
				ink := t.Text
				if (now.UnixMilli()/caretBeat.Milliseconds())%2 == 1 {
					ink = ui.Transparent
				}
				ui.Text(c.Key("caret"), " ").Background(ink)
			})
			c.After(typeEvery)
		}
	}
	switch {
	case para.stored != nil:
		text, typing := para.shownText(now, reduce)
		body(text, typing || para.writing)
	case para.failed:
		theme.Paragraph(c, theme.Body, t.TextTertiary, m.say.SummaryFailed(), 0).MinHeight(44)
	default:
		body("", !reduce)
	}
}

// drawTail draws the stepped tail of the bubble at 3 pt a cell, 4 x 5 cells: the frame's color
// for the edge, the bubble's for the inside.
func drawTail(p *ui.Painter, r ui.Rect, edge, fill ui.Color) {
	cell := func(x, y, w, h float32, c ui.Color) {
		p.Fill(ui.Rect{X: r.X + x*bubbleEdge, Y: r.Y + y*bubbleEdge, W: w * bubbleEdge, H: h * bubbleEdge}, c, 0)
	}
	cell(0, 2, 1, 1, edge)
	cell(1, 1, 1, 3, edge)
	cell(2, 0, 2, 5, edge)
	cell(2, 1, 2, 3, fill)
	cell(1, 2, 1, 1, fill)
}
