package core

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// TestPagesMatchTheDesktop is acceptance 1: on app/data's demo data, the phone's JSON decodes to
// exactly what the desktop's data.Real and page builders produce for the same database, and the
// golden files pin the shape.
func TestPagesMatchTheDesktop(t *testing.T) {
	dir := demoDatabase(t, demoNow)
	phone, err := open(dir, nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	defer func() { _ = phone.Close() }()
	desktop, err := data.OpenReal(context.Background(), phone.dbPath, data.RealOptions{
		Now: func() time.Time { return demoNow },
	})
	if err != nil {
		t.Fatalf("open the desktop store: %v", err)
	}
	defer func() { _ = desktop.Close() }()
	snap, err := desktop.Snapshot()
	if err != nil {
		t.Fatalf("desktop snapshot: %v", err)
	}

	t.Run("Mine", func(t *testing.T) {
		raw, err := phone.Mine()
		if err != nil {
			t.Fatal(err)
		}
		var got data.Mine
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Mine: %v", err)
		}
		want := data.BuildMine(data.MineInput{Snapshot: snap, Now: demoNow, NoDateOpen: true}) // the phone shows the fold open
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Mine differs from the desktop's\nphone: %+v\nwant: %+v", got, want)
		}
		checkGolden(t, "mine.golden.json", raw)
	})

	t.Run("Agents", func(t *testing.T) {
		raw, err := phone.Agents()
		if err != nil {
			t.Fatal(err)
		}
		var got data.AgentsScreen
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Agents: %v", err)
		}
		want := data.BuildAgents(data.MineInput{Snapshot: snap, Now: demoNow})
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Agents differs from the desktop's\nphone: %+v\nwant: %+v", got, want)
		}
		checkGolden(t, "agents.golden.json", raw)
	})

	t.Run("Projects", func(t *testing.T) {
		raw, err := phone.Projects()
		if err != nil {
			t.Fatal(err)
		}
		var got []data.Project
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Projects: %v", err)
		}
		if !reflect.DeepEqual(got, snap.Projects) {
			t.Errorf("Projects = %+v\nwant %+v", got, snap.Projects)
		}
		checkGolden(t, "projects.golden.json", raw)
	})

	t.Run("Categories", func(t *testing.T) {
		raw, err := phone.Categories()
		if err != nil {
			t.Fatal(err)
		}
		var got []data.Category
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Categories: %v", err)
		}
		if !reflect.DeepEqual(got, snap.Categories) {
			t.Errorf("Categories differs from the snapshot\nphone: %+v\nwant: %+v", got, snap.Categories)
		}
		checkGolden(t, "categories.golden.json", raw)
	})

	t.Run("Project", func(t *testing.T) {
		raw, err := phone.Project(1)
		if err != nil {
			t.Fatal(err)
		}
		var got projectPage
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Project: %v", err)
		}
		want := projectPageOf(snap, 1)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Project(1) differs from the desktop's\nphone: %+v\nwant: %+v", got, want)
		}
		checkGolden(t, "project-1.golden.json", raw)
	})

	t.Run("Task", func(t *testing.T) {
		raw, err := phone.Task(1)
		if err != nil {
			t.Fatal(err)
		}
		var got taskPage
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Task: %v", err)
		}
		want := taskPageOf(snap, 1)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Task(1) differs from the desktop's\nphone: %+v\nwant: %+v", got, want)
		}
		checkGolden(t, "task-1.golden.json", raw)
	})

	t.Run("Search", func(t *testing.T) {
		raw, err := phone.Search("#12")
		if err != nil {
			t.Fatal(err)
		}
		var got []data.Task
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Search: %v", err)
		}
		want, err := desktop.SearchTasks("#12", 0)
		if err != nil {
			t.Fatalf("desktop search: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Search differs from the desktop's\nphone: %+v\nwant: %+v", got, want)
		}
		checkGolden(t, "search.golden.json", raw)
	})

	t.Run("missing", func(t *testing.T) {
		if _, err := phone.Task(9999); err == nil {
			t.Error("Task(9999) found a task")
		}
		if _, err := phone.Project(9999); err == nil {
			t.Error("Project(9999) found a project")
		}
	})
}

// projectPageOf is the desktop's project page: the project, its tasks and its milestones.
func projectPageOf(snap data.Snapshot, id int) projectPage {
	out := projectPage{Tasks: []data.Task{}, Milestones: []data.Milestone{}, Agents: map[string]string{}}
	for _, p := range append(append([]data.Project{}, snap.Projects...), snap.Archived...) {
		if p.ID == id {
			out.Project = p
			break
		}
	}
	for _, task := range snap.Tasks {
		if task.ProjectID != nil && *task.ProjectID == id {
			out.Tasks = append(out.Tasks, task)
			if r := snap.LatestReport(task.ID); r != nil {
				out.Agents[strconv.Itoa(task.ID)] = r.Agent
			}
		}
	}
	out.Milestones = append(out.Milestones, snap.Milestones[id]...)
	return out
}

// taskPageOf is the desktop's inspector content: the task, its subtasks oldest first and its
// reports newest first.
func taskPageOf(snap data.Snapshot, id int) taskPage {
	out := taskPage{Subtasks: []data.Task{}, Reports: []data.Report{}}
	for _, task := range snap.Tasks {
		if task.ID == id {
			out.Task = task
		}
		if task.ParentID != nil && *task.ParentID == id {
			out.Subtasks = append(out.Subtasks, task)
		}
	}
	if len(out.Subtasks) > 1 {
		for i := 1; i < len(out.Subtasks); i++ {
			for j := i; j > 0 && out.Subtasks[j].ID < out.Subtasks[j-1].ID; j-- {
				out.Subtasks[j], out.Subtasks[j-1] = out.Subtasks[j-1], out.Subtasks[j]
			}
		}
	}
	out.Reports = append(out.Reports, snap.Reports[id]...)
	return out
}
