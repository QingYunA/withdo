package page

import (
	"slices"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

var t0 = time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

func task(id int, s data.Status, mods ...func(*data.Task)) data.Task {
	t := data.Task{ID: id, Status: s, Source: data.SourceUser, Priority: data.P2, UpdatedAt: t0.Add(time.Duration(id) * time.Minute)}
	for _, m := range mods {
		m(&t)
	}
	return t
}

func agent(t *data.Task) { t.Source = data.SourceAgent }

func finished(hours int) func(*data.Task) {
	return func(t *data.Task) {
		at := t0.Add(-time.Duration(hours) * time.Hour)
		t.CompletedAt = &at
	}
}

func ids(ts []data.Task) []int {
	var out []int
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}

func secIDs(p Page, i int) []int { return ids(p.Sections[i].Tasks) }

func TestKindFollowsTheScreenAndTheSearchOverIt(t *testing.T) {
	some := []data.Task{task(1, data.Todo)}
	cases := []struct {
		name string
		in   In
		want Kind
		comp bool
	}{
		{"mine", In{Screen: Mine}, KindMine, true},
		{"agents", In{Screen: Agents}, KindAgents, false},
		{"completed", In{Screen: Completed}, KindCompleted, false},
		{"calendar", In{Screen: Calendar}, KindCalendar, false},
		{"summary", In{Screen: Summary}, KindSummary, false},
		{"recent", In{Screen: Recent}, KindRecent, false},
		{"inbox", In{Screen: Inbox, Tasks: some}, KindList, false},
		{"project with tasks", In{Screen: Project, Tasks: some}, KindColumns, true},
		{"project as board", In{Screen: Project, Tasks: some, Board: true, Cards: some}, KindBoard, true},
		{"empty project reads as a list", In{Screen: Project, Board: true}, KindList, true},
		{"search over mine", In{Screen: Mine, Searching: true}, KindSearch, false},
		{"search over the calendar", In{Screen: Calendar, Searching: true}, KindSearch, false},
		{"search over a board", In{Screen: Project, Tasks: some, Board: true, Searching: true}, KindSearch, false},
	}
	for _, c := range cases {
		p := Build(c.in)
		if p.Kind != c.want || p.Composer != c.comp {
			t.Errorf("%s: kind %v composer %v, want %v %v", c.name, p.Kind, p.Composer, c.want, c.comp)
		}
	}
}

func TestMineHasTheHeroInputAndNoOtherPageDoes(t *testing.T) {
	some := []data.Task{task(1, data.Todo)}
	for name, in := range map[string]In{
		"agents": {Screen: Agents}, "inbox": {Screen: Inbox, Tasks: some}, "calendar": {Screen: Calendar},
		"project": {Screen: Project, Tasks: some}, "empty project": {Screen: Project},
	} {
		if Build(in).Hero {
			t.Errorf("%s has the hero input", name)
		}
	}
	if p := Build(In{Screen: Mine}); !p.Hero || !p.Composer {
		t.Errorf("mine: hero %v composer %v", p.Hero, p.Composer)
	}
}

func TestOnlyPagesWithRowsAreSelectable(t *testing.T) {
	for _, s := range []Screen{Calendar, Summary, Recent, Mine, Agents, Completed} {
		if Build(In{Screen: s}).Selectable() {
			t.Errorf("screen %v with no rows is selectable", s)
		}
	}
	if !Build(In{Screen: Inbox, Tasks: []data.Task{task(1, data.Todo)}}).Selectable() {
		t.Error("a list with a row is not selectable")
	}
	found := Found{Tasks: []data.Task{task(1, data.Todo)}, Name: func(*int) string { return "" }}
	if !Build(In{Screen: Calendar, Searching: true, Found: found}).Selectable() {
		t.Error("search results over the calendar are not selectable")
	}
}

func TestAListGroupsByStatusInPaintOrderAndFoldsDoneAndBacklog(t *testing.T) {
	tasks := []data.Task{
		task(1, data.Todo), task(2, data.Doing), task(3, data.Blocked), task(4, data.Review),
		task(5, data.Done, finished(1)), task(6, data.Done, finished(2)), task(7, data.Done, finished(3)), task(8, data.Done, finished(4)),
		task(9, data.Backlog), task(10, data.Cancelled),
	}
	p := Build(In{Screen: Inbox, Tasks: tasks})
	var order []data.Status
	for _, s := range p.Sections {
		order = append(order, s.Status)
	}
	if want := []data.Status{data.Blocked, data.Review, data.Doing, data.Todo, data.Done, data.Backlog}; !slices.Equal(order, want) {
		t.Fatalf("sections %v, want %v", order, want)
	}
	done, backlog := p.Sections[4], p.Sections[5]
	if !slices.Equal(ids(done.Tasks), []int{5, 6, 7}) || done.Total != 4 || done.Hidden() != 1 {
		t.Errorf("Done %v of %d, want the newest three of four", ids(done.Tasks), done.Total)
	}
	if backlog.Open || len(backlog.Tasks) != 0 || backlog.Total != 1 {
		t.Errorf("a closed Backlog %+v keeps no rows", backlog)
	}
	if want := []int{3, 4, 2, 1, 5, 6, 7}; !slices.Equal(p.IDs(), want) {
		t.Errorf("ids %v, want %v", p.IDs(), want)
	}
	if c := p.Cols[0]; c.Total != 10 || c.Active != 4 {
		t.Errorf("counts %+v, want 10 tasks of which 4 active", c)
	}

	p = Build(In{Screen: Inbox, Tasks: tasks, Folds: [2]Fold{{Backlog: true, AllDone: true}}})
	if want := []int{3, 4, 2, 1, 5, 6, 7, 8, 9}; !slices.Equal(p.IDs(), want) {
		t.Errorf("opened: ids %v, want %v", p.IDs(), want)
	}
	if p.Sections[4].Hidden() != 0 || !p.Sections[5].Open {
		t.Error("Show all and the Backlog fold did not open")
	}
}

func TestADoneFoldKeepsNothingBackWhenItFits(t *testing.T) {
	p := Build(In{Screen: Inbox, Tasks: []data.Task{task(1, data.Done, finished(1)), task(2, data.Done, finished(2)), task(3, data.Done, finished(3))}})
	if s := p.Sections[0]; len(s.Tasks) != 3 || s.Hidden() != 0 {
		t.Errorf("Done %+v, want all three", s)
	}
}

func TestAListSortsWhatIsDueFirstThenUrgentThenRecent(t *testing.T) {
	due := t0.Add(time.Hour)
	p := Build(In{Screen: Inbox, Tasks: []data.Task{
		task(1, data.Todo), // oldest, no date, default priority
		task(2, data.Todo, func(t *data.Task) { t.Priority = data.P0 }),
		task(3, data.Todo, func(t *data.Task) { t.DueAt = &due }),
		task(4, data.Todo), // newest
	}})
	if want := []int{3, 2, 4, 1}; !slices.Equal(p.IDs(), want) {
		t.Errorf("ids %v, want %v", p.IDs(), want)
	}
}

func TestColumnsSplitBySourceAndWaitingLeadsTheAgentsColumn(t *testing.T) {
	handed := t0.Add(-time.Hour)
	waiting := func(t *data.Task) { t.Source = data.SourceAgent; t.HandedTo = "claude"; t.HandedAt = &handed }
	tasks := []data.Task{
		task(1, data.Todo), task(2, data.Doing, agent), task(3, data.Todo, waiting), task(4, data.Blocked, agent),
		task(5, data.Done, finished(1)), task(6, data.Done, agent, finished(1)), task(7, data.Backlog, agent),
	}
	p := Build(In{Screen: Project, Tasks: tasks})
	if p.Kind != KindColumns {
		t.Fatalf("kind %v", p.Kind)
	}
	if got, want := p.IDs(), []int{1, 5, 3, 4, 2, 6}; !slices.Equal(got, want) {
		t.Errorf("ids %v, want Mine's rows, then Agents' with Waiting first: %v", got, want)
	}
	if mine, ag := p.Of(0), p.Of(1); len(mine) != 2 || len(ag) != 5 || ag[0].Kind != SecWaiting {
		t.Errorf("columns have %d and %d sections, the second leading with Waiting", len(mine), len(ag))
	}
	if p.Cols[0] != (Col{Total: 2, Active: 1}) || p.Cols[1] != (Col{Total: 5, Active: 3}) {
		t.Errorf("counts %+v, want Mine 2/1 and Agents 5/3 (Waiting counted in both)", p.Cols)
	}
}

func TestEachColumnFoldsItsOwnBacklogAndDone(t *testing.T) {
	var tasks []data.Task
	for i := 0; i < 5; i++ {
		tasks = append(tasks, task(10+i, data.Done, finished(i+1)), task(20+i, data.Done, agent, finished(i+1)), task(30+i, data.Backlog, agent))
	}
	p := Build(In{Screen: Project, Tasks: tasks, Folds: [2]Fold{{}, {AllDone: true, Backlog: true}}})
	mine, ag := p.Of(0), p.Of(1)
	if len(mine[0].Tasks) != ListDoneRows || mine[0].Hidden() != 2 {
		t.Errorf("Mine's Done %v", ids(mine[0].Tasks))
	}
	if len(ag[0].Tasks) != 5 || len(ag[1].Tasks) != 5 || !ag[1].Open {
		t.Errorf("Agents' Done %v and Backlog %v, both open", ids(ag[0].Tasks), ids(ag[1].Tasks))
	}
}

func TestTheBoardHasFourFixedLanesAndBlockedLeadsDoing(t *testing.T) {
	var tasks []data.Task
	for i := 0; i < 7; i++ {
		tasks = append(tasks, task(100+i, data.Done, finished(i+1)))
	}
	tasks = append(tasks, task(1, data.Doing), task(2, data.Blocked), task(3, data.Backlog), task(4, data.Cancelled))
	p := Build(In{Screen: Project, Board: true, Tasks: tasks, Cards: tasks})
	var lanes []data.Status
	for _, s := range p.Sections {
		lanes = append(lanes, s.Status)
		if s.Kind != SecLane {
			t.Errorf("%v is not a lane", s.Status)
		}
	}
	if want := []data.Status{data.Todo, data.Doing, data.Review, data.Done}; !slices.Equal(lanes, want) {
		t.Fatalf("lanes %v, want %v", lanes, want)
	}
	if got := secIDs(p, 1); !slices.Equal(got, []int{2, 1}) {
		t.Errorf("Doing %v, want the blocked card first", got)
	}
	if done := p.Sections[3]; len(done.Tasks) != LaneDoneRows || done.Total != 7 {
		t.Errorf("Done lane shows %d of %d", len(done.Tasks), done.Total)
	}
	if p.Sections[0].Total != 0 {
		t.Error("an empty lane is not empty")
	}
	p = Build(In{Screen: Project, Board: true, Tasks: tasks, Cards: tasks, Folds: [2]Fold{{AllDone: true}}})
	if len(p.Sections[3].Tasks) != 7 {
		t.Error("Show all did not open the Done lane")
	}
}

func TestTheBoardReadsTheCardsNotTheListTasks(t *testing.T) {
	list := []data.Task{task(1, data.Todo)}
	cards := []data.Task{task(2, data.Todo)}
	p := Build(In{Screen: Project, Board: true, Tasks: list, Cards: cards})
	if got := p.IDs(); !slices.Equal(got, []int{2}) {
		t.Errorf("ids %v, want the card of the board's own completion state", got)
	}
}

func TestSearchGroupsByProjectInTheOrderFirstSeenAndTheInboxLast(t *testing.T) {
	proj := func(id int) func(*data.Task) { return func(t *data.Task) { t.ProjectID = data.IntPtr(id) } }
	found := Found{
		Tasks: []data.Task{task(1, data.Todo), task(2, data.Todo, proj(7)), task(3, data.Todo, proj(5)), task(4, data.Todo, proj(7)), task(5, data.Todo)},
		More:  true,
		Name: func(id *int) string {
			if id == nil {
				return ""
			}
			return map[int]string{5: "five", 7: "seven"}[*id]
		},
	}
	p := Build(In{Screen: Calendar, Searching: true, Found: found})
	if p.Total != 5 || !p.More || len(p.Sections) != 3 {
		t.Fatalf("total %d more %v sections %d", p.Total, p.More, len(p.Sections))
	}
	if p.Sections[0].Title != "seven" || p.Sections[1].Title != "five" || !p.Sections[2].Inbox {
		t.Errorf("groups %q %q inbox=%v", p.Sections[0].Title, p.Sections[1].Title, p.Sections[2].Inbox)
	}
	if want := []int{2, 4, 3, 1, 5}; !slices.Equal(p.IDs(), want) {
		t.Errorf("ids %v, want %v", p.IDs(), want)
	}
}

func TestSearchTakesTheScreenOverAnyOtherState(t *testing.T) {
	tasks := []data.Task{task(1, data.Todo)}
	found := Found{Tasks: []data.Task{task(9, data.Todo)}, Name: func(*int) string { return "" }}
	p := Build(In{Screen: Project, Tasks: tasks, Searching: true, Found: found})
	if got := p.IDs(); !slices.Equal(got, []int{9}) || p.Composer {
		t.Errorf("ids %v composer %v, want only the results and no composer", got, p.Composer)
	}
}

func TestCompletedGroupsByDayNewestFirst(t *testing.T) {
	p := Build(In{Screen: Completed, Now: t0, Done: []data.Task{
		task(1, data.Done, finished(1)), task(2, data.Done, finished(2)), task(3, data.Done, finished(30)),
	}})
	if len(p.Sections) != 2 || p.Sections[0].Kind != SecDay {
		t.Fatalf("sections %+v", p.Sections)
	}
	if !slices.Equal(secIDs(p, 0), []int{1, 2}) || !slices.Equal(secIDs(p, 1), []int{3}) {
		t.Errorf("days %v and %v", secIDs(p, 0), secIDs(p, 1))
	}
	if !slices.Equal(p.IDs(), []int{1, 2, 3}) {
		t.Errorf("ids %v", p.IDs())
	}
}

func TestScopePicksTheInboxOrTheProjectAndItsMilestone(t *testing.T) {
	inProject := func(p, ms int) data.Task {
		t := task(1, data.Todo, func(t *data.Task) { t.ProjectID = data.IntPtr(p) })
		if ms != 0 {
			t.MilestoneID = data.IntPtr(ms)
		}
		return t
	}
	if !(Scope{Inbox: true}).Has(task(1, data.Todo)) || (Scope{Inbox: true}).Has(inProject(1, 0)) {
		t.Error("the Inbox scope is wrong")
	}
	if !(Scope{Project: 1}).Has(inProject(1, 4)) || (Scope{Project: 1}).Has(inProject(2, 0)) {
		t.Error("the project scope is wrong")
	}
	if !(Scope{Project: 1, Milestone: 4}).Has(inProject(1, 4)) || (Scope{Project: 1, Milestone: 4}).Has(inProject(1, 5)) || (Scope{Project: 1, Milestone: 4}).Has(inProject(1, 0)) {
		t.Error("the milestone narrows wrongly")
	}
	sub := inProject(1, 0)
	sub.ParentID = data.IntPtr(9)
	if (Scope{Project: 1}).Has(sub) {
		t.Error("a subtask is listed apart from its parent")
	}
}
