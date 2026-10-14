package domain_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

var ctx = context.Background()

// clock is a settable time source.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

var t0 = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func newCore(t *testing.T) (*domain.Core, *clock) {
	c, clk, _ := newCoreDB(t)
	return c, clk
}

// newCoreDB also returns the raw database for tests that poke at rows directly.
func newCoreDB(t *testing.T) (*domain.Core, *clock, *sql.DB) {
	t.Helper()
	testutil.IsolateGit(t)
	db, _ := testutil.OpenDB(t)
	clk := &clock{t: t0}
	return domain.New(db, domain.Options{Now: clk.now}), clk, db
}

func exec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func must[T any](t *testing.T) func(T, error) T {
	t.Helper()
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func ptr[T any](v T) *T { return &v }

func mustProject(t *testing.T, c *domain.Core, slug string) *domain.Project {
	t.Helper()
	return must[*domain.Project](t)(c.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: slug}))
}

func mustTask(t *testing.T, c *domain.Core, in domain.CreateTaskInput) *domain.Task {
	t.Helper()
	if in.Title == "" {
		in.Title = "t"
	}
	return must[*domain.Task](t)(c.CreateTask(ctx, in))
}

func eq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func eqPtr[T comparable](t *testing.T, name string, got *T, want *T) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Errorf("%s = %v, want %v", name, got, want)
	case *got != *want:
		t.Errorf("%s = %v, want %v", name, *got, *want)
	}
}

func wantFieldError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error naming %q", field)
	}
	if got := err.Error(); len(got) < len(field) || !contains(got, field) {
		t.Fatalf("error %q does not name %q", got, field)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
