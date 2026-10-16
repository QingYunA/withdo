package views

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

const trayDir = "/Users/demo/code/picpocket/.claude/worktrees/tray-race"

func TestAnswerOnABlockedRowOpensTheAgentsAppAtItsWorkingDirectory(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	click(t, tt, "Open Fix tray race on launch")
	if want := []openCall{{OpenAgentApp, "Claude", trayDir}}; !slices.Equal(rec.opens, want) {
		t.Errorf("opened %v, want %v", rec.opens, want)
	}
	if m.openID != 0 {
		t.Error("the button also opened the inspector")
	}
}

func TestReviewWithoutAPullRequestOpensTheSettingAndFallsBackToTheProjectFolder(t *testing.T) {
	app, store, rec := newTestApp(t)
	task, err := store.CreateTask(data.CreateTaskInput{Title: "Tidy the docs", Source: data.SourceAgent, Status: data.Review, ProjectID: data.IntPtr(1)})
	if err != nil {
		t.Fatal(err)
	}
	store.ReportAs(task.ID, "codex", "docs are tidy", fixedNow().Add(-time.Minute))
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)

	click(t, tt, "Open Tidy the docs")
	if want := (openCall{OpenAgentApp, "Codex", "~/Code/adhd-todo"}); len(rec.opens) != 1 || rec.opens[0] != want {
		t.Fatalf("opened %v, want %v: no report named a folder, so the project's", rec.opens, want)
	}

	app.SetOpenWith(OpenTerminal)
	click(t, tt, "Open Tidy the docs")
	if got := rec.opens[len(rec.opens)-1]; got.how != OpenTerminal || got.app != "Codex" {
		t.Errorf("after choosing the terminal: %v", got)
	}
}

func TestTheNewestReportThatNamesAFolderGivesTheWorkingDirectory(t *testing.T) {
	app, store, _ := newTestApp(t)
	task, _ := store.CreateTask(data.CreateTaskInput{Title: "Elsewhere", Source: data.SourceAgent, Status: data.Review, ProjectID: data.IntPtr(1)})
	store.AddReport(data.Report{TaskID: task.ID, Agent: "claude-code", Summary: "old", Cwd: "/old", CreatedAt: fixedNow().Add(-3 * time.Hour)})
	store.AddReport(data.Report{TaskID: task.ID, Agent: "claude-code", Summary: "new", Cwd: "/new", CreatedAt: fixedNow().Add(-2 * time.Hour)})
	store.AddReport(data.Report{TaskID: task.ID, Agent: "claude-code", Summary: "newest, says nothing", CreatedAt: fixedNow().Add(-time.Hour)})
	app.Refresh()
	got, _ := app.Task(task.ID)
	if dir := app.workDir(got); dir != "/new" {
		t.Errorf("workDir = %q, want /new", dir)
	}
	inbox, _ := store.CreateTask(data.CreateTaskInput{Title: "Nowhere", Source: data.SourceAgent})
	app.Refresh()
	got, _ = app.Task(inbox.ID)
	if dir := app.workDir(got); dir != "" {
		t.Errorf("workDir = %q for a task with no report and no project", dir)
	}
}

func TestTheOpenWithChoiceIsKeptBetweenRuns(t *testing.T) {
	app, _, _ := newTestApp(t)
	prefs := &data.MemoryPrefs{}
	app.Prefs = prefs
	if got := app.OpenWith(); got != OpenAgentApp {
		t.Fatalf("default %q, want the agent's app", got)
	}
	app.SetOpenWith(OpenEditor)
	again := NewApp(data.NewFake(fixedNow), &recorder{}, testConfig())
	again.Prefs = prefs
	if got := again.OpenWith(); got != OpenEditor {
		t.Errorf("after a restart %q, want editor", got)
	}
	stored, _ := prefs.LoadPrefs()
	if stored.OpenWith != "editor" {
		t.Errorf("stored %+v", stored)
	}
}

func TestAFailureToOpenShowsOneQuietLineUntilTheNextSuccessOrTheRowIsOpened(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	rec.openErr = errors.New("boom")
	click(t, tt, "Open Fix tray race on launch")
	has(t, tt, "Couldn't open it: boom")

	rec.openErr = nil
	click(t, tt, "Open Fix tray race on launch")
	hasNot(t, tt, "Couldn't open it: boom")

	rec.openErr = ErrNoDir
	click(t, tt, "Open Fix tray race on launch")
	has(t, tt, "Nothing to open: the agent did not say where it works.")
	m.openRow(3)
	tt.Frame()
	hasNot(t, tt, "Nothing to open: the agent did not say where it works.")
}

func TestTheTaskMenuOfAnAgentTaskOpensItsWorkWhereverYouPick(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Open in…")
	has(t, tt, "Open in…", "The agent's app", "Terminal", "Editor", "Finder")
	hasNot(t, tt, "Remind me…")
	click(t, tt, "Editor")
	if want := []openCall{{OpenEditor, "Claude", trayDir}}; !slices.Equal(rec.opens, want) {
		t.Errorf("opened %v, want %v", rec.opens, want)
	}
	if m.pop.open {
		t.Error("the menu stays open after the choice")
	}
}

func TestTheOpenInPageIsQuietWhereTheTaskHasNoFolder(t *testing.T) {
	app, store, rec := newTestApp(t)
	task, _ := store.CreateTask(data.CreateTaskInput{Title: "Lost agent", Source: data.SourceAgent, Status: data.Doing})
	store.AddReport(data.Report{TaskID: task.ID, Agent: "gemini-cli", Summary: "hi", CreatedAt: fixedNow().Add(-time.Minute)})
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenInbox})
	tt.Frame()
	rightClick(t, tt, "Lost agent")
	click(t, tt, "Open in…")
	has(t, tt, "The agent's app", "Terminal", "Editor", "Finder")
	for _, name := range []string{"The agent's app", "Terminal", "Editor", "Finder"} {
		click(t, tt, name)
	}
	if len(rec.opens) != 0 {
		t.Errorf("opened %v: Gemini has no app and the task no folder", rec.opens)
	}
	if !m.pop.open {
		t.Error("a quiet choice closed the menu")
	}
}

func TestTheMenuOfMyOwnTaskHasNoOpenIn(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenInbox})
	tt.Frame()
	rightClick(t, tt, "Book dentist")
	hasNot(t, tt, "Open in…")
}

func TestSettingsAgentsChoosesWhereToOpenAgentWork(t *testing.T) {
	r := agentsRig(t)
	r.app.Prefs = &data.MemoryPrefs{}
	r.tt.Scroll(340, 300, 0, 3000) // the row is under the agents
	r.tt.Frame()
	has(t, r.tt, "Open agent work in", "The agent's app", "Brings the agent's app forward. It cannot open the exact session.")
	click(t, r.tt, "The agent's app")
	has(t, r.tt, "Terminal", "Editor", "Finder")
	click(t, r.tt, "Terminal")
	if got := r.app.OpenWith(); got != OpenTerminal {
		t.Errorf("chose %q", got)
	}
}

func TestAgentAppNamesTheDesktopAppOfAnAgent(t *testing.T) {
	for agent, want := range map[string]string{
		"claude-code": "Claude", "Claude Desktop": "Claude", "codex": "Codex", "cursor": "Cursor", "windsurf": "Windsurf",
		"zed": "Zed", "Zed-Agent": "Zed", "github-copilot": "Visual Studio Code", "Visual Studio Code": "Visual Studio Code",
		"gemini-cli": "", "opencode": "", "qwen-code": "", "kimi-cli": "", "cline": "", "agent": "", "": "",
	} {
		if got := agentApp(agent); got != want {
			t.Errorf("agentApp(%q) = %q, want %q", agent, got, want)
		}
	}
}
