package mobilesync

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
)

func desktopFixture(t *testing.T) (string, int64, int64, int64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "desktop.db")
	db, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	core := domain.New(db, domain.Options{Now: func() time.Time { return time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC) }})
	project, err := core.UpsertProject(t.Context(), domain.UpsertProjectInput{RepoSlug: "fixture/desktop", Name: "Real source project"})
	if err != nil {
		t.Fatal(err)
	}
	branch := "codex/android-client"
	parent, err := core.CreateTask(t.Context(), domain.CreateTaskInput{Title: "Existing Agent task", ProjectID: &project.ID, Source: domain.SourceAgent, Status: domain.StatusDoing, Branch: &branch})
	if err != nil {
		t.Fatal(err)
	}
	child, err := core.CreateTask(t.Context(), domain.CreateTaskInput{Title: "Existing subtask", ParentID: &parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	status := domain.StatusDoing
	if _, err := core.AddReport(t.Context(), domain.AddReportInput{TaskID: parent.ID, Agent: "codex", Summary: "Existing real progress report", Status: &status}); err != nil {
		t.Fatal(err)
	}
	return path, project.ID, parent.ID, child.ID
}

func files(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestDesktopImportReadOnlyRelationsReportsAndIdempotency(t *testing.T) {
	path, projectID, parentID, childID := desktopFixture(t)
	s := openTestStore(t)
	server := httptest.NewServer(handler(t, s))
	defer server.Close()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeInfo, _ := os.Stat(path)
	beforeFiles := files(t, filepath.Dir(path))
	const sourceID = "fixture-desktop"
	for attempt := 0; attempt < 2; attempt++ {
		summary, err := ImportDesktop(t.Context(), path, sourceID, server.URL, testToken)
		if err != nil || summary.Accepted != 4 || summary.Conflicts != 0 || summary.Rejected != 0 || len(summary.Issues) != 0 {
			t.Fatalf("import %d: %+v %v", attempt, summary, err)
		}
	}
	after, _ := os.ReadFile(path)
	afterInfo, _ := os.Stat(path)
	if !bytes.Equal(before, after) || !reflect.DeepEqual(beforeFiles, files(t, filepath.Dir(path))) || beforeInfo.ModTime() != afterInfo.ModTime() {
		t.Fatal("read-only import modified the source database or created sidecars")
	}
	snapshot, err := s.Snapshot(t.Context())
	if err != nil || snapshot.Cursor != 4 || len(snapshot.Records) != 4 {
		t.Fatalf("duplicate records or changes: %+v %v", snapshot, err)
	}
	for _, record := range snapshot.Records {
		if !validID(record.ID) || record.Version != 1 {
			t.Fatal(record)
		}
		switch record.Type {
		case "project":
			var p project
			if decodeEntity(record.Data, &p) != nil || p.ID != sourceEntityID(sourceID, "project", projectID) || p.RepoSlug != "fixture/desktop" {
				t.Fatal(string(record.Data))
			}
		case "task":
			var task task
			if decodeEntity(record.Data, &task) != nil || task.ProjectID == nil || *task.ProjectID != sourceEntityID(sourceID, "project", projectID) {
				t.Fatal(string(record.Data))
			}
			if task.ID == sourceEntityID(sourceID, "task", childID) {
				if task.ParentID == nil || *task.ParentID != sourceEntityID(sourceID, "task", parentID) {
					t.Fatal("subtask mapping lost", task)
				}
			} else if task.Source != "agent" || task.Status != "doing" || task.Branch != "codex/android-client" {
				t.Fatal("agent metadata lost", task)
			}
		case "report":
			var report report
			if decodeEntity(record.Data, &report) != nil || report.TaskID != sourceEntityID(sourceID, "task", parentID) || report.Agent != "codex" || report.Status == nil || *report.Status != "doing" {
				t.Fatal("report mapping lost", string(record.Data))
			}
		}
	}
	if sourceEntityID("another-desktop", "task", parentID) == sourceEntityID(sourceID, "task", parentID) {
		t.Fatal("integer IDs from independent desktops collide")
	}
}

func TestDesktopImportPreservesRemoteEditsAndPhysicalSourceDeletes(t *testing.T) {
	path, _, parentID, childID := desktopFixture(t)
	s := openTestStore(t)
	server := httptest.NewServer(handler(t, s))
	defer server.Close()
	const sourceID = "fixture-desktop"
	if _, err := ImportDesktop(t.Context(), path, sourceID, server.URL, testToken); err != nil {
		t.Fatal(err)
	}
	remoteID := sourceEntityID(sourceID, "task", parentID)
	record, err := getRecord(t.Context(), s.db, "task", remoteID)
	if err != nil {
		t.Fatal(err)
	}
	var mobile task
	_ = decodeEntity(record.Data, &mobile)
	mobile.Title = "Edited on Android"
	wantStatus(t, push(t, s, change("task", remoteID, 1, mobile))[0], "accepted", "")
	// Replaying a previously acknowledged source snapshot is harmless even after
	// a phone changed the record: the server returns the original receipt.
	if summary, err := ImportDesktop(t.Context(), path, sourceID, server.URL, testToken); err != nil || summary.Conflicts != 0 {
		t.Fatalf("identical snapshot retry failed: %+v %v", summary, err)
	}
	db, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	core := domain.New(db, domain.Options{})
	title := "Changed later on desktop"
	if _, err := core.UpdateTask(t.Context(), parentID, domain.TaskPatch{Title: &title}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.AddReport(t.Context(), domain.AddReportInput{TaskID: parentID, Agent: "codex", Summary: "New append-only source report"}); err != nil {
		t.Fatal(err)
	}
	if err := core.DeleteTask(t.Context(), childID); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	summary, err := ImportDesktop(t.Context(), path, sourceID, server.URL, testToken)
	if err != nil || summary.Conflicts != 1 || summary.Rejected != 0 || len(summary.Issues) != 1 || summary.Issues[0].EntityID != remoteID {
		t.Fatalf("changed source did not report conflict: %+v %v", summary, err)
	}
	record, _ = getRecord(t.Context(), s.db, "task", remoteID)
	var actual task
	_ = decodeEntity(record.Data, &actual)
	if actual.Title != mobile.Title || record.Version != 2 {
		t.Fatal("import overwrote an Android edit", actual)
	}
	child, err := getRecord(t.Context(), s.db, "task", sourceEntityID(sourceID, "task", childID))
	if err != nil || child == nil || child.Deleted {
		t.Fatalf("physical source deletion propagated: %+v %v", child, err)
	}
	snapshot, _ := s.Snapshot(t.Context())
	reports := 0
	for _, record := range snapshot.Records {
		if record.Type == "report" {
			reports++
		}
	}
	if reports != 2 {
		t.Fatal("append-only reports not retained", reports)
	}
}

func TestDesktopImportRejectsUncheckpointedAndMissingSourcesBeforeHTTP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "active-desktop.db")
	db, err := store.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++ }))
	defer server.Close()
	if _, err := ImportDesktop(t.Context(), path, "desktop", server.URL, testToken); err == nil || !strings.Contains(err.Error(), "WAL") {
		t.Fatalf("accepted active WAL source: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "missing.db")
	if _, err := ImportDesktop(t.Context(), missing, "desktop", server.URL, testToken); err == nil {
		t.Fatal("accepted missing source")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("read-only source open created a database")
	}
	if requests != 0 {
		t.Fatal("uploaded before validating source")
	}
}
