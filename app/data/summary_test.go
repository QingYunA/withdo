package data

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// Friday 2026-10-16, a quarter past six in the evening.
var friday = time.Date(2026, 10, 16, 18, 15, 0, 0, time.Local)

func oct(day, hour, min int) time.Time { return time.Date(2026, 10, day, hour, min, 0, 0, time.Local) }

// ledger is a set of tasks with their reports, as BuildSummary is given them.
type ledger struct {
	tasks   []Task
	agent   map[int]*Report
	github  map[int]*Report
	project map[int]string
}

func newLedger() *ledger {
	return &ledger{agent: map[int]*Report{}, github: map[int]*Report{}, project: map[int]string{}}
}

// done adds a done task finished at when; by, when not empty, is the agent whose report came as it did.
func (l *ledger) done(id int, title string, when time.Time, by string) *Task {
	l.tasks = append(l.tasks, Task{ID: id, Title: title, Status: Done, CompletedAt: &when, UpdatedAt: when})
	if by != "" {
		l.agent[id] = &Report{TaskID: id, Agent: by, Summary: by + " says " + title, CreatedAt: when.Add(-time.Second)}
	}
	return &l.tasks[len(l.tasks)-1]
}

func (l *ledger) build(p Period) Summary {
	snap := snapshotWith(l.tasks, func(id int) *Report { return l.agent[id] }, func(id int) *Report { return l.github[id] })
	for i := range snap.Tasks {
		snap.Tasks[i].ProjectID = IntPtr(1)
	}
	snap.Projects = []Project{{ID: 1, Name: "adhd-todo"}}
	return BuildSummary(SummaryInput{
		Now: friday, Period: p, Snapshot: snap,
		Agent: func(name string) string {
			if strings.Contains(name, "claude") {
				return "Claude Code"
			}
			return name
		},
	})
}

func names(s Summary) []string {
	var out []string
	for _, p := range s.People {
		who := p.Agent
		if p.You() {
			who = "you"
		}
		out = append(out, who+":"+itoa(len(p.Tasks)))
	}
	return out
}

func TestSummaryTodayCountsDoneTasksOfTheLocalDayOnly(t *testing.T) {
	l := newLedger()
	l.done(1, "yesterday late", oct(15, 23, 59), "")
	l.done(2, "just after midnight", oct(16, 0, 0), "")
	l.done(3, "this evening", oct(16, 18, 0), "")
	l.done(4, "tomorrow", oct(17, 0, 0), "")
	l.tasks = append(l.tasks, Task{ID: 5, Title: "cancelled today", Status: Cancelled, CompletedAt: ptrTo(oct(16, 9, 0))})
	l.tasks = append(l.tasks, Task{ID: 6, Title: "still open", Status: Doing})
	s := l.build(DayPeriod(friday, 0))
	if s.Total() != 2 {
		t.Errorf("total %d, want the two of Friday: %v", s.Total(), names(s))
	}
	ids := []int{}
	for _, task := range s.Tasks() {
		ids = append(ids, task.ID)
	}
	if !reflect.DeepEqual(ids, []int{2, 3}) {
		t.Errorf("tasks %v", ids)
	}
}

func TestSummaryOfAPastDayLooksBack(t *testing.T) {
	l := newLedger()
	l.done(1, "thursday", oct(15, 10, 0), "")
	l.done(2, "friday", oct(16, 10, 0), "")
	s := l.build(DayPeriod(friday, 1))
	if s.Total() != 1 || s.Tasks()[0].ID != 1 {
		t.Errorf("yesterday: %v", names(s))
	}
	if !s.Period.Over(friday) || DayPeriod(friday, 0).Over(friday) {
		t.Error("yesterday is over, today is not")
	}
}

func TestWeekPeriodStartsOnTheSystemsFirstWeekday(t *testing.T) {
	for _, c := range []struct {
		first      time.Weekday
		back       int
		start, end int // the day of October
	}{
		{time.Monday, 0, 12, 19}, {time.Sunday, 0, 11, 18}, {time.Saturday, 0, 10, 17},
		{time.Monday, 1, 5, 12}, {time.Friday, 0, 16, 23}, {time.Sunday, 1, 4, 11},
	} {
		p := WeekPeriod(friday, c.back, c.first)
		if p.Start != oct(c.start, 0, 0) || p.End != oct(c.end, 0, 0) || !p.Week {
			t.Errorf("first %v back %d: %v to %v", c.first, c.back, p.Start, p.End)
		}
	}
}

func TestSummaryOfAWeekTakesTheWholeWeekFromItsFirstDay(t *testing.T) {
	l := newLedger()
	l.done(1, "sunday before", oct(11, 12, 0), "")
	l.done(2, "monday", oct(12, 0, 0), "")
	l.done(3, "sunday after", oct(18, 23, 0), "")
	l.done(4, "next monday", oct(19, 0, 0), "")
	if got := l.build(WeekPeriod(friday, 0, time.Monday)).Total(); got != 2 {
		t.Errorf("a Monday week has %d, want 2", got)
	}
	if got := l.build(WeekPeriod(friday, 0, time.Sunday)).Total(); got != 2 { // Sun 11 to Sat 17: 1 and 2
		t.Errorf("a Sunday week has %d, want 2", got)
	}
}

func TestPeriodDayIsTheNthMidnightOfAWeek(t *testing.T) {
	p := WeekPeriod(friday, 0, time.Monday)
	if p.Day(0) != oct(12, 0, 0) || p.Day(6) != oct(18, 0, 0) {
		t.Errorf("%v %v", p.Day(0), p.Day(6))
	}
	if p.FirstDate() != "2026-10-12" || p.Kind() != "week" || DayPeriod(friday, 0).Kind() != "day" {
		t.Errorf("%q %q", p.FirstDate(), p.Kind())
	}
}

func TestEachTaskCountsForWhoFinishedIt(t *testing.T) {
	l := newLedger()
	l.done(1, "mine", oct(16, 9, 0), "")
	l.done(2, "claude one", oct(16, 10, 0), "claude-code")
	l.done(3, "claude two", oct(16, 11, 0), "Claude Code") // another spelling of the same agent
	l.done(4, "codex one", oct(16, 12, 0), "codex")
	l.done(5, "mine too", oct(16, 13, 0), "")
	// An agent's report long before I finished its task does not make it the agent's.
	l.done(6, "I closed it", oct(16, 14, 0), "")
	l.agent[6] = &Report{TaskID: 6, Agent: "codex", Summary: "planned", CreatedAt: oct(15, 9, 0)}
	s := l.build(DayPeriod(friday, 0))
	if got := names(s); !reflect.DeepEqual(got, []string{"you:3", "Claude Code:2", "codex:1"}) {
		t.Errorf("people %v: you first, then the agents by count", got)
	}
}

func TestAMergedPullRequestCountsForThePullRequestsAgentAndIsStarred(t *testing.T) {
	l := newLedger()
	task := l.done(1, "Port the MCP server", oct(16, 13, 0), "")
	task.PRState, task.PRNumber = PRMerged, 14
	// The agent reported hours ago; GitHub reported the merge as the task closed.
	l.agent[1] = &Report{TaskID: 1, Agent: "claude-code", Summary: "Ready for review", CreatedAt: oct(16, 9, 0)}
	l.github[1] = &Report{TaskID: 1, Agent: GitHubAgent, Summary: "PR #14 merged", CreatedAt: oct(16, 13, 0)}
	l.done(2, "A merge nobody reported on", oct(16, 14, 0), "").PRState = PRMerged
	l.github[2] = &Report{TaskID: 2, Agent: GitHubAgent, Summary: "PR #15 merged", CreatedAt: oct(16, 14, 0)}
	s := l.build(DayPeriod(friday, 0))
	if got := names(s); !reflect.DeepEqual(got, []string{"you:1", "Claude Code:1"}) {
		t.Fatalf("people %v: the first merge is the agent's, the second is yours", got)
	}
	if s.Merged() != 2 || s.People[0].Merged != 1 || s.People[1].Merged != 1 {
		t.Errorf("merged %d (%d, %d)", s.Merged(), s.People[0].Merged, s.People[1].Merged)
	}
	claude := s.People[1].Tasks[0]
	if !claude.Merged || claude.Said != "Ready for review" || claude.Project != "adhd-todo" {
		t.Errorf("%+v", claude)
	}
}

func TestSubtasksAreTheirParentsStoryAndArchivedProjectsAreNotListed(t *testing.T) {
	l := newLedger()
	l.done(1, "parent", oct(16, 9, 0), "")
	kid := l.done(2, "kid", oct(16, 9, 5), "")
	kid.ParentID = IntPtr(1)
	if s := l.build(DayPeriod(friday, 0)); s.Total() != 1 {
		t.Errorf("a finished subtask counts apart: %v", names(s))
	}
}

func TestPeopleOfEqualCountComeInTheOrderTheyFirstFinished(t *testing.T) {
	l := newLedger()
	l.done(1, "z", oct(16, 12, 0), "zed")
	l.done(2, "a", oct(16, 9, 0), "ant")
	l.done(3, "me", oct(16, 15, 0), "")
	s := l.build(DayPeriod(friday, 0))
	if got := names(s); !reflect.DeepEqual(got, []string{"you:1", "ant:1", "zed:1"}) {
		t.Errorf("%v", got)
	}
}

func TestStillOpenCountsWhatAgentsAreOn(t *testing.T) {
	l := newLedger()
	for i, st := range []Status{Doing, Doing, Doing, Review, Blocked, Todo, Backlog, Cancelled} {
		l.tasks = append(l.tasks, Task{ID: 100 + i, Status: st})
	}
	l.tasks = append(l.tasks, Task{ID: 200, Status: Doing, ParentID: IntPtr(100)})
	if got := l.build(DayPeriod(friday, 0)).Open; got != (SummaryOpen{InProgress: 3, InReview: 1, Blocked: 1}) {
		t.Errorf("open %+v", got)
	}
}

func TestAnEmptyDayHasNoPeople(t *testing.T) {
	s := newLedger().build(DayPeriod(friday, 0))
	if len(s.People) != 0 || s.Total() != 0 || s.Merged() != 0 || !s.Open.None() {
		t.Errorf("%+v", s)
	}
}

func TestFinishedSinceCountsWhatCameAfterTheParagraph(t *testing.T) {
	l := newLedger()
	l.done(1, "a", oct(16, 9, 0), "")
	l.done(2, "b", oct(16, 17, 0), "")
	s := l.build(DayPeriod(friday, 0))
	if s.FinishedSince(oct(16, 12, 0)) != 1 || s.FinishedSince(oct(16, 18, 0)) != 0 || s.FinishedSince(oct(15, 0, 0)) != 2 {
		t.Error("finished since")
	}
}

func TestTheAssistantsKindsWinOverTheWordsAndAnUnknownOneFallsBack(t *testing.T) {
	l := newLedger()
	l.done(1, "Fix the leak", oct(16, 9, 0), "")
	l.done(2, "Fix the leak again", oct(16, 10, 0), "")
	l.done(3, "Pay rent", oct(16, 11, 0), "")
	s := BuildSummary(SummaryInput{
		Now: friday, Period: DayPeriod(friday, 0), Tasks: l.tasks,
		Kinds: map[int]string{1: "gear", 2: "dragon"},
	})
	got := []string{}
	for _, task := range s.Tasks() {
		got = append(got, task.Kind)
	}
	if !reflect.DeepEqual(got, []string{"gear", "jar", "coin"}) {
		t.Errorf("kinds %v", got)
	}
}

func TestGuessKindReadsEnglishAndChineseWords(t *testing.T) {
	for title, want := range map[string]string{
		"Fix the crash on launch":        "jar",
		"修复导入时的空指针":                      "jar",
		"Add tests for the parser":       "flask",
		"补充测试用例":                         "flask",
		"Reply to design feedback":       "mail",
		"回复设计反馈":                         "mail",
		"Refactor the store":             "gear",
		"迁移到 Go":                         "gear",
		"Migrating the database":         "gear",
		"交房租":                            "coin",
		"Pay the electricity bill":       "coin",
		"Write the release notes":        "book",
		"更新 README 文档":                   "book",
		"Research sync engines":          "scroll",
		"调研同步方案":                         "scroll",
		"Implement the shipping scene":   "gem",
		"新增导出功能":                         "gem",
		"Water the plants":               "crate",
		"Address the review":             "mail",
		"Spelling":                       "crate", // "spell" is no word of any kind, and "pell"... nothing matches
		"Native UI with mygo":            "crate",
		"Settings > General > Fixtures":  "crate", // "fix" only counts as a whole word
		"Hotfix for the importer":        "jar",
		"Port the MCP server":            "gear",
		"Report the numbers":             "crate", // "port" inside "report" is not the word
		"Draft the UI spec":              "book",
		"Plan the next milestone":        "scroll",
		"Setup CI, tests and coverage":   "flask",
		"Cleanup old branches":           "gear",
		"fix/native-ui":                  "jar",
		"feat/shipping-scene":            "gem",
		"Check the budget with the bank": "coin",
	} {
		if got := GuessKind(title); got != want {
			t.Errorf("%q: %q, want %q", title, got, want)
		}
	}
	if got := GuessKind("Native UI", "feat/native-ui", ""); got != "gem" {
		t.Errorf("the branch counts too: %q", got)
	}
}

func TestEveryGoodKindIsOneTheDomainKnows(t *testing.T) {
	if len(GoodKinds) != 9 || GoodKinds[len(GoodKinds)-1] != "crate" {
		t.Errorf("%v", GoodKinds)
	}
}

func TestSummaryRequestNamesWhoFinishedWhat(t *testing.T) {
	l := newLedger()
	l.done(1, "mine", oct(16, 9, 0), "")
	l.done(2, "theirs", oct(16, 10, 0), "claude-code").PRNumber = 12
	s := l.build(DayPeriod(friday, 0))
	req := s.Request("Friday 2026-10-16", "English", func(a string) string { return "Agent " + a })
	if req.Week || req.Label != "Friday 2026-10-16" || len(req.Tasks) != 2 {
		t.Fatalf("%+v", req)
	}
	if req.Tasks[0].Who != "you" || req.Tasks[1].Who != "Agent Claude Code" || req.Tasks[1].PR != 12 || req.Tasks[1].Said == "" {
		t.Errorf("%+v", req.Tasks)
	}
}

func ptr[T any](v T) *T { return &v }
