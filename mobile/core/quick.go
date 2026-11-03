package core

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/internal/quickadd"
)

// quickDraftJSON is one task read out of the composer's text.
type quickDraftJSON struct {
	Title      string   `json:"title"`
	RemindAt   string   `json:"remind_at"` // RFC 3339, "" for none
	DueAt      string   `json:"due_at"`    // RFC 3339, "" for none
	CategoryID int64    `json:"category_id"`
	Category   string   `json:"category"` // the #name as spelled in the category list, "" for none
	Important  bool     `json:"important"`
	Ranges     [][2]int `json:"ranges"` // [start, end) in UTF-16 units of the text, the words read
}

// quickJSON is the JSON of ParseQuick.
type quickJSON struct {
	Drafts []quickDraftJSON `json:"drafts"`
}

// ParseQuick reads what the composer holds without a model: the desktop's rule parser
// (internal/quickadd) with the user's category names, as the desktop's composer runs it, and the
// device's clock. It returns what the phone needs to mark the words that were read and to show what
// will be stored:
//
//	{"drafts":[{"title":"交房租","remind_at":"2026-10-21T09:00:00+08:00","due_at":"",
//	            "category_id":0,"category":"","important":false,"ranges":[[0,2]]}]}
//
// There is one draft for a plain sentence and one for each task when the text holds explicit
// separators (";", "然后"); no draft for blank text. title is the text with the words read taken
// out, remind_at and due_at are when it was read to remind or to be due (usually one of them),
// category_id is the id of a "#name" that names a category (0 for none), important says a closing
// "!" or "重要" was read (the composer stores priority P1), and ranges are the words that were read
// as [start, end) offsets into the text in UTF-16 units, the offsets of a Kotlin String, so
// "明天9点交房租" gives [[0,5]]. A draft with empty ranges read nothing: it is the text as typed.
// CreateTask takes title, remind_at, due_at, category_id and priority "P1" to store a draft.
func (c *Core) ParseQuick(text string) (string, error) {
	names, ids, err := c.quickCategories()
	if err != nil {
		return "", shortError(err)
	}
	out := quickJSON{Drafts: []quickDraftJSON{}}
	for _, d := range quickadd.Parse(text, c.nowTime(), names) {
		draft := quickDraftJSON{
			Title: d.Title, RemindAt: timeText(d.RemindAt), DueAt: timeText(d.Due), Category: d.Category,
			CategoryID: ids[strings.ToLower(d.Category)], Important: d.Important, Ranges: [][2]int{},
		}
		for _, sp := range d.Spans {
			draft.Ranges = append(draft.Ranges, [2]int{utf16Offset(text, sp.Start), utf16Offset(text, sp.End)})
		}
		out.Drafts = append(out.Drafts, draft)
	}
	return marshal(out)
}

// quickCategories are the names a #tag may name, as the desktop's composer offers them: the
// categories as the user wrote them, and Work and Life in both languages of the app. ids finds the
// category by its lowercase name.
func (c *Core) quickCategories() (names []string, ids map[string]int64, err error) {
	cats, err := c.dom.ListCategories(context.Background())
	if err != nil {
		return nil, nil, err
	}
	ids = map[string]int64{}
	add := func(name string, id int64) {
		if key := strings.ToLower(name); name != "" && ids[key] == 0 {
			names, ids[key] = append(names, name), id
		}
	}
	for _, cat := range cats {
		switch cat.Builtin {
		case data.BuiltinWork:
			add(locales.In("zh-Hans").CategoryWork(), cat.ID)
			add(locales.In("").CategoryWork(), cat.ID)
		case data.BuiltinLife:
			add(locales.In("zh-Hans").CategoryLife(), cat.ID)
			add(locales.In("").CategoryLife(), cat.ID)
		}
		add(cat.Name, cat.ID)
	}
	return names, ids, nil
}

// utf16Offset is the offset in UTF-16 units of byte offset at in s.
func utf16Offset(s string, at int) int {
	n := 0
	for _, r := range s[:at] {
		n++
		if r >= 0x10000 && r != utf8.RuneError {
			n++
		}
	}
	return n
}

// timeText is a time as RFC 3339, "" for none.
func timeText(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
