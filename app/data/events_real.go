package data

import (
	"fmt"
	"maps"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// eventOf says whether a report is an event, and which. An agent's report that moves its task to In
// review or Blocked is one; the poller's reports of a failing check or a merge are, told apart by
// their kind rather than their text. Everything else, the core's housekeeping included, is not.
func eventOf(r domain.ReportEvent) (Event, bool) {
	e := Event{TaskID: int(r.TaskID), Title: r.TaskTitle, ProjectName: str(r.ProjectName)}
	if r.PRNumber != nil {
		e.PRNumber = int(*r.PRNumber)
	}
	switch {
	case r.Agent == domain.GitHubAgent:
		return githubEvent(e, r.Kind)
	case r.Agent == "" || r.Agent == domain.HousekeepingAgent || r.Status == nil:
		return Event{}, false
	case *r.Status == domain.StatusReview:
		e.Kind, e.Detail = EventHandedBack, r.Summary
	case *r.Status == domain.StatusBlocked:
		e.Kind, e.Detail = EventStuck, firstText(str(r.NextStep), r.Summary)
	default:
		return Event{}, false
	}
	e.Agent = r.Agent
	return e, true
}

func githubEvent(e Event, kind domain.ReportKind) (Event, bool) {
	switch kind {
	case domain.KindChecksFailing:
		e.Kind = EventChecksFailing
	case domain.KindPRMerged:
		e.Kind = EventMerged
	default:
		return Event{}, false
	}
	return e, true
}

func firstText(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// SubscribeEvents implements Store.
func (r *Real) SubscribeEvents(fn func(Event)) func() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextSub++
	id := r.nextSub
	r.evSubs[id] = fn
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.evSubs, id)
	}
}

// eventMark is how far the reports were told: the time of the newest, and which of the reports at
// exactly that time were. Ids say nothing, as a deleted report's id is reused by the next one.
type eventMark struct {
	at   string
	seen map[string]bool
}

// reportKey tells the reports at one time apart. Both ids can be reused after a delete, so the
// words of the report and its task go in too.
func reportKey(r domain.ReportEvent) string {
	return fmt.Sprintf("%d/%d/%s/%s", r.ID, r.TaskID, r.TaskTitle, r.Summary)
}

// advance is the mark after the reports rs (oldest first, none older than m.at) were read.
func (m eventMark) advance(rs []domain.ReportEvent) eventMark {
	if len(rs) == 0 {
		return m
	}
	out := eventMark{at: rs[len(rs)-1].CreatedAt, seen: map[string]bool{}}
	if out.at == m.at {
		maps.Copy(out.seen, m.seen) // more reports at the time already told
	}
	for _, r := range rs {
		if r.CreatedAt == out.at {
			out.seen[reportKey(r)] = true
		}
	}
	return out
}

// markEvents sets the mark to the newest report: what is there already is not told.
func (r *Real) markEvents() {
	at, err := r.core.LatestReportAt(r.ctx())
	if err != nil {
		return
	}
	rs, err := r.core.ReportsSince(r.ctx(), at)
	if err != nil {
		return
	}
	r.eventMark = eventMark{at: at, seen: map[string]bool{}}.advance(rs)
}

// deliverEvents tells the subscribers of the reports filed since the last call. It runs from the
// watcher, one change at a time: it reads the reports at or after the mark's time and skips those at
// that very time that were told. A failed read leaves the mark, and the next change reads again.
func (r *Real) deliverEvents() {
	rs, err := r.core.ReportsSince(r.ctx(), r.eventMark.at)
	if err != nil {
		return
	}
	var fresh []domain.ReportEvent
	for _, rep := range rs {
		if rep.CreatedAt != r.eventMark.at || !r.eventMark.seen[reportKey(rep)] {
			fresh = append(fresh, rep)
		}
	}
	if len(fresh) == 0 {
		return
	}
	r.eventMark = r.eventMark.advance(fresh)
	r.mu.Lock()
	fns := make([]func(Event), 0, len(r.evSubs))
	for _, fn := range r.evSubs {
		fns = append(fns, fn)
	}
	r.mu.Unlock()
	for _, rep := range fresh {
		if e, ok := eventOf(rep); ok {
			for _, fn := range fns {
				fn(e)
			}
		}
	}
}
