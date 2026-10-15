package words

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/i18n"
)

var now = time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local) // a Wednesday

func at(days int, hour, min int) time.Time {
	return time.Date(2026, 10, 14+days, hour, min, 0, 0, time.Local)
}

func check(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

func TestEnglishWords(t *testing.T) {
	k := New("en", i18n.Region{})
	check(t, "Age 30s", k.Age(now.Add(-30*time.Second), now), "now")
	check(t, "Age 2m", k.Age(now.Add(-2*time.Minute), now), "2m")
	check(t, "Age 3h", k.Age(now.Add(-3*time.Hour), now), "3h")
	check(t, "Age 2d", k.Age(now.Add(-48*time.Hour), now), "2d")
	check(t, "Age 10d", k.Age(now.Add(-10*24*time.Hour), now), "Oct 4")
	check(t, "Span 0", k.Span(0), "1h")
	check(t, "Span 5", k.Span(5), "5h")
	check(t, "Span 72", k.Span(72), "3d")
	check(t, "Stale 54", k.Stale(54), "2d stale")
	check(t, "Stale 30", k.Stale(30), "30h stale")
	check(t, "Heading", k.Heading(now), "Wednesday, Oct 14")
	check(t, "Time", k.Time(at(0, 16, 0)), "4:00 PM")
	check(t, "DayInline tomorrow", k.DayInline(at(1, 9, 0), now), "tomorrow")
	check(t, "DayInline Saturday", k.DayInline(at(3, 9, 0), now), "Saturday")
	check(t, "DayInline far", k.DayInline(at(9, 9, 0), now), "Oct 23")
	check(t, "DayInline next year", k.DayInline(time.Date(2027, 1, 2, 9, 0, 0, 0, time.Local), now), "Jan 2, 2027")
}

func TestEnglishWhenAndMoments(t *testing.T) {
	k := New("en", i18n.Region{})
	for _, c := range []struct {
		in   time.Time
		want string
	}{
		{at(0, 16, 0), "Today 4:00 PM"},
		{at(1, 9, 0), "Tomorrow 9:00 AM"},
		{at(2, 13, 30), "Fri 1:30 PM"},
		{at(-1, 9, 0), "Yesterday 9:00 AM"},
		{at(10, 9, 0), "Oct 24 9:00 AM"},
		{time.Date(2027, 1, 2, 9, 0, 0, 0, time.Local), "Jan 2, 2027 9:00 AM"},
	} {
		check(t, "When", k.When(c.in, now), c.want)
	}
	check(t, "DayShort", k.DayShort(at(2, 18, 0), now), "Fri")
	check(t, "DayShort today", k.DayShort(at(0, 18, 0), now), "Today")
	check(t, "Moment day", k.Moment(at(1, 0, 0), now, false), "Tomorrow")
	check(t, "Moment reminder", k.Moment(at(2, 10, 0), now, true), "Friday, 10:00 AM")
}

func TestDueLabel(t *testing.T) {
	k := New("en", i18n.Region{})
	due := func(d time.Time) data.Task { return data.Task{DueAt: &d} }
	remind := func(d time.Time) data.Task { return data.Task{RemindAt: &d} }
	for _, c := range []struct {
		task data.Task
		want string
	}{
		{due(at(0, 17, 0)), "Today"},
		{due(at(1, 9, 0)), "Tomorrow"},
		{due(at(2, 9, 0)), "Fri"},
		{due(at(-1, 9, 0)), "Yesterday"},
		{due(at(7, 9, 0)), "Oct 21"},
		{remind(at(0, 16, 0)), "4:00 PM"},
		{remind(at(1, 9, 0)), "Tomorrow"},
		{data.Task{}, ""},
	} {
		check(t, "DueLabel", k.DueLabel(c.task, now), c.want)
	}
}

func TestReminderSentences(t *testing.T) {
	k := New("en", i18n.Region{})
	check(t, "today", k.Reminder(at(0, 13, 30), now), "You asked to be reminded at 1:30 PM")
	check(t, "yesterday", k.Reminder(at(-1, 14, 30), now), "You asked to be reminded yesterday at 2:30 PM")
	check(t, "earlier", k.Reminder(at(-3, 14, 30), now), "You asked to be reminded on Oct 11 at 2:30 PM")
}

func TestChineseWords(t *testing.T) {
	k := New("zh-Hans", i18n.Region{HourCycle: 12})
	check(t, "Age 30s", k.Age(now.Add(-30*time.Second), now), "刚刚")
	check(t, "Age 2m", k.Age(now.Add(-2*time.Minute), now), "2分钟前")
	check(t, "Age 3h", k.Age(now.Add(-3*time.Hour), now), "3小时前")
	check(t, "Age 2d", k.Age(now.Add(-48*time.Hour), now), "2天前")
	check(t, "Age 10d", k.Age(now.Add(-10*24*time.Hour), now), "10月4日")
	check(t, "Stale", k.Stale(80), "停滞3天")
	check(t, "Heading", k.Heading(now), "10月14日星期三")
	check(t, "Time", k.Time(at(0, 16, 0)), "下午4:00")
	check(t, "When", k.When(at(1, 9, 0), now), "明天上午9:00")
	check(t, "Reminder", k.Reminder(at(0, 16, 0), now.Add(2*time.Hour)), "你设置了下午4:00的提醒")
	due := func(d time.Time) data.Task { return data.Task{DueAt: &d} }
	check(t, "DueLabel today", k.DueLabel(due(at(0, 17, 0)), now), "今天")
	check(t, "DueLabel Fri", k.DueLabel(due(at(2, 9, 0)), now), "周五")
	check(t, "DueLabel date", k.DueLabel(due(at(7, 9, 0)), now), "10月21日")
	check(t, "DayInline", k.DayInline(at(3, 9, 0), now), "星期六")
	check(t, "Moment", k.Moment(at(2, 10, 0), now, true), "星期五 上午10:00")
}

func TestRegionsHourCycle(t *testing.T) {
	r := i18n.Region{Locale: "en-GB", HourCycle: 24}
	check(t, "24 hours", New("en", r).Time(at(0, 16, 0)), "16:00")
	check(t, "Chinese 24 hours", New("zh-Hans", i18n.Region{Locale: "zh-CN", HourCycle: 24}).Time(at(0, 16, 0)), "16:00")
	check(t, "Chinese 12 hours", New("zh-Hans", i18n.Region{Locale: "zh-CN", HourCycle: 12}).Time(at(0, 16, 0)), "下午4:00")
}

func TestDayHeadsTheLogbook(t *testing.T) {
	en, zh := New("en", i18n.Region{}), New("zh-Hans", i18n.Region{})
	check(t, "today", en.Day(at(0, 9, 0), now), "Today")
	check(t, "yesterday", en.Day(at(-1, 23, 0), now), "Yesterday")
	check(t, "two days ago", en.Day(at(-2, 9, 0), now), "Oct 12")
	check(t, "another year", en.Day(time.Date(2025, 12, 31, 9, 0, 0, 0, time.Local), now), "Dec 31, 2025")
	check(t, "Chinese today", zh.Day(at(0, 9, 0), now), "今天")
	check(t, "Chinese yesterday", zh.Day(at(-1, 9, 0), now), "昨天")
	check(t, "Chinese date", zh.Day(at(-2, 9, 0), now), "10月12日")
}

func TestCalendarWordsInEnglish(t *testing.T) {
	k := New("en", i18n.Region{})
	check(t, "MonthTitle", k.MonthTitle(now), "October 2026")
	check(t, "CalendarDay", k.CalendarDay(at(2, 0, 0)), "Fri, Oct 16")
	check(t, "MonthDay", k.MonthDay(time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)), "Nov 1")
	check(t, "Weekday", k.Weekday(time.Date(2026, 10, 12, 0, 0, 0, 0, time.Local)), "Mon")
}

func TestCalendarWordsInChinese(t *testing.T) {
	k := New("zh-Hans", i18n.Region{})
	check(t, "MonthTitle", k.MonthTitle(now), "2026 年 10 月")
	check(t, "CalendarDay", k.CalendarDay(at(2, 0, 0)), "10 月 16 日周五")
	check(t, "MonthDay", k.MonthDay(time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)), "11 月 1 日")
	check(t, "Weekday", k.Weekday(time.Date(2026, 10, 12, 0, 0, 0, 0, time.Local)), "周一")
}

func TestWeeksStartOnMondayInChineseAndSundayInEnglishUnlessTheSystemSaysOtherwise(t *testing.T) {
	for _, tc := range []struct {
		lang   string
		region i18n.Region
		want   time.Weekday
	}{
		{"en", i18n.Region{}, time.Sunday},
		{"zh-Hans", i18n.Region{}, time.Monday},
		{"en", i18n.Region{Locale: "en-GB", FirstWeekday: time.Monday}, time.Monday},
		{"zh-Hans", i18n.Region{Locale: "zh-CN", FirstWeekday: time.Sunday}, time.Sunday},
	} {
		if got := FirstWeekday(tc.lang, tc.region); got != tc.want {
			t.Errorf("FirstWeekday(%q, %+v) = %v, want %v", tc.lang, tc.region, got, tc.want)
		}
	}
}
