package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
)

func TestAProjectIsTwoColumnsMineAndAgentsBySource(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	left := func(a, b string) bool {
		ra, _ := tt.Find(a)
		rb, ok := tt.Find(b)
		return ok && ra.X < rb.X
	}
	for _, mine := range []string{"Write the release notes", "Decide on the activation policy", "Draft the UI spec"} {
		for _, agents := range []string{"Native UI with mygo", "Fix tray race on launch", "Port the MCP server"} {
			if !left(mine, agents) {
				t.Errorf("%q (mine) is not left of %q (an agent's)", mine, agents)
			}
		}
	}
	has(t, tt, "Mine", "Agents")
	if a, b := m.pg.Cols[sideMine].Active, m.pg.Cols[sideAgents].Active; a != 2 || b != 4 {
		t.Errorf("active counts %d and %d, want 2 and 4", a, b)
	}
	if m.pg.Kind != page.KindColumns || m.nav.projectID != 1 {
		t.Error("the project is not drawn as columns")
	}
}

func TestTheBoardStaysAsItWas(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	m.board[1] = true
	tt.Frame()
	if m.pg.Kind == page.KindColumns {
		t.Error("the board is two columns")
	}
}

func TestEachColumnHasItsOwnDoneAndBacklog(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	tt.SetSize(1100, 1600)
	tt.Frame()
	has(t, tt, "Done", "Spike the mygo shell", "Draft the UI spec", "Backlog 2")
	hasNot(t, tt, "Poll gh for PR status")
	if err := tt.Click("show Backlog 2"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "Poll gh for PR status", "Add a Windows build")
	if !m.expanded[m.nav] {
		t.Error("Mine's Backlog is not open")
	}
	if m.foldMap(sideAgents)[m.nav] {
		t.Error("opening Mine's Backlog opened the Agents column's")
	}
}

func TestTheAgentsDoneGroupShowsThreeAndShowAllOpensTheHistory(t *testing.T) {
	f := demoStore(t, fixedNow, false)
	p := f.SeedProject("scratch", "me/scratch")
	for i := 0; i < 5; i++ {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: "Agent made " + itoa(i), Status: data.Done, Source: data.SourceAgent, ProjectID: data.IntPtr(p.ID)})
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetSize(1100, 1000)
	m.goTo(nav{screen: screenProject, projectID: p.ID})
	tt.Frame()
	has(t, tt, "Show all")
	if err := tt.Click("Show all"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if want := (nav{screen: screenCompleted, projectID: p.ID, agents: true}); m.nav != want {
		t.Fatalf("nav %+v, want the history of the project's agents %+v", m.nav, want)
	}
	if ids := m.pg.IDs(); len(ids) != 5 {
		t.Errorf("ids %v in the history", ids)
	}
	has(t, tt, "scratch", "Agent made 0")
}

func TestAWaitingTaskHasItsOwnGroupFirstAndLeavesTodo(t *testing.T) {
	r := newHandRig(t, true)
	if _, err := r.store.HandOff(5); err != nil {
		t.Fatal(err)
	}
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "Waiting for an agent", "Write the release notes", "Tell any agent “do #5” to pick it up")
	if !above(t, r.tt, "Waiting for an agent", "Blocked") {
		t.Error("Waiting for an agent comes before Blocked")
	}
	if above(t, r.tt, "To do", "Write the release notes") && !above(t, r.tt, "Write the release notes", "Fix tray race on launch") {
		t.Error("the waiting task is still listed under To do")
	}
	waiting := 0
	for _, s := range r.m.pg.Sections {
		if s.Kind == page.SecWaiting {
			waiting = len(s.Tasks)
		}
	}
	if mine := r.m.pg.Cols[sideMine].Total; waiting != 1 || mine != 5 {
		t.Errorf("waiting %d, mine %d", waiting, mine)
	}
	// Its first report moves it on.
	r.now = r.now.Add(time.Minute)
	r.store.SeedReportAs(5, "claude-code", "Reading the notes", r.now)
	r.app.Refresh()
	r.tt.Frame()
	hasNot(t, r.tt, "Waiting for an agent")
}

func TestAnEmptyColumnKeepsItsHeaderAndOneQuietLine(t *testing.T) {
	// Only an agent's task: Mine is empty.
	f := demoStore(t, fixedNow, false)
	p := f.SeedProject("scratch", "me/scratch")
	_, _ = f.CreateTask(data.CreateTaskInput{Title: "Agent only", Source: data.SourceAgent, Status: data.Doing, ProjectID: data.IntPtr(p.ID)})
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: p.ID})
	tt.Frame()
	has(t, tt, "Mine", "Agents", "Nothing for you here — write one above", "Agent only")
	hasNot(t, tt, "No agent has reported in this project yet", "Install MCP →")

	// Only a task of mine: Agents is empty, with the link to install the MCP.
	g := demoStore(t, fixedNow, false)
	q := g.SeedProject("local", "me/local")
	_, _ = g.CreateTask(data.CreateTaskInput{Title: "Mine only", ProjectID: data.IntPtr(q.ID)})
	app2 := NewApp(g, &recorder{}, testConfig())
	app2.Now = fixedNow
	tt2, m2, _ := newTesterFor(app2, 1100)
	m2.goTo(nav{screen: screenProject, projectID: q.ID})
	tt2.Frame()
	has(t, tt2, "Mine", "Agents", "No agent has reported in this project yet", "Install MCP →", "Mine only")
	hasNot(t, tt2, "Nothing for you here")
	if err := tt2.Click("Install MCP →"); err != nil {
		t.Fatal(err)
	}
	rec := app2.Act.(*recorder)
	if len(rec.panes) != 1 || rec.panes[0] != PaneAgents {
		t.Errorf("opened settings %v, want the Agents pane", rec.panes)
	}
}

func TestAProjectWithNoTaskKeepsTheCentredEmptyState(t *testing.T) {
	app, p := emptyProjectApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: p.ID})
	tt.Frame()
	if m.pg.Kind == page.KindColumns {
		t.Error("an empty project is drawn as columns")
	}
	has(t, tt, "No tasks yet.")
	hasNot(t, tt, "Nothing for you here")
}

func TestTheColumnsStackBelow640PointsOfContent(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	head := func(s string) (x, y float32) {
		r, _ := tt.Find(s)
		return r.X, r.Y
	}
	mx, my := head("Write the release notes")
	ax, ay := head("Native UI with mygo")
	if !(ax > mx+200 && absInt(int(ay-my)) < 400) {
		t.Fatalf("side by side at 1100: mine (%v, %v), agents (%v, %v)", mx, my, ax, ay)
	}
	m.openRow(5) // the inspector takes 361 pt: 522 pt of content
	tt.Frame()
	mx, my = head("Write the release notes")
	ax, ay = head("Native UI with mygo")
	if absInt(int(m.cols.geo.tray.X-m.cols.geo.mine.X)) > 2 || ay <= my {
		t.Errorf("stacked: mine (%v, %v), agents (%v, %v): the tray is under Mine, at its left edge", mx, my, ax, ay)
	}
	if dx, dy := head("Draft the UI spec"); dy >= ay || dx != mx {
		t.Errorf("Mine's last row (%v, %v) is not above the tray (%v)", dx, dy, ay)
	}
}

func TestTheColumnsShareTheEdgesOfTheComposerAndStopAt980(t *testing.T) {
	tt, m, _ := projectTester(t, 1)
	tt.SetSize(1800, 800)
	tt.Frame()
	geo := m.cols.geo
	if got := geo.tray.X + geo.tray.W - geo.mine.X; got < theme.BoardMax-1 || got > theme.BoardMax+1 {
		t.Errorf("the columns span %v pt, want the block's %v", got, theme.BoardMax)
	}
	if gap := geo.tray.X - (geo.mine.X + geo.mine.W); gap < theme.ColumnsGap-1 || gap > theme.ColumnsGap+1 {
		t.Errorf("gap %v, want %v", gap, theme.ColumnsGap)
	}
	if d := geo.mine.W - geo.tray.W; d < -1 || d > 1 {
		t.Errorf("columns of %v and %v pt are not equal", geo.mine.W, geo.tray.W)
	}
}
