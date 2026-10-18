package views

import (
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// Pages an agent attached to a report (see Links from agents in the spec): the inspector shows each
// as a chip, and a task in review gets an Open page button. They open in the default browser; the
// app never renders or fetches them.

// linkChipHeight is the height of a link chip.
const linkChipHeight float32 = 26

// linkTipFor is how long the "File not found" tip stays after a click.
const linkTipFor = 2 * time.Second

// fileExists says whether a local file is there.
func (a *App) fileExists(p string) bool {
	if a.FileExists != nil {
		return a.FileExists(p)
	}
	_, err := os.Stat(p)
	return err == nil
}

// linkFile returns the path a file:// link names, and false for any other link.
func linkFile(l data.Link) (string, bool) {
	u, err := url.Parse(l.URL)
	if err != nil || !strings.EqualFold(u.Scheme, "file") {
		return "", false
	}
	return u.Path, true
}

// linkLabel is what a chip says: the title the agent gave, else the file's name, else the host and
// the end of the path ("github.com › pull/19").
func linkLabel(l data.Link) string {
	if l.Title != "" {
		return l.Title
	}
	if p, ok := linkFile(l); ok {
		return path.Base(p)
	}
	u, err := url.Parse(l.URL)
	if err != nil || u.Host == "" {
		return l.URL
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	if segs[0] == "" {
		return u.Host
	}
	return u.Host + " › " + strings.Join(segs[max(0, len(segs)-2):], "/")
}

// linkAddress is the full address a chip's tip shows: a file under the home folder as "~/…".
func (m *Main) linkAddress(l data.Link) string {
	if p, ok := linkFile(l); ok {
		return withTilde(p, m.app.Config.Home)
	}
	return l.URL
}

// openLink opens a page: a file with the app macOS has for it, any other link in the default browser.
func (m *Main) openLink(l data.Link) {
	if p, ok := linkFile(l); ok {
		m.app.Act.OpenFile(p)
		return
	}
	m.app.Act.OpenURL(l.URL)
}

// linkChips draws the pages a report attached, one chip each, wrapping.
func (m *Main) linkChips(c *ui.Context, t *theme.Tokens, r data.Report) {
	ui.Row(c.Key("links-"+itoa(r.ID))).Gap(6).Wrap().Margin(6, 0, 0).Children(func() {
		for i, l := range r.Links {
			m.linkChip(c, t, "link-"+itoa(r.ID)+"-"+itoa(i), l)
		}
	})
}

// linkChip is one page: the pixel file in accent and its name on a subtle well. Hovered, it takes an
// accent wash and a tip with the full address. A local file that is gone is drawn in tertiary, and a
// click says so in a tip and does nothing else.
func (m *Main) linkChip(c *ui.Context, t *theme.Tokens, key string, l data.Link) {
	in := &m.inspector
	file, isFile := linkFile(l)
	missing := isFile && !m.app.fileExists(file)
	label := linkLabel(l)
	btn := ui.ButtonBase(c.Key(key)).Height(linkChipHeight).Padding(0, 9, 0, 7).Radius(7).Shrink(1).MinWidth(0).
		Background(t.BgSubtle).Label(label).Cursor(ui.CursorPointer)
	icon, ink := t.Accent, t.Text
	switch {
	case missing:
		icon, ink = t.TextTertiary, t.TextTertiary
	case btn.Hovered():
		btn.Background(t.Accent.Alpha(0.12))
		icon, ink = t.Accent, t.Accent
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() {
		ui.Row(c).Gap(6).MinWidth(0).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, theme.IconFile).Size(theme.IconSmall, theme.IconSmall).Shrink(0).TextColor(icon)
			theme.Text(c, theme.Meta, ink, label).Shrink(1).MinWidth(0).Ellipsis("…")
		})
	})
	age := m.app.Now().Sub(in.linkTipAt)
	showing := in.linkTip == key && age < linkTipFor
	if showing {
		c.After(linkTipFor - age)
		m.linkTipOver(c, t, btn)
	} else {
		btn.Tooltip(m.linkAddress(l))
	}
	if btn.Clicked() {
		if missing {
			in.linkTip, in.linkTipAt = key, m.app.Now()
			return
		}
		m.openLink(l)
	}
}

// linkTipOver is the tip "File not found" above a chip: drawn by hand, because the tip of a hover
// is hidden by the click that asks for it.
func (m *Main) linkTipOver(c *ui.Context, t *theme.Tokens, anchor ui.Element) {
	ui.Overlay(c, func() {
		tip := ui.Box(c.Key("link-tip")).Absolute().PassThrough().Padding(4, 8).Radius(6).Background(t.Text).Role(ui.RoleTooltip)
		tip.AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, 6, 0)
		tip.Children(func() { theme.Text(c, theme.Meta, t.Bg, m.say.LinkNotFound()) })
	})
}
