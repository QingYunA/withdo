package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/QingYunA/adhd-todo/app/data"
)

// Mine returns the Mine screen as JSON: app/data.Mine, built from the same snapshot and builders
// the desktop uses. Its keys are the Go field names:
//
//	{"FromAgents":[Item...],"Today":[Item...],"Later":[Item...],"MoreLater":0,
//	 "Done":[Item...],"MoreDone":0}
//
// An Item is data.Item with its data.Task fields promoted to the top level, plus Project, Kind,
// Summary, NextStep, Words, Since, Stale, Hours, Live and By. Kind is data.NeedKind as an integer
// (Blocked 0, Reminder 1, PRClosed 2, PRFailing 3, PRChanges 4, Review 5, Stale 6). Times are
// RFC 3339 with the device's offset. The user's replies are in Task's activity but are not the
// agent's latest word, so they do not appear in Item.Since or reset Stale.
func (c *Core) Mine() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	m := data.BuildMine(data.MineInput{Snapshot: snap, Now: c.nowTime()})
	return marshal(m)
}

// Agents returns the Agents screen as JSON: app/data.AgentsScreen, built like the desktop's.
//
//	{"Working":[AgentRow...],"Done":[AgentRow...],"MoreDone":0}
//
// An AgentRow is data.AgentRow: data.Item's fields plus "Subs":[Item...]. The replies the user
// wrote do not count as an agent's latest word.
func (c *Core) Agents() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	a := data.BuildAgents(data.MineInput{Snapshot: snap, Now: c.nowTime()})
	return marshal(a)
}

// Projects returns the sidebar's projects as JSON: a []data.Project of the projects that are not
// archived, by name. A Project has ID, Name, RepoSlug, RepoPath, CreatedAt, UpdatedAt, ArchivedAt
// and ParentID.
func (c *Core) Projects() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	return marshal(snap.Projects)
}

// projectPage is the JSON of Project(id).
type projectPage struct {
	Project    data.Project
	Tasks      []data.Task // newest first
	Milestones []data.Milestone
}

// Project returns one project's page as JSON: its data.Project, its tasks (newest first, subtasks
// included) and its milestones. An archived project is returned too; its tasks are then empty
// because the snapshot leaves them out of the lists, as it does on the desktop.
//
//	{"Project":{...},"Tasks":[Task...],"Milestones":[Milestone...]}
func (c *Core) Project(id int64) (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	var project *data.Project
	for i := range snap.Projects {
		if int64(snap.Projects[i].ID) == id {
			project = &snap.Projects[i]
			break
		}
	}
	if project == nil {
		for i := range snap.Archived {
			if int64(snap.Archived[i].ID) == id {
				project = &snap.Archived[i]
				break
			}
		}
	}
	if project == nil {
		return "", fmt.Errorf("project %d not found", id)
	}
	page := projectPage{Project: *project, Tasks: []data.Task{}, Milestones: []data.Milestone{}}
	for _, t := range snap.Tasks {
		if t.ProjectID != nil && int64(*t.ProjectID) == id {
			page.Tasks = append(page.Tasks, t)
		}
	}
	page.Milestones = append(page.Milestones, snap.Milestones[int(id)]...)
	return marshal(page)
}

// taskPage is the JSON of Task(id): the inspector's content.
type taskPage struct {
	Task     data.Task
	Subtasks []data.Task   // oldest first
	Reports  []data.Report // newest first, the activity timeline
}

// Task returns one task's inspector content as JSON: its properties, its subtasks (oldest first)
// and its reports (newest first, the activity). A report with Agent "you" is the user's reply.
//
//	{"Task":{...},"Subtasks":[Task...],"Reports":[Report...]}
//
// A Report has ID, TaskID, Agent, Status, Summary, NextStep, CreatedAt, Cwd and Links.
func (c *Core) Task(id int64) (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	var task *data.Task
	for i := range snap.Tasks {
		if int64(snap.Tasks[i].ID) == id {
			task = &snap.Tasks[i]
			break
		}
	}
	if task == nil {
		return "", fmt.Errorf("task %d not found", id)
	}
	page := taskPage{Task: *task, Subtasks: []data.Task{}, Reports: []data.Report{}}
	for _, t := range snap.Tasks {
		if t.ParentID != nil && int64(*t.ParentID) == id {
			page.Subtasks = append(page.Subtasks, t)
		}
	}
	sort.SliceStable(page.Subtasks, func(i, j int) bool { return page.Subtasks[i].ID < page.Subtasks[j].ID })
	page.Reports = append(page.Reports, snap.Reports[int(id)]...)
	return marshal(page)
}

// Search returns the tasks matching every word of query as JSON: a []data.Task, newest first,
// whatever their status, tasks of archived projects left out. It is data.Real.SearchTasks, which
// the desktop's search uses. A blank query returns [].
func (c *Core) Search(query string) (string, error) {
	tasks, err := c.real.SearchTasks(query, 0)
	if err != nil {
		return "", shortError(err)
	}
	if tasks == nil {
		tasks = []data.Task{}
	}
	return marshal(tasks)
}

// marshal is the one encoder of page data.
func marshal(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", errors.New("could not encode the page")
	}
	return string(raw), nil
}
