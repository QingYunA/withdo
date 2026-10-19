package domain_test

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

// syncStep is one write against a linked database.
type syncStep func(t *testing.T, c *domain.Core, db *sql.DB)

// wantChange is a SyncChange without its seq and created_at, which the tests do not pin.
type wantChange struct {
	Entity  string
	RowID   int64
	Op      string
	Columns []string
}

// enableSync marks the database as linked to a server, as S3's linking will.
func enableSync(t *testing.T, db *sql.DB) {
	t.Helper()
	exec(t, db, "INSERT INTO sync_state (key, value) VALUES ('enabled', '1')")
}

// enableSyncWithIDs also leases a block per synced entity. The bases are far from the ids of the
// rows the tests make before linking, so an expected id is easy to read in a test.
func enableSyncWithIDs(t *testing.T, db *sql.DB) {
	t.Helper()
	enableSync(t, db)
	for _, b := range []struct {
		entity string
		next   int64
	}{
		{"tasks", 1000}, {"projects", 2000}, {"milestones", 3000}, {"categories", 4000}, {"reports", 5000},
	} {
		exec(t, db, "INSERT INTO sync_id_blocks (entity, next, end) VALUES (?, ?, ?)", b.entity, b.next, b.next+100)
	}
}

func readOutbox(t *testing.T, c *domain.Core) []domain.SyncChange {
	t.Helper()
	return must[[]domain.SyncChange](t)(c.Outbox(ctx, math.MaxInt64))
}

func clearOutbox(t *testing.T, db *sql.DB) {
	t.Helper()
	exec(t, db, "DELETE FROM sync_outbox")
}

// wantChanges checks the outbox entries against want; seq and created_at only have to be sane, and
// the entries are compared as a set because a cascading delete's order is SQLite's business.
func wantChanges(t *testing.T, got []domain.SyncChange, want []wantChange) {
	t.Helper()
	for i, ch := range got {
		if ch.Seq <= 0 || ch.CreatedAt == "" {
			t.Errorf("entry %d has no seq or created_at: %+v", i, ch)
		}
		if i > 0 && ch.Seq <= got[i-1].Seq {
			t.Errorf("seq is not increasing: %d after %d", ch.Seq, got[i-1].Seq)
		}
	}
	key := func(entity string, rowID int64, op string, columns []string) string {
		return fmt.Sprintf("%s/%d/%s/%s", entity, rowID, op, strings.Join(columns, ","))
	}
	g := append([]domain.SyncChange(nil), got...)
	w := append([]wantChange(nil), want...)
	if len(g) != len(w) {
		t.Fatalf("outbox = %s, want %s", describeChanges(g), describeChanges(w))
	}
	sort.Slice(g, func(i, j int) bool {
		return key(g[i].Entity, g[i].RowID, g[i].Op, g[i].Columns) < key(g[j].Entity, g[j].RowID, g[j].Op, g[j].Columns)
	})
	sort.Slice(w, func(i, j int) bool {
		return key(w[i].Entity, w[i].RowID, w[i].Op, w[i].Columns) < key(w[j].Entity, w[j].RowID, w[j].Op, w[j].Columns)
	})
	for i := range g {
		if g[i].Entity != w[i].Entity || g[i].RowID != w[i].RowID || g[i].Op != w[i].Op || !slices.Equal(g[i].Columns, w[i].Columns) {
			t.Errorf("entry %d = %s, want %s", i, describeChange(g[i]), describeChange(w[i]))
		}
	}
}

func describeChanges[T any](changes []T) string {
	parts := make([]string, len(changes))
	for i, ch := range changes {
		switch v := any(ch).(type) {
		case domain.SyncChange:
			parts[i] = describeChange(v)
		case wantChange:
			parts[i] = describeChange(v)
		}
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func describeChange[T any](ch T) string {
	switch v := any(ch).(type) {
	case domain.SyncChange:
		return fmt.Sprintf("%s#%d %s %v", v.Entity, v.RowID, v.Op, v.Columns)
	case wantChange:
		return fmt.Sprintf("%s#%d %s %v", v.Entity, v.RowID, v.Op, v.Columns)
	}
	return ""
}

// TestEveryWritePathIsCaptured is acceptance criterion 1: every exported Core method that writes a
// synced table records what it changed on a linked database. Each case sets its rows up while the
// database is still unlinked, links it, then checks the entries the one write adds.
func TestEveryWritePathIsCaptured(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange)
	}{
		{
			name: "create a task",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					mustTask(t, c, domain.CreateTaskInput{Title: "one"})
				}, []wantChange{{"tasks", 1000, "put", nil}}
			},
		},
		{
			name: "update a task",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				task := mustTask(t, c, domain.CreateTaskInput{Title: "one"})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Title: ptr("two")}))
				}, []wantChange{{"tasks", task.ID, "put", []string{"title", "updated_at"}}}
			},
		},
		{
			name: "complete a task",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				task := mustTask(t, c, domain.CreateTaskInput{Title: "one"})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Status: ptr(domain.StatusDone)}))
				}, []wantChange{{"tasks", task.ID, "put", []string{"status", "updated_at", "completed_at"}}}
			},
		},
		{
			name: "delete a task",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				task := mustTask(t, c, domain.CreateTaskInput{Title: "one"})
				report := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "s"}))
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					if err := c.DeleteTask(ctx, task.ID); err != nil {
						t.Fatal(err)
					}
				}, []wantChange{{"reports", report.ID, "delete", nil}, {"tasks", task.ID, "delete", nil}}
			},
		},
		{
			name: "restore a task",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				task := mustTask(t, c, domain.CreateTaskInput{Title: "one"})
				report := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "s"}))
				snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, task.ID))
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					if err := c.DeleteTask(ctx, task.ID); err != nil {
						t.Fatal(err)
					}
					clearOutbox(t, db) // the delete is another case; only the restore is under test here
					must[*domain.Task](t)(c.RestoreTask(ctx, *snap))
				}, []wantChange{{"reports", report.ID, "put", nil}, {"tasks", task.ID, "put", nil}}
			},
		},
		{
			name: "bulk update tasks",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
				b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						must[[]domain.Task](t)(c.UpdateTasks(ctx, []int64{a.ID, b.ID}, domain.TaskPatch{Title: ptr("x")}))
					}, []wantChange{
						{"tasks", a.ID, "put", []string{"title", "updated_at"}},
						{"tasks", b.ID, "put", []string{"title", "updated_at"}},
					}
			},
		},
		{
			name: "bulk delete tasks",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
				b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
				ra := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: a.ID, Agent: "x", Summary: "a"}))
				rb := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: b.ID, Agent: "x", Summary: "b"}))
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						must[[]domain.TaskSnapshot](t)(c.DeleteTasks(ctx, []int64{a.ID, b.ID}))
					}, []wantChange{
						{"reports", ra.ID, "delete", nil}, {"tasks", a.ID, "delete", nil},
						{"reports", rb.ID, "delete", nil}, {"tasks", b.ID, "delete", nil},
					}
			},
		},
		{
			name: "add a report",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				task := mustTask(t, c, domain.CreateTaskInput{Title: "one"})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "s"}))
				}, []wantChange{{"reports", 5000, "put", nil}}
			},
		},
		{
			name: "accept a handed task with a report",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				task := handedTask(t, c)
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "on it"}))
					}, []wantChange{
						{"reports", 5000, "put", nil},
						{"tasks", task.ID, "put", []string{"handed_to", "handed_at"}},
					}
			},
		},
		{
			name: "create a project",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "Garden"}))
				}, []wantChange{{"projects", 2000, "put", nil}}
			},
		},
		{
			name: "rename a project",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				p := mustProject(t, c, "acme/widgets")
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Project](t)(c.RenameProject(ctx, p.ID, "Widget factory"))
				}, []wantChange{{"projects", p.ID, "put", []string{"name", "updated_at"}}}
			},
		},
		{
			name: "archive a project",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				p := mustProject(t, c, "acme/widgets")
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Project](t)(c.ArchiveProject(ctx, p.ID))
				}, []wantChange{{"projects", p.ID, "put", []string{"updated_at", "archived_at"}}}
			},
		},
		{
			name: "create a sub-project",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				parent := mustProject(t, c, "acme/widgets")
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "Widgets docs", ParentID: &parent.ID}))
				}, []wantChange{{"projects", 2000, "put", nil}}
			},
		},
		{
			name: "delete a project with its reports, tasks and milestones",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				p := mustProject(t, c, "acme/widgets")
				m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
				a := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID, MilestoneID: &m.ID})
				b := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID})
				ra := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: a.ID, Agent: "x", Summary: "a"}))
				rb := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: b.ID, Agent: "x", Summary: "b"}))
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						if _, err := c.DeleteProject(ctx, p.ID); err != nil {
							t.Fatal(err)
						}
					}, []wantChange{
						{"reports", ra.ID, "delete", nil}, {"reports", rb.ID, "delete", nil},
						{"tasks", a.ID, "delete", nil}, {"tasks", b.ID, "delete", nil},
						{"milestones", m.ID, "delete", nil}, {"projects", p.ID, "delete", nil},
					}
			},
		},
		{
			name: "create a milestone",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				p := mustProject(t, c, "acme/widgets")
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
				}, []wantChange{{"milestones", 3000, "put", nil}}
			},
		},
		{
			name: "update a milestone",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				p := mustProject(t, c, "acme/widgets")
				m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Milestone](t)(c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{Title: ptr("v2")}))
				}, []wantChange{{"milestones", m.ID, "put", []string{"title"}}}
			},
		},
		{
			name: "create a category",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Category](t)(c.CreateCategory(ctx, domain.CreateCategoryInput{Name: "Reading", Icon: dot}))
				}, []wantChange{{"categories", 4000, "put", nil}}
			},
		},
		{
			name: "update a category",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				cat := newCategory(t, c, "Reading")
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Category](t)(c.UpdateCategory(ctx, cat.ID, domain.CategoryPatch{Name: ptr("Books")}))
				}, []wantChange{{"categories", cat.ID, "put", []string{"name"}}}
			},
		},
		{
			name: "reorder categories",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				newCategory(t, c, "Reading")
				garden := newCategory(t, c, "Garden")
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						if err := c.MoveCategory(ctx, garden.ID, 0); err != nil {
							t.Fatal(err)
						}
					}, []wantChange{
						{"categories", 1, "put", []string{"position"}},
						{"categories", 2, "put", []string{"position"}},
						{"categories", 3, "put", []string{"position"}},
						{"categories", 4, "put", []string{"position"}},
					}
			},
		},
		{
			name: "delete a category and unsort its tasks",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				reading := newCategory(t, c, "Reading")
				garden := newCategory(t, c, "Garden")
				task := mustTask(t, c, domain.CreateTaskInput{Title: "one", CategoryID: &reading.ID})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						must[domain.DeletedCategory](t)(c.DeleteCategory(ctx, reading.ID))
					}, []wantChange{
						{"categories", reading.ID, "delete", nil},
						{"tasks", task.ID, "put", []string{"category_id"}},
						{"categories", garden.ID, "put", []string{"position"}},
					}
			},
		},
		{
			name: "close a task and its subtasks",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				parent := mustTask(t, c, domain.CreateTaskInput{Title: "parent"})
				sub := mustTask(t, c, domain.CreateTaskInput{Title: "sub", ParentID: &parent.ID})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						must[*domain.Task](t)(c.UpdateTask(ctx, parent.ID, domain.TaskPatch{Status: ptr(domain.StatusDone)}))
					}, []wantChange{
						{"tasks", parent.ID, "put", []string{"status", "updated_at", "completed_at"}},
						{"tasks", sub.ID, "put", []string{"status", "updated_at", "completed_at"}},
						{"reports", 5000, "put", nil},
					}
			},
		},
		{
			name: "record pull request state",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				url := "https://github.com/acme/widgets/pull/7"
				task := mustTask(t, c, domain.CreateTaskInput{Title: "one", PRURL: &url})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
						must[bool](t)(c.RecordPR(ctx, task.ID, domain.PRUpdate{
							URL:      url,
							Snapshot: domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksPassing, Review: domain.PRReviewNone},
							Reports:  []domain.PRReport{{Kind: domain.KindChecksFailing, Summary: "checks failed"}},
						}))
					}, []wantChange{
						{"tasks", task.ID, "put", []string{"pr_state", "pr_checks", "pr_review", "pr_checked_at"}},
						{"reports", 5000, "put", nil},
					}
			},
		},
		{
			name: "hand off a task",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				p := mustProject(t, c, "acme/widgets")
				task := mustTask(t, c, domain.CreateTaskInput{Title: "ship it", ProjectID: &p.ID, Status: domain.StatusDone})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Task](t)(c.HandOff(ctx, task.ID))
				}, []wantChange{{"tasks", task.ID, "put", []string{"status", "source", "updated_at", "completed_at", "handed_to", "handed_at"}}}
			},
		},
		{
			name: "reparent a task",
			prepare: func(t *testing.T, c *domain.Core, db *sql.DB) (syncStep, []wantChange) {
				parent := mustTask(t, c, domain.CreateTaskInput{Title: "parent"})
				child := mustTask(t, c, domain.CreateTaskInput{Title: "child"})
				return func(t *testing.T, c *domain.Core, db *sql.DB) {
					must[*domain.Task](t)(c.UpdateTask(ctx, child.ID, domain.TaskPatch{ParentID: domain.Set(parent.ID)}))
				}, []wantChange{{"tasks", child.ID, "put", []string{"updated_at", "parent_id"}}}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, clk, db := newCoreDB(t)
			write, want := tc.prepare(t, c, db)
			enableSyncWithIDs(t, db)
			clk.advance(time.Hour) // a later stamp makes updated_at a changed synced column
			write(t, c, db)
			wantChanges(t, readOutbox(t, c), want)
		})
	}
}

// TestLocalOnlyChangesRecordNothing is acceptance criterion 2: the device-local columns are not
// synced, so changing only one of them records nothing.
func TestLocalOnlyChangesRecordNothing(t *testing.T) {
	t.Run("claiming a due reminder", func(t *testing.T) {
		c, _, db := newCoreDB(t)
		due := domain.FormatTime(t0)
		task := mustTask(t, c, domain.CreateTaskInput{Title: "ring", RemindAt: &due})
		enableSync(t, db)
		claimed := must[[]domain.Reminder](t)(c.ClaimDueReminders(ctx, t0.Add(time.Minute)))
		eq(t, "claimed", len(claimed), 1)
		eq(t, "task", claimed[0].TaskID, task.ID)
		wantChanges(t, readOutbox(t, c), nil)
	})

	t.Run("a project's repo_path update from git", func(t *testing.T) {
		c, _, db := newCoreDB(t)
		p := mustProject(t, c, "acme/widgets")
		dir := widgetsRepo(t, "main")
		enableSync(t, db)
		resolved := must[*domain.ResolvedProject](t)(c.ResolveProjectFromCwd(ctx, dir))
		eq(t, "project", resolved.Project.ID, p.ID)
		eqPtr(t, "repo_path", resolved.Project.RepoPath, &dir)
		wantChanges(t, readOutbox(t, c), nil)
	})

	t.Run("a report's cwd", func(t *testing.T) {
		c, _, db := newCoreDB(t)
		task := mustTask(t, c, domain.CreateTaskInput{Title: "one"})
		report := must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "s"}))
		enableSync(t, db)
		// No Core method changes a report after it is filed; this is the trigger's contract for cwd.
		exec(t, db, "UPDATE reports SET cwd = '/elsewhere' WHERE id = ?", report.ID)
		wantChanges(t, readOutbox(t, c), nil)
	})
}

// TestUnlinkedOrApplyingWritesRecordNothing is acceptance criterion 3: an unlinked database and a
// write made while applying server changes add no outbox rows, and the applying flag ends up off.
func TestUnlinkedOrApplyingWritesRecordNothing(t *testing.T) {
	t.Run("unlinked", func(t *testing.T) {
		c, _ := newCore(t)
		p := mustProject(t, c, "acme/widgets")
		task := mustTask(t, c, domain.CreateTaskInput{Title: "one", ProjectID: &p.ID})
		must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Title: ptr("two")}))
		must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "s"}))
		must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
		newCategory(t, c, "Reading")
		if err := c.DeleteTask(ctx, task.ID); err != nil {
			t.Fatal(err)
		}
		wantChanges(t, readOutbox(t, c), nil)
	})

	t.Run("applying", func(t *testing.T) {
		c, _, db := newCoreDB(t)
		enableSync(t, db)
		err := c.ApplyChanges(ctx, func(q store.Querier) error {
			if _, err := q.ExecContext(ctx, "INSERT INTO projects (id, name, created_at, updated_at) VALUES (1, 'from server', 't', 't')"); err != nil {
				return err
			}
			_, err := q.ExecContext(ctx, "INSERT INTO tasks (id, title, source, created_at, updated_at) VALUES (1, 'from server', 'user', 't', 't')")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "projects written", scalarCount(t, db, "SELECT COUNT(*) FROM projects"), 1)
		eq(t, "tasks written", scalarCount(t, db, "SELECT COUNT(*) FROM tasks"), 1)
		wantChanges(t, readOutbox(t, c), nil)
		eq(t, "applying flag", applyingFlag(t, db), "")
	})

	t.Run("applying is reset when the write fails", func(t *testing.T) {
		c, _, db := newCoreDB(t)
		enableSync(t, db)
		before := scalarCount(t, db, "SELECT COUNT(*) FROM projects")
		boom := errors.New("boom")
		err := c.ApplyChanges(ctx, func(q store.Querier) error {
			if _, err := q.ExecContext(ctx, "INSERT INTO projects (id, name, created_at, updated_at) VALUES (1, 'from server', 't', 't')"); err != nil {
				return err
			}
			return boom
		})
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
		eq(t, "projects written", scalarCount(t, db, "SELECT COUNT(*) FROM projects"), before)
		wantChanges(t, readOutbox(t, c), nil)
		eq(t, "applying flag", applyingFlag(t, db), "")
	})
}

func applyingFlag(t *testing.T, db *sql.DB) string {
	t.Helper()
	var value string
	if err := db.QueryRowContext(ctx,
		"SELECT COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '')").Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

// TestOutboxAckDeletesThroughSeq covers the uploader's other half: Outbox reads up to a seq and
// DeleteOutbox acknowledges exactly those entries.
func TestOutboxAckDeletesThroughSeq(t *testing.T) {
	c, _, db := newCoreDB(t)
	enableSyncWithIDs(t, db)
	mustTask(t, c, domain.CreateTaskInput{Title: "one"})
	second := mustTask(t, c, domain.CreateTaskInput{Title: "two"})
	changes := readOutbox(t, c)
	eq(t, "changes", len(changes), 2)

	if err := c.DeleteOutbox(ctx, changes[0].Seq); err != nil {
		t.Fatal(err)
	}
	left := readOutbox(t, c)
	eq(t, "changes left", len(left), 1)
	eq(t, "seq left", left[0].Seq, changes[1].Seq)
	eq(t, "row left", left[0].RowID, second.ID)

	if err := c.DeleteOutbox(ctx, left[0].Seq); err != nil {
		t.Fatal(err)
	}
	eq(t, "changes left after the last ack", len(readOutbox(t, c)), 0)
}

// TestIdBlocksHandOutIdsAndFailWhenEmpty is acceptance criterion 4.
func TestIdBlocksHandOutIdsAndFailWhenEmpty(t *testing.T) {
	t.Run("a block hands out its ids then fails", func(t *testing.T) {
		c, _, db := newCoreDB(t)
		enableSync(t, db)
		exec(t, db, "INSERT INTO sync_id_blocks (entity, next, end) VALUES ('tasks', 1000, 1003)")
		for i, want := range []int64{1000, 1001, 1002} {
			task := mustTask(t, c, domain.CreateTaskInput{Title: fmt.Sprintf("t%d", i)})
			eq(t, "id", task.ID, want)
		}
		if n := scalarCount(t, db, "SELECT COUNT(*) FROM sync_id_blocks WHERE entity = 'tasks'"); n != 0 {
			t.Errorf("the spent block is still there: %d rows", n)
		}
		tasksBefore := scalarCount(t, db, "SELECT COUNT(*) FROM tasks")
		outboxBefore := len(readOutbox(t, c))
		_, err := c.CreateTask(ctx, domain.CreateTaskInput{Title: "four"})
		if !errors.Is(err, domain.ErrNoSyncIDs) {
			t.Fatalf("err = %v, want ErrNoSyncIDs", err)
		}
		eq(t, "tasks written", scalarCount(t, db, "SELECT COUNT(*) FROM tasks"), tasksBefore)
		eq(t, "outbox entries", len(readOutbox(t, c)), outboxBefore)
	})

	t.Run("an unlinked database assigns ids as before", func(t *testing.T) {
		c, _ := newCore(t)
		for i, want := range []int64{1, 2, 3} {
			task := mustTask(t, c, domain.CreateTaskInput{Title: fmt.Sprintf("t%d", i)})
			eq(t, "id", task.ID, want)
		}
	})

	t.Run("the smallest block is used first", func(t *testing.T) {
		c, _, db := newCoreDB(t)
		enableSync(t, db)
		exec(t, db, "INSERT INTO sync_id_blocks (entity, next, end) VALUES ('tasks', 2000, 2002)")
		exec(t, db, "INSERT INTO sync_id_blocks (entity, next, end) VALUES ('tasks', 1000, 1002)")
		for i, want := range []int64{1000, 1001, 2000} {
			task := mustTask(t, c, domain.CreateTaskInput{Title: fmt.Sprintf("t%d", i)})
			eq(t, "id", task.ID, want)
		}
	})
}

// TestLinkedCoresNeverHandOutTheSameID is acceptance criterion 6: two Cores on one file, as two
// processes would be, spend one linked id block without handing out the same id.
func TestLinkedCoresNeverHandOutTheSameID(t *testing.T) {
	testutil.IsolateGit(t)
	path := filepath.Join(t.TempDir(), "shared.db")
	dbA, _ := testutil.OpenDBAt(t, path)
	dbB, _ := testutil.OpenDBAt(t, path)
	cores := []*domain.Core{domain.New(dbA, domain.Options{}), domain.New(dbB, domain.Options{})}
	enableSync(t, dbA)
	exec(t, dbA, "INSERT INTO sync_id_blocks (entity, next, end) VALUES ('tasks', 1000, 1100)")

	const perCore = 20
	var wg sync.WaitGroup
	ids := make(chan int64, 2*perCore)
	errs := make(chan error, 2*perCore)
	for i, c := range cores {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perCore; j++ {
				task, err := c.CreateTask(ctx, domain.CreateTaskInput{Title: fmt.Sprintf("c%d-%d", i, j)})
				if err != nil {
					errs <- err
					continue
				}
				ids <- task.ID
			}
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	seen := map[int64]bool{}
	for id := range ids {
		if id < 1000 || id >= 1100 {
			t.Errorf("id %d is outside the block", id)
		}
		if seen[id] {
			t.Errorf("id %d was handed out twice", id)
		}
		seen[id] = true
	}
	eq(t, "tasks", len(seen), 2*perCore)
	eq(t, "outbox entries", len(readOutbox(t, cores[0])), 2*perCore)
}

func scalarCount(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
