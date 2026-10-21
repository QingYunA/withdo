// Package views draws the adhd-todo windows with mygo native UI: the main window (the sidebar,
// the Mine and Agents screens, the project lists and board, the inbox and the inspector) and the Settings
// window. It reads and writes through data.Store.
package views

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
)

// Actions is what the views ask the shell to do.
type Actions interface {
	OpenTask(id int)        // show the main window with the task open in the inspector
	OpenSettings(pane Pane) // show the Settings window on a pane
	OpenURL(url string)
	ShowInFinder(path string)
	// OpenFile opens a file with the app macOS has for it: Preview, for a picture.
	OpenFile(path string)
	// SaveText asks where to save text, as a file proposed under name, and writes it there. It
	// returns at once: the user answers in a panel of its own.
	SaveText(name, text string)
	// OpenIn brings up where an agent works: its own desktop app (agentApp, "" when it has none), or
	// a terminal, an editor or Finder at dir. The agent's app cannot be asked for the exact session,
	// only brought forward; without that app, or when it is not installed, a terminal opens.
	OpenIn(how OpenWith, agentApp, dir string) error
	// Post runs fn on the UI thread and redraws. The assistant answers through it.
	Post(fn func())
	// Languages are the languages the interface is written in. Language is the one the user
	// chose, empty while the interface follows the system's; SetLanguage chooses one, or "" to
	// follow the system's again.
	Languages() []string
	Language() string
	SetLanguage(tag string)
	// NotificationsDenied says macOS refused the last system notification, so the switch in
	// Settings does nothing until it is allowed in System Settings.
	NotificationsDenied() bool
	// CanUpdate says this build updates itself: it was built with updates in mygo.json. Without, it
	// offers no updates at all.
	CanUpdate() bool
	// AutomaticUpdates says the app checks for updates in the background; SetAutomaticUpdates turns
	// that on or off. The update plugin keeps the choice, not the app's prefs.
	AutomaticUpdates() bool
	SetAutomaticUpdates(on bool)
}

// Config is what the Settings window shows.
type Config struct {
	ConnectCommand string // the claude mcp add command
	DataFolder     string
	// AssistantMemory is the file the assistant keeps its notes about the user in; empty hides its row.
	AssistantMemory string
	Home            string // the user's home folder, which the data folder is shown relative to
	Version         string
	// AgentSetup finds the coding agents on this Mac and installs the MCP into them; nil finds none.
	AgentSetup *agentsetup.Env
}

// App is the state the windows share: the store, a snapshot of its data, and the tasks that are
// playing their completion animation.
type App struct {
	Store  data.Store
	Act    Actions
	Config Config
	Now    func() time.Time
	// FileExists says whether a local file is there, for the pages agents link to; nil asks the
	// file system.
	FileExists func(path string) bool
	// Assistant is the built-in agent behind the composer; nil leaves the composer adding plain
	// tasks. Models is Settings > Assistant; nil leaves that pane empty.
	Assistant data.Assistant
	Models    data.ModelSettings
	// GitHub is the pull request poller behind Settings > GitHub; nil leaves the pane empty.
	GitHub data.GitHub
	// Sync is the sync service behind Settings > Sync and the sidebar cloud; nil leaves the pane empty.
	Sync SyncService
	// Seen keeps which handed-over items the user has seen; nil keeps that for this run only.
	Seen data.SeenStore
	// Dismissed keeps which handed-over rows the user put aside with Not now; nil keeps that for
	// this run only.
	Dismissed data.SeenStore
	// Prefs keeps the choices made in Settings that the app keeps itself; nil keeps them for this
	// run only.
	Prefs data.PrefsStore
	// Sound makes the Summary scene's sounds; nil keeps the scene silent. SystemSounds says whether
	// macOS plays user interface sound effects, asked once per play off the UI thread, and WindowKey
	// whether the main window has keyboard focus, asked before each cue; nil means yes.
	Sound                   CuePlayer
	SystemSounds, WindowKey func() bool
	// Recent reads the repositories of the projects for the Recent changes page, and RecentCache keeps
	// what the assistant wrote about them (a cache in memory when nil). With no Recent the page has
	// nothing to read.
	Recent      data.Recent
	RecentCache *data.RecentCache

	seen            data.Seen
	seenLoaded      bool
	dismissed       data.Seen
	dismissedLoaded bool
	prefsMu         sync.Mutex // the notifier reads the prefs from its own goroutine
	prefs           data.Prefs
	prefsLoaded     bool
	weekStart       atomic.Int32 // the first weekday, as the window draws it
	snap            snapshot
	rev             int // counts the snapshots, so a search or Completed knows its answer is old
	completing      map[int]completion
}

// snapshot is the data the views read, loaded on every change rather than every frame. The fields
// but all and images are views of all, for the code that reads them by name.
type snapshot struct {
	loaded     bool
	err        error
	all        data.Snapshot
	projects   []data.Project
	categories []data.Category
	archived   []data.Project
	tasks      []data.Task
	milestones map[int][]data.Milestone
	details    map[int]data.TaskDetail
	images     map[int][]data.TaskImage // the pictures of the tasks that have any
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
	var s snapshot
	all, err := a.Store.Snapshot()
	if err == nil {
		s = snapshot{
			all: all, projects: all.Projects, categories: all.Categories, archived: all.Archived,
			tasks: all.Tasks, milestones: all.Milestones, details: all.Details(),
		}
		ids := make([]int, len(s.tasks))
		for i, t := range s.tasks {
			ids[i] = t.ID
		}
		if imgs, ierr := a.Store.ImagesOf(ids); ierr == nil { // one read for every list and card
			s.images = imgs
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
	a.rev++
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
func (a *App) SetStatus(t data.Task, s data.Status) { a.setStatus(t, s, byGlyph) }

// Accept marks a task done from its Accept button, which stamps before the row folds away. A task
// already being accepted is left alone, so a second click does not start the stamp again.
func (a *App) Accept(t data.Task) {
	if _, busy := a.completing[t.ID]; !busy {
		a.setStatus(t, data.Done, byAccept)
	}
}

func (a *App) setStatus(t data.Task, s data.Status, kind completionKind) {
	if s == data.Done && t.Status != data.Done {
		a.startCompletion(t, kind)
	}
	_ = a.Update(t.ID, data.Patch{Status: data.StatusPtr(s)})
}

// KeepClosedPR dismisses the From your agents row of a task whose pull request was closed, and refreshes.
func (a *App) KeepClosedPR(id int) {
	if err := a.Store.KeepClosedPR(id); err != nil {
		a.snap.err = err
		return
	}
	a.Refresh()
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

// AddTasks creates the tasks in order and refreshes once. It returns the ids it made, which are all
// of them unless the error says one failed; the ones made stay.
func (a *App) AddTasks(ins []data.CreateTaskInput) ([]int, error) {
	var ids []int
	var err error
	for _, in := range ins {
		var t data.Task
		if t, err = a.Store.CreateTask(in); err != nil {
			a.snap.err = err
			break
		}
		ids = append(ids, t.ID)
	}
	a.Refresh()
	return ids, err
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

// Images are the pictures of a task, oldest first; none for most tasks.
func (a *App) Images(id int) []data.TaskImage { return a.snap.images[id] }

// Project finds a project in the snapshot.
func (a *App) Project(id int) (data.Project, bool) {
	for _, p := range a.snap.projects {
		if p.ID == id {
			return p, true
		}
	}
	return data.Project{}, false
}

// Category finds a category in the snapshot.
func (a *App) Category(id int) (data.Category, bool) {
	for _, c := range a.snap.categories {
		if c.ID == id {
			return c, true
		}
	}
	return data.Category{}, false
}

// builtinID is the id of the built-in category data.BuiltinWork or data.BuiltinLife, 0 when the
// snapshot has none (the store is unreachable).
func (a *App) builtinID(key string) int {
	for _, c := range a.snap.categories {
		if c.Builtin == key {
			return c.ID
		}
	}
	return 0
}

// workID is the id of the built-in Work, which a task in a repository's project counts under.
func (a *App) workID() int { return a.builtinID(data.BuiltinWork) }

func (a *App) projectName(id *int) string { return a.snap.all.ProjectName(id) }

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

// latestReport is the newest report an agent filed against a task, githubReport the newest GitHub
// filed; see data.TaskDetail.
func (a *App) latestReport(id int) *data.Report { return a.snap.all.LatestReport(id) }
func (a *App) githubReport(id int) *data.Report { return a.snap.all.GitHubReport(id) }

// mineInput is what the Mine and Agents screens are built from. A task that was just marked done
// stays in its place while its completion plays.
func (a *App) mineInput() data.MineInput {
	snap := a.snap.all
	snap.Tasks = a.withCompleting(a.snap.tasks, func(data.Task) bool { return true })
	return data.MineInput{Snapshot: snap, Now: a.Now()}
}

// buildMine sorts the snapshot into the sections of the Mine screen, without the handed-over rows
// the user put aside, so the count, the badge and the hop leave them out too.
func (a *App) buildMine() data.Mine { return a.buildMineOf(nil) }

// buildMineOf is buildMine for the tasks keep accepts; nil keeps them all.
func (a *App) buildMineOf(keep func(data.Task) bool) data.Mine {
	in := a.mineInput()
	in.Keep = keep
	return data.BuildMine(in).WithoutDismissed(a.dismissedSet())
}

// buildAgents lists what the agents are doing, with their subtasks.
func (a *App) buildAgents() data.AgentsScreen { return a.buildAgentsOf(nil) }

// buildAgentsOf is buildAgents for the tasks keep accepts; nil keeps them all.
func (a *App) buildAgentsOf(keep func(data.Task) bool) data.AgentsScreen {
	in := a.mineInput()
	in.Keep = keep
	return data.BuildAgents(in)
}

// topLevel are the tasks that get a row of their own: the snapshot without the subtasks, which live
// under their parent.
func topLevel(tasks []data.Task) []data.Task {
	out := make([]data.Task, 0, len(tasks))
	for _, t := range tasks {
		if !t.IsSubtask() {
			out = append(out, t)
		}
	}
	return out
}

// parentOf is the task a subtask belongs to, when it is in the snapshot.
func (a *App) parentOf(t data.Task) (data.Task, bool) {
	if t.ParentID == nil {
		return data.Task{}, false
	}
	return a.Task(*t.ParentID)
}
