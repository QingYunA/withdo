package data

import (
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// AgentActivity is what one agent has reported: when it last did, and how many reports came
// since a moment (Settings asks for "today").
type AgentActivity struct {
	Agent string // the name it reported under, such as "claude-code"
	Last  time.Time
	Since int
}

// AgentSource is the part of a Store that knows which agents reported. A Store may leave it out;
// Settings then lists no agents.
type AgentSource interface {
	// AgentActivity lists every agent that ever reported, the latest first, counting the reports
	// at or after since. The GitHub poller is not an agent and is left out.
	AgentActivity(since time.Time) ([]AgentActivity, error)
}

var (
	_ AgentSource = (*Real)(nil)
	_ AgentSource = (*Fake)(nil)
)

// AgentActivity implements AgentSource.
func (r *Real) AgentActivity(since time.Time) ([]AgentActivity, error) {
	rows, err := r.core.AgentActivity(r.ctx(), domain.FormatTime(since))
	if err != nil {
		return nil, err
	}
	out := make([]AgentActivity, len(rows))
	for i, a := range rows {
		out[i] = AgentActivity{Agent: a.Agent, Last: parseTime(a.Last), Since: a.Since}
	}
	return out, nil
}

// AgentActivity implements AgentSource.
func (f *Fake) AgentActivity(since time.Time) ([]AgentActivity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	byAgent := map[string]*AgentActivity{}
	var out []*AgentActivity
	for _, r := range f.reports {
		if r.Agent == GitHubAgent || r.Agent == "" {
			continue
		}
		a := byAgent[r.Agent]
		if a == nil {
			a = &AgentActivity{Agent: r.Agent}
			byAgent[r.Agent] = a
			out = append(out, a)
		}
		if r.CreatedAt.After(a.Last) {
			a.Last = r.CreatedAt
		}
		if !r.CreatedAt.Before(since) {
			a.Since++
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Last.After(out[j].Last) })
	res := make([]AgentActivity, len(out))
	for i, a := range out {
		res[i] = *a
	}
	return res, nil
}

// ReportAs files a report by an agent against a task, for demos and tests of Settings > Agents.
func (f *Fake) ReportAs(taskID int, agent, summary string, at time.Time) {
	f.mu.Lock()
	f.reports = append(f.reports, Report{ID: len(f.reports) + 1, TaskID: taskID, Agent: agent, Summary: summary, CreatedAt: at})
	f.picked(taskID)
	f.mu.Unlock()
	f.notify()
}
