// Package views draws the adhd-todo window with mygo native UI: the sidebar, the Now screen, the
// project lists and board, the inbox, the inspector and the settings sheet. It reads and writes
// through data.Store.
package views

import (
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// Actions is what the views ask the shell to do.
type Actions interface {
	OpenTask(id int) // show the main window with the task open in the inspector
	OpenSettings()
	OpenURL(url string)
	// Post runs fn on the UI thread and redraws. The assistant answers through it.
	Post(fn func())
}

// Config is what the settings sheet shows.
type Config struct {
	ConnectCommand string // the claude mcp add command
	DataFolder     string
	Version        string
}

// App is the state the windows share: the store, a snapshot of its data, and the tasks that are
// playing their completion animation.
type App struct {
	Store  data.Store
	Act    Actions
	Config Config
	Now    func() time.Time
	// Assistant is the built-in agent behind the ask field; nil leaves the field adding Inbox
	// tasks. Models is Settings > Models; nil hides that tab.
	Assistant data.Assistant
	Models    data.ModelSettings

	snap       snapshot
	completing map[int]completion
}

// snapshot is the data the views read, loaded on every change rather than every frame.
type snapshot struct {
	loaded     bool
	err        error
	projects   []data.Project
	tasks      []data.Task
	overview   data.Overview
	milestones map[int][]data.Milestone
	details    map[int]data.TaskDetail
}

// NewApp loads the first snapshot.
func NewApp(store data.Store, act Actions, cfg Config) *App {
	a := &App{Store: store, Act: act, Config: cfg, Now: time.Now, completing: map[int]completion{}}
	a.Refresh()
	return a
}

// Refresh reloads the snapshot. The shell calls it, on the main thread, when the store reports a
// change.
func (a *App) Refresh() {
	s := snapshot{milestones: map[int][]data.Milestone{}, details: map[int]data.TaskDetail{}}
	var err error
	if s.projects, err = a.Store.ListProjects(); err == nil {
		if s.tasks, err = a.Store.ListTasks(data.Filter{}); err == nil {
			s.overview, err = a.Store.Overview()
		}
	}
	for _, p := range s.projects {
		if ms, merr := a.Store.ListMilestones(p.ID); merr == nil {
			s.milestones[p.ID] = ms
		}
	}
	for _, t := range s.tasks {
		if d, derr := a.Store.GetTask(t.ID); derr == nil {
			s.details[t.ID] = d
		}
	}
	s.loaded, s.err = true, err
	if err != nil {
		// Keep showing what we had while the store is unreachable.
		old := a.snap
		old.err = err
		a.snap = old
		return
	}
	a.snap = s
}

// Update patches a task and refreshes. An error leaves the snapshot as it was.
func (a *App) Update(id int, p data.Patch) error {
	if _, err := a.Store.UpdateTask(id, p); err != nil {
		a.snap.err = err
		return err
	}
	a.Refresh()
	return nil
}

// SetStatus changes a task's status. Marking a task done starts its completion animation.
func (a *App) SetStatus(t data.Task, s data.Status) {
	if s == data.Done && t.Status != data.Done {
		a.startCompletion(t)
	}
	_ = a.Update(t.ID, data.Patch{Status: data.StatusPtr(s)})
}

// AddToInbox creates a task without a project.
func (a *App) AddToInbox(title string) error {
	return a.AddTask(data.CreateTaskInput{Title: title, Source: data.SourceUser})
}

// AddTask creates a task and refreshes. An error leaves the snapshot as it was.
func (a *App) AddTask(in data.CreateTaskInput) error {
	if _, err := a.Store.CreateTask(in); err != nil {
		a.snap.err = err
		return err
	}
	a.Refresh()
	return nil
}

// Task finds a task in the snapshot.
func (a *App) Task(id int) (data.Task, bool) {
	for _, t := range a.snap.tasks {
		if t.ID == id {
			return t, true
		}
	}
	return data.Task{}, false
}

// Project finds a project in the snapshot.
func (a *App) Project(id int) (data.Project, bool) {
	for _, p := range a.snap.projects {
		if p.ID == id {
			return p, true
		}
	}
	return data.Project{}, false
}

func (a *App) projectName(id *int) string {
	if id == nil {
		return ""
	}
	if p, ok := a.Project(*id); ok {
		return p.Name
	}
	return ""
}

// Err is the last store error, shown as a calm banner.
func (a *App) Err() error { return a.snap.err }

// ClearErr dismisses the banner after a successful retry.
func (a *App) ClearErr() { a.snap.err = nil }

// byPriorityThenRecent orders tasks for lists: most urgent first, then most recently touched.
func byPriorityThenRecent(ts []data.Task) {
	sort.SliceStable(ts, func(i, j int) bool {
		if ts[i].Priority != ts[j].Priority {
			return ts[i].Priority < ts[j].Priority
		}
		return ts[i].UpdatedAt.After(ts[j].UpdatedAt)
	})
}

// latestReport is a task's newest report in the snapshot, or nil.
func (a *App) latestReport(id int) *data.Report {
	if d, ok := a.snap.details[id]; ok && len(d.Reports) > 0 {
		return &d.Reports[0]
	}
	return nil
}

// buildNow sorts the snapshot into the sections of the Now screen. A task that was just marked
// done stays in its place while its completion plays.
func (a *App) buildNow() data.Now {
	tasks := a.withCompleting(a.snap.tasks, func(data.Task) bool { return true })
	return data.BuildNow(data.NowInput{
		Now: a.Now(), Tasks: tasks, StaleHours: a.snap.overview.StaleHours,
		Latest: a.latestReport, ProjectName: a.projectName,
	})
}
