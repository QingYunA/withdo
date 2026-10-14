package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

const noticeFor = 1500 * time.Millisecond

// Panel is the quick panel: one input, what agents are doing, what is on my plate. A new Panel is
// made every time the flyout opens, so it always starts clean.
type Panel struct {
	app      *App
	query    string
	selected int // index into the entries; -1 means none
	noticeAt time.Time
	scroll   ui.ScrollState
	moved    bool // the selection moved this frame, so scroll it into view
}

// NewPanel makes a panel with nothing typed and nothing selected.
func NewPanel(a *App) *Panel { return &Panel{app: a, selected: -1} }

// View builds the panel for one frame.
func (p *Panel) View(c *ui.Context) {
	t := theme.Resolve(c)
	c.Root().Background(ui.Transparent)
	model := p.app.buildPanel(p.query)
	ids := model.entries()
	p.selected = min(p.selected, len(ids)-1)
	p.handleKeys(c, ids)

	shell := ui.Column(c).Fill()
	if !c.Vibrancy() {
		shell.Background(t.Bg).Radius(theme.RadiusPanel).Border(1, t.Border).Clip()
	}
	shell.Children(func() {
		p.inputRow(c, t, ids)
		hairline(c, t)
		p.errorStrip(c, t)
		p.list(c, t, model, ids)
		hairline(c, t)
		p.footer(c, t)
	})
}

// handleKeys takes the keys the input leaves alone: arrows move the highlight, Cmd+Enter opens
// the highlighted task, Escape closes the panel.
func (p *Panel) handleKeys(c *ui.Context, ids []int) {
	switch {
	case c.Shortcut(0, ui.KeyDown):
		p.selected, p.moved = moveSelection(p.selected, 1, len(ids)), true
	case c.Shortcut(0, ui.KeyUp):
		p.selected, p.moved = moveSelection(p.selected, -1, len(ids)), true
	case c.Shortcut(ui.Cmd, ui.KeyEnter):
		p.openSelected(ids)
	case c.Shortcut(0, ui.KeyEscape):
		p.app.Act.ClosePanel()
	}
}

func (p *Panel) openSelected(ids []int) {
	if p.selected >= 0 && p.selected < len(ids) {
		p.open(ids[p.selected])
	}
}

func (p *Panel) open(id int) {
	p.app.Act.OpenTask(id)
	p.app.Act.ClosePanel()
}

// submit is Enter: it opens the highlighted row if the arrow keys chose one, and otherwise adds
// what is typed to the inbox.
func (p *Panel) submit(ids []int) {
	if p.selected >= 0 {
		p.openSelected(ids)
		return
	}
	title := strings.TrimSpace(p.query)
	if title == "" {
		return
	}
	if err := p.app.AddToInbox(title); err != nil {
		return
	}
	p.query, p.noticeAt = "", p.app.Now()
}

func (p *Panel) inputRow(c *ui.Context, t *theme.Tokens, ids []int) {
	ui.Row(c).Height(56).Shrink(0).Padding(0, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			in := theme.Input.Apply(ui.TextInputBase(c.Key("input"), &p.query)).TextColor(t.Text).
				Label("Add a task").AutoFocus()
			if in.Changed() {
				p.selected = -1
			}
			if in.Submitted() {
				p.submit(ids)
			}
			if p.query == "" {
				theme.Text(c, theme.Input, t.TextTertiary, "Add a task…").Absolute().Left(0).Top(0).PassThrough()
			}
		})
		p.notice(c, t)
	})
}

// notice is the inline "Added to Inbox" confirmation, shown for 1.5 s and fading at the end.
func (p *Panel) notice(c *ui.Context, t *theme.Tokens) {
	if p.noticeAt.IsZero() {
		return
	}
	age := p.app.Now().Sub(p.noticeAt)
	if age >= noticeFor {
		return
	}
	c.After(noticeFor - age)
	opacity := float32(1)
	if age > noticeFor-300*time.Millisecond {
		opacity = float32(noticeFor-age) / float32(300*time.Millisecond)
	}
	ui.Row(c).Gap(6).Shrink(0).Opacity(opacity).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.TextSecondary)
		theme.Text(c, theme.Meta, t.TextSecondary, "Added to Inbox")
	})
}

func (p *Panel) errorStrip(c *ui.Context, t *theme.Tokens) {
	if p.app.Err() == nil {
		return
	}
	ui.Row(c).Shrink(0).Padding(8, 16).Gap(8).Background(t.BgSubtle).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, "Couldn't reach the data store. Retrying…")
		ui.Spacer(c)
		if link(c, t, "Retry").Clicked() {
			p.app.Refresh()
		}
	})
}

func (p *Panel) list(c *ui.Context, t *theme.Tokens, m panelModel, ids []int) {
	scroll := ui.Scroll(c).Grow(1).MinHeight(0).TrackScroll(&p.scroll)
	scroll.Children(func() {
		switch {
		case !p.app.snap.loaded:
			p.skeleton(c, t)
		case m.empty() && strings.TrimSpace(p.query) != "":
			p.noMatches(c, t)
		default:
			p.agents(c, t, m, ids)
			p.mine(c, t, m, ids, len(ids)-len(m.Mine))
		}
		ui.Box(c).Height(8).Shrink(0)
	})
}

func (p *Panel) noMatches(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Padding(20, 16).Gap(4).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, "Nothing matches")
		theme.Text(c, theme.Body, t.TextSecondary, "Press Enter to add it to your inbox.")
	})
}

func (p *Panel) skeleton(c *ui.Context, t *theme.Tokens) {
	for i := 0; i < 4; i++ {
		bar := ui.Box(c.Key(fmt.Sprint("sk", i))).Height(10).Margin(14, 16, 0).Radius(5).Background(t.BgSubtle).
			Width(float32(200 + (i%2)*80))
		bar.Opacity(0.5 + 0.5*bar.Loop("pulse", 1400*time.Millisecond, ui.Bounce(ui.EaseInOut)))
	}
}

func (p *Panel) agents(c *ui.Context, t *theme.Tokens, m panelModel, ids []int) {
	if len(m.Groups) == 0 && strings.TrimSpace(p.query) != "" {
		return
	}
	right := ""
	if m.Active > 0 {
		right = fmt.Sprintf("%d active", m.Active)
	}
	sectionHeader(c, t, "Agents", right)
	if len(m.Groups) == 0 {
		p.agentsEmpty(c, t)
		return
	}
	index := 0
	for _, g := range m.Groups {
		theme.Text(c, theme.Meta, t.TextSecondary, g.Name).Padding(6, 16, 2).Shrink(0)
		for _, r := range g.Rows {
			p.agentRow(c, t, r, index == p.selected)
			index++
		}
	}
}

func (p *Panel) agentsEmpty(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Padding(2, 16, 8).Gap(2).Shrink(0).Children(func() {
		theme.Text(c, theme.Body, t.Text, "No agent is reporting yet.")
		ui.Row(c).Gap(4).Children(func() {
			theme.Text(c, theme.Meta, t.TextSecondary, "Coding agents report here through the adhd-todo MCP server.")
		})
		if link(c, t, "How to connect").Margin(2, 0, 0).Clicked() {
			p.app.Act.OpenURL(readmeURL)
		}
	})
}

func (p *Panel) mine(c *ui.Context, t *theme.Tokens, m panelModel, ids []int, offset int) {
	if len(m.Mine) == 0 && strings.TrimSpace(p.query) != "" {
		return
	}
	sectionHeader(c, t, "Mine", "")
	if len(m.Mine) == 0 {
		ui.Column(c).Padding(2, 16, 8).Shrink(0).Children(func() {
			theme.Text(c, theme.Body, t.TextSecondary, "Nothing on your plate. Type above to add a task.")
		})
		return
	}
	for i, task := range m.Mine {
		p.mineRow(c, t, task, offset+i == p.selected)
	}
}

func (p *Panel) footer(c *ui.Context, t *theme.Tokens) {
	ui.Row(c).Height(36).Shrink(0).Padding(0, 16).Gap(16).AlignItems(ui.Center).Children(func() {
		primary := "add"
		if p.selected >= 0 {
			primary = "open"
		}
		hint(c, t, "↵", primary)
		hint(c, t, "↑↓", "select")
		if p.selected < 0 {
			hint(c, t, "⌘↵", "open")
		}
		ui.Spacer(c)
		hint(c, t, "esc", "close")
	})
}
