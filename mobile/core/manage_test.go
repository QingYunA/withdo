package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
)

// phoneWithRecorder is demoPhone with a listener that counts OnChange.
func phoneWithRecorder(t *testing.T) (*Core, *changeRecorder) {
	t.Helper()
	phone := demoPhone(t)
	recorder := newChangeRecorder()
	phone.SetListener(recorder)
	return phone, recorder
}

// expectChange waits for at least one OnChange after a write: the listener hears every write.
func expectChange(t *testing.T, recorder *changeRecorder, what string) {
	t.Helper()
	select {
	case <-recorder.done:
	case <-time.After(5 * time.Second):
		t.Fatalf("no OnChange after %s", what)
	}
	time.Sleep(50 * time.Millisecond)
	for len(recorder.done) > 0 { // drain the rest of this write's signals
		<-recorder.done
	}
}

type projectRowForTest struct {
	Project data.Project
	Open    int
	Total   int
	Blocked bool
	Depth   int
}

func projectList(t *testing.T, phone *Core) []projectRowForTest {
	t.Helper()
	raw, err := phone.ProjectList()
	if err != nil {
		t.Fatal(err)
	}
	var rows []projectRowForTest
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatalf("decode ProjectList: %v", err)
	}
	return rows
}

// TestProjectListIsTheSidebarsRows: the counts and the red dot are the desktop sidebar's.
func TestProjectListIsTheSidebarsRows(t *testing.T) {
	phone := demoPhone(t)
	rows := projectList(t, phone)
	snap, err := phone.real.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(snap.Projects) {
		t.Fatalf("%d rows for %d projects", len(rows), len(snap.Projects))
	}
	for _, row := range rows {
		open, total, blocked := 0, 0, false
		for _, task := range snap.Tasks {
			if task.ProjectID == nil || *task.ProjectID != row.Project.ID {
				continue
			}
			total++
			if !task.IsSubtask() && !task.Status.Terminal() && task.Status != data.Backlog {
				open++
				blocked = blocked || task.Status == data.Blocked
			}
		}
		if row.Open != open || row.Total != total || row.Blocked != blocked {
			t.Errorf("%s: open %d total %d blocked %v, want %d %d %v", row.Project.Name, row.Open, row.Total, row.Blocked, open, total, blocked)
		}
	}
	checkGolden(t, "project-list.golden.json", mustString(t)(phone.ProjectList()))
}

func mustString(t *testing.T) func(string, error) string {
	return func(s string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
}

// TestProjectLifecycle: create (with a sub-project), rename, archive, list the archive, unarchive,
// delete; every write reaches the listener, and the sub-project nests under its parent.
func TestProjectLifecycle(t *testing.T) {
	phone, recorder := phoneWithRecorder(t)

	raw, err := phone.CreateProject("Garden", 0)
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "CreateProject")
	var garden data.Project
	if err := json.Unmarshal([]byte(raw), &garden); err != nil || garden.Name != "Garden" || garden.ID == 0 || garden.ParentID != nil {
		t.Fatalf("created %+v (%v)", garden, err)
	}
	raw, err = phone.CreateProject("Roses", int64(garden.ID))
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "CreateProject of a sub-project")
	var roses data.Project
	_ = json.Unmarshal([]byte(raw), &roses)
	if roses.ParentID == nil || *roses.ParentID != garden.ID {
		t.Fatalf("sub-project = %+v", roses)
	}
	rows := projectList(t, phone)
	for i, row := range rows {
		if row.Project.ID == garden.ID {
			if rows[i+1].Project.ID != roses.ID || rows[i+1].Depth != 1 || row.Depth != 0 {
				t.Errorf("the sub-project does not follow its parent: %+v", rows)
			}
		}
	}

	if _, err := phone.CreateProject("  ", 0); err == nil || !strings.Contains(err.Error(), "name must not be empty") {
		t.Errorf("an empty name gave %v", err)
	}

	if err := phone.RenameProject(int64(garden.ID), "Yard"); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "RenameProject")

	if err := phone.ArchiveProject(int64(garden.ID)); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "ArchiveProject")
	for _, row := range projectList(t, phone) {
		if row.Project.ID == garden.ID || row.Project.ID == roses.ID {
			t.Errorf("%s is still listed after Archive", row.Project.Name)
		}
	}
	var archived []data.Project
	if err := json.Unmarshal([]byte(mustString(t)(phone.ArchivedProjects())), &archived); err != nil || len(archived) != 2 {
		t.Fatalf("archived = %+v (%v)", archived, err)
	}
	if err := phone.UnarchiveProject(int64(garden.ID)); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "UnarchiveProject")
	if got := mustString(t)(phone.ArchivedProjects()); got != "[]" {
		t.Errorf("archived after Unarchive = %s, want []", got)
	}

	// Delete reports what went with the project.
	task := mustString(t)(phone.CreateTask(`{"title":"Prune","project_id":` + itoa(garden.ID) + `}`))
	_ = task
	out, err := phone.DeleteProject(int64(garden.ID))
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "DeleteProject")
	var removed data.Removed
	if err := json.Unmarshal([]byte(out), &removed); err != nil || removed.Tasks != 1 {
		t.Errorf("removed = %+v (%v)", removed, err)
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// TestMilestones: create with and without a date, rename, move the date, clear it, close, reopen.
func TestMilestones(t *testing.T) {
	phone, recorder := phoneWithRecorder(t)

	var m data.Milestone
	if err := json.Unmarshal([]byte(mustString(t)(phone.CreateMilestone(1, "v0.13", "2026-11-14"))), &m); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "CreateMilestone")
	if m.Title != "v0.13" || m.ProjectID != 1 || !m.Open || m.DueAt == nil || m.DueAt.Format("2006-01-02") != "2026-11-14" {
		t.Fatalf("created %+v", m)
	}
	var plain data.Milestone
	if err := json.Unmarshal([]byte(mustString(t)(phone.CreateMilestone(1, "later", ""))), &plain); err != nil || plain.DueAt != nil {
		t.Fatalf("a milestone with no date: %+v (%v)", plain, err)
	}
	if _, err := phone.CreateMilestone(1, " ", ""); err == nil {
		t.Error("an empty title was accepted")
	}

	update := func(patch string) data.Milestone {
		t.Helper()
		var got data.Milestone
		if err := json.Unmarshal([]byte(mustString(t)(phone.UpdateMilestone(int64(m.ID), patch))), &got); err != nil {
			t.Fatal(err)
		}
		expectChange(t, recorder, "UpdateMilestone "+patch)
		return got
	}
	if got := update(`{"title":"v0.14","due_at":"2026-12-01"}`); got.Title != "v0.14" || got.DueAt == nil || got.DueAt.Format("2006-01-02") != "2026-12-01" || !got.Open {
		t.Errorf("rename and move: %+v", got)
	}
	if got := update(`{"closed":true}`); got.Open || got.Title != "v0.14" || got.DueAt == nil {
		t.Errorf("close: %+v", got)
	}
	if got := update(`{"closed":false,"due_at":null}`); !got.Open || got.DueAt != nil {
		t.Errorf("reopen and clear the date: %+v", got)
	}
}

type categoryRowForTest struct {
	Category data.Category
	Open     int
}

func categoryRows(t *testing.T, phone *Core) []categoryRowForTest {
	t.Helper()
	var rows []categoryRowForTest
	if err := json.Unmarshal([]byte(mustString(t)(phone.CategoriesWithCounts())), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// TestCategories: the counts are the desktop Settings', and create, update, move, delete and
// restore work with the desktop's rules and sentences.
func TestCategories(t *testing.T) {
	phone, recorder := phoneWithRecorder(t)
	snap, err := phone.real.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	rows := categoryRows(t, phone)
	if len(rows) != len(snap.Categories) {
		t.Fatalf("%d rows for %d categories", len(rows), len(snap.Categories))
	}
	checkGolden(t, "categories-counts.golden.json", mustString(t)(phone.CategoriesWithCounts()))

	var made data.Category
	if err := json.Unmarshal([]byte(mustString(t)(phone.CreateCategory(`{"name":"Health","color":"green","icon":"heart"}`))), &made); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "CreateCategory")
	heart, _ := libraryIcon("heart")
	if made.Name != "Health" || made.Color != "green" || made.Builtin != "" || made.Icon != heart || made.Position != len(rows)+1 {
		t.Fatalf("created %+v", made)
	}
	if _, err := phone.CreateCategory(`{"name":"Health"}`); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("a taken name gave %v", err)
	}
	if _, err := phone.CreateCategory(`{"name":""}`); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("an empty name gave %v", err)
	}
	if _, err := phone.CreateCategory(`{"name":"X","icon":"no-such-icon"}`); err == nil {
		t.Error("an unknown icon was accepted")
	}
	// A new category with nothing but a name gets the library's first curated icon and a hashed color.
	var plain data.Category
	if err := json.Unmarshal([]byte(mustString(t)(phone.CreateCategory(`{"name":"Plain"}`))), &plain); err != nil || plain.Color == "" || plain.Icon == ([12][12]byte{}) {
		t.Errorf("plain category = %+v (%v)", plain, err)
	}

	var renamed data.Category
	if err := json.Unmarshal([]byte(mustString(t)(phone.UpdateCategory(int64(made.ID), `{"name":"Fitness","color":"red"}`))), &renamed); err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Fitness" || renamed.Color != "red" || renamed.Icon != heart {
		t.Errorf("updated %+v", renamed)
	}
	if _, err := phone.UpdateCategory(1, `{"name":"Job"}`); err == nil {
		t.Error("Work was renamed")
	}

	// Sort a task under it, move the category to the front, delete it, and take it all back.
	if err := phone.UpdateTask(2, `{"category":`+itoa(made.ID)+`}`); err != nil {
		t.Fatal(err)
	}
	counted := false
	for _, row := range categoryRows(t, phone) {
		if row.Category.ID == made.ID {
			counted = row.Open == 1
		}
	}
	if !counted {
		t.Errorf("the new category does not count its task: %+v", categoryRows(t, phone))
	}
	if err := phone.MoveCategory(int64(made.ID), 0); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "MoveCategory")
	if first := categoryRows(t, phone)[0].Category; first.ID != made.ID {
		t.Errorf("first category = %+v after moving to 0", first)
	}

	snapshot, err := phone.DeleteCategory(int64(made.ID))
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "DeleteCategory")
	if mustTask(t, phone, 2).CategoryID != 0 {
		t.Error("the deleted category's task is not unsorted")
	}
	if _, err := phone.DeleteCategory(1); err == nil {
		t.Error("Work was deleted")
	}
	if err := phone.RestoreCategory(snapshot); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "RestoreCategory")
	if mustTask(t, phone, 2).CategoryID != made.ID {
		t.Error("the restored category did not take its task back")
	}
	if first := categoryRows(t, phone)[0].Category; first.ID != made.ID {
		t.Errorf("the restored category is not back at its place: %+v", first)
	}
}

// TestCategoryIconLibrary: the library lists the curated groups, searches by name and Chinese
// keyword, and every listed name is one CreateCategory accepts.
func TestCategoryIconLibrary(t *testing.T) {
	phone := demoPhone(t)
	type library struct {
		Groups []struct {
			Name  string
			Icons []struct {
				Name     string
				Keywords string
				Icon     [12][12]byte
			}
		}
	}
	read := func(query string) library {
		var l library
		if err := json.Unmarshal([]byte(mustString(t)(phone.CategoryIcons(query))), &l); err != nil {
			t.Fatal(err)
		}
		return l
	}
	curated := read("")
	if len(curated.Groups) != 8 || curated.Groups[0].Name == "" || len(curated.Groups[0].Icons) == 0 {
		t.Fatalf("curated groups = %d", len(curated.Groups))
	}
	icon := curated.Groups[0].Icons[0]
	if picture, ok := libraryIcon(icon.Name); !ok || picture != icon.Icon {
		t.Errorf("%s: icon differs from the library's", icon.Name)
	}
	found := read("heart")
	if len(found.Groups) != 1 || found.Groups[0].Name != "" || len(found.Groups[0].Icons) == 0 {
		t.Fatalf("search heart = %+v", found)
	}
	if none := read("zzzzzz"); len(none.Groups) != 1 || len(none.Groups[0].Icons) != 0 {
		t.Errorf("a search with no match = %+v", none)
	}
}

// TestBulkChanges: UpdateTasks changes several tasks as one and its Undo puts them back; DeleteTasks
// and RestoreTasks do the same for a delete; DoTodayMany moves the tasks of mine.
func TestBulkChanges(t *testing.T) {
	phone, recorder := phoneWithRecorder(t)

	before1, before2 := mustTask(t, phone, 4), mustTask(t, phone, 5)
	undo, err := phone.UpdateTasks("[4,5]", `{"status":"done","priority":"P1"}`)
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "UpdateTasks")
	for _, id := range []int64{4, 5} {
		if task := mustTask(t, phone, id); task.Status != data.Done || task.Priority != data.P1 || task.CompletedAt == nil {
			t.Errorf("task %d after the bulk change: %+v", id, task)
		}
	}
	if err := phone.RestoreTaskStates(undo); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "RestoreTaskStates")
	for _, want := range []data.Task{before1, before2} {
		if got := mustTask(t, phone, int64(want.ID)); got.Status != want.Status || got.Priority != want.Priority || got.CompletedAt != nil {
			t.Errorf("task %d after Undo: %+v, want status %s priority %s", want.ID, got, want.Status, want.Priority)
		}
	}

	if _, err := phone.UpdateTasks("[4,5]", `{"project_id":2}`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{4, 5} {
		if p := mustTask(t, phone, id).ProjectID; p == nil || *p != 2 {
			t.Errorf("task %d project = %v, want 2", id, p)
		}
	}
	if _, err := phone.UpdateTasks("[4,5]", `{"project_id":0}`); err != nil {
		t.Fatal(err)
	}
	if p := mustTask(t, phone, 4).ProjectID; p != nil {
		t.Errorf("task 4 project = %v, want the Inbox", *p)
	}
	for _, bad := range [][2]string{{"[]", `{}`}, {"nope", `{}`}, {"[4]", `{"status":"nonsense"}`}} {
		if _, err := phone.UpdateTasks(bad[0], bad[1]); err == nil {
			t.Errorf("UpdateTasks(%s, %s) was accepted", bad[0], bad[1])
		}
	}

	// Delete several, undo them all, with their reports.
	reportsBefore := len(mustReports(t, phone, 20))
	snapshots, err := phone.DeleteTasks("[20,21]")
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "DeleteTasks")
	if gone, err := phone.dom.GetTask(context.Background(), 20); err == nil && gone != nil {
		t.Error("task 20 survived DeleteTasks")
	}
	if err := phone.RestoreTasks(snapshots); err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "RestoreTasks")
	if got := len(mustReports(t, phone, 20)); got != reportsBefore || got == 0 {
		t.Errorf("task 20 has %d reports after Undo, want %d", got, reportsBefore)
	}
	if mustTask(t, phone, 21).Title != "Send the packaging answer" {
		t.Error("task 21 did not come back")
	}
}

// TestDoTodayMany moves the tasks of mine that DoToday would, keeps the others, and hands back the
// old dates.
func TestDoTodayMany(t *testing.T) {
	phone, recorder := phoneWithRecorder(t)
	made := func(input string) int64 {
		var task data.Task
		if err := json.Unmarshal([]byte(mustString(t)(phone.CreateTask(input))), &task); err != nil {
			t.Fatal(err)
		}
		return int64(task.ID)
	}
	undated := made(`{"title":"a"}`)
	dated := made(`{"title":"b","due_at":"2026-10-23T09:30:00Z"}`)
	today := made(`{"title":"c","due_at":"2026-10-20T11:00:00Z"}`)
	agents := int64(20) // written by an agent and finished: neither is offered

	out, err := phone.DoTodayMany(fmtInts([]int{int(undated), int(dated), int(today), int(agents), 99999}))
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, recorder, "DoTodayMany")
	var moved []doneToday
	if err := json.Unmarshal([]byte(out), &moved); err != nil {
		t.Fatal(err)
	}
	if len(moved) != 2 || moved[0].ID != undated || moved[0].Before != "" || moved[1].ID != dated || moved[1].Before != "2026-10-23T09:30:00Z" {
		t.Fatalf("moved = %+v", moved)
	}
	if due := mustTask(t, phone, dated).DueAt; due == nil || !due.Equal(time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)) {
		t.Errorf("b is due %v, want today 9:30", due)
	}
	if got := mustString(t)(phone.DoTodayMany(fmtInts([]int{int(today)}))); got != "[]" {
		t.Errorf("a task due today moved: %s", got)
	}
	if _, err := phone.DoTodayMany("[]"); err == nil {
		t.Error("an empty selection was accepted")
	}
}

func mustReports(t *testing.T, phone *Core, id int64) []domain.Report {
	t.Helper()
	reports, err := phone.dom.ListReports(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return reports
}
