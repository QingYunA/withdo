package data

import (
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

// markEvents sets the watermark to the newest report: what is there already is not told.
func (r *Real) markEvents() {
	if id, err := r.core.LatestReportID(r.ctx()); err == nil {
		r.eventMark = id
	}
}

// deliverEvents tells the subscribers of the reports filed since the last call. It runs from the
// watcher, one change at a time. A report whose id is at or below the watermark was already told; if
// the newest report is below it, rows were deleted and ids may be reused, so the watermark steps back.
// A failed read leaves the watermark, and the next change reads again.
func (r *Real) deliverEvents() {
	latest, err := r.core.LatestReportID(r.ctx())
	if err != nil {
		return
	}
	r.eventMark = min(r.eventMark, latest)
	if latest == r.eventMark {
		return
	}
	rs, err := r.core.ReportsAfter(r.ctx(), r.eventMark)
	if err != nil {
		return
	}
	r.eventMark = latest
	r.mu.Lock()
	fns := make([]func(Event), 0, len(r.evSubs))
	for _, fn := range r.evSubs {
		fns = append(fns, fn)
	}
	r.mu.Unlock()
	for _, rep := range rs {
		if e, ok := eventOf(rep); ok {
			for _, fn := range fns {
				fn(e)
			}
		}
	}
}
