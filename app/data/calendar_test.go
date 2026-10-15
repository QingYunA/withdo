package data

import (
	"testing"
	"time"
)

func day(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.Local) }
func at(m time.Month, d, h, min int) *time.Time {
	t := time.Date(2026, m, d, h, min, 0, 0, time.Local)
	return &t
}

func TestMonthGridStartsOnTheWeekStartAndHasFourToSixWeeks(t *testing.T) {
	tests := []struct {
		name  string
		month time.Time
		start time.Weekday
		first time.Time
		weeks int
	}{
		{"october 2026, Monday first", day(time.October, 1), time.Monday, day(time.September, 28), 5},
		{"october 2026, Sunday first", day(time.October, 15), time.Sunday, day(time.September, 27), 5},
		{"a month that starts on the week start", day(time.June, 1), time.Monday, day(time.June, 1), 5},
		{"february 2027 fills four Monday weeks exactly", time.Date(2027, 2, 9, 0, 0, 0, 0, time.Local), time.Monday, time.Date(2027, 2, 1, 0, 0, 0, 0, time.Local), 4},
		{"a long month that starts late needs six", day(time.August, 1), time.Monday, day(time.July, 27), 6},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			first, weeks := MonthGrid(tc.month, tc.start)
			if !first.Equal(tc.first) || weeks != tc.weeks {
				t.Errorf("MonthGrid = %v, %d weeks; want %v, %d", first.Format("2006-01-02"), weeks, tc.first.Format("2006-01-02"), tc.weeks)
			}
			if first.Weekday() != tc.start {
				t.Errorf("first day is a %v, want %v", first.Weekday(), tc.start)
			}
		})
	}
}

func buildCal(tasks []Task, ms []Milestone) []CalDay {
	return BuildCalendar(CalendarInput{
		Month: day(time.October, 1), Now: fixedNow(), WeekStart: time.Monday, Tasks: tasks, Milestones: ms,
		Agent: func(id int) string {
			if id == 4 {
				return "codex"
			}
			return ""
		},
	})
}

func dayOf(days []CalDay, m time.Month, d int) CalDay {
	for _, cd := range days {
		if cd.Date.Month() == m && cd.Date.Day() == d {
			return cd
		}
	}
	return CalDay{}
}

func TestCalendarDaysKnowTheirMonthWeekendAndToday(t *testing.T) {
	days := buildCal(nil, nil)
	if len(days) != 35 {
		t.Fatalf("%d days, want 35", len(days))
	}
	if d := dayOf(days, time.September, 30); d.InMonth || d.Today {
		t.Errorf("Sep 30 %+v", d)
	}
	if d := dayOf(days, time.October, 14); !d.Today || !d.InMonth || d.Weekend {
		t.Errorf("Oct 14 %+v", d)
	}
	if d := dayOf(days, time.October, 17); !d.Weekend {
		t.Error("Saturday Oct 17 is not a weekend")
	}
	if d := dayOf(days, time.October, 18); !d.Weekend {
		t.Error("Sunday Oct 18 is not a weekend")
	}
}

func TestItemsLandOnTheirDays(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "due", Status: Todo, DueAt: at(time.October, 16, 18, 0), Source: SourceUser},
		{ID: 2, Title: "remind", Status: Todo, RemindAt: at(time.October, 16, 9, 0), Source: SourceUser},
		{ID: 3, Title: "done", Status: Done, CompletedAt: at(time.October, 12, 10, 0), Source: SourceUser},
		{ID: 4, Title: "agent", Status: Doing, DueAt: at(time.October, 20, 18, 0), Source: SourceAgent},
		{ID: 5, Title: "cancelled", Status: Cancelled, DueAt: at(time.October, 16, 18, 0), Source: SourceUser},
		{ID: 6, Title: "undated", Status: Todo, Source: SourceUser},
	}
	ms := []Milestone{
		{ID: 1, Title: "v0.3", Open: true, DueAt: at(time.October, 23, 0, 0)},
		{ID: 2, Title: "closed", Open: false, DueAt: at(time.October, 23, 0, 0)},
	}
	days := buildCal(tasks, ms)
	titles := func(d CalDay) []string {
		var out []string
		for _, it := range d.Items {
			if it.Kind == CalMilestone {
				out = append(out, "ms:"+it.Milestone.Title)
				continue
			}
			out = append(out, it.Task.Title)
		}
		return out
	}
	for _, tc := range []struct {
		m    time.Month
		d    int
		want []string
	}{
		{time.October, 16, []string{"remind", "due"}}, // by time of day; cancelled is not here
		{time.October, 12, []string{"done"}},
		{time.October, 20, []string{"agent"}},
		{time.October, 23, []string{"ms:v0.3"}},
		{time.October, 15, nil},
	} {
		if got := titles(dayOf(days, tc.m, tc.d)); !equalStrings(got, tc.want) {
			t.Errorf("%v %d: %v, want %v", tc.m, tc.d, got, tc.want)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestATaskDueAndRemindedOnTheSameDayShowsOnceAsTheReminder(t *testing.T) {
	task := Task{ID: 1, Title: "both", Status: Todo, Source: SourceUser,
		DueAt: at(time.October, 16, 18, 0), RemindAt: at(time.October, 16, 9, 0)}
	d := dayOf(buildCal([]Task{task}, nil), time.October, 16)
	if len(d.Items) != 1 || d.Items[0].Kind != CalReminder {
		t.Fatalf("items %+v", d.Items)
	}
	// On different days it shows on both.
	task.RemindAt = at(time.October, 15, 9, 0)
	days := buildCal([]Task{task}, nil)
	if a, b := dayOf(days, time.October, 15), dayOf(days, time.October, 16); len(a.Items) != 1 || a.Items[0].Kind != CalReminder || len(b.Items) != 1 || b.Items[0].Kind != CalDue {
		t.Errorf("15: %+v, 16: %+v", a.Items, b.Items)
	}
}

func TestOnlyOpenTasksPastDueAreOverdue(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "late", Status: Todo, DueAt: at(time.October, 9, 18, 0)},
		{ID: 2, Title: "today later", Status: Todo, DueAt: at(time.October, 14, 18, 0)},
		{ID: 3, Title: "late but done", Status: Done, DueAt: at(time.October, 9, 18, 0), CompletedAt: at(time.October, 9, 10, 0)},
		{ID: 4, Title: "late reminded the same day", Status: Todo, DueAt: at(time.October, 10, 18, 0), RemindAt: at(time.October, 10, 9, 0)},
	}
	days := buildCal(tasks, nil)
	nine := dayOf(days, time.October, 9).Items
	if len(nine) != 2 {
		t.Fatalf("Oct 9 items %+v", nine)
	}
	for _, it := range nine {
		if it.Overdue != (it.Task.ID == 1) {
			t.Errorf("%q overdue = %v", it.Task.Title, it.Overdue)
		}
	}
	if dayOf(days, time.October, 14).Items[0].Overdue {
		t.Error("a task due this evening is overdue at 14:30")
	}
	if it := dayOf(days, time.October, 10).Items[0]; it.Kind != CalReminder || !it.Overdue {
		t.Errorf("a reminder item of a task past due: %+v", it)
	}
}

func TestMilestonesComeFirstAndDoneItemsLast(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "done early", Status: Done, CompletedAt: at(time.October, 12, 8, 0)},
		{ID: 2, Title: "due", Status: Todo, DueAt: at(time.October, 12, 18, 0)},
		{ID: 3, Title: "remind", Status: Todo, RemindAt: at(time.October, 12, 15, 0)},
	}
	ms := []Milestone{{ID: 1, Title: "m", Open: true, DueAt: at(time.October, 12, 0, 0)}}
	d := dayOf(buildCal(tasks, ms), time.October, 12)
	var kinds []CalKind
	for _, it := range d.Items {
		kinds = append(kinds, it.Kind)
	}
	want := []CalKind{CalMilestone, CalReminder, CalDue, CalDone}
	if len(kinds) != len(want) {
		t.Fatalf("kinds %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds %v, want %v", kinds, want)
		}
	}
}

func TestAgentIsTheNameOfWhoReportedOnATaskAnAgentWrote(t *testing.T) {
	tasks := []Task{
		{ID: 4, Title: "agent", Status: Todo, Source: SourceAgent, DueAt: at(time.October, 20, 18, 0)},
		{ID: 104, Title: "mine", Status: Todo, Source: SourceUser, DueAt: at(time.October, 21, 18, 0)},
	}
	days := buildCal(tasks, nil)
	if a := dayOf(days, time.October, 20).Items[0].Agent; a != "codex" {
		t.Errorf("agent task by %q, want codex", a)
	}
	if a := dayOf(days, time.October, 21).Items[0].Agent; a != "" {
		t.Errorf("my task by %q", a)
	}
	// An agent task with no report is by "agent".
	days = BuildCalendar(CalendarInput{Month: day(time.October, 1), Now: fixedNow(), Tasks: tasks[:1], Agent: func(int) string { return "" }})
	if a := dayOf(days, time.October, 20).Items[0].Agent; a != "agent" {
		t.Errorf("agent task with no report by %q, want agent", a)
	}
}

func TestMovingAMomentToAnotherDayKeepsItsTimeOfDay(t *testing.T) {
	from := time.Date(2026, 10, 16, 9, 30, 0, 0, time.Local)
	got := OnDay(from, day(time.October, 23))
	if want := time.Date(2026, 10, 23, 9, 30, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("OnDay = %v, want %v", got, want)
	}
	// A moment stored in another zone keeps the clock the user sees.
	utc := time.Date(2026, 10, 16, 1, 30, 0, 0, time.UTC)
	got = OnDay(utc, day(time.October, 23))
	l := utc.In(time.Local)
	if want := time.Date(2026, 10, 23, l.Hour(), l.Minute(), 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("OnDay(utc) = %v, want %v", got, want)
	}
}

func TestADayWithNoTimeWordsIsDueAtSixInTheEvening(t *testing.T) {
	got := EndOfDay(day(time.October, 16))
	if want := time.Date(2026, 10, 16, 18, 0, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("EndOfDay = %v, want %v", got, want)
	}
}

func TestTheStoreListsEveryDoneTaskForTheCalendar(t *testing.T) {
	f := NewFake(fixedNow)
	old := fixedNow().AddDate(0, -3, 0)
	task, _ := f.CreateTask(CreateTaskInput{Title: "ancient", Source: SourceUser})
	if _, err := f.UpdateTask(task.ID, Patch{Status: StatusPtr(Done)}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	tt, _ := f.find(task.ID)
	tt.CompletedAt = &old
	f.mu.Unlock()
	all, err := f.ListTasks(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range all {
		found = found || x.ID == task.ID
	}
	if !found {
		t.Error("ListTasks leaves an old Done task out; the calendar would miss it")
	}
}

func TestCreateTaskTakesADueDate(t *testing.T) {
	f := NewFake(fixedNow)
	due := time.Date(2026, 10, 16, 18, 0, 0, 0, time.Local)
	task, err := f.CreateTask(CreateTaskInput{Title: "x", Source: SourceUser, DueAt: &due})
	if err != nil || task.DueAt == nil || !task.DueAt.Equal(due) {
		t.Fatalf("task %+v, %v", task, err)
	}
}
