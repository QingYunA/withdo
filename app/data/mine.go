package data

import (
	"sort"
	"time"
)

// NeedKind says why a task is in From your agents. The order is the order of the section.
type NeedKind int

const (
	NeedBlocked   NeedKind = iota // an agent is stuck and waits for an answer
	NeedReminder                  // a reminder of an agent's task came due
	NeedPRClosed                  // the pull request was closed without merging
	NeedPRFailing                 // the checks of the pull request are failing
	NeedPRChanges                 // changes were requested on the pull request
	NeedReview                    // an agent finished and waits for a look
	NeedStale                     // an agent went quiet
)

// Limits of the Mine and Agents screens.
const (
	LiveWithin      = 10 * time.Minute // an agent report this fresh draws the live dot
	MaxUndated      = 7                // undated tasks of mine shown in Later before "Show all"
	MaxDoneToday    = 5                // rows of Done today before "Show all in Completed"
	finishWindow    = 5 * time.Second  // a report this close to a completion is what closed the task
	YoursHorizonDay = 7                // dated tasks of mine appear in Later when due within this many days
)

// Item is one row of the Mine or the Agents screen.
type Item struct {
	Task
	Project  string // project name, empty for the inbox
	Kind     NeedKind
	Summary  string    // an agent task: what its latest report says
	NextStep string    // an agent task: what its latest report says comes next
	Words    string    // a blocked agent task: what it said it waits for
	Since    time.Time // the latest report, a due reminder's time, or the last update; of a handed-over row, when it was handed over
	Stale    bool      // no report for the stale threshold
	Hours    int       // hours since Since
	Live     bool      // an agent task whose last report is under LiveWithin old
	By       string    // a Done today row: the agent that finished it, empty for me
}

// Mine is what the home screen shows. A section with no items is not drawn.
type Mine struct {
	FromAgents []Item // what agents handed over: stuck, done and waiting for a look, a pull request, a reminder
	Today      []Item // mine, due or reminded today (or overdue), and what I am doing
	Later      []Item // mine, due in the next week, then undated by priority
	// MoreLater counts undated tasks of mine that did not fit; the screen offers "Show all" for them.
	MoreLater int
	// Parked are the tasks parked in the backlog that have no project: no project page folds them,
	// so Today does, and nothing is unreachable now that there is no Inbox. ParkedTotal is how many
	// there are; Parked holds the rows when the fold is open (see MineInput.ParkedOpen).
	Parked      []Item
	ParkedTotal int
	// Done is what was finished since local midnight, by agents and by me, newest first, at most
	// MaxDoneToday; MoreDone counts the rest. Since is when it was finished, Summary what the
	// report that closed it said. It is no part of Count or Empty: it is a record, not a to-do.
	Done     []Item
	MoreDone int
}

// Count is what the sidebar shows beside Mine when nothing is unseen: the rows that ask for
// attention now, handed over or due today. Later waits.
func (m Mine) Count() int { return len(m.FromAgents) + len(m.Today) }

// Empty reports whether the screen shows "All clear".
func (m Mine) Empty() bool { return len(m.FromAgents)+len(m.Today)+len(m.Later) == 0 }

// WithoutDismissed is m without the handed-over rows the user put aside: a row stays away until its
// task is handed over again (a newer Since). The other sections are not the agents' to dismiss.
func (m Mine) WithoutDismissed(d Seen) Mine {
	m.FromAgents = Unseen(m.FromAgents, d)
	return m
}

// Only is m with the rows of the tasks keep rejects left out, for the counts of a slice. MoreLater
// and MoreDone stay as they are: build the slice itself (MineInput.Keep) to have them right.
func (m Mine) Only(keep func(Task) bool) Mine {
	m.FromAgents, m.Today, m.Later, m.Done = only(m.FromAgents, keep), only(m.Today, keep), only(m.Later, keep), only(m.Done, keep)
	m.Parked = only(m.Parked, keep)
	return m
}

func only(items []Item, keep func(Task) bool) []Item {
	var out []Item
	for _, it := range items {
		if keep(it.Task) {
			out = append(out, it)
		}
	}
	return out
}

// IDs lists the task ids in the order the rows are drawn.
func (m Mine) IDs() []int {
	ids := make([]int, 0, len(m.FromAgents)+len(m.Today)+len(m.Later)+len(m.Done)+len(m.Parked))
	for _, section := range [][]Item{m.FromAgents, m.Today, m.Later, m.Done, m.Parked} {
		for _, it := range section {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

// AgentRow is an agent's top-level task with the subtasks under it.
type AgentRow struct {
	Item
	Subs []Item // oldest first; cancelled ones are left out
}

// AgentsScreen is what the Agents screen shows.
type AgentsScreen struct {
	Working []AgentRow // agent tasks in doing or blocked, newest report first
	Done    []AgentRow // agent tasks finished since midnight, newest first, at most MaxDoneToday
	// MoreDone counts the finished ones that did not fit.
	MoreDone int
}

// Count is the number of agent tasks at work; the sidebar shows it beside Agents.
func (a AgentsScreen) Count() int { return len(a.Working) }

// Empty reports whether nothing is at work and nothing was finished today.
func (a AgentsScreen) Empty() bool { return len(a.Working)+len(a.Done) == 0 }

// AgentTally is one agent's share of the Agents screen.
type AgentTally struct {
	Name    string // the name the agent goes by
	Working int    // how many of its tasks are at work
}

// Tally lists the agents on the screen in the order the screen shows them: those at work, newest
// report first, then those that only finished something today. name says which agent a task is
// of; tasks that share a name are one agent.
func (a AgentsScreen) Tally(name func(Task) string) []AgentTally {
	var out []AgentTally
	at := map[string]int{}
	for i, rows := range [][]AgentRow{a.Working, a.Done} {
		for _, r := range rows {
			n := name(r.Task)
			j, ok := at[n]
			if !ok {
				j = len(out)
				at[n] = j
				out = append(out, AgentTally{Name: n})
			}
			if i == 0 {
				out[j].Working++
			}
		}
	}
	return out
}

// IDs lists the task ids in the order the rows are drawn: a task, then its subtasks.
func (a AgentsScreen) IDs() []int {
	var ids []int
	for _, rows := range [][]AgentRow{a.Working, a.Done} {
		for _, r := range rows {
			ids = append(ids, r.ID)
			for _, s := range r.Subs {
				ids = append(ids, s.ID)
			}
		}
	}
	return ids
}

// MineInput is what BuildMine and BuildAgents read: the snapshot, whose Tasks are the ones to sort
// (the views add the ones that are just done), with its reports and project names.
type MineInput struct {
	Snapshot
	Now time.Time
	// Keep, when set, limits the screen to the top-level tasks it accepts; their subtasks go where
	// they go. The screen is built from those tasks alone, so its counts and overflow are its own.
	Keep func(Task) bool
	// AllLater lifts the cap on the undated tasks of Later; ParkedOpen fills Mine.Parked.
	AllLater, ParkedOpen bool
}

// BuildMine sorts tasks into the sections of the Mine screen. Subtasks never get a row of their
// own: they belong to their parent. A task appears once, in the first section that wants it.
func BuildMine(in MineInput) Mine {
	b := newBuilder(in)
	var out Mine
	var mine, parked []Task
	for _, t := range b.Tasks {
		switch {
		case t.IsSubtask() || t.Status.Terminal():
		case t.Status == Backlog:
			if t.ProjectID == nil {
				parked = append(parked, t)
			}
		case b.handed(t, &out):
		case t.Source == SourceUser && (t.Status == Todo || t.Status == Doing || t.Status == Blocked):
			mine = append(mine, t)
		}
	}
	sortHanded(out.FromAgents)
	out.Today, out.Later, out.MoreLater = b.mine(mine)
	out.Parked, out.ParkedTotal = b.parked(parked)
	out.Done, out.MoreDone = b.doneToday(func(Task) bool { return true })
	return out
}

// BuildAgents lists what agents are doing: their tasks in doing or blocked with the subtasks under
// them, then what they finished today.
func BuildAgents(in MineInput) AgentsScreen {
	b := newBuilder(in)
	var out AgentsScreen
	for _, t := range b.Tasks {
		if t.IsSubtask() || t.Source != SourceAgent || (t.Status != Doing && t.Status != Blocked) {
			continue
		}
		out.Working = append(out.Working, AgentRow{Item: b.agent(t), Subs: b.subs(t.ID)})
	}
	sort.SliceStable(out.Working, func(i, j int) bool { return out.Working[i].Since.After(out.Working[j].Since) })
	done, more := b.doneToday(func(t Task) bool { return t.Source == SourceAgent })
	out.MoreDone = more
	for _, it := range done {
		out.Done = append(out.Done, AgentRow{Item: it, Subs: b.subs(it.ID)})
	}
	return out
}

type nowBuilder struct{ MineInput }

func newBuilder(in MineInput) nowBuilder {
	in.Tasks = keepSlice(in.Tasks, in.Keep)
	b := nowBuilder{MineInput: in}
	return b
}

// keepSlice is the tasks of a slice: the top-level tasks keep accepts and their subtasks. A nil keep
// takes them all.
func keepSlice(tasks []Task, keep func(Task) bool) []Task {
	if keep == nil {
		return tasks
	}
	kept := map[int]bool{}
	for _, t := range tasks {
		if !t.IsSubtask() && keep(t) {
			kept[t.ID] = true
		}
	}
	var out []Task
	for _, t := range tasks {
		if kept[t.ID] || (t.IsSubtask() && kept[*t.ParentID]) {
			out = append(out, t)
		}
	}
	return out
}

// item fills what every row has: the project, and when the task last showed a sign of life.
func (b nowBuilder) item(t Task) Item {
	it := Item{Task: t, Project: b.ProjectName(t.ProjectID), Since: t.UpdatedAt}
	if r := b.LatestReport(t.ID); r != nil {
		it.Since = r.CreatedAt
	}
	it.Hours = int(b.Now.Sub(it.Since) / time.Hour)
	it.Stale = t.Source == SourceAgent && t.Status.Active() && it.Hours >= b.StaleHours()
	return it
}

// subs lists a task's subtasks, oldest first, without the cancelled ones.
func (b nowBuilder) subs(id int) []Item {
	var out []Item
	for _, s := range Subtasks(b.Tasks, id) {
		if s.Status == Cancelled {
			continue
		}
		it := b.item(s)
		it.Live = s.Status == Doing && b.Now.Sub(it.Since) < LiveWithin
		out = append(out, it)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// handed files a task under From your agents when it belongs there: an agent's task that is stuck,
// finished, quiet or has a reminder due, and any task whose pull request needs a look.
func (b nowBuilder) handed(t Task, out *Mine) bool {
	agent := t.Source == SourceAgent
	var kind NeedKind
	switch {
	case t.HandedTo != "":
		// It waits for the agent it was given to, which is the user's own doing: nothing to hand back.
		return false
	case agent && t.Status == Blocked:
		kind = NeedBlocked
	case agent && DueReminder(t, b.Now):
		kind = NeedReminder
	case t.PRClosedUnseen():
		kind = NeedPRClosed
	case t.PRChecksFailing():
		kind = NeedPRFailing
	case t.PRChangesRequested():
		kind = NeedPRChanges
	case agent && t.Status == Review:
		kind = NeedReview
	case agent && t.Status == Doing && b.item(t).Stale:
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
	if kind == NeedBlocked {
		it.Words = blockedWords(b.LatestReport(t.ID))
	}
	out.FromAgents = append(out.FromAgents, it)
	return true
}

// dateByEvent ages a pull request row from the report GitHub filed about it, not from the agent's
// last word: what it waits on is the pull request.
func (b nowBuilder) dateByEvent(it *Item) {
	it.Stale = false
	if r := b.GitHubReport(it.ID); r != nil {
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

func sortHanded(ns []Item) {
	sort.SliceStable(ns, func(i, j int) bool {
		if ns[i].Kind != ns[j].Kind {
			return ns[i].Kind < ns[j].Kind
		}
		return ns[i].Since.Before(ns[j].Since) // the longest wait first
	})
}

func (b nowBuilder) agent(t Task) Item {
	it := b.item(t)
	it.Live = b.Now.Sub(it.Since) < LiveWithin
	if r := b.LatestReport(t.ID); r != nil {
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

// mine splits my tasks into Today and Later. Today is what I am doing and what is due or reminded
// today or earlier, dated ones first. Later is the dated ones of the next week, then the undated
// by priority, at most MaxUndated of them; more says how many did not fit. Tasks due further away
// stay out of both.
func (b nowBuilder) mine(tasks []Task) (today, later []Item, more int) {
	var todayDated, todayDoing, laterDated, undated []Item
	for _, t := range tasks {
		it := b.item(t)
		doing := t.Status != Todo
		at, ok := NextMoment(t)
		if ok {
			it.Since = at
		}
		switch days := dayIndex(at.In(b.Now.Location())) - dayIndex(b.Now); {
		case ok && days <= 0:
			todayDated = append(todayDated, it)
		case doing:
			todayDoing = append(todayDoing, it)
		case !ok:
			undated = append(undated, it)
		case days <= YoursHorizonDay:
			laterDated = append(laterDated, it)
		}
	}
	byMoment := func(s []Item) {
		sort.SliceStable(s, func(i, j int) bool { return s[i].Since.Before(s[j].Since) })
	}
	byPriority := func(s []Item) {
		sort.SliceStable(s, func(i, j int) bool {
			if s[i].Priority != s[j].Priority {
				return s[i].Priority < s[j].Priority
			}
			return s[i].ID > s[j].ID
		})
	}
	byMoment(todayDated)
	byMoment(laterDated)
	byPriority(todayDoing)
	byPriority(undated)
	if len(undated) > MaxUndated && !b.AllLater {
		more, undated = len(undated)-MaxUndated, undated[:MaxUndated]
	}
	return append(todayDated, todayDoing...), append(laterDated, undated...), more
}

// parked sorts the backlog tasks that have no project, newest first; the rows are kept only when
// the fold is open.
func (b nowBuilder) parked(tasks []Task) (rows []Item, total int) {
	for _, t := range tasks {
		rows = append(rows, b.item(t))
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	total = len(rows)
	if !b.ParkedOpen {
		rows = nil
	}
	return rows, total
}

// doneToday lists the tasks finished since local midnight that want accepts, newest first, as many
// as the screen shows, and how many more there are. Subtasks are the parent's business.
func (b nowBuilder) doneToday(want func(Task) bool) (rows []Item, more int) {
	y, m, d := b.Now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, b.Now.Location())
	for _, t := range b.Tasks {
		if t.IsSubtask() || t.Status != Done || t.CompletedAt == nil || t.CompletedAt.Before(midnight) || !want(t) {
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
	return CompletedBy(t, b.LatestReport(t.ID), b.GitHubReport(t.ID))
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
