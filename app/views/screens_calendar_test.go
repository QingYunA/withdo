package views

import (
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// The Calendar screens, in light and dark, in English and Chinese, at the normal window and the
// narrowest one, with a day selected and with the inspector open. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensCalendar
//
// The month is October 2026, the clock Wed Oct 14, and the data is the one of docs/design/mockups/v4.png.

// calendarShowcase is a store with a month of tasks written by me and by agents.
func calendarShowcase(t *testing.T) *App {
	t.Helper()
	f := data.NewEmptyFake(fixedNow)
	proj := f.AddProject("adhd-todo", "QingYunA/adhd-todo")
	local := func(m time.Month, d, h, min int) *time.Time {
		at := time.Date(2026, m, d, h, min, 0, 0, time.Local)
		return &at
	}
	add := func(title string, src data.Source, st data.Status, agent string, edit func(*data.Task)) {
		task := data.Task{Title: title, Source: src, Status: st, Priority: data.P2, ProjectID: &proj.ID}
		edit(&task)
		task = f.AddTask(task)
		if agent != "" {
			f.AddReport(data.Report{TaskID: task.ID, Agent: agent, Summary: "Working on it", CreatedAt: fixedNow().Add(-time.Hour)})
		}
	}
	doneOn := func(m time.Month, d int) func(*data.Task) {
		return func(t *data.Task) { t.CompletedAt = local(m, d, 10, 0) }
	}
	dueOn := func(m time.Month, d, h, min int) func(*data.Task) {
		return func(t *data.Task) { t.DueAt = local(m, d, h, min) }
	}
	remindOn := func(m time.Month, d, h, min int) func(*data.Task) {
		return func(t *data.Task) { t.RemindAt = local(m, d, h, min) }
	}
	const claude, codex = "claude-code", "codex"
	add("Pay the electricity bill", data.SourceUser, data.Done, "", doneOn(time.September, 30))
	add("M2 assistant is live", data.SourceAgent, data.Done, claude, doneOn(time.October, 2))
	add("M3 settings window", data.SourceAgent, data.Done, claude, doneOn(time.October, 8))
	add("Health check report", data.SourceUser, data.Done, "", doneOn(time.October, 8))
	add("Reply to the landlord", data.SourceUser, data.Todo, "", dueOn(time.October, 9, 18, 0))
	add("Store migration tests", data.SourceAgent, data.Done, codex, doneOn(time.October, 12))
	add("M4 GitHub polling", data.SourceAgent, data.Done, claude, doneOn(time.October, 13))
	add("Pay the rent", data.SourceUser, data.Done, "", doneOn(time.October, 13))
	add("Interface restraint v3", data.SourceAgent, data.Done, claude, doneOn(time.October, 14))
	add("M5 calendar and avatars", data.SourceAgent, data.Doing, claude, dueOn(time.October, 15, 18, 0))
	add("Dentist follow-up", data.SourceUser, data.Todo, "", remindOn(time.October, 15, 14, 0))
	add("Parser edge cases", data.SourceAgent, data.Doing, codex, dueOn(time.October, 15, 18, 0))
	add("Book the train", data.SourceUser, data.Todo, "", dueOn(time.October, 15, 18, 0))
	add("Call Sam back", data.SourceUser, data.Todo, "", dueOn(time.October, 15, 18, 0))
	add("Fix the login redirect loop", data.SourceUser, data.Todo, "", dueOn(time.October, 16, 18, 0))
	add("Review PR 12", data.SourceUser, data.Todo, "", remindOn(time.October, 16, 9, 0))
	add("Go for a run", data.SourceUser, data.Todo, "", remindOn(time.October, 17, 10, 0))
	add("Write the v0.3 release notes", data.SourceUser, data.Todo, "", dueOn(time.October, 20, 18, 0))
	add("Renew the domain", data.SourceAgent, data.Todo, "cursor", remindOn(time.October, 22, 20, 0))
	add("Quarterly review", data.SourceUser, data.Todo, "", dueOn(time.October, 27, 18, 0))
	if _, err := f.CreateMilestone(proj.ID, "v0.3 release", local(time.October, 23, 0, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CreateMilestone(proj.ID, "v0.2 native", local(time.October, 28, 0, 0)); err != nil {
		t.Fatal(err)
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	return app
}

// calendarShot renders the Calendar page at 2x, size points wide and high, and the language chosen.
func calendarShot(t *testing.T, name string, dark, chinese bool, w, h int, setup func(*ui.Tester, *Main)) {
	t.Helper()
	m := NewMain(calendarShowcase(t))
	tt := ui.NewTester(m.View, w, h)
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	if chinese {
		tt.SetLocale("zh-Hans")
		tt.SetRegion(i18n.Region{Locale: "zh-CN", FirstWeekday: time.Monday, HourCycle: 12})
	}
	m.goTo(nav{screen: screenCalendar})
	tt.Frame()
	if setup != nil {
		setup(tt, m)
		tt.Frame()
	}
	settle(tt)
	shot(t, name, tt.Image(), false, dark)
}

func TestScreensCalendar(t *testing.T) {
	screensTest(t)
	pickFriday := func(tt *ui.Tester, m *Main) { m.selectDay(time.Date(2026, 10, 16, 0, 0, 0, 0, time.Local)) }
	for _, a := range appearances() {
		for _, lang := range []struct {
			suffix  string
			chinese bool
		}{{"", false}, {"-zh", true}} {
			calendarShot(t, "calendar"+lang.suffix+"-"+a.suffix+".png", a.dark, lang.chinese, 1100, 720, pickFriday)
			calendarShot(t, "calendar-narrow"+lang.suffix+"-"+a.suffix+".png", a.dark, lang.chinese, 860, 600, pickFriday)
		}
		calendarShot(t, "calendar-inspector-"+a.suffix+".png", a.dark, false, 1100, 720, func(tt *ui.Tester, m *Main) {
			pickFriday(tt, m)
			m.OpenTask(taskID(t, m, "Fix the login redirect loop"))
			m.nav = nav{screen: screenCalendar}
		})
		calendarShot(t, "calendar-tall-"+a.suffix+".png", a.dark, false, 1100, 900, pickFriday)
		calendarShot(t, "calendar-day-card-"+a.suffix+".png", a.dark, false, 1100, 720, openDayCardOf15)
		if !a.dark {
			calendarShot(t, "calendar-day-card-zh-"+a.suffix+".png", false, true, 1100, 720, openDayCardOf15)
		}
	}
}

// openDayCardOf15 opens the day card of Thu Oct 15, the day with the most items, by its "N more"
// line (the animation is off: the card is full size).
func openDayCardOf15(tt *ui.Tester, m *Main) {
	for _, s := range tt.Texts() {
		if (strings.HasPrefix(s, "Show all ") && strings.HasSuffix(s, " on Oct 15")) || (strings.HasPrefix(s, "显示 10 月 15 日")) {
			_ = tt.Click(s)
			return
		}
	}
	panic("no \"more\" line on the 15th in " + strings.Join(tt.Texts(), " | "))
}

func taskID(t *testing.T, m *Main, title string) int {
	t.Helper()
	for _, task := range m.app.snap.tasks {
		if task.Title == title {
			return task.ID
		}
	}
	t.Fatalf("no task %q", title)
	return 0
}
