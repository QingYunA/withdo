package data

import (
	"sort"
	"time"
)

// NeedKind says why a task is in the Needs you section. The order is the order of the section.
type NeedKind int

const (
	NeedBlocked   NeedKind = iota // something is stuck
	NeedReminder                  // a reminder came due
	NeedPRClosed                  // the pull request was closed without merging
	NeedPRFailing                 // the checks of the pull request are failing
	NeedPRChanges                 // changes were requested on the pull request
	NeedReview                    // an agent finished and waits for a look
	NeedStale                     // an agent went quiet
)

// Limits of the Now screen.
const (
	LiveWithin      = 10 * time.Minute // an agent report this fresh draws the live dot
	MaxUndated      = 7                // undated tasks of mine shown before "Show all in Inbox"
	MaxDoneToday    = 5                // rows of Done today before "Show all in Completed"
	finishWindow    = 5 * time.Second  // a report this close to a completion is what closed the task
	YoursHorizonDay = 7                // dated tasks of mine appear when due within this many days
)

// NowItem is one row of the Now screen.
type NowItem struct {
	Task
	Project  string // project name, empty for the inbox
	Kind     NeedKind
	Summary  string    // an agent task: what its latest report says
	NextStep string    // an agent task: what its latest report says comes next
	Words    string    // a blocked agent task: what it said it waits for
	Since    time.Time // the latest report, a due reminder's time, or the last update
	Stale    bool      // no report for the stale threshold
	Hours    int       // hours since Since
	Live     bool      // an agent task whose last report is under LiveWithin old
	By       string    // a Done today row: the agent that finished it, empty for me
}

// Now is what the home screen shows. A section with no items is not drawn.
type Now struct {
	Needs  []NowItem // Needs you
	Agents []NowItem // Agents at work
	Yours  []NowItem // Yours
	// MoreUndated counts undated tasks of mine that did not fit; the screen links to the Inbox.
	MoreUndated int
	// Done is what was finished since local midnight, by agents and by me, newest first, at most
	// MaxDoneToday; MoreDone counts the rest. Since is when it was finished, Summary what the
	// report that closed it said. It is no part of Count or Empty: it is a record, not a to-do.
	Done     []NowItem
	MoreDone int
}

// Count is the number of rows of the three sections that ask for attention; the sidebar shows it.
func (n Now) Count() int { return len(n.Needs) + len(n.Agents) + len(n.Yours) }

// Empty reports whether the screen shows "All clear".
func (n Now) Empty() bool { return n.Count() == 0 }

// IDs lists the task ids in the order the rows are drawn.
func (n Now) IDs() []int {
	ids := make([]int, 0, n.Count()+len(n.Done))
	for _, section := range [][]NowItem{n.Needs, n.Agents, n.Yours, n.Done} {
		for _, it := range section {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

// NowInput is what BuildNow reads. Latest returns a task's newest report, or nil.
type NowInput struct {
	Now    time.Time
	Tasks  []Task
	Latest func(taskID int) *Report
	// Event returns the newest report GitHub filed against a task, or nil; it dates the pull
	// request rows. Latest should leave GitHub's reports out: they are not the agent speaking.
	Event       func(taskID int) *Report
	StaleHours  int // DefaultStaleHours when 0
	ProjectName func(*int) string
}

// BuildNow sorts tasks into the three sections of the Now screen. A task appears once, in the
// first section that wants it: blocked, due reminder, review, stale, then agents at work, then
// mine.
func BuildNow(in NowInput) Now {
	b := nowBuilder{NowInput: in}
	if b.StaleHours == 0 {
		b.StaleHours = DefaultStaleHours
	}
	var out Now
	var mine []Task
	for _, t := range in.Tasks {
		switch {
		case t.Status.Terminal() || t.Status == Backlog:
		case b.need(t, &out):
		case t.Source == SourceAgent && t.Status == Doing:
			out.Agents = append(out.Agents, b.agent(t))
		case t.Source == SourceUser && (t.Status == Todo || t.Status == Doing):
			mine = append(mine, t)
		}
	}
	sortNeeds(out.Needs)
	sort.SliceStable(out.Agents, func(i, j int) bool { return out.Agents[i].Since.After(out.Agents[j].Since) })
	out.Yours, out.MoreUndated = b.yours(mine)
	out.Done, out.MoreDone = b.doneToday(in.Tasks)
	return out
}

type nowBuilder struct{ NowInput }

// item fills what every row has: the project, and when the task last showed a sign of life.
func (b nowBuilder) item(t Task) NowItem {
	it := NowItem{Task: t, Project: b.ProjectName(t.ProjectID), Since: t.UpdatedAt}
	if r := b.Latest(t.ID); r != nil {
		it.Since = r.CreatedAt
	}
	it.Hours = int(b.Now.Sub(it.Since) / time.Hour)
	it.Stale = t.Source == SourceAgent && t.Status.Active() && it.Hours >= b.StaleHours
	return it
}

// need files a task under Needs you when it belongs there.
func (b nowBuilder) need(t Task, out *Now) bool {
	var kind NeedKind
	switch {
	case t.Status == Blocked:
		kind = NeedBlocked
	case DueReminder(t, b.Now):
		kind = NeedReminder
	case t.PRClosedUnseen():
		kind = NeedPRClosed
	case t.PRChecksFailing():
		kind = NeedPRFailing
	case t.PRChangesRequested():
		kind = NeedPRChanges
	case t.Source == SourceAgent && t.Status == Review:
		kind = NeedReview
	case t.Source == SourceAgent && t.Status == Doing && b.item(t).Stale:
		kind = NeedStale
	default:
		return false
	}
	it := b.item(t)
	it.Kind = kind
	if kind == NeedReminder {
		it.Since = *t.RemindAt
		it.Hours = int(b.Now.Sub(it.Since) / time.Hour)
	}
	if kind >= NeedPRClosed && kind <= NeedPRChanges {
		b.dateByEvent(&it)
	}
	if kind == NeedBlocked && t.Source == SourceAgent {
		it.Words = blockedWords(b.Latest(t.ID))
	}
	out.Needs = append(out.Needs, it)
	return true
}

// dateByEvent ages a pull request row from the report GitHub filed about it, not from the agent's
// last word: what it waits on is the pull request.
func (b nowBuilder) dateByEvent(it *NowItem) {
	it.Stale = false
	if b.Event == nil {
		return
	}
	if r := b.Event(it.ID); r != nil {
		it.Since = r.CreatedAt
		it.Hours = int(b.Now.Sub(it.Since) / time.Hour)
	}
}

// blockedWords is what a blocked agent task said it waits for: the next step of its latest
// report, else the summary.
func blockedWords(r *Report) string {
	if r == nil {
		return ""
	}
	if r.NextStep != "" {
		return r.NextStep
	}
	return r.Summary
}

func sortNeeds(ns []NowItem) {
	sort.SliceStable(ns, func(i, j int) bool {
		if ns[i].Kind != ns[j].Kind {
			return ns[i].Kind < ns[j].Kind
		}
		return ns[i].Since.Before(ns[j].Since) // the longest wait first
	})
}

func (b nowBuilder) agent(t Task) NowItem {
	it := b.item(t)
	it.Live = b.Now.Sub(it.Since) < LiveWithin
	if r := b.Latest(t.ID); r != nil {
		it.Summary, it.NextStep = r.Summary, r.NextStep
	}
	return it
}

// DueReminder reports whether a reminder is due and the task still open.
func DueReminder(t Task, now time.Time) bool {
	return t.RemindAt != nil && !t.RemindAt.After(now) && !t.Status.Terminal()
}

// NextMoment is the time a task asks for attention: its reminder or due time, whichever comes
// first. ok is false for a task with neither.
func NextMoment(t Task) (at time.Time, ok bool) {
	for _, p := range []*time.Time{t.RemindAt, t.DueAt} {
		if p != nil && (!ok || p.Before(at)) {
			at, ok = *p, true
		}
	}
	return at, ok
}

// yours orders my tasks: due or reminded today and overdue, then the next week, then undated.
func (b nowBuilder) yours(tasks []Task) (rows []NowItem, more int) {
	var dated, undated []NowItem
	for _, t := range tasks {
		it := b.item(t)
		at, ok := NextMoment(t)
		if !ok {
			undated = append(undated, it)
			continue
		}
		if dayIndex(at.In(b.Now.Location()))-dayIndex(b.Now) > YoursHorizonDay {
			continue
		}
		it.Since = at
		dated = append(dated, it)
	}
	sort.SliceStable(dated, func(i, j int) bool { return dated[i].Since.Before(dated[j].Since) })
	sort.SliceStable(undated, func(i, j int) bool {
		if undated[i].Priority != undated[j].Priority {
			return undated[i].Priority < undated[j].Priority
		}
		return undated[i].ID > undated[j].ID
	})
	if len(undated) > MaxUndated {
		more, undated = len(undated)-MaxUndated, undated[:MaxUndated]
	}
	return append(dated, undated...), more
}

// doneToday lists the tasks finished since local midnight, newest first, as many as the screen
// shows, and how many more there are.
func (b nowBuilder) doneToday(tasks []Task) (rows []NowItem, more int) {
	y, m, d := b.Now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, b.Now.Location())
	for _, t := range tasks {
		if t.Status != Done || t.CompletedAt == nil || t.CompletedAt.Before(midnight) {
			continue
		}
		it := b.item(t)
		it.Since = *t.CompletedAt
		it.By, it.Summary = b.completedBy(t)
		rows = append(rows, it)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].Since.Equal(rows[j].Since) {
			return rows[i].Since.After(rows[j].Since)
		}
		return rows[i].ID > rows[j].ID
	})
	if len(rows) > MaxDoneToday {
		more, rows = len(rows)-MaxDoneToday, rows[:MaxDoneToday]
	}
	return rows, more
}

func (b nowBuilder) completedBy(t Task) (by, summary string) {
	var event *Report
	if b.Event != nil {
		event = b.Event(t.ID)
	}
	return CompletedBy(t, b.Latest(t.ID), event)
}

// CompletedBy says who finished a done task and what was said when it was. agent is the newest
// report of an agent and github the newest GitHub filed, either may be nil. When one of them was
// filed as the task closed (within five seconds of it, or later) the report is the finish: by is
// the agent's name (GitHub's when only GitHub spoke, as for a merge) and summary what the newest of
// them says. Otherwise it was me: by is empty and summary too.
func CompletedBy(t Task, agent, github *Report) (by, summary string) {
	if t.CompletedAt == nil {
		return "", ""
	}
	closing := agent
	if github != nil && (closing == nil || github.CreatedAt.After(closing.CreatedAt)) {
		closing = github
	}
	if closing == nil || closing.CreatedAt.Before(t.CompletedAt.Add(-finishWindow)) {
		return "", ""
	}
	if agent != nil {
		return agent.Agent, closing.Summary
	}
	return closing.Agent, closing.Summary
}
