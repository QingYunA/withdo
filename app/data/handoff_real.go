package data

import (
	"fmt"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/domain"
)

// HandOff implements Store.
func (r *Real) HandOff(id int) (HandOffResult, error) {
	t, err := r.core.GetTask(r.ctx(), int64(id))
	if err != nil {
		return HandOffResult{}, err
	}
	if t == nil {
		return HandOffResult{}, fmt.Errorf("task %d not found", id)
	}
	agent, err := r.pickAgent(t.ProjectID)
	if err != nil {
		return HandOffResult{}, err
	}
	handed, err := r.core.HandOff(r.ctx(), t.ID, agent.ID)
	if err != nil {
		return HandOffResult{}, err
	}
	return HandOffResult{Task: fromTask(*handed), Agent: agent.ID, AgentName: agent.Name}, nil
}

// TakeBack implements Store.
func (r *Real) TakeBack(id int) (Task, error) {
	t, err := r.core.TakeBack(r.ctx(), int64(id))
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}

// RestoreHandOff implements Store.
func (r *Real) RestoreHandOff(id int, agent string, at *time.Time, source Source) (Task, error) {
	t, err := r.core.RestoreHandOff(r.ctx(), int64(id), agent, formatTimePtr(at), domain.Source(source))
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}

// pickAgent chooses who gets a task of the project: the agent that last reported in it, else the
// first one found on this Mac, else Claude Code.
func (r *Real) pickAgent(projectID *int64) (agentsetup.Agent, error) {
	if projectID != nil {
		name, err := r.core.LastReportAgent(r.ctx(), *projectID)
		if err != nil {
			return agentsetup.Agent{}, err
		}
		if a, ok := agentsetup.Match(name); ok {
			return a, nil
		}
	}
	for _, a := range agentsetup.Agents {
		if r.opts.Agents.Detect(a).Found {
			return a, nil
		}
	}
	a, _ := agentsetup.ByID("claude")
	return a, nil
}
