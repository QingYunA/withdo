package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

func oct(d, h, min int) *time.Time {
	t := time.Date(2026, 10, d, h, min, 0, 0, time.Local)
	return &t
}

// calendarApp is an empty store with a few dated tasks around fixedNow (Wed Oct 14, 14:30).
func calendarApp(t *testing.T) (*App, *data.Fake) {
	t.Helper()
	f := data.NewEmptyFake(fixedNow)
	add := func(title string, src data.Source, p data.Patch) data.Task {
		task, err := f.CreateTask(data.CreateTaskInput{Title: title, Source: src})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.UpdateTask(task.ID, p); err != nil {
			t.Fatal(err)
		}
		task, _ = f.UpdateTask(task.ID, data.Patch{})
		return task
	}
	add("Review PR 12", data.SourceUser, data.Patch{DueAt: oct(16, 18, 0)})
	add("Dentist", data.SourceUser, data.Patch{RemindAt: oct(16, 9, 30)})
	add("Ship notes", data.SourceUser, data.Patch{Status: data.StatusPtr(data.Done)})
	add("Call the landlord", data.SourceUser, data.Patch{DueAt: oct(9, 18, 0)})
	m, err := f.CreateMilestone(f.AddProject("scratch", "me/scratch").ID, "v0.3 release", oct(23, 0, 0))
	if err != nil || m.ID == 0 {
		t.Fatal(err)
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	return app, f
}

func calendarTester(t *testing.T, app *App, width int) (*ui.Tester, *Main) {
	t.Helper()
	tt, m, _ := newTesterFor(app, width)
	m.goTo(nav{screen: screenCalendar})
	tt.Frame()
	return tt, m
}

func taskByTitle(t *testing.T, app *App, title string) data.Task {
	t.Helper()
	for _, task := range app.snap.tasks {
		if task.Title == title {
			return task
		}
	}
	t.Fatalf("no task %q", title)
	return data.Task{}
}

// dragOnto drags the thing showing from onto the thing showing to.
func dragOnto(t *testing.T, tt *ui.Tester, from, to string) {
	t.Helper()
	a, ok := tt.Find(from)
	if !ok {
		t.Fatalf("no %q in %q", from, tt.Texts())
	}
	b, ok := tt.Find(to)
	if !ok {
		t.Fatalf("no %q in %q", to, tt.Texts())
	}
	fx, fy := center(a)
	tx, ty := center(b)
	tt.Press(fx, fy)
	tt.Move(fx+8, fy+8)
	tt.Move(tx, ty+8)
	tt.Release(tx, ty+8)
}

func TestTheSidebarHasACalendarRowBetweenInboxAndCompleted(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, "Inbox", "Calendar", "Completed")
	if !above(t, tt, "Inbox", "Calendar") || !above(t, tt, "Calendar", "Completed") {
		t.Error("Calendar is not between Inbox and Completed")
	}
	if err := tt.Click("Calendar"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "October 2026", "Today")
}

func TestTheMonthIsTheTitleAndTheGridHasTheSundayFirstInEnglish(t *testing.T) {
	app, _ := calendarApp(t)
	tt, m := calendarTester(t, app, 1100)
	has(t, tt, "October 2026", "Sun", "Mon", "Sat", "Oct 1")
	if sun, _ := tt.Find("Sun"); sun.X > func() float32 { r, _ := tt.Find("Mon"); return r.X }() {
		t.Error("Sunday is not the first column")
	}
	days := m.calendarDays()
	if len(days) != 35 || days[0].Date.Weekday() != time.Sunday || days[0].Date.Day() != 27 {
		t.Fatalf("the grid starts %v with %d days", days[0].Date, len(days))
	}
}

func TestChineseWeeksStartOnMondayAndTheTitleHasSpaces(t *testing.T) {
	app, _ := calendarApp(t)
	tt, m := calendarTester(t, app, 1100)
	tt.SetLocale("zh-Hans")
	tt.Frame()
	has(t, tt, "2026 年 10 月", "周一", "周日", "今天", "日历")
	if days := m.calendarDays(); days[0].Date.Weekday() != time.Monday || days[0].Date.Day() != 28 {
		t.Fatalf("the grid starts %v", days[0].Date)
	}
	if mon, _ := tt.Find("周一"); mon.X > func() float32 { r, _ := tt.Find("周日"); return r.X }() {
		t.Error("Monday is not before Sunday")
	}
}

func TestTheSystemsFirstWeekdayWinsWhenItNamesARegion(t *testing.T) {
	app, _ := calendarApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetRegion(i18n.Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24})
	m.goTo(nav{screen: screenCalendar})
	tt.Frame()
	if days := m.calendarDays(); days[0].Date.Weekday() != time.Monday {
		t.Errorf("the grid starts on a %v", days[0].Date.Weekday())
	}
}

func TestItemsAreInTheirCells(t *testing.T) {
	app, _ := calendarApp(t)
	tt, _ := calendarTester(t, app, 1100)
	has(t, tt, "Review PR 12", "Dentist", "9:30 AM", "Ship notes", "Call the landlord", "v0.3 release")
	// The done task is struck through and sits on the day it was finished (today), not on a due date.
	r, _ := tt.Find("Ship notes")
	today, _ := tt.Find("14")
	if r.Y < today.Y || r.X < today.X-30 {
		t.Errorf("Ship notes is at %v, the 14th at %v", r, today)
	}
}

func TestACellListsThreeItemsThenHowManyMore(t *testing.T) {
	app, f := calendarApp(t)
	for _, title := range []string{"One", "Two", "Three", "Four", "Five"} {
		task, _ := f.CreateTask(data.CreateTaskInput{Title: title, Source: data.SourceUser})
		_, _ = f.UpdateTask(task.ID, data.Patch{DueAt: oct(21, 18, 0)})
	}
	app.Refresh()
	tt, _ := calendarTester(t, app, 1100)
	tt.SetSize(1100, 1000) // tall enough for three items and the "more" line in every cell
	tt.Frame()
	has(t, tt, "One", "Two", "Three", "2 more")
	hasNot(t, tt, "Four", "Five")
}

func TestInTheDefaultWindowACellKeepsRoomForTheMoreLine(t *testing.T) {
	app, f := calendarApp(t)
	for _, title := range []string{"One", "Two", "Three", "Four", "Five"} {
		task, _ := f.CreateTask(data.CreateTaskInput{Title: title, Source: data.SourceUser})
		_, _ = f.UpdateTask(task.ID, data.Patch{DueAt: oct(21, 18, 0)})
	}
	app.Refresh()
	tt, _ := calendarTester(t, app, 1100)
	more, ok := tt.Find("3 more")
	if !ok {
		t.Fatalf("no \"3 more\" in %q", tt.Texts())
	}
	day, _ := tt.Find("Wed, Oct 21")
	if more.Y+more.H > day.Y+day.H {
		t.Errorf("the more line (%v) is cut off by its cell (%v)", more, day)
	}
}

func TestHowManyItemsFitInACell(t *testing.T) {
	for _, tc := range []struct{ n, lines, shown, more int }{
		{0, 4, 0, 0},
		{3, 4, 3, 0},
		{4, 4, 3, 1}, // a "more" line takes the fourth line
		{5, 6, 3, 2},
		{3, 2, 1, 2},
		{2, 2, 2, 0},
		{2, 1, 0, 2},
		{3, 0, 0, 3},
	} {
		if s, m := visibleItems(tc.n, tc.lines); s != tc.shown || m != tc.more {
			t.Errorf("visibleItems(%d, %d) = %d, %d; want %d, %d", tc.n, tc.lines, s, m, tc.shown, tc.more)
		}
	}
}

func TestAnAuthorHasTheirOwnAvatar(t *testing.T) {
	app, f := calendarApp(t)
	agentTask, _ := f.CreateTask(data.CreateTaskInput{Title: "By codex", Source: data.SourceAgent})
	app.Refresh()
	m := NewMain(app)
	mine := data.CalItem{Task: taskByTitle(t, app, "Dentist")}
	if got := m.calAvatar(mine); got.Kind != theme.AvatarYou {
		t.Errorf("my task has %v", got.Kind)
	}
	theirs := data.CalItem{Task: agentTask, Agent: "codex"}
	if got := m.calAvatar(theirs); got.Kind != theme.AvatarCodex {
		t.Errorf("codex's task has %v", got.Kind)
	}
	if got := m.calAvatar(data.CalItem{Task: agentTask, Agent: "agent"}); got.Kind != theme.AvatarOther {
		t.Errorf("an unnamed agent has %v", got.Kind)
	}
}

func TestClickingADaySelectsItAndTheComposerAddsToIt(t *testing.T) {
	app, _ := calendarApp(t)
	app.Assistant = &data.FakeAssistant{Ready: true}
	tt, m := calendarTester(t, app, 1100)
	has(t, tt, "Add to Wed, Oct 14, or ask me anything") // today is selected on arrival
	if err := tt.Click("16"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "Add to Fri, Oct 16, or ask me anything")
	if d := m.calendarDay(); d.Day() != 16 {
		t.Errorf("selected %v", d)
	}
}

func TestChineseComposerNamesTheSelectedDay(t *testing.T) {
	app, _ := calendarApp(t)
	tt, _ := calendarTester(t, app, 1100)
	tt.SetLocale("zh-Hans")
	tt.Frame()
	if err := tt.Click("16"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "添加到10 月 16 日周五，或者问我任何事")
}

func TestTheAssistantIsToldTheSelectedDay(t *testing.T) {
	app, _ := calendarApp(t)
	fa := &data.FakeAssistant{Ready: true}
	app.Assistant = fa
	tt, _ := calendarTester(t, app, 1100)
	if err := tt.Click("16"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("buy milk")
	tt.Key(0, ui.KeyEnter)
	if got := fa.Places(); len(got) != 1 || got[0] != (data.Place{Kind: data.PlaceCalendar, Day: "2026-10-16"}) {
		t.Errorf("places %+v", got)
	}
}

func TestWithoutAModelAPlainTaskIsDueAtSixOnTheSelectedDay(t *testing.T) {
	app, _ := calendarApp(t)
	tt, _ := calendarTester(t, app, 1100)
	if err := tt.Click("16"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("Water the plants")
	tt.Key(0, ui.KeyEnter)
	task := taskByTitle(t, app, "Water the plants")
	want := time.Date(2026, 10, 16, 18, 0, 0, 0, time.Local)
	if task.DueAt == nil || !task.DueAt.Equal(want) || task.ProjectID != nil {
		t.Errorf("task %+v, want due %v in the Inbox", task, want)
	}
	has(t, tt, "Water the plants") // it is on the grid now
}

func TestOtherPagesAddPlainTasksWithNoDueDate(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.Assistant = nil
	tt, _, _ := newTesterFor(app, 1100)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Type("No date")
	tt.Key(0, ui.KeyEnter)
	if task := taskByTitle(t, app, "No date"); task.DueAt != nil {
		t.Errorf("due %v", task.DueAt)
	}
}

func TestClickingAnItemOpensTheInspector(t *testing.T) {
	app, _ := calendarApp(t)
	tt, m := calendarTester(t, app, 1100)
	if err := tt.Click("Review PR 12"); err != nil {
		t.Fatal(err)
	}
	if want := taskByTitle(t, app, "Review PR 12").ID; m.openID != want {
		t.Errorf("open %d, want %d", m.openID, want)
	}
	if d := m.calendarDay(); d.Day() == 16 {
		t.Error("clicking an item also selected its day")
	}
}

func TestDraggingADueItemToAnotherDayKeepsItsTime(t *testing.T) {
	app, _ := calendarApp(t)
	tt, _ := calendarTester(t, app, 1100)
	dragOnto(t, tt, "Review PR 12", "20")
	task := taskByTitle(t, app, "Review PR 12")
	if want := time.Date(2026, 10, 20, 18, 0, 0, 0, time.Local); task.DueAt == nil || !task.DueAt.Equal(want) {
		t.Errorf("due %v, want %v", task.DueAt, want)
	}
}

func TestDraggingAReminderMovesTheReminderNotTheDueDate(t *testing.T) {
	app, f := calendarApp(t)
	task := taskByTitle(t, app, "Dentist")
	_, _ = f.UpdateTask(task.ID, data.Patch{DueAt: oct(28, 18, 0)})
	app.Refresh()
	tt, _ := calendarTester(t, app, 1100)
	dragOnto(t, tt, "Dentist", "21")
	got := taskByTitle(t, app, "Dentist")
	if want := time.Date(2026, 10, 21, 9, 30, 0, 0, time.Local); got.RemindAt == nil || !got.RemindAt.Equal(want) {
		t.Errorf("remind %v, want %v", got.RemindAt, want)
	}
	if want := time.Date(2026, 10, 28, 18, 0, 0, 0, time.Local); got.DueAt == nil || !got.DueAt.Equal(want) {
		t.Errorf("the due date moved: %v", got.DueAt)
	}
}

func TestDoneItemsDoNotDrag(t *testing.T) {
	app, _ := calendarApp(t)
	before := taskByTitle(t, app, "Ship notes")
	tt, _ := calendarTester(t, app, 1100)
	dragOnto(t, tt, "Ship notes", "21")
	after := taskByTitle(t, app, "Ship notes")
	if after.DueAt != nil || !after.CompletedAt.Equal(*before.CompletedAt) || after.Status != data.Done {
		t.Errorf("the done task changed: %+v", after)
	}
}

func TestMonthButtonsAndToday(t *testing.T) {
	app, _ := calendarApp(t)
	tt, m := calendarTester(t, app, 1100)
	if err := tt.Click("Next month"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "November 2026")
	hasNot(t, tt, "Review PR 12") // October's items are not on a November grid
	if err := tt.Click("Previous month"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Previous month"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "September 2026")
	if err := tt.Click("Today"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "October 2026")
	if d := m.calendarDay(); d.Day() != 14 || d.Month() != time.October {
		t.Errorf("Today selected %v", d)
	}
}

func TestTheKeyboardWalksTheDaysAndMonths(t *testing.T) {
	app, _ := calendarApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.Key(ui.Cmd, ui.Key4)
	if m.nav.screen != screenCalendar {
		t.Fatal("Cmd+4 did not open the calendar")
	}
	day := func() string { return m.calendarDay().Format("2006-01-02") }
	for _, step := range []struct {
		mods ui.Modifiers
		key  ui.Key
		want string
	}{
		{0, ui.KeyRight, "2026-10-15"},
		{0, ui.KeyDown, "2026-10-22"},
		{0, ui.KeyLeft, "2026-10-21"},
		{0, ui.KeyUp, "2026-10-14"},
		{ui.Cmd, ui.KeyRight, "2026-11-14"},
		{ui.Cmd, ui.KeyLeft, "2026-10-14"},
	} {
		tt.Key(step.mods, step.key)
		if got := day(); got != step.want {
			t.Fatalf("after %v: %s, want %s", step.key, got, step.want)
		}
	}
	// Walking past the end of the month turns the page.
	m.selectDay(time.Date(2026, 10, 31, 0, 0, 0, 0, time.Local))
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	has(t, tt, "November 2026")
}

func TestThePageKeepsItsDayWhenYouLeaveAndComeBack(t *testing.T) {
	app, _ := calendarApp(t)
	tt, m := calendarTester(t, app, 1100)
	if err := tt.Click("16"); err != nil {
		t.Fatal(err)
	}
	m.goTo(nav{})
	tt.Frame()
	m.goTo(nav{screen: screenCalendar})
	tt.Frame()
	if m.calendarDay().Day() != 16 {
		t.Errorf("the selected day was forgotten: %v", m.calendarDay())
	}
}

func TestOverdueOpenTasksAreRed(t *testing.T) {
	app, _ := calendarApp(t)
	m := NewMain(app)
	tok := theme.For(false, ui.Transparent)
	late := data.CalItem{Task: taskByTitle(t, app, "Call the landlord"), Kind: data.CalDue, Overdue: true}
	if got := m.calInk(tok, late); got != tok.Blocked() {
		t.Errorf("overdue title is %v", got)
	}
	done := data.CalItem{Task: taskByTitle(t, app, "Ship notes"), Kind: data.CalDone}
	if got := m.calInk(tok, done); got != tok.TextTertiary {
		t.Errorf("done title is %v", got)
	}
	plain := data.CalItem{Task: taskByTitle(t, app, "Review PR 12"), Kind: data.CalDue}
	if got := m.calInk(tok, plain); got != tok.Text {
		t.Errorf("plain title is %v", got)
	}
}

func TestTheCalendarHasAComposerAndNotTheOtherPages(t *testing.T) {
	app, _ := calendarApp(t)
	_, m := calendarTester(t, app, 1100)
	if !m.pageHasComposer() {
		t.Error("the calendar has no composer")
	}
}
