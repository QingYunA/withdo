package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The Summary page (ui-spec, Summary): the day or the week as a shipping scene, the assistant's
// paragraph, and the ledger. This file is what the page knows: which period it shows, the summary of
// it, and the scene that plays it. summary_page.go draws the page, summary_text.go is the paragraph,
// summary_ledger.go the ledger and summary_export.go the Markdown.

// summaryState is the Summary page: its period, the scene being played, the paragraphs written,
// the ledger rows that are open and the export sheet.
type summaryState struct {
	week bool
	back int // periods back from this one; 0 is today or this week

	// playing is what the scene is showing: the period, and how many times it was replayed. start is
	// when it began, and sky the sky it began under: it does not change while it plays.
	playing string
	replays int
	start   time.Time
	sky     theme.SkyKind
	sceneW  float32 // the scene's width as last drawn, for the pointer
	scene   ui.Rect // where it was drawn, for the tests that point at it
	// freeze, when set, is the time of the scene in place of the clock's: the screenshots' still frames.
	freeze *time.Duration

	paras  map[string]*paragraph
	open   map[string]bool // ledger rows that show their tasks, by period and person
	export exportSheet

	cache summaryCache
}

// summaryCache keeps the built summary while nothing it is built from changes.
type summaryCache struct {
	rev   int
	key   string
	kinds *data.StoredSummary
	start time.Time // the scene it was built for: a replay builds again, under the sky of the moment
	view  *summaryView
}

// summaryView is the period as the page shows it: what was finished, and the plan of its scene.
type summaryView struct {
	period data.Period
	key    string
	sum    data.Summary
	tasks  []data.SummaryTask    // in the order they were finished
	byDay  [7][]data.SummaryTask // of a week, by day
	day    *theme.DayPlan
	week   *theme.WeekPlan
	// who is the index, among the people, of the one who finished each task, by task id.
	who map[int]int
}

// periodOf is the period the page shows at now.
func (m *Main) periodOf(now time.Time) data.Period {
	if m.sum.week {
		return data.WeekPeriod(now, m.sum.back, m.weekStart)
	}
	return data.DayPeriod(now, m.sum.back)
}

// summaryKey names a period for what is kept per period.
func summaryKey(p data.Period) string { return p.Kind() + " " + p.FirstDate() }

// summaryFor is the view of the period on screen, which starts the scene when it is not playing:
// built again only when the data, the period, the paragraph's kinds or the scene changed.
func (m *Main) summaryFor(now time.Time) *summaryView {
	p := m.periodOf(now)
	key := summaryKey(p)
	m.beginScene(now, key)
	para := m.paragraphOf(p)
	ca := &m.sum.cache
	if ca.view != nil && ca.rev == m.app.rev && ca.key == key && ca.kinds == para.stored && ca.start.Equal(m.sum.start) {
		return ca.view
	}
	var kinds map[int]string
	if para.stored != nil {
		kinds = para.stored.Kinds
	}
	sum := data.BuildSummary(data.SummaryInput{
		Now: now, Period: p, Tasks: m.app.snap.tasks,
		Latest: m.app.latestReport, Event: m.app.githubReport, ProjectName: m.app.projectName,
		Kinds: kinds, Agent: agentKey,
	})
	v := &summaryView{period: p, key: key, sum: sum, tasks: sum.Tasks(), who: map[int]int{}}
	for i, person := range sum.People {
		for _, t := range person.Tasks {
			v.who[t.ID] = i
		}
	}
	if p.Week {
		for _, t := range v.tasks {
			v.byDay[weekDayOf(p, t.At)] = append(v.byDay[weekDayOf(p, t.At)], t)
		}
		v.week = theme.NewWeekPlan(weekScene(v))
	} else {
		v.day = theme.NewDayPlan(dayScene(v, m.sum.sky))
	}
	*ca = summaryCache{rev: m.app.rev, key: key, kinds: para.stored, start: m.sum.start, view: v}
	return v
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

// actorOfPerson is how a person of the ledger is drawn in the scene.
func actorOfPerson(p data.SummaryPerson) theme.Actor {
	if p.You() {
		return theme.YouActor
	}
	return theme.ActorOf(p.Agent)
}

// dayScene is the day as the scene plays it: everyone who finished something, and each task in the
// order it was finished, thrown by the one who finished it.
func dayScene(v *summaryView, sky theme.SkyKind) theme.DayScene {
	s := theme.DayScene{Sky: sky}
	for _, p := range v.sum.People {
		s.Actors = append(s.Actors, actorOfPerson(p))
	}
	for _, t := range v.tasks {
		s.Throws = append(s.Throws, theme.DayThrow{Actor: v.who[t.ID], Kind: t.Kind, Starred: t.Merged})
	}
	return s
}

// weekScene is the week as the scene stacks it: the goods of each day, and whoever finished most.
func weekScene(v *summaryView) theme.WeekScene {
	var s theme.WeekScene
	for d, tasks := range v.byDay {
		for _, t := range tasks {
			s.Days[d] = append(s.Days[d], t.Kind)
		}
	}
	lead := -1
	for i, p := range v.sum.People {
		if lead < 0 || len(p.Tasks) > len(v.sum.People[lead].Tasks) {
			lead = i
		}
	}
	if lead >= 0 {
		a := actorOfPerson(v.sum.People[lead])
		s.Lead = &a
	}
	return s
}

// agentKey says which agent a name is, as the avatars and the names do: "claude-code" and
// "Claude Code" are one person.
func agentKey(name string) string {
	if name == "" {
		return ""
	}
	return agentName(name)
}

// beginScene starts the scene over when the period it plays is not the one on screen, which is the
// page opening, the period changing or a replay. The sky is the one of the moment it starts: day,
// dusk or night by the clock for today, dusk for a day that is over and night for a week.
func (m *Main) beginScene(now time.Time, key string) {
	st := &m.sum
	if st.playing == key {
		return
	}
	st.playing, st.start = key, now
	switch {
	case m.sum.week:
		st.sky = theme.SkyNight
	case m.sum.back > 0:
		st.sky = theme.SkyDusk
	default:
		st.sky = theme.SkyAt(now)
	}
}

// replay plays the scene again.
func (m *Main) replayScene() {
	m.sum.replays++
	m.sum.playing = ""
}

// leaveSummary forgets what the scene was playing when another page opens, so the next visit plays it.
func (m *Main) leaveSummary() { m.sum.playing = "" }

// sceneTime is how long the scene has been playing, and whether it is drawn calm: with Reduce Motion
// it is the last frame at once.
func (m *Main) sceneTime(c *ui.Context, v *summaryView, now time.Time) (time.Duration, bool) {
	calm := c.Preferences().ReduceMotion
	switch {
	case m.sum.freeze != nil:
		return *m.sum.freeze, calm
	case calm:
		return v.calmTime(), true
	}
	return max(0, now.Sub(m.sum.start)), false
}

// calmTime is the time of the last frame of the scene.
func (v *summaryView) calmTime() time.Duration {
	if v.week != nil {
		return v.week.Calm()
	}
	return v.day.Calm()
}

// sceneEnd is when the scene has played and the ledger starts to light up.
func (v *summaryView) sceneEnd() time.Duration {
	if v.week != nil {
		return v.week.End()
	}
	return v.day.End()
}
