// Package quickadd reads a typed sentence into task drafts with plain rules, for the input when no
// model is connected: dates and times, "remind me", "by Friday", #category, "!" and a few explicit
// separators. It is conservative on purpose: words it does not surely understand stay in the
// title. Chinese and English are both read. The package holds no wording and no clock: the caller
// passes "now" and words the drafts itself.
package quickadd

import (
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	// DueHour is when a day without a time is a deadline: the end of the working day.
	DueHour = 18
	// RemindHour is when a day without a time is a reminder: the morning.
	RemindHour = 9
)

// Draft is one task the sentence asks for.
type Draft struct {
	Title     string
	Due       *time.Time
	RemindAt  *time.Time
	Category  string // a name from the list given to Parse, as spelled there; "" for none
	Important bool
	// Spans are the words that were read, as byte ranges of the text given to Parse, in order and
	// not touching: a screen can mark them in what was typed. Empty when nothing was read.
	Spans []Span
}

// Span is a byte range of a text: Start is in the range, End is not.
type Span struct{ Start, End int }

// separators are the explicit marks between two tasks. A bare "and" is not one: "salt and pepper"
// is one thing, and neither is a line break, which the composer keeps inside one title.
var separators = regexp.MustCompile(`(?i)[;；]+|[，,]\s*还有|[，,]?\s*然后|,\s*and also\s|,\s*then\s`)

// Parse reads text into drafts, one per task the sentence names. now fixes "tomorrow" and every
// time without a date; categories are the names a #tag may match, case-insensitively. A sentence
// with nothing in it gives no drafts.
func Parse(text string, now time.Time, categories []string) []Draft {
	var out []Draft
	last := 0
	cut := func(part string, base int) {
		if d, ok := parseOne(part, base, now, categories); ok {
			out = append(out, d)
		}
	}
	for _, m := range separators.FindAllStringIndex(text, -1) {
		cut(text[last:m[0]], last)
		last = m[1]
	}
	cut(text[last:], last)
	return out
}

var remindWords = regexp.MustCompile(`(?i)\bremind me(?:\s+to)?\b|提醒我`)

// parseOne reads one task; part starts at byte base of the text. A part that holds nothing but
// recognized words keeps its own text as the title and sets no field: it is better to add what was
// typed than an empty task.
func parseOne(part string, base int, now time.Time, categories []string) (Draft, bool) {
	orig := strings.TrimSpace(part)
	if orig == "" {
		return Draft{}, false
	}
	sc := newScan(orig, base+len(part)-len(strings.TrimLeftFunc(part, unicode.IsSpace)))
	var d Draft
	d.Category = sc.category(categories)
	remind := false
	if m := sc.find(remindWords, nil); m != nil {
		remind = true
		sc.cut(m[0], m[1])
	}
	day, hasDay := sc.date(now)
	tm, hasTime := sc.clock(day.period)
	if at, ok := moment(now, day, hasDay, tm, hasTime, remind && !(day.deadline || tm.deadline)); ok {
		if remind && !(day.deadline || tm.deadline) {
			d.RemindAt = &at
		} else {
			d.Due = &at
		}
	}
	title, important := takeImportant(sc.s)
	d.Important = important
	d.Title = tidy(title)
	if d.Title == "" {
		return Draft{Title: orig}, true
	}
	rest := len(title)
	for rest < len(sc.s) && sc.s[rest] == ' ' {
		rest++
	}
	sc.mark(rest, len(sc.s)) // the "!" or "重要" taken off the end
	d.Spans = mergeSpans(sc.spans)
	return d, true
}

// mergeSpans puts spans in order and joins those that overlap or touch.
func mergeSpans(spans []Span) []Span {
	sort.Slice(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
	var out []Span
	for _, sp := range spans {
		if n := len(out); n > 0 && sp.Start <= out[n-1].End {
			out[n-1].End = max(out[n-1].End, sp.End)
			continue
		}
		out = append(out, sp)
	}
	return out
}

// moment is the instant the words name. A day with no time is the end of the day for a deadline
// and the morning for a reminder; a time with no day is the next time it is on the clock.
func moment(now time.Time, day dayWord, hasDay bool, tm clockWord, hasTime, remind bool) (time.Time, bool) {
	loc := now.Location()
	switch {
	case hasDay && hasTime:
		return time.Date(day.y, day.m, day.d, tm.h, tm.min, 0, 0, loc), true
	case hasDay && day.dflt != nil:
		return time.Date(day.y, day.m, day.d, day.dflt.h, day.dflt.min, 0, 0, loc), true
	case hasDay:
		hour := DueHour
		if remind {
			hour = RemindHour
		}
		return time.Date(day.y, day.m, day.d, hour, 0, 0, 0, loc), true
	case hasTime:
		y, m, d := now.Date()
		at := time.Date(y, m, d, tm.h, tm.min, 0, 0, loc)
		if !at.After(now) {
			at = time.Date(y, m, d+1, tm.h, tm.min, 0, 0, loc)
		}
		return at, true
	}
	return time.Time{}, false
}

// takeImportant strips the "!", "！" or "重要" the text ends with.
func takeImportant(s string) (string, bool) {
	important := false
	for {
		t := strings.TrimRightFunc(s, unicode.IsSpace)
		switch {
		case strings.HasSuffix(t, "!") || strings.HasSuffix(t, "！"):
			t = strings.TrimRight(t, "!！")
		case strings.HasSuffix(t, "重要"):
			t = strings.TrimSuffix(t, "重要")
		default:
			return s, important
		}
		s, important = t, true
	}
}

// tidy is the title after the recognized words were cut out: single spaces, no punctuation or
// dangling connecting word at either end.
func tidy(s string) string {
	for {
		t := strings.Join(strings.Fields(s), " ")
		t = strings.TrimFunc(t, func(r rune) bool { return unicode.IsSpace(r) || strings.ContainsRune("，,。.;；:：、-—~", r) })
		t = strings.TrimPrefix(t, "的")
		words := strings.Fields(t)
		if len(words) > 1 {
			switch strings.ToLower(words[0]) {
			case "to", "and", "then":
				words = words[1:]
			}
		}
		if len(words) > 1 {
			switch strings.ToLower(words[len(words)-1]) {
			case "on", "at", "by", "before", "due", "and", "then":
				words = words[:len(words)-1]
			}
		}
		t = strings.Join(words, " ")
		if t == s {
			return t
		}
		s = t
	}
}
