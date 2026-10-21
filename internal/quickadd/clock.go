package quickadd

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// clockWord is a time of day. h may be 24, which is midnight of the next day.
type clockWord struct {
	h, min   int
	deadline bool // "by", "before", "前"... stood next to it
}

const (
	zhNumeral = `[零一二两三四五六七八九十]{1,3}`
	zhPeriod  = `凌晨|早上|早晨|上午|中午|下午|傍晚|晚上`
)

var (
	zhHour    = regexp.MustCompile(`(` + zhPeriod + `)?\s?(\d{1,2}|` + zhNumeral + `)(点钟|点|:|：)(?:(\d{1,2})分?|(半)|(一刻))?`)
	enAMPM    = regexp.MustCompile(`(?i)\b(\d{1,2})(?::(\d{2}))?\s*([ap])\.?m\b\.?`)
	enAt      = regexp.MustCompile(`(?i)\bat\s+(\d{1,2})(?::(\d{2}))?\b`)
	enNoon    = regexp.MustCompile(`(?i)\bnoon\b`)
	zhPeriods = regexp.MustCompile(zhPeriod)
	ampm      = regexp.MustCompile(`(?i)^\s*[ap]\.?m\b`)
)

// periodHours are the times of day the words alone mean, the ones the assistant's prompt uses too.
var periodHours = map[string]int{"早上": 9, "早晨": 9, "上午": 9, "中午": 12, "下午": 14, "傍晚": 18, "晚上": 19}

// clock finds a time of day and cuts it out with the words around it. dayPeriod is the 晚上 or 早上
// that 今晚 or 明早 already said, which a bare hour after it takes.
func (sc *scan) clock(dayPeriod string) (clockWord, bool) {
	if m := sc.find(zhHour, sc.zhHourOK); m != nil {
		g := sc.groups(m)
		h, _ := zhNum(g[2])
		period := g[1]
		if period == "" {
			period = dayPeriod
		}
		minute := num(g[4])
		switch {
		case g[5] != "":
			minute = 30
		case g[6] != "":
			minute = 15
		}
		t := clockWord{h: hour24(h, period, g[3] != ":" && g[3] != "："), min: minute}
		t.deadline = sc.cutWords(m[0], m[1])
		return t, true
	}
	if m := sc.find(enAMPM, nil); m != nil {
		g := sc.groups(m)
		h := num(g[1])
		if h < 1 || h > 12 || num(g[2]) > 59 {
			return clockWord{}, false
		}
		h %= 12
		if strings.EqualFold(g[3], "p") {
			h += 12
		}
		t := clockWord{h: h, min: num(g[2])}
		t.deadline = sc.cutWords(m[0], m[1])
		return t, true
	}
	if m := sc.find(enAt, func(m []int) bool { g := sc.groups(m); return num(g[1]) <= 24 && num(g[2]) <= 59 }); m != nil {
		g := sc.groups(m)
		t := clockWord{h: hour24(num(g[1]), dayPeriod, true), min: num(g[2])}
		t.deadline = sc.cutWords(m[0], m[1])
		return t, true
	}
	if m := sc.find(enNoon, nil); m != nil {
		t := clockWord{h: 12}
		t.deadline = sc.cutWords(m[0], m[1])
		return t, true
	}
	if m := sc.find(zhPeriods, func(m []int) bool { return sc.s[m[0]:m[1]] != "凌晨" }); m != nil {
		t := clockWord{h: periodHours[sc.s[m[0]:m[1]]]}
		t.deadline = sc.cutWords(m[0], m[1])
		return t, true
	}
	return clockWord{}, false
}

// zhHourOK keeps real times and rejects 一点 ("a bit"), numbers that are the tail of a longer one,
// hours past 24 and minutes past 59.
func (sc *scan) zhHourOK(m []int) bool {
	g := sc.groups(m)
	h, ok := zhNum(g[2])
	if !ok || h > 24 || num(g[4]) > 59 {
		return false
	}
	colon := g[3] == ":" || g[3] == "："
	if colon && (len(g[4]) != 2 || !isDigits(g[2])) {
		return false
	}
	if ampm.MatchString(sc.s[m[1]:]) {
		return false // 3:30pm is the English rule's
	}
	if g[2] == "一" && g[1] == "" && g[4] == "" && g[5] == "" && g[6] == "" {
		return false
	}
	if g[1] == "" {
		prev, _ := utf8.DecodeLastRuneInString(sc.s[:m[0]])
		if prev >= '0' && prev <= '9' {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// hour24 is an hour as the 24-hour clock says it. With a period (afternoon, evening) an hour before
// noon is the later half of the day; with none, a small hour such as "3点" is the afternoon too,
// since nobody means 3 in the morning.
func hour24(h int, period string, guess bool) int {
	switch period {
	case "下午", "傍晚", "晚上":
		switch {
		case h == 12 && period == "晚上":
			return 24
		case h < 12:
			return h + 12
		}
	case "中午":
		if h < 11 {
			return h + 12
		}
	case "":
		if guess && h >= 1 && h <= 6 {
			return h + 12
		}
	}
	return h
}

// zhNum reads 9, 九, 十, 十二, 二十, 二十五.
func zhNum(s string) (int, bool) {
	if isDigits(s) {
		return num(s), true
	}
	digit := map[rune]int{'零': 0, '一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	r := []rune(s)
	val := func(x rune) (int, bool) { v, ok := digit[x]; return v, ok }
	switch {
	case len(r) == 1 && r[0] == '十':
		return 10, true
	case len(r) == 1:
		return val(r[0])
	case len(r) == 2 && r[0] == '十':
		v, ok := val(r[1])
		return 10 + v, ok
	case len(r) == 2 && r[1] == '十':
		v, ok := val(r[0])
		return v * 10, ok
	case len(r) == 3 && r[1] == '十':
		a, ok1 := val(r[0])
		b, ok2 := val(r[2])
		return a*10 + b, ok1 && ok2
	}
	return 0, false
}
