package core

import (
	"encoding/json"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

// TestSeedDemo pins the phone's demo seed: a fresh folder gets app/data's workspace, a second call
// leaves it alone, and a database the reviewer already changed is never overwritten.
func TestSeedDemo(t *testing.T) {
	dir := t.TempDir()
	if err := SeedDemo(dir); err != nil {
		t.Fatalf("SeedDemo: %v", err)
	}
	phone, err := Open(dir, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	raw, err := phone.Mine()
	if err != nil {
		t.Fatalf("Mine: %v", err)
	}
	var mine data.Mine
	if err := json.Unmarshal([]byte(raw), &mine); err != nil {
		t.Fatalf("decode Mine: %v", err)
	}
	if len(mine.FromAgents) == 0 || len(mine.Today) == 0 {
		t.Errorf("the seeded Mine is empty: FromAgents %d, Today %d", len(mine.FromAgents), len(mine.Today))
	}
	raw, err = phone.Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	var projects []data.Project
	if err := json.Unmarshal([]byte(raw), &projects); err != nil {
		t.Fatalf("decode Projects: %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("seeded projects = %d, want 3", len(projects))
	}

	// A change the reviewer made survives the next demo launch.
	if _, err := phone.CreateTask(`{"title":"Keep the demo change"}`); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := phone.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := SeedDemo(dir); err != nil {
		t.Fatalf("SeedDemo again: %v", err)
	}
	phone, err = Open(dir, nil)
	if err != nil {
		t.Fatalf("Open again: %v", err)
	}
	defer func() { _ = phone.Close() }()
	raw, err = phone.Search("Keep the demo change")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	var found []data.Task
	if err := json.Unmarshal([]byte(raw), &found); err != nil {
		t.Fatalf("decode Search: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("tasks after a second SeedDemo = %d, want 1; the demo overwrote the database", len(found))
	}
	raw, err = phone.Projects()
	if err != nil {
		t.Fatalf("Projects again: %v", err)
	}
	if err := json.Unmarshal([]byte(raw), &projects); err != nil {
		t.Fatalf("decode Projects again: %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("projects after a second SeedDemo = %d, want 3", len(projects))
	}
}
