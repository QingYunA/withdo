package domain_test

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestAgentActivityGroupsReportsByAgentAndSkipsTheGitHubPoller(t *testing.T) {
	c, clk := newCore(t)
	task := progress(t, c, domain.ProgressInput{Summary: "start", Agent: "claude-code", Cwd: widgetsRepo(t, "feat/a")}).Task
	clk.advance(48 * time.Hour)
	add := func(agent string, ago time.Duration) {
		t.Helper()
		must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{
			TaskID: task.ID, Agent: agent, Summary: "s", CreatedAt: domain.FormatTime(clk.now().Add(-ago)),
		}))
	}
	add("claude-code", 2*time.Hour)
	add("claude-code", 30*time.Hour)
	add("codex", 5*time.Minute)
	add(domain.GitHubAgent, time.Minute)

	got := must[[]domain.AgentActivity](t)(c.AgentActivity(ctx, domain.FormatTime(clk.now().Add(-24*time.Hour))))
	if len(got) != 2 {
		t.Fatalf("got %+v, want codex and claude-code only", got)
	}
	eq(t, "first", got[0].Agent, "codex")
	eq(t, "codex since", got[0].Since, 1)
	eq(t, "second", got[1].Agent, "claude-code")
	eq(t, "claude since", got[1].Since, 1) // the start report and the 30 h old one are before the window
	eq(t, "claude last", got[1].Last, domain.FormatTime(clk.now().Add(-2*time.Hour)))
}

func TestAgentActivityIsEmptyBeforeAnyReport(t *testing.T) {
	c, _ := newCore(t)
	got := must[[]domain.AgentActivity](t)(c.AgentActivity(ctx, domain.FormatTime(t0)))
	eq(t, "len", len(got), 0)
}
