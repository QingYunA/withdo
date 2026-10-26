package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/remind"
)

// Mine returns the Mine screen as JSON: app/data.Mine, built from the same snapshot and builders
// the desktop uses. Its keys are the Go field names:
//
//	{"FromAgents":[Item...],"Today":[Item...],"Later":[Item...],"MoreLater":0,
//	 "Done":[Item...],"MoreDone":0}
//
// The desktop's v34 Home splits Later into Next (dated within a week) and the No date fold; the
// phone keeps its one Later list: the wire holds both (Next, NoDate) and Later is their sum, so the
// Android app is unchanged and no task is out of its reach.
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
	m := data.BuildMine(data.MineInput{Snapshot: snap, Now: c.nowTime(), NoDateOpen: true})
	return marshal(mobileMine{Mine: mineWithLists(m), Later: append(append([]data.Item{}, m.Next...), m.NoDate...)})
}

// mobileMine is data.Mine with the phone's Later list.
type mobileMine struct {
	data.Mine
	Later     []data.Item
	MoreLater int
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
	return marshal(agentsWithLists(a))
}

// Projects returns the sidebar's projects as JSON: a []data.Project of the projects that are not
// archived, by name. A Project has ID, Name, RepoSlug, RepoPath, CreatedAt, UpdatedAt, ArchivedAt
// and ParentID.
func (c *Core) Projects() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	return marshal(emptyIfNil(snap.Projects))
}

// Categories returns the snapshot's categories as JSON: a []data.Category in the user's order.
// A Category has ID, Builtin ("work", "life" or ""), Name, Icon (a 12 x 12 pixmap.Picture),
// Color (one of CategoryColors) and Position.
func (c *Core) Categories() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	return marshal(emptyIfNil(snap.Categories))
}

// remindPresetJSON is one choice of the Remind me page: its kind and the moment it means in Unix
// milliseconds.
type remindPresetJSON struct {
	Kind string `json:"kind"`
	At   int64  `json:"at"`
}

// RemindPresets returns the task menu's ready-made reminders for now as JSON:
//
//	[{"kind":"later","at":1792480800000}, …]
//
// The kinds are later, tomorrow, monday and weekend. The moments follow the desktop's menu: Later
// today is 18:00 while it is over half an hour away, Tomorrow is 9:00, Next Monday is 9:00 and
// This weekend is Saturday 10:00. A choice that would repeat another (Monday on a Sunday, the
// weekend on a Sunday, the evening under half an hour away) is left out.
func (c *Core) RemindPresets() (string, error) {
	presets := remind.Presets(c.nowTime())
	out := make([]remindPresetJSON, 0, len(presets))
	for _, p := range presets {
		out = append(out, remindPresetJSON{Kind: p.Kind.String(), At: p.At.UnixMilli()})
	}
	return marshal(out)
}

// projectPage is the JSON of Project(id).
type projectPage struct {
	Project    data.Project
	Tasks      []data.Task // newest first
	Milestones []data.Milestone
	Agents     map[string]string // task id → the agent of its latest report, for the row's avatar
}

// Project returns one project's page as JSON: its data.Project, its tasks (newest first, subtasks
// included) and its milestones. An archived project is returned too; its tasks are then empty
// because the snapshot leaves them out of the lists, as it does on the desktop.
//
//	{"Project":{...},"Tasks":[Task...],"Milestones":[Milestone...],"Agents":{"3":"claude-code"}}
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
	page := projectPage{Project: *project, Tasks: []data.Task{}, Milestones: []data.Milestone{}, Agents: map[string]string{}}
	for _, t := range snap.Tasks {
		if t.ProjectID != nil && int64(*t.ProjectID) == id {
			page.Tasks = append(page.Tasks, t)
			if r := snap.LatestReport(t.ID); r != nil {
				page.Agents[strconv.Itoa(t.ID)] = r.Agent
			}
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

// mineWithLists and agentsWithLists make every list of the screen an array even when empty (see
// emptyIfNil). A list added to data.Mine or data.AgentsScreen is caught by
// TestJSONOfANewDatabaseHasNoNull until it is added here.
func mineWithLists(m data.Mine) data.Mine {
	m.FromAgents, m.Today, m.Next = emptyIfNil(m.FromAgents), emptyIfNil(m.Today), emptyIfNil(m.Next)
	m.NoDate, m.Done = emptyIfNil(m.NoDate), emptyIfNil(m.Done)
	return m
}

func agentsWithLists(a data.AgentsScreen) data.AgentsScreen {
	a.Working, a.Queued = emptyIfNil(a.Working), emptyIfNil(a.Queued)
	a.Backlog, a.Done = emptyIfNil(a.Backlog), emptyIfNil(a.Done)
	return a
}

// emptyIfNil keeps an empty list a JSON array: encoding/json writes a nil slice as null, which the
// app's list decoders refuse, so a phone with no projects could not open a task.
func emptyIfNil[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}

// marshal is the one encoder of page data.
func marshal(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", errors.New("could not encode the page")
	}
	return string(raw), nil
}
