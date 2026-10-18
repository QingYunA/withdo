package data

import (
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The Summary page (ui-spec, Summary): what was finished in a day or a week, by whom, as a ledger.
// Everything here is data; the views word it and draw it.

// Period is a local day or a week. Start is local midnight of its first day and End the midnight
// after its last: a moment is in the period from Start up to, not including, End.
type Period struct {
	Week       bool
	Start, End time.Time
}

// DayPeriod is the local day back days before now's: 0 is today.
func DayPeriod(now time.Time, back int) Period {
	y, m, d := now.Date()
	return Period{Start: time.Date(y, m, d-back, 0, 0, 0, 0, now.Location()), End: time.Date(y, m, d-back+1, 0, 0, 0, 0, now.Location())}
}

// WeekPeriod is the week back weeks before now's, a week starting on the weekday first (the
// system's first day of the week): 0 is the week of now.
func WeekPeriod(now time.Time, back int, first time.Weekday) Period {
	y, m, d := now.Date()
	d -= (int(now.Weekday())-int(first)+7)%7 + 7*back
	return Period{Week: true, Start: time.Date(y, m, d, 0, 0, 0, 0, now.Location()), End: time.Date(y, m, d+7, 0, 0, 0, 0, now.Location())}
}

// Kind is what a summary is stored under: domain.PeriodDay or domain.PeriodWeek.
func (p Period) Kind() string {
	if p.Week {
		return domain.PeriodWeek
	}
	return domain.PeriodDay
}

// FirstDate is the date the period starts on, as stored: 2026-10-17.
func (p Period) FirstDate() string { return p.Start.Format("2006-01-02") }

// Contains reports whether t is in the period.
func (p Period) Contains(t time.Time) bool { return !t.Before(p.Start) && t.Before(p.End) }

// Over reports whether the period has ended at now: more may still be finished until then.
func (p Period) Over(now time.Time) bool { return !now.Before(p.End) }

// Day is the n-th day of a week period (0 is its first), as local midnight.
func (p Period) Day(n int) time.Time {
	y, m, d := p.Start.Date()
	return time.Date(y, m, d+n, 0, 0, 0, 0, p.Start.Location())
}

// SummaryTask is a finished task with what the ledger and the scene say about it.
type SummaryTask struct {
	Task
	// By is who finished it: an agent's name as it reported, "" for you.
	By      string
	Project string
	// Said is what the agent that finished it last reported, "" when it was you or nothing was said.
	Said string
	// At is when it was finished.
	At time.Time
	// Merged says its pull request merged: the good wears a gold star.
	Merged bool
	// Kind is the good it became: the assistant's pick, else the one its words give (see GuessKind).
	Kind string
}

// SummaryPerson is one of the people who finished something: you, or an agent.
type SummaryPerson struct {
	// Agent is the agent's name, "" for you.
	Agent string
	// Tasks are the tasks it finished, in the order they were finished.
	Tasks []SummaryTask
	// Merged counts the tasks whose pull request merged.
	Merged int
}

// You reports whether the person is the user.
func (p SummaryPerson) You() bool { return p.Agent == "" }

// SummaryOpen counts what is still open: the tasks an agent is working on, waits on or is stuck on.
type SummaryOpen struct {
	InProgress, InReview, Blocked int
}

// None reports whether nothing is open.
func (o SummaryOpen) None() bool { return o.InProgress+o.InReview+o.Blocked == 0 }

// Summary is a period's ledger. People are you first, then the agents by how many they finished.
type Summary struct {
	Period Period
	People []SummaryPerson
	Open   SummaryOpen
}

// Total is how many tasks were finished.
func (s Summary) Total() int {
	n := 0
	for _, p := range s.People {
		n += len(p.Tasks)
	}
	return n
}

// Merged is how many of them went in as a merged pull request.
func (s Summary) Merged() int {
	n := 0
	for _, p := range s.People {
		n += p.Merged
	}
	return n
}

// Tasks lists every finished task in the order they were finished.
func (s Summary) Tasks() []SummaryTask {
	var out []SummaryTask
	for _, p := range s.People {
		out = append(out, p.Tasks...)
	}
	sort.SliceStable(out, func(i, j int) bool { return finishedBefore(out[i], out[j]) })
	return out
}

// FinishedSince counts the tasks finished after t: what happened since a paragraph was written.
func (s Summary) FinishedSince(t time.Time) int {
	n := 0
	for _, task := range s.Tasks() {
		if task.At.After(t) {
			n++
		}
	}
	return n
}

func finishedBefore(a, b SummaryTask) bool {
	if !a.At.Equal(b.At) {
		return a.At.Before(b.At)
	}
	return a.ID < b.ID
}

// SummaryInput is what a summary is built from.
type SummaryInput struct {
	Now    time.Time
	Period Period
	Tasks  []Task
	// Latest is the newest report an agent filed against a task, Event the newest GitHub filed;
	// either may be nil or return nil.
	Latest, Event func(id int) *Report
	ProjectName   func(*int) string
	// Kinds are the goods the assistant picked, by task id.
	Kinds map[int]string
	// Agent says which agent a name it reported under is, so that "claude-code" and "Claude Code" are
	// one person; nil takes names as they are.
	Agent func(name string) string
}

// BuildSummary lists what was finished in a period. Only done tasks count, cancelled ones do not,
// and a subtask is part of its parent's story. Who finished one is CompletedBy (an agent's report
// within five seconds of completion, or the agent whose report closed it, else you); a task a
// merged pull request completed counts for the agent that reported on it, and for you when none did.
func BuildSummary(in SummaryInput) Summary {
	canon := in.Agent
	if canon == nil {
		canon = func(name string) string { return name }
	}
	byWho := map[string]*SummaryPerson{}
	var order []string
	for _, t := range in.Tasks {
		at := finishTime(t)
		if t.IsSubtask() || t.Status != Done || !in.Period.Contains(at) {
			continue
		}
		task := summaryTask(t, at, in, canon)
		person := byWho[task.By]
		if person == nil {
			person = &SummaryPerson{Agent: task.By}
			byWho[task.By] = person
			order = append(order, task.By)
		}
		person.Tasks = append(person.Tasks, task)
		if task.Merged {
			person.Merged++
		}
	}
	out := Summary{Period: in.Period, Open: openCounts(in.Tasks)}
	for _, key := range order {
		p := byWho[key]
		sort.SliceStable(p.Tasks, func(i, j int) bool { return finishedBefore(p.Tasks[i], p.Tasks[j]) })
		out.People = append(out.People, *p)
	}
	sort.SliceStable(out.People, func(i, j int) bool { return personBefore(out.People[i], out.People[j]) })
	return out
}

// personBefore puts you first, then the agents with the most finished, then the one that finished
// something first.
func personBefore(a, b SummaryPerson) bool {
	switch {
	case a.You() != b.You():
		return a.You()
	case len(a.Tasks) != len(b.Tasks):
		return len(a.Tasks) > len(b.Tasks)
	case !a.Tasks[0].At.Equal(b.Tasks[0].At):
		return a.Tasks[0].At.Before(b.Tasks[0].At)
	}
	return a.Agent < b.Agent
}

func finishTime(t Task) time.Time {
	if t.CompletedAt != nil {
		return *t.CompletedAt
	}
	return t.UpdatedAt
}

func summaryTask(t Task, at time.Time, in SummaryInput, canon func(string) string) SummaryTask {
	var agent, github *Report
	if in.Latest != nil {
		agent = in.Latest(t.ID)
	}
	if in.Event != nil {
		github = in.Event(t.ID)
	}
	by, _ := CompletedBy(t, agent, github)
	if by == GitHubAgent {
		by = "" // a merge nobody reported on: the user's own pull request
	}
	task := SummaryTask{Task: t, By: canon(by), At: at, Merged: t.PRState == PRMerged}
	if in.ProjectName != nil {
		task.Project = in.ProjectName(t.ProjectID)
	}
	if by != "" && agent != nil {
		task.Said = agent.Summary
	}
	if kind, ok := in.Kinds[t.ID]; ok && domain.IsGoodKind(kind) {
		task.Kind = kind
	} else {
		task.Kind = GuessKind(t.Title, t.Branch, task.Said)
	}
	return task
}

// openCounts counts the open tasks that are not parked: what an agent works on, waits on or is stuck
// on. Subtasks are their parent's business.
func openCounts(tasks []Task) SummaryOpen {
	var o SummaryOpen
	for _, t := range tasks {
		if t.IsSubtask() {
			continue
		}
		switch t.Status {
		case Doing:
			o.InProgress++
		case Review:
			o.InReview++
		case Blocked:
			o.Blocked++
		}
	}
	return o
}

// You is the name the requests for a paragraph give the user.
const You = "you"

// SummaryRequest is what the assistant is asked to write a paragraph about.
type SummaryRequest struct {
	Week bool
	// Label says which period it is, as words ("Friday 2026-10-17").
	Label string
	// Language is the language to write in, by name ("English", "Simplified Chinese").
	Language string
	Tasks    []SummaryRequestTask
	Open     SummaryOpen
}

// SummaryRequestTask is one finished task of a request.
type SummaryRequestTask struct {
	ID      int
	Title   string
	Project string
	Who     string // You, or the agent's name
	PR      int
	Merged  bool
	Said    string
}

// SummaryText is what the assistant wrote: the paragraph, the goods it picked by task id, and the
// model that wrote them.
type SummaryText struct {
	Paragraph string
	Kinds     map[int]string
	Model     string
}

// Request is the summary as the assistant is asked about it; who words an agent's name.
func (s Summary) Request(label, language string, who func(agent string) string) SummaryRequest {
	req := SummaryRequest{Week: s.Period.Week, Label: label, Language: language, Open: s.Open}
	for _, t := range s.Tasks() {
		name := You
		if t.By != "" {
			name = who(t.By)
		}
		req.Tasks = append(req.Tasks, SummaryRequestTask{
			ID: t.ID, Title: t.Title, Project: t.Project, Who: name, PR: t.PRNumber, Merged: t.Merged, Said: t.Said,
		})
	}
	return req
}

// StoredSummary is the paragraph the assistant wrote for a period, as kept.
type StoredSummary struct {
	Paragraph string
	Model     string
	// Kinds are the goods it picked, by task id.
	Kinds     map[int]string
	CreatedAt time.Time
}

// Summaries is what the assistant wrote about periods, kept so that a visit shows it at once.
type Summaries interface {
	// GetSummary reads the paragraph of a period, nil when none was written.
	GetSummary(Period) (*StoredSummary, error)
	// PutSummary keeps the paragraph of a period, replacing the one it had, and returns it as stored.
	PutSummary(Period, StoredSummary) (StoredSummary, error)
}
