package core

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
)

// defaultCompleted is how many finished tasks Completed returns when limit is not positive: the
// desktop's page size.
const defaultCompleted = 100

// completedTask is one row of Completed: the finished task with who finished it. Like data.Item it
// carries the data.Task fields at the top level.
type completedTask struct {
	data.Task
	// Project is the name of the task's project, "" for the Inbox.
	Project string
	// By is the agent that finished it as it reported, "" when it was you (data.CompletedBy);
	// ByName is the same as words: the agent's name, or "you".
	By     string
	ByName string
	// Summary is what the agent's closing report said, "" when it was you or nothing was said.
	Summary string
}

// completedDay is the tasks finished on one local day.
type completedDay struct {
	Day   string // 2026-10-20, in the device's time zone
	Tasks []completedTask
}

// completedPageJSON is the JSON of Completed.
type completedPageJSON struct {
	Days []completedDay
	More bool
}

// Completed returns the desktop's Completed page as JSON: the done and cancelled tasks, newest
// completion first, in a group for each local day (the desktop's data.LogbookOf and page.Build).
// projectID 0 is everything; a project's id narrows it to that project; agents narrows it to the
// tasks agents wrote. limit is how many tasks to load (100 when it is not positive): More says
// older ones exist, and asking again with a larger limit loads them. A finished subtask is left
// out, as it is part of its parent's story.
//
//	{"Days":[{"Day":"2026-10-20","Tasks":[Row...]}],"More":false}
//
// A Row is the task's fields (data.Task, at the top level as in Mine's Items) plus Project, By,
// ByName and Summary: By is the agent that finished the task as it reported ("" when it was you),
// ByName is "you" or the agent's name, and Summary is what its closing report said.
func (c *Core) Completed(projectID int64, agents bool, limit int64) (string, error) {
	if limit <= 0 {
		limit = defaultCompleted
	}
	tasks, err := c.real.LogbookOf(int(projectID), agents, int(limit)+1)
	if err != nil {
		return "", shortError(err)
	}
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	out := completedPageJSON{Days: []completedDay{}}
	if int64(len(tasks)) > limit {
		out.More, tasks = true, tasks[:limit]
	}
	var top []data.Task
	for _, t := range tasks {
		if !t.IsSubtask() {
			top = append(top, t)
		}
	}
	pg := page.Build(page.In{Screen: page.Completed, Done: top, Now: c.nowTime()})
	for _, section := range pg.Sections {
		day := completedDay{Day: section.Day.Format("2006-01-02"), Tasks: make([]completedTask, 0, len(section.Tasks))}
		for _, t := range section.Tasks {
			by, said := data.CompletedBy(t, snap.LatestReport(t.ID), snap.GitHubReport(t.ID))
			name := "you"
			if by != "" {
				name = data.AgentName(by)
			}
			day.Tasks = append(day.Tasks, completedTask{
				Task: t, Project: snap.ProjectName(t.ProjectID), By: by, ByName: name, Summary: said,
			})
		}
		out.Days = append(out.Days, day)
	}
	return marshal(out)
}
