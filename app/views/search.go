package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// searchLimit is how many results the page shows.
const searchLimit = 100

// searchState is the sidebar search: what is typed, and the results found for it, which are looked
// up again when the query or the data changes.
type searchState struct {
	query string // bound to the field
	was   bool   // the page showed results last frame
	for_  string // the query the results answer
	rev   int    // the snapshot they answer
	found page.Found
}

// searching is true while there is something in the search field.
func (m *Main) searching() bool { return strings.TrimSpace(m.manage.search.query) != "" }

// clearSearch ends the search.
func (m *Main) clearSearch() { m.manage.search.query = "" }

// searchFound is the results for the query, looked up again when the query or the data changes.
func (m *Main) searchFound() page.Found {
	s := &m.manage.search
	q := strings.TrimSpace(s.query)
	if q == s.for_ && m.app.rev == s.rev && s.rev != 0 {
		return s.found
	}
	s.for_, s.rev = q, m.app.rev
	s.found = page.Found{Name: m.app.projectName}
	if q == "" {
		return s.found
	}
	tasks, err := m.app.Store.SearchTasks(q, searchLimit+1)
	if err != nil {
		m.app.snap.err = err
		return s.found
	}
	if len(tasks) > searchLimit {
		tasks, s.found.More = tasks[:searchLimit], true
	}
	s.found.Tasks = tasks
	return s.found
}

// searchField is the well at the top of the sidebar. Cmd+F puts the cursor in it.
func (m *Main) searchField(c *ui.Context, t *theme.Tokens) {
	s := &m.manage.search
	focused := m.searchRef.Focused(c)
	row := ui.Row(c.Key("search")).Height(30).Shrink(0).Margin(0, 10, 8).Padding(0, 8).Gap(6).AlignItems(ui.Center).
		Radius(7).Background(t.Text.Alpha(0.06)).Border(theme.Hairline, ui.Transparent)
	if focused {
		row.Border(theme.Hairline, t.Accent.Alpha(0.5))
	}
	row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	row.Children(func() {
		ui.Icon(c, theme.IconSearch).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary).Shrink(0)
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			theme.Compact.Apply(ui.TextInputBase(c.Key("search-input"), &s.query)).TextColor(t.Text).
				Label(m.say.ManageSearchLabel()).Bind(&m.searchRef).OnTextKey(m.searchKey)
			if s.query == "" {
				theme.Text(c, theme.Compact, t.TextTertiary, m.say.ManageSearch()).Absolute().Left(0).Top(0).PassThrough()
			}
		})
		switch {
		case s.query != "":
			clear := ui.ButtonBase(c.Key("search-clear")).Size(theme.HitTarget, theme.HitTarget).Margin(-4).Shrink(0).Center().
				Radius(theme.RadiusControl).TextColor(t.TextTertiary).Label(m.say.ManageSearchClear()).Cursor(ui.CursorPointer)
			clear.Children(func() { ui.Icon(c, theme.IconClose).Size(theme.IconSmall, theme.IconSmall) })
			if clear.Clicked() {
				m.clearSearch()
			}
		case !focused:
			theme.Text(c, theme.Meta, t.TextTertiary, keyHint("⌘F")).Shrink(0)
		}
	})
}

// searchKey is the keyboard of the field: Enter opens the highlighted result, or the first; Esc
// clears the text and leaves the field; Down walks into the results.
func (m *Main) searchKey(k ui.TextKey) bool {
	if k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyEnter:
		ids := m.pageNow().IDs()
		switch {
		case m.sel != 0 && containsID(ids, m.sel):
			m.openRow(m.sel)
		case len(ids) > 0:
			m.openRow(ids[0])
		}
	case ui.KeyEscape:
		m.clearSearch()
		k.SetText("")
		m.queue.leave = true
	case ui.KeyDown:
		m.queue.move = 1
	default:
		return false
	}
	return true
}

func containsID(ids []int, id int) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// searchScreen is the page of results, in place of the screen under it.
func (m *Main) searchScreen(c *ui.Context, t *theme.Tokens) {
	pg := m.pg
	sub := ""
	if pg.Total > 0 {
		sub = m.say.ManageSearchCount(pg.Total, strings.TrimSpace(m.manage.search.query))
	}
	m.pageHeader(c, t, m.say.ManageSearch(), sub, nil)
	if pg.Total == 0 {
		emptyState(c, t, m.say.ManageSearchNone(strings.TrimSpace(m.manage.search.query)), m.say.ManageSearchNoneBody(), 56, nil)
		return
	}
	for _, g := range pg.Sections {
		m.searchGroup(c, t, g)
	}
	if pg.More {
		theme.Paragraph(c, theme.Meta, t.TextTertiary, m.say.ManageSearchMore(searchLimit), 0).Margin(20, 0, 0)
	}
}

// searchGroup is a header, a hash (the Inbox's tray) and the project's name and count, and its rows.
func (m *Main) searchGroup(c *ui.Context, t *theme.Tokens, g page.Section) {
	name, key := g.Title, g.Title
	if g.NoProject {
		name, key = m.say.NavNoProject(), "inbox"
	}
	ui.Row(c.Key("search-group-"+key)).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(16, theme.Body.Line).Shrink(0).Center().Children(func() {
			if g.NoProject {
				ui.Icon(c, theme.IconInbox).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary)
				return
			}
			theme.Text(c, theme.Body, t.TextTertiary, "#")
		})
		theme.Text(c, theme.BodyStrong, t.Text, name)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(g.Tasks)))
	})
	for _, task := range g.Tasks {
		m.searchRow(c, t, task)
	}
}

// searchRow is a result: its status glyph, title with branch and pull request, and how long ago it
// last changed.
func (m *Main) searchRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	ph := m.rowPhase(c, task)
	row := m.rowShell(c, t, task.ID, plainRow, ph).AlignItems(ui.Center)
	row.Children(func() {
		m.glyphSlot(c, t, task, ph.fill)
		body(c, func() { m.titleLine(c, t, task, m.parentTitle(task), true) })
		right(c, func() {
			m.rowThumb(c, t, task.ID)
			plainText(c, t, m.clock.Age(task.UpdatedAt, m.app.Now()))
		})
	})
	if m.rowClicked(row, task.ID) {
		m.openRow(task.ID)
	}
}

// parentTitle is what a result says about where it belongs when it is a subtask: its parent's title,
// in the place a title keeps its project name. Any other task has none.
func (m *Main) parentTitle(task data.Task) string {
	if parent, ok := m.app.parentOf(task); ok {
		return parent.Title
	}
	return ""
}
