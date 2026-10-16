package data

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/handoff"
)

// DefaultLauncher opens agents in Terminal, from scripts kept under the data folder.
func DefaultLauncher(dataFolder string) handoff.Launcher {
	return handoff.Terminal{Folder: filepath.Join(dataFolder, "tmp", "handoff")}
}

// HandOff implements Store.
func (r *Real) HandOff(id int, prompt string) (HandOffResult, error) {
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
	// Record it before the agent starts, so that its first report finds the task waiting.
	handed, err := r.core.HandOff(r.ctx(), t.ID, agent.ID)
	if err != nil {
		return HandOffResult{}, err
	}
	return HandOffResult{
		Task: fromTask(*handed), Agent: agent.ID, AgentName: agent.Name,
		Launched: r.launch(t.ProjectID, agent, prompt),
	}, nil
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

// launch starts the agent in the project's folder with the prompt, and says whether it did. It
// does when a launcher is set, the folder exists, the agent is one that can be started this way
// and its command is found; a failure to open Terminal is not an error: the prompt goes to the
// clipboard instead.
func (r *Real) launch(projectID *int64, agent agentsetup.Agent, prompt string) bool {
	if r.opts.Launcher == nil || projectID == nil || !handoff.Launchable(agent.ID) {
		return false
	}
	p, err := r.core.GetProject(r.ctx(), *projectID)
	if err != nil || p == nil || p.RepoPath == nil {
		return false
	}
	if fi, err := os.Stat(*p.RepoPath); err != nil || !fi.IsDir() {
		return false
	}
	exe := ""
	for _, c := range agent.Commands {
		if exe, err = r.opts.Agents.Find(c); err == nil {
			break
		}
	}
	if err != nil {
		return false
	}
	argv, _ := handoff.Command(agent.ID, exe, prompt)
	err = r.opts.Launcher.Launch(r.ctx(), handoff.Request{
		Dir: *p.RepoPath, Argv: argv, PathDirs: r.opts.Agents.ToolDirs(exe),
	})
	return err == nil
}
