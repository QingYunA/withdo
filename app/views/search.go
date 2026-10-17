package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
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
	model searchModel
}

// searchGroup is the results that belong to one project, or the Inbox.
type searchGroup struct {
	name  string
	inbox bool
	tasks []data.Task
}

type searchModel struct {
	groups []searchGroup
	total  int
	more   bool // there were more results than the page shows
}

// searching is true while there is something in the search field.
func (m *Main) searching() bool { return strings.TrimSpace(m.manage.search.query) != "" }

// clearSearch ends the search.
func (m *Main) clearSearch() { m.manage.search.query = "" }

// searchModel is the results for the query, grouped by project, newest group first, the Inbox last.
func (m *Main) searchModel() searchModel {
	s := &m.manage.search
	q := strings.TrimSpace(s.query)
	if q == s.for_ && m.app.rev == s.rev && s.rev != 0 {
		return s.model
	}
	s.for_, s.rev = q, m.app.rev
	s.model = searchModel{}
	if q == "" {
		return s.model
	}
	tasks, err := m.app.Store.SearchTasks(q, searchLimit+1)
	if err != nil {
		m.app.snap.err = err
		return s.model
	}
	if len(tasks) > searchLimit {
		tasks, s.model.more = tasks[:searchLimit], true
	}
	s.model.total = len(tasks)
	index := map[int]int{} // project id (0 for the Inbox) -> group
	for _, t := range tasks {
		id := 0
		if t.ProjectID != nil {
			id = *t.ProjectID
		}
		i, ok := index[id]
		if !ok {
			i = len(s.model.groups)
			index[id] = i
			s.model.groups = append(s.model.groups, searchGroup{name: m.app.projectName(t.ProjectID), inbox: id == 0})
		}
		s.model.groups[i].tasks = append(s.model.groups[i].tasks, t)
	}
	for i, g := range s.model.groups { // the Inbox goes last
		if g.inbox {
			s.model.groups = append(append(s.model.groups[:i:i], s.model.groups[i+1:]...), g)
			break
		}
	}
	return s.model
}

// searchIDs is the task ids of the results in the order they are drawn.
func (m *Main) searchIDs() []int {
	var ids []int
	for _, g := range m.searchModel().groups {
		for _, t := range g.tasks {
			ids = append(ids, t.ID)
		}
	}
	return ids
}

// searchField is the well at the top of the sidebar. Cmd+F puts the cursor in it.
func (m *Main) searchField(c *ui.Context, t *theme.Tokens) {
	s := &m.manage.search
	focused := m.searchRef.Focused(c)
	row := ui.Row(c.Key("search")).Height(30).Shrink(0).Margin(0, 10, 8).Padding(0, 8).Gap(6).AlignItems(ui.Center).
		Radius(7).Background(t.Text.Alpha(0.06)).Border(1, ui.Transparent)
	if focused {
		row.Border(1, t.Accent.Alpha(0.5))
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
			theme.Text(c, theme.Meta, t.TextTertiary, "⌘F").Shrink(0)
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
		ids := m.searchIDs()
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
	model := m.searchModel()
	sub := ""
	if model.total > 0 {
		sub = m.say.ManageSearchCount(model.total, strings.TrimSpace(m.manage.search.query))
	}
	m.pageHeader(c, t, m.say.ManageSearch(), sub, nil)
	if model.total == 0 {
		emptyState(c, t, m.say.ManageSearchNone(strings.TrimSpace(m.manage.search.query)), m.say.ManageSearchNoneBody(), 56, nil)
		return
	}
	for _, g := range model.groups {
		m.searchGroup(c, t, g)
	}
	if model.more {
		theme.Paragraph(c, theme.Meta, t.TextTertiary, m.say.ManageSearchMore(searchLimit), 0).Margin(20, 0, 0)
	}
}

// searchGroup is a header, a hash (the Inbox's tray) and the project's name and count, and its rows.
func (m *Main) searchGroup(c *ui.Context, t *theme.Tokens, g searchGroup) {
	name, key := g.name, g.name
	if g.inbox {
		name, key = m.say.NavInbox(), "inbox"
	}
	ui.Row(c.Key("search-group-"+key)).Margin(20, 0, 4).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(16, theme.Body.Line).Shrink(0).Center().Children(func() {
			if g.inbox {
				ui.Icon(c, theme.IconInbox).Size(theme.IconSmall, theme.IconSmall).TextColor(t.TextTertiary)
				return
			}
			theme.Text(c, theme.Body, t.TextTertiary, "#")
		})
		theme.Text(c, theme.BodyStrong, t.Text, name)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(g.tasks)))
	})
	for _, task := range g.tasks {
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
		right(c, func() { plainText(c, t, m.clock.Age(task.UpdatedAt, m.app.Now())) })
	})
	if row.Clicked() {
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
