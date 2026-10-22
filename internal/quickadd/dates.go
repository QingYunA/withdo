package quickadd

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// dayWord is the calendar day the words name.
type dayWord struct {
	y        int
	m        time.Month
	d        int
	deadline bool       // "by", "before", "前"... stood next to it
	period   string     // 晚上 or 早上 when the word itself says so (今晚, 明早), for a bare hour after it
	dflt     *clockWord // the time of day the word means alone: 今晚 is 20:00
}

var (
	isoDate  = regexp.MustCompile(`\b(\d{4})-(\d{1,2})-(\d{1,2})\b`)
	zhMonDay = regexp.MustCompile(`(\d{1,2})月(\d{1,2})[日号]?`)
	zhDay    = regexp.MustCompile(`(\d{1,2})号`)
	monthRe  = `(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)`
	enMonDay = regexp.MustCompile(`(?i)\b` + monthRe + `\.?\s+(\d{1,2})(?:st|nd|rd|th)?\b`)
	enDayMon = regexp.MustCompile(`(?i)\b(\d{1,2})(?:st|nd|rd|th)?\s+(?:of\s+)?` + monthRe + `\b`)
	numDate  = regexp.MustCompile(`(\d{1,2})[./-](\d{1,2})`)
	zhRel    = regexp.MustCompile(`大后天|后天|明天|明日|今天|今日|今晚|明晚|今早|明早`)
	enRel    = regexp.MustCompile(`(?i)\bday after tomorrow\b|\btomorrow\b|\btoday\b|\btonight\b`)
	zhWeek   = regexp.MustCompile(`(下下|下|这|本|上)?(?:个)?(?:周|星期|礼拜)([一二三四五六日天])`)
	enWeek   = regexp.MustCompile(`(?i)\b(?:(next|this|last)\s+)?(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`)
)

var months = map[string]time.Month{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

var zhWeekdays = map[string]time.Weekday{
	"一": time.Monday, "二": time.Tuesday, "三": time.Wednesday, "四": time.Thursday, "五": time.Friday, "六": time.Saturday,
	"日": time.Sunday, "天": time.Sunday,
}

var enWeekdays = map[string]time.Weekday{
	"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday, "thursday": time.Thursday,
	"friday": time.Friday, "saturday": time.Saturday, "sunday": time.Sunday,
}

// date finds the day the words name and cuts it out with the words around it.
func (sc *scan) date(now time.Time) (dayWord, bool) {
	rules := []struct {
		re  *regexp.Regexp
		day func(g []string) (dayWord, bool)
		ok  func(m []int) bool
	}{
		{isoDate, func(g []string) (dayWord, bool) { return exact(num(g[1]), num(g[2]), num(g[3]), now) }, nil},
		{zhMonDay, func(g []string) (dayWord, bool) { return nextMonthDay(num(g[1]), num(g[2]), now) }, sc.notAfterDigit},
		{zhDay, func(g []string) (dayWord, bool) { return nextDayOfMonth(num(g[1]), now) }, sc.zhDayOK},
		{enMonDay, func(g []string) (dayWord, bool) { return nextMonthDay(int(months[mon(g[1])]), num(g[2]), now) }, nil},
		{enDayMon, func(g []string) (dayWord, bool) { return nextMonthDay(int(months[mon(g[2])]), num(g[1]), now) }, nil},
		{numDate, func(g []string) (dayWord, bool) { return nextMonthDay(num(g[1]), num(g[2]), now) }, sc.numDateOK},
		{zhRel, func(g []string) (dayWord, bool) { return relative(g[0], now) }, nil},
		{enRel, func(g []string) (dayWord, bool) { return relative(strings.ToLower(g[0]), now) }, nil},
		{zhWeek, func(g []string) (dayWord, bool) { return weekday(zhWeekdays[g[2]], zhMode(g[1]), now) }, nil},
		{enWeek, func(g []string) (dayWord, bool) {
			return weekday(enWeekdays[strings.ToLower(g[2])], strings.ToLower(g[1]), now)
		}, nil},
	}
	for _, r := range rules {
		for _, m := range r.re.FindAllStringSubmatchIndex(sc.s, -1) {
			if r.ok != nil && !r.ok(m) {
				continue
			}
			day, ok := r.day(sc.groups(m))
			if !ok {
				continue
			}
			day.deadline = sc.cutWords(m[0], m[1])
			return day, true
		}
	}
	return dayWord{}, false
}

func num(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func mon(s string) string { return strings.ToLower(s)[:3] }

// notAfterDigit rejects a match that is the tail of a longer number.
func (sc *scan) notAfterDigit(m []int) bool {
	r, _ := utf8.DecodeLastRuneInString(sc.s[:m[0]])
	return !(r >= '0' && r <= '9')
}

// zhDayOK rejects "15号线" (a metro line), "3号楼" and the like: only a day of the month is read.
func (sc *scan) zhDayOK(m []int) bool {
	next, _ := utf8.DecodeRuneInString(sc.s[m[1]:])
	return sc.notAfterDigit(m) && !strings.ContainsRune("线楼院门店口层", next)
}

// numDateOK keeps "10.31" and "10-31" and rejects the numbers that only look like one: "3.5kg",
// "1.2.3", the tail of "2026-10-31", the minutes of "10:31".
func (sc *scan) numDateOK(m []int) bool {
	prev, _ := utf8.DecodeLastRuneInString(sc.s[:m[0]])
	if asciiWord(prev) || prev == '.' || prev == '-' || prev == '/' || prev == ':' {
		return false
	}
	next, size := utf8.DecodeRuneInString(sc.s[m[1]:])
	if asciiWord(next) || next == ':' {
		return false
	}
	if next == '.' || next == '-' || next == '/' {
		after, _ := utf8.DecodeRuneInString(sc.s[m[1]+size:])
		return !(after >= '0' && after <= '9')
	}
	return true
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func dayOf(t time.Time) dayWord {
	y, m, d := t.Date()
	return dayWord{y: y, m: m, d: d}
}

// exact is a date with its year; one that does not exist is not read.
func exact(y, mo, d int, now time.Time) (dayWord, bool) {
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, now.Location())
	if t.Year() != y || int(t.Month()) != mo || t.Day() != d {
		return dayWord{}, false
	}
	return dayOf(t), true
}

// nextMonthDay is the first time the date comes round, today included: one already gone this year
// is next year's. A date that does not exist (Feb 30) is not read.
func nextMonthDay(mo, d int, now time.Time) (dayWord, bool) {
	today := midnight(now)
	for y := now.Year(); y <= now.Year()+4; y++ {
		t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, now.Location())
		if int(t.Month()) != mo || t.Day() != d {
			continue // Feb 29 in a year without one
		}
		if !t.Before(today) {
			return dayOf(t), true
		}
	}
	return dayWord{}, false
}

// nextDayOfMonth is "15号": this month's 15th when it is not over, else the next month that has one.
func nextDayOfMonth(d int, now time.Time) (dayWord, bool) {
	today := midnight(now)
	for i := 0; i < 12; i++ {
		t := time.Date(now.Year(), now.Month()+time.Month(i), d, 0, 0, 0, 0, now.Location())
		if t.Day() == d && !t.Before(today) {
			return dayOf(t), true
		}
	}
	return dayWord{}, false
}

var (
	evening = clockWord{h: 20}
	morning = clockWord{h: 9}
)

// relative is today, tomorrow and the days around them, and the evening and morning words.
func relative(w string, now time.Time) (dayWord, bool) {
	days, period, dflt := 0, "", (*clockWord)(nil)
	switch w {
	case "大后天":
		days = 3
	case "后天", "day after tomorrow":
		days = 2
	case "明天", "明日", "tomorrow":
		days = 1
	case "明晚":
		days, period, dflt = 1, "晚上", &evening
	case "明早":
		days, period, dflt = 1, "早上", &morning
	case "今晚", "tonight":
		period, dflt = "晚上", &evening
	case "今早":
		period, dflt = "早上", &morning
	}
	y, m, d := now.Date()
	day := dayOf(time.Date(y, m, d+days, 0, 0, 0, 0, now.Location()))
	day.period, day.dflt = period, dflt
	return day, true
}

// zhMode is the week word before 周五 as the English one says it.
func zhMode(w string) string {
	switch w {
	case "下下":
		return "next next"
	case "下":
		return "next"
	case "这", "本":
		return "this"
	case "上":
		return "last"
	}
	return ""
}

// weekday is a day of the week. The week runs Monday to Sunday: "next Monday" on a Sunday is
// tomorrow, "this Monday" on a Wednesday is gone and means the one coming, and a bare "Friday" is
// the first one after today.
func weekday(wd time.Weekday, mode string, now time.Time) (dayWord, bool) {
	cur, tgt := (int(now.Weekday())+6)%7, (int(wd)+6)%7
	var delta int
	switch mode {
	case "":
		if delta = (tgt - cur + 7) % 7; delta == 0 {
			delta = 7
		}
	case "this":
		if delta = tgt - cur; delta < 0 {
			delta += 7
		}
	case "next":
		delta = 7 - cur + tgt
	case "next next":
		delta = 14 - cur + tgt
	default: // last: the past is not read
		return dayWord{}, false
	}
	y, m, d := now.Date()
	return dayOf(time.Date(y, m, d+delta, 0, 0, 0, 0, now.Location())), true
}
