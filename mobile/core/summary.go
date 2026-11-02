package core

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
)

// The Summary page (ui-spec, Summary) is built from the desktop's data.BuildSummary over the same
// snapshot; the phone draws the scene and the ledger from this JSON.

// summaryWeekStart is the weekday the weeks of Summary start on. The desktop uses the system's
// first day of the week; the core has no locale, so the phone's weeks start on Monday.
const summaryWeekStart = time.Monday

// summaryTaskJSON is one finished task of the ledger.
type summaryTaskJSON struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Project  string `json:"project"`
	At       string `json:"at"` // when it was finished, RFC 3339
	Kind     string `json:"kind"`
	Merged   bool   `json:"merged"`
	PRNumber int    `json:"pr_number"`
	Said     string `json:"said"`
}

// summaryPersonJSON is one row of the ledger: you or an agent.
type summaryPersonJSON struct {
	Agent  string            `json:"agent"` // the agent as people write it, "" for you
	Name   string            `json:"name"`  // "you" or the same as agent
	You    bool              `json:"you"`
	Count  int               `json:"count"`
	Merged int               `json:"merged"`
	Tasks  []summaryTaskJSON `json:"tasks"`
}

// summaryGoodJSON is one good thrown in the scene.
type summaryGoodJSON struct {
	Actor   int    `json:"actor"` // index into ledger
	Kind    string `json:"kind"`
	Starred bool   `json:"starred"`
	Day     int    `json:"day"` // of a week, 0 to 6 from its first day; 0 for a day
	TaskID  int64  `json:"task_id"`
}

// summaryOpenJSON is what is still open.
type summaryOpenJSON struct {
	InProgress int `json:"in_progress"`
	InReview   int `json:"in_review"`
	Blocked    int `json:"blocked"`
}

// summaryJSON is the JSON of Summary.
type summaryJSON struct {
	Week      bool                `json:"week"`
	Back      int64               `json:"back"`
	From      string              `json:"from"`
	To        string              `json:"to"`
	Over      bool                `json:"over"`
	Total     int                 `json:"total"`
	Merged    int                 `json:"merged"`
	Ledger    []summaryPersonJSON `json:"ledger"`
	Open      summaryOpenJSON     `json:"open"`
	Paragraph string              `json:"paragraph"`
	Goods     []summaryGoodJSON   `json:"goods"`
}

// summaryOf builds the summary of the period, with the paragraph the assistant wrote for it (nil
// when none was) and the snapshot it was built from.
func (c *Core) summaryOf(week bool, back int64) (data.Summary, *data.StoredSummary, data.Snapshot, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return data.Summary{}, nil, snap, shortError(err)
	}
	back = max(back, 0)
	now := c.nowTime()
	period := data.DayPeriod(now, int(back))
	if week {
		period = data.WeekPeriod(now, int(back), summaryWeekStart)
	}
	stored, err := c.real.GetSummary(period)
	if err != nil {
		return data.Summary{}, nil, snap, shortError(err)
	}
	in := data.SummaryInput{Snapshot: snap, Now: now, Period: period, Agent: summaryAgentKey}
	if stored != nil {
		in.Kinds = stored.Kinds
	}
	return data.BuildSummary(in), stored, snap, nil
}

// summaryAgentKey says which agent a name is, as the avatars and names do: "claude-code" and
// "Claude Code" are one person.
func summaryAgentKey(name string) string {
	if name == "" {
		return ""
	}
	return data.AgentName(name)
}

// Summary returns the desktop's Summary page for a day or a week as JSON. back counts periods back
// from this one: 0 is today or this week (weeks start on Monday), 1 yesterday or last week.
//
//	{"week":false,"back":0,"from":"2026-10-20","to":"2026-10-20","over":false,
//	 "total":3,"merged":1,
//	 "ledger":[{"agent":"","name":"you","you":true,"count":1,"merged":0,
//	            "tasks":[{"id":4,"title":"…","project":"…","at":"…","kind":"…",
//	                      "merged":false,"pr_number":0,"said":""}]}, …],
//	 "open":{"in_progress":0,"in_review":0,"blocked":0},
//	 "paragraph":"",
//	 "goods":[{"actor":0,"kind":"…","starred":false,"day":0,"task_id":4}, …]}
//
// from and to are the first and last day (equal for a day), in the device's time zone; over says
// the period has ended. The ledger is you first, then the agents by how many they finished, as
// the desktop lists it; agent is "" for you and the agent's name as people write it otherwise,
// and each task is in the order it was finished. goods are the things the scene throws, one per
// finished task in that order: actor indexes ledger, kind is the good (the assistant's pick when
// a summary row exists, else the one the task's words give), starred says its pull request
// merged, and day is the weekday a week's good stacks on. paragraph is the assistant's text, ""
// when none has synced.
func (c *Core) Summary(week bool, back int64) (string, error) {
	sum, stored, _, err := c.summaryOf(week, back)
	if err != nil {
		return "", err
	}
	p := sum.Period
	out := summaryJSON{
		Week: week, Back: max(back, 0), From: p.FirstDate(), To: p.Day(0).Format("2006-01-02"),
		Over: p.Over(c.nowTime()), Total: sum.Total(), Merged: sum.Merged(),
		Ledger: make([]summaryPersonJSON, 0, len(sum.People)), Goods: []summaryGoodJSON{},
		Open: summaryOpenJSON{InProgress: sum.Open.InProgress, InReview: sum.Open.InReview, Blocked: sum.Open.Blocked},
	}
	if week {
		out.To = p.Day(6).Format("2006-01-02")
	}
	if stored != nil {
		out.Paragraph = strings.TrimSpace(stored.Paragraph)
	}
	who := map[int]int{}
	for i, person := range sum.People {
		row := summaryPersonJSON{
			Agent: person.Agent, Name: "you", You: person.You(), Count: len(person.Tasks), Merged: person.Merged,
			Tasks: make([]summaryTaskJSON, 0, len(person.Tasks)),
		}
		if !person.You() {
			row.Name = person.Agent
		}
		for _, t := range person.Tasks {
			who[t.ID] = i
			row.Tasks = append(row.Tasks, summaryTaskJSON{
				ID: int64(t.ID), Title: t.Title, Project: t.Project, At: t.At.Format(time.RFC3339Nano), Kind: t.Kind,
				Merged: t.Merged, PRNumber: t.PRNumber, Said: t.Said,
			})
		}
		out.Ledger = append(out.Ledger, row)
	}
	for _, t := range sum.Tasks() {
		good := summaryGoodJSON{Actor: who[t.ID], Kind: t.Kind, Starred: t.Merged, TaskID: int64(t.ID)}
		if week {
			good.Day = weekDayOf(p, t.At)
		}
		out.Goods = append(out.Goods, good)
	}
	return marshal(out)
}

// weekDayOf is the day of a week period, 0 to 6, that a moment is in.
func weekDayOf(p data.Period, at time.Time) int {
	for d := 6; d > 0; d-- {
		if !at.Before(p.Day(d)) {
			return d
		}
	}
	return 0
}

// SummaryMarkdown returns the Markdown of the same period Summary(week, back) describes: what the
// desktop's Export Markdown sheet shows. The text is "# Daily summary, 2026-10-20", the paragraph
// when one synced, a section for each person with a line per task (project, pull request, what
// the agent said and the links its reports attached), then what is still open. lang is the app's
// language, "en" or "zh-Hans" (anything starting with "zh" is Chinese); this is the one place the
// core words something, as the desktop's export does.
func (c *Core) SummaryMarkdown(week bool, back int64, lang string) (string, error) {
	sum, stored, snap, err := c.summaryOf(week, back)
	if err != nil {
		return "", err
	}
	locale := ""
	if strings.HasPrefix(lang, "zh") {
		locale = "zh-Hans"
	}
	paragraph := ""
	if stored != nil {
		paragraph = stored.Paragraph
	}
	return sum.Markdown(locales.In(locale), paragraph, func(id int) []data.Link {
		return data.TaskLinks(snap.Reports[id])
	}), nil
}
