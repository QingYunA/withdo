package data

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// NeedKind says why a task is in the Needs you section. The order is the order of the section.
type NeedKind int

const (
	NeedBlocked  NeedKind = iota // something is stuck
	NeedReminder                 // a reminder came due
	NeedReview                   // an agent finished and waits for a look
	NeedStale                    // an agent went quiet
)

// Limits of the Now screen.
const (
	LiveWithin      = 10 * time.Minute // an agent report this fresh draws the live dot
	MaxUndated      = 7                // undated tasks of mine shown before "Show all in Inbox"
	YoursHorizonDay = 7                // dated tasks of mine appear when due within this many days
)

// NowItem is one row of the Now screen.
type NowItem struct {
	Task
	Project string // project name, empty for the inbox
	Kind    NeedKind
	Summary string    // the second line, empty when there is nothing to say
	Since   time.Time // the latest report, a due reminder's time, or the last update
	Stale   bool      // no report for the stale threshold
	Hours   int       // hours since Since
	Live    bool      // an agent task whose last report is under LiveWithin old
	Label   string    // the due label of my own task: "Today", "Tomorrow", "Fri", "Oct 20"
}

// Now is what the home screen shows. A section with no items is not drawn.
type Now struct {
	Needs  []NowItem // Needs you
	Agents []NowItem // Agents at work
	Yours  []NowItem // Yours
	// MoreUndated counts undated tasks of mine that did not fit; the screen links to the Inbox.
	MoreUndated int
}

// Count is the number of rows the screen draws.
func (n Now) Count() int { return len(n.Needs) + len(n.Agents) + len(n.Yours) }

// Empty reports whether the screen shows "All clear".
func (n Now) Empty() bool { return n.Count() == 0 }

// IDs lists the task ids in the order the rows are drawn.
func (n Now) IDs() []int {
	ids := make([]int, 0, n.Count())
	for _, section := range [][]NowItem{n.Needs, n.Agents, n.Yours} {
		for _, it := range section {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

// NowInput is what BuildNow reads. Latest returns a task's newest report, or nil.
type NowInput struct {
	Now         time.Time
	Tasks       []Task
	Latest      func(taskID int) *Report
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
	it.Summary = b.needSummary(it)
	out.Needs = append(out.Needs, it)
	return true
}

func (b nowBuilder) needSummary(it NowItem) string {
	switch it.Kind {
	case NeedBlocked:
		return blockedSummary(it.Task, b.Latest(it.ID))
	case NeedReminder:
		return "You asked to be reminded " + reminderWhen(*it.RemindAt, b.Now)
	case NeedReview:
		if it.PRNumber > 0 {
			return fmt.Sprintf("PR #%d is ready for your review", it.PRNumber)
		}
		return "Ready for your review"
	}
	return "No update for " + Short(it.Hours)
}

func blockedSummary(t Task, r *Report) string {
	words := ""
	if r != nil {
		words = r.NextStep
		if words == "" {
			words = r.Summary
		}
	}
	switch {
	case t.Source != SourceAgent:
		return "Marked as blocked"
	case words == "":
		return "Agent is waiting"
	}
	return "Agent is waiting: “" + words + "”"
}

// reminderWhen reads "at 4:00 PM", "yesterday at 4:00 PM" or "on Oct 12 at 4:00 PM".
func reminderWhen(at, now time.Time) string {
	clock := at.In(now.Location()).Format("3:04 PM")
	switch day := DayWord(at, now); day {
	case "today":
		return "at " + clock
	case "yesterday":
		return "yesterday at " + clock
	default:
		return "on " + day + " at " + clock
	}
}

// Short writes a span of hours as "5h" or "3d".
func Short(hours int) string {
	if hours >= 48 {
		return fmt.Sprintf("%dd", hours/24)
	}
	return fmt.Sprintf("%dh", max(hours, 1))
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
		it.Summary = r.Summary
		if r.NextStep != "" {
			it.Summary += "; next: " + lowerFirst(r.NextStep)
		}
	}
	return it
}

func lowerFirst(s string) string {
	if s == "" || strings.ToUpper(s[:1]) != s[:1] {
		return s
	}
	// Keep acronyms and names ("MCP tools") as they are.
	if len(s) > 1 && strings.ToUpper(s[1:2]) == s[1:2] && strings.ToLower(s[1:2]) != s[1:2] {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
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
		it.Label = dueLabel(t, at, b.Now)
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

// dueLabel is the right-hand label of one of my tasks: "Today", "Tomorrow", "Fri", "Oct 20". A
// reminder that is still ahead today reads as its time of day.
func dueLabel(t Task, at, now time.Time) string {
	if t.RemindAt != nil && at.Equal(*t.RemindAt) && DayWord(at, now) == "today" {
		return at.In(now.Location()).Format("3:04 PM")
	}
	switch day := DayWord(at, now); day {
	case "today", "tomorrow", "yesterday":
		return Capital(day)
	default:
		if days := dayIndex(at.In(now.Location())) - dayIndex(now); days > 1 && days < 7 {
			return at.In(now.Location()).Format("Mon") // "Friday" -> "Fri"
		}
		return day
	}
}

// Capital upper-cases the first letter of a word.
func Capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
