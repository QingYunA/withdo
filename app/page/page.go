// Package page answers "what is this page showing": from the screen the sidebar chose, the search
// over it, the tasks and the fold state it returns the page, its sections and their rows in the order
// they are drawn. Painting and the arrow keys read the same sections, so they cannot drift apart. The
// package is pure: no windows, no Store, no clock.
package page

import (
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// Screen is what the sidebar selected.
type Screen int

const (
	Mine Screen = iota
	Agents
	Inbox
	Project
	Completed
	Calendar
	Summary
	Recent
)

// Kind is what the page is: the screen, or the search results over it, and for a project how it is
// drawn.
type Kind int

const (
	KindMine Kind = iota
	KindAgents
	KindCompleted
	KindCalendar
	KindSummary
	KindRecent
	KindSearch
	KindList    // the Inbox, or a project as statuses
	KindColumns // a project with tasks: Mine and Agents side by side
	KindBoard   // a project as lanes
)

// The Done rows a page shows before "Show all": a list or a column shows three, a board lane five.
const (
	ListDoneRows = 3
	LaneDoneRows = 5
)

// SectionKind says what a section is.
type SectionKind int

const (
	SecStatus  SectionKind = iota // the tasks of one status; Backlog is folded into one line
	SecWaiting                    // handed to an agent that has not reported yet
	SecLane                       // a board lane, drawn even when empty
	SecProject                    // search results of one project, or of the Inbox
	SecDay                        // tasks finished on one day
)

// Section is a header and its rows.
type Section struct {
	Kind   SectionKind
	Status data.Status // SecStatus and SecLane
	Col    int         // 0, or 1 for the Agents column
	Title  string      // SecProject: the project's name
	Inbox  bool        // SecProject: the Inbox
	Day    time.Time   // SecDay: the day, in the location of the Now the page was built for
	Tasks  []data.Task // the rows drawn, which key the arrows; none for a Backlog that is folded
	Total  int         // the rows there are, before the fold
	Open   bool        // Backlog: its rows are shown
}

// Hidden is how many rows the fold keeps back.
func (s Section) Hidden() int { return s.Total - len(s.Tasks) }

// Fold is what the user opened in a list or column: Backlog, and every Done task.
type Fold struct{ Backlog, AllDone bool }

// Col counts one column of a project, or the whole list in Cols[0].
type Col struct {
	Total  int // every task, whatever its status
	Active int // those neither finished nor in the backlog
}

// Scope picks the tasks a list shows: the Inbox or one project, narrowed to a milestone.
type Scope struct {
	Inbox     bool
	Project   int
	Milestone int // 0 for all
}

// Has says whether the scope shows the task. A subtask lives under its parent.
func (s Scope) Has(t data.Task) bool {
	switch {
	case t.IsSubtask():
		return false
	case s.Inbox:
		return t.ProjectID == nil
	case t.ProjectID == nil || *t.ProjectID != s.Project:
		return false
	}
	return s.Milestone == 0 || (t.MilestoneID != nil && *t.MilestoneID == s.Milestone)
}

// Found is the tasks a search found, in the order the Store gave them.
type Found struct {
	Tasks []data.Task
	More  bool              // there were more than it lists
	Name  func(*int) string // the name of a project, for a project id
}

// In is everything a page is made of.
type In struct {
	Screen    Screen
	Searching bool // a search is on: its results replace the screen
	Board     bool // the project is shown as a board

	// Tasks are the scope's tasks as lists show them while a task completes; Cards the same as the
	// board shows them. Only a project's board reads Cards.
	Tasks, Cards []data.Task
	// Folds are the list's, and for a project drawn as columns Mine's and the Agents column's.
	Folds [2]Fold

	Mine   data.Mine         // the open page of Mine
	Agents data.AgentsScreen // and of Agents
	Found  Found             // the results, while Searching
	Done   []data.Task       // the finished tasks, newest first, on Completed
	Now    time.Time         // gives Completed its days their location
}

// Page is what a screen shows.
type Page struct {
	Kind     Kind
	Composer bool // the page has a composer
	Sections []Section
	Cols     [2]Col
	Total    int  // Search: the results
	More     bool // Search: there were more than it lists
	ids      []int
}

// Build makes the page.
func Build(in In) Page {
	p := Page{Kind: kindOf(in)}
	switch p.Kind {
	case KindSearch:
		p.search(in.Found)
	case KindMine:
		p.ids = in.Mine.IDs()
	case KindAgents:
		p.ids = in.Agents.IDs()
	case KindCompleted:
		p.days(in.Done, in.Now)
	case KindBoard:
		p.board(in.Cards, in.Folds[0].AllDone)
	case KindColumns:
		p.columns(in.Tasks, in.Folds)
	case KindList:
		p.column(0, in.Tasks, in.Folds[0])
	}
	p.Composer = hasComposer(p.Kind)
	for _, s := range p.Sections {
		for _, t := range s.Tasks {
			p.ids = append(p.ids, t.ID)
		}
	}
	return p
}

func kindOf(in In) Kind {
	if in.Searching {
		return KindSearch
	}
	switch in.Screen {
	case Mine:
		return KindMine
	case Agents:
		return KindAgents
	case Completed:
		return KindCompleted
	case Calendar:
		return KindCalendar
	case Summary:
		return KindSummary
	case Recent:
		return KindRecent
	case Project:
		switch {
		case len(in.Tasks) == 0: // an empty project reads the same in every view
		case in.Board:
			return KindBoard
		default:
			return KindColumns
		}
	}
	return KindList
}

// hasComposer: Mine, Agents, a project, the Inbox and the Calendar do; Completed, Summary, Recent
// changes and the search results do not.
func hasComposer(k Kind) bool {
	switch k {
	case KindMine, KindAgents, KindList, KindColumns, KindBoard, KindCalendar:
		return true
	}
	return false
}

// IDs are the task ids of the rows in the order they are drawn, which is the order the arrow keys
// walk. The Calendar, Summary and Recent changes have none: their keys, or the mouse, do the work.
func (p Page) IDs() []int { return p.ids }

// Selectable says whether the page lists tasks that can be selected together.
func (p Page) Selectable() bool { return len(p.ids) > 0 }

// Of returns the sections of one column.
func (p Page) Of(col int) []Section {
	var out []Section
	for _, s := range p.Sections {
		if s.Col == col {
			out = append(out, s)
		}
	}
	return out
}

// DoneAt is when a finished task was finished.
func DoneAt(t data.Task) time.Time {
	if t.CompletedAt != nil {
		return *t.CompletedAt
	}
	return t.UpdatedAt
}

// HandedAt is when the task was handed over.
func HandedAt(t data.Task) time.Time {
	if t.HandedAt != nil {
		return *t.HandedAt
	}
	return t.UpdatedAt
}

// foldDone is the one Done fold: the newest limit rows, or all once "Show all" was clicked.
func foldDone(done []data.Task, limit int, all bool) []data.Task {
	if all || len(done) <= limit {
		return done
	}
	return done[:limit]
}

// listGroups are the statuses every list draws, in order.
var listGroups = []data.Status{data.Blocked, data.Review, data.Doing, data.Todo}

// column appends the sections of the Inbox, a project or one of its columns: the statuses, Done, and
// Backlog folded into a line.
func (p *Page) column(col int, tasks []data.Task, f Fold) {
	by := map[data.Status][]data.Task{}
	for _, t := range tasks {
		by[t.Status] = append(by[t.Status], t)
	}
	c := &p.Cols[col]
	c.Total += len(tasks)
	for _, s := range listGroups {
		ts := by[s]
		if len(ts) == 0 {
			continue
		}
		SortForList(ts)
		c.Active += len(ts)
		p.Sections = append(p.Sections, Section{Kind: SecStatus, Status: s, Col: col, Tasks: ts, Total: len(ts)})
	}
	done, backlog := by[data.Done], by[data.Backlog]
	if len(done) > 0 {
		sort.SliceStable(done, func(i, j int) bool { return DoneAt(done[i]).After(DoneAt(done[j])) })
		p.Sections = append(p.Sections, Section{Kind: SecStatus, Status: data.Done, Col: col,
			Tasks: foldDone(done, ListDoneRows, f.AllDone), Total: len(done)})
	}
	if len(backlog) > 0 {
		SortForList(backlog)
		s := Section{Kind: SecStatus, Status: data.Backlog, Col: col, Total: len(backlog), Open: f.Backlog}
		if f.Backlog {
			s.Tasks = backlog
		}
		p.Sections = append(p.Sections, s)
	}
}

// columns splits a project's tasks by who wrote them: Mine's, and the Agents', which lead with the
// ones handed over that no agent has reported on yet.
func (p *Page) columns(tasks []data.Task, folds [2]Fold) {
	var mine, agents, waiting []data.Task
	for _, t := range tasks {
		switch {
		case t.Source != data.SourceAgent:
			mine = append(mine, t)
		case t.HandedTo != "" && !t.Status.Terminal():
			waiting = append(waiting, t)
		default:
			agents = append(agents, t)
		}
	}
	sort.SliceStable(waiting, func(i, j int) bool { return HandedAt(waiting[i]).After(HandedAt(waiting[j])) })
	p.column(0, mine, folds[0])
	if len(waiting) > 0 {
		p.Sections = append(p.Sections, Section{Kind: SecWaiting, Col: 1, Tasks: waiting, Total: len(waiting)})
	}
	p.column(1, agents, folds[1])
	p.Cols[1].Total += len(waiting)
	p.Cols[1].Active += len(waiting)
}

// lanes are the board's four fixed lanes. A blocked task sits in Doing, first; Backlog and Cancelled
// have no lane.
var lanes = []data.Status{data.Todo, data.Doing, data.Review, data.Done}

// Lanes are the board's lanes in order.
func Lanes() []data.Status { return lanes }

// LaneOf is the lane a status is drawn in.
func LaneOf(s data.Status) data.Status {
	if s == data.Blocked {
		return data.Doing
	}
	return s
}

// board is the four lanes; Done shows the newest five until "Show all".
func (p *Page) board(tasks []data.Task, allDone bool) {
	by := map[data.Status][]data.Task{}
	for _, t := range tasks {
		by[t.Status] = append(by[t.Status], t)
	}
	p.Cols[0] = Col{Total: len(tasks)}
	for _, s := range lanes {
		ts := by[s]
		switch s {
		case data.Done:
			sort.SliceStable(ts, func(i, j int) bool { return DoneAt(ts[i]).After(DoneAt(ts[j])) })
		case data.Doing:
			SortForList(by[data.Blocked])
			SortForList(ts)
			ts = append(append([]data.Task(nil), by[data.Blocked]...), ts...)
		default:
			SortForList(ts)
		}
		sec := Section{Kind: SecLane, Status: s, Tasks: ts, Total: len(ts)}
		if s == data.Done {
			sec.Tasks = foldDone(ts, LaneDoneRows, allDone)
		}
		p.Sections = append(p.Sections, sec)
	}
}

// SortForList puts what asks for a date first, in date order, then the most urgent, then the most
// recently touched.
func SortForList(ts []data.Task) {
	sort.SliceStable(ts, func(i, j int) bool {
		ai, iok := data.NextMoment(ts[i])
		aj, jok := data.NextMoment(ts[j])
		switch {
		case iok != jok:
			return iok
		case iok && !ai.Equal(aj):
			return ai.Before(aj)
		case ts[i].Priority != ts[j].Priority:
			return ts[i].Priority < ts[j].Priority
		}
		return ts[i].UpdatedAt.After(ts[j].UpdatedAt)
	})
}

// search groups the results by project, in the order the first result of each came, the Inbox last.
func (p *Page) search(f Found) {
	p.Total, p.More = len(f.Tasks), f.More
	index := map[int]int{} // project id (0 for the Inbox) -> section
	for _, t := range f.Tasks {
		id := 0
		if t.ProjectID != nil {
			id = *t.ProjectID
		}
		i, ok := index[id]
		if !ok {
			i = len(p.Sections)
			index[id] = i
			p.Sections = append(p.Sections, Section{Kind: SecProject, Title: f.Name(t.ProjectID), Inbox: id == 0})
		}
		p.Sections[i].Tasks = append(p.Sections[i].Tasks, t)
		p.Sections[i].Total++
	}
	for i, s := range p.Sections { // the Inbox goes last
		if s.Inbox {
			p.Sections = append(append(p.Sections[:i:i], p.Sections[i+1:]...), s)
			break
		}
	}
}

// days groups the finished tasks by the day they were finished, in the order they come.
func (p *Page) days(done []data.Task, now time.Time) {
	last := ""
	for _, t := range done {
		at := DoneAt(t).In(now.Location())
		if key := at.Format("2006-01-02"); key != last {
			p.Sections, last = append(p.Sections, Section{Kind: SecDay, Day: at}), key
		}
		s := &p.Sections[len(p.Sections)-1]
		s.Tasks = append(s.Tasks, t)
		s.Total++
	}
}
