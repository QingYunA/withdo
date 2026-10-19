package data

import (
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

var _ AgentSource = (*Real)(nil)

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
