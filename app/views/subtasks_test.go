package views

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The seed's "Port core to Go" (task 1) has subtasks 22 (done), 23 and 24 (doing); mineApp adds a
// fourth handed back (in review). Task 20 "Port the MCP server" is done today with subtask 25.
var subtaskTitles = []string{
	"Port the store and migrations", "Port the MCP tools", "Watch the database for changes",
	"Vendor icons and one-click install", "Cover the tools with tests",
}

func TestAgentsListsWorkingTasksWithTheirSubtasksUnderThem(t *testing.T) {
	app, _ := mineApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1000)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	has(t, tt, "Agents", "3 working", "Working", "Port core to Go", "Issue #218 adaptation", "Fix tray race on launch",
		"Port the store and migrations", "Port the MCP tools", "Watch the database for changes", "Vendor icons and one-click install",
		"Done today", "Port the MCP server", "1 subtask")
	if !above(t, tt, "Port core to Go", "Port the store and migrations") || !above(t, tt, "Vendor icons and one-click install", "Issue #218 adaptation") {
		t.Errorf("subtasks belong under their parent: %q", tt.Texts())
	}
	if !above(t, tt, "Working", "Port core to Go") || !above(t, tt, "Fix tray race on launch", "Done today") {
		t.Errorf("sections out of order: %q", tt.Texts())
	}
	hasNot(t, tt, "Cover the tools with tests") // a finished task's subtasks are a count, not rows
	if !tt.HasText("Done") || !tt.HasText("Doing") || !tt.HasText("Handed back") || !tt.HasText("Blocked") {
		t.Errorf("status pills missing in %q", tt.Texts())
	}
}

func TestAgentsComposerAsksAboutTheAgents(t *testing.T) {
	app, _ := mineApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	has(t, tt, "Ask what your agents are doing")
	hasNot(t, tt, "Tell me anything — I'll turn it into tasks")
	m.goTo(nav{})
	tt.Frame()
	has(t, tt, "Tell me anything — I'll turn it into tasks")
	hasNot(t, tt, "Ask what your agents are doing")
}

func TestAgentsWithNothingAtWorkSaysSo(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	has(t, tt, "No agents at work", "When an agent reports its progress, it shows up here.", "How to connect")
	hasNot(t, tt, "working", "Working")
}

func TestClickingASubtaskOpensItAndNotItsParent(t *testing.T) {
	app, _ := mineApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1000)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	click(t, tt, "Port the MCP tools")
	if m.openID != 23 {
		t.Fatalf("open %d, want the subtask 23", m.openID)
	}
	// The inspector names the parent, as a link that opens it.
	has(t, tt, "Parent")
	click(t, tt, "Open Port core to Go")
	if m.openID != 1 {
		t.Errorf("the parent link opened %d, want 1", m.openID)
	}
}

func TestTheInspectorOfAParentListsItsSubtasksAndAClickOpensOne(t *testing.T) {
	app, _ := mineApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1000)
	m.goTo(nav{screen: screenAgents})
	m.OpenTask(1)
	m.nav = nav{screen: screenAgents}
	tt.Frame()
	has(t, tt, "Subtasks")
	// Every subtask is in the inspector, in the order they were made; the Agents screen shows them
	// too, so each title is drawn twice here.
	var titles []string
	for _, s := range app.subtasksOf(1) {
		titles = append(titles, s.Title)
	}
	if strings.Join(titles, "|") != strings.Join(subtaskTitles[:4], "|") {
		t.Errorf("subtasks %q", titles)
	}
	hasNot(t, tt, "Parent") // a parent has no parent
	// Click the row in the inspector: only it is labelled as a button.
	if err := tt.Click("Open Port the MCP tools"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 23 {
		t.Errorf("open %d, want 23", m.openID)
	}
}

func TestATaskWithoutSubtasksHasNoSubtasksList(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.OpenTask(3)
	tt.Frame()
	hasNot(t, tt, "Subtasks", "Parent")
}

func TestSubtasksAreNeverTopLevelRowsInListsBoardsInboxAndCompleted(t *testing.T) {
	app, store := mineApp(t)
	// One in the Inbox too, and done ones, so every place has something to leave out.
	inboxParent, _ := store.CreateTask(data.CreateTaskInput{Title: "Inbox parent", Source: data.SourceAgent, Status: data.Doing})
	store.CreateTask(data.CreateTaskInput{Title: "Inbox child", ParentID: data.IntPtr(inboxParent.ID), Source: data.SourceAgent, Status: data.Doing})
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1400)
	hidden := append(subtaskTitles, "Inbox child")
	for name, to := range map[string]nav{
		"list":      {screen: screenProject, projectID: 1},
		"inbox":     {screen: screenInbox},
		"completed": {screen: screenCompleted},
		"calendar":  {screen: screenCalendar},
	} {
		m.goTo(to)
		tt.Frame()
		for _, title := range hidden {
			if tt.HasText(title) {
				t.Errorf("%s: subtask %q is a row", name, title)
			}
		}
	}
	m.goTo(nav{screen: screenProject, projectID: 1})
	m.board[1] = true
	tt.Frame()
	for _, title := range hidden {
		if tt.HasText(title) {
			t.Errorf("board: subtask %q is a card", title)
		}
	}
	has(t, tt, "Port core to Go")
}

func TestTheSidebarCountsLeaveSubtasksOut(t *testing.T) {
	app, store := mineApp(t)
	before, _ := openCount(app.snap.tasks, func(task data.Task) bool { return task.ProjectID != nil && *task.ProjectID == 1 })
	store.CreateTask(data.CreateTaskInput{Title: "Another child", ProjectID: data.IntPtr(1), ParentID: data.IntPtr(1), Source: data.SourceAgent, Status: data.Doing})
	app.Refresh()
	after, _ := openCount(app.snap.tasks, func(task data.Task) bool { return task.ProjectID != nil && *task.ProjectID == 1 })
	if after != before {
		t.Errorf("project count went from %d to %d with a new subtask", before, after)
	}
}

func TestSearchFindsSubtasksAndNamesTheirParent(t *testing.T) {
	app, _ := mineApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.Frame()
	searchFor(t, tt, "MCP tools")
	has(t, tt, "Port the MCP tools", "Port core to Go")
	if got := m.pg.Total; got != 1 {
		t.Errorf("results %d", got)
	}
	if parent := m.parentTitle(data.Task{ParentID: data.IntPtr(1)}); parent != "Port core to Go" {
		t.Errorf("parent title %q", parent)
	}
	if parent := m.parentTitle(data.Task{}); parent != "" {
		t.Errorf("a top-level task names a parent: %q", parent)
	}
}

func TestSubtasksOfAParentWhoseSubtaskWasCancelledAreStillListedInTheInspector(t *testing.T) {
	app, store := mineApp(t)
	if _, err := store.UpdateTask(23, data.Patch{Status: data.StatusPtr(data.Cancelled)}); err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	if got := len(app.subtasksOf(1)); got != 4 {
		t.Errorf("the inspector lists %d subtasks, want all 4", got)
	}
	agents := app.buildAgents()
	if got := len(agents.Working[0].Subs); got != 3 {
		t.Errorf("Agents lists %d, want the 3 that were not cancelled", got)
	}
}

func TestChineseAgentsScreen(t *testing.T) {
	app, _ := mineApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1000)
	inChinese(tt)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	has(t, tt, "Agent", "3 个在做", "在做", "完成", "进行中", "交回主代理", "1 个子任务", "今天做完", "问问 Agent 们在做什么")
	hasNot(t, tt, "Working", "Handed back", "Done today")
	empty, _ := newEmptyApp(t)
	tt2, m2, _ := newTesterFor(empty, 1100)
	inChinese(tt2)
	m2.goTo(nav{screen: screenAgents})
	tt2.Frame()
	has(t, tt2, "没有 Agent 在做事", "Agent 汇报进度后，会出现在这里。")
}
