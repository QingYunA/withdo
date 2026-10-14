package domain_test

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestFormatTaskLine(t *testing.T) {
	base := domain.Task{ID: 12, Status: domain.StatusDoing, Priority: domain.P2, Title: "Add login"}
	withBranch := base
	withBranch.Branch = ptr("feat/login")
	withPR := withBranch
	withPR.PRNumber = ptr(int64(3))
	withURL := base
	withURL.PRURL = ptr("https://example.com/x")
	tests := []struct {
		name string
		task domain.Task
		want string
	}{
		{"plain", base, "#12 [doing] P2 — Add login"},
		{"branch", withBranch, "#12 [doing] P2 feat/login — Add login"},
		{"pr number", withPR, "#12 [doing] P2 feat/login — Add login (PR #3)"},
		{"non-github url", withURL, "#12 [doing] P2 — Add login (https://example.com/x)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { eq(t, "line", domain.FormatTaskLine(tt.task), tt.want) })
	}
}

func TestFormatOverview(t *testing.T) {
	c, clk := newCore(t)
	repo := widgetsRepo(t, "feat/login")
	st := domain.StatusDoing
	progress(t, c, domain.ProgressInput{Summary: "wiring up login", Cwd: repo, Agent: "claude-code", NextStep: "write tests", Status: &st})
	progress(t, c, domain.ProgressInput{Summary: "inbox work", Agent: "claude-code"})
	clk.advance(50 * 3600 * 1e9)

	text := domain.FormatOverview(must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{})))
	for _, want := range []string{
		"widgets (acme/widgets)", "feat/login", "STALE (50h)",
		"last: wiring up login (claude-code, 2026-10-01T00:00:00.000Z)", "next: write tests", "Inbox",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("overview is missing %q:\n%s", want, text)
		}
	}
	if strings.HasSuffix(text, "\n") {
		t.Error("overview should not end with a newline")
	}
}

func TestFormatOverviewEmptyAndNamedOnly(t *testing.T) {
	empty := domain.FormatOverview(&domain.Overview{})
	if !strings.Contains(empty, "No active tasks") {
		t.Errorf("empty overview = %q", empty)
	}
	noSlug := domain.FormatOverview(&domain.Overview{Projects: []domain.OverviewGroup{{Project: &domain.OverviewProject{Name: "local"}}}})
	eq(t, "no slug", noSlug, "local")
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"2026-12-01", "2026-12-01T00:00:00.000Z", true},
		{"2026-11-30T09:00:00+08:00", "2026-11-30T01:00:00.000Z", true},
		{"2026-12-31T00:00:00Z", "2026-12-31T00:00:00.000Z", true},
		{"2026-12-31T00:00:00.5Z", "2026-12-31T00:00:00.500Z", true},
		{"2026-12-31T10:30Z", "2026-12-31T10:30:00.000Z", true},
		{"tomorrow", "", false},
		{"2026-13-01", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := domain.ParseTimestamp(tt.in)
			if (err == nil) != tt.ok || got != tt.want {
				t.Errorf("got (%q, %v), want (%q, ok=%v)", got, err, tt.want, tt.ok)
			}
		})
	}
}

func TestStatusAndPriorityHelpers(t *testing.T) {
	for _, s := range domain.Statuses {
		if !s.Valid() {
			t.Errorf("%s should be valid", s)
		}
	}
	if domain.Status("x").Valid() || domain.Priority("P9").Valid() {
		t.Error("unknown values must be invalid")
	}
	if !domain.StatusDone.Terminal() || domain.StatusDoing.Terminal() {
		t.Error("Terminal misclassifies")
	}
	if !domain.StatusReview.Active() || domain.StatusTodo.Active() {
		t.Error("Active misclassifies")
	}
}
