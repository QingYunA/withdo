package data

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestRealAgentActivityCountsTodayAndSkipsTheGitHubPoller(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	r, core := realPair(t, func() time.Time { return now })
	task, err := core.ReportProgress(bg, domain.ProgressInput{Summary: "start", Agent: "codex", Cwd: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []domain.AddReportInput{
		{TaskID: task.Task.ID, Agent: "claude-code", Summary: "a", CreatedAt: domain.FormatTime(now.Add(-time.Hour))},
		{TaskID: task.Task.ID, Agent: "claude-code", Summary: "b", CreatedAt: domain.FormatTime(now.Add(-26 * time.Hour))},
		{TaskID: task.Task.ID, Agent: domain.GitHubAgent, Summary: "PR merged", CreatedAt: domain.FormatTime(now)},
	} {
		if _, err := core.AddReport(bg, in); err != nil {
			t.Fatal(err)
		}
	}
	got, err := r.AgentActivity(now.Add(-12 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Agent != "codex" || got[0].Since != 1 || got[1].Agent != "claude-code" || got[1].Since != 1 {
		t.Fatalf("got %+v", got)
	}
	if want := now.Add(-time.Hour); !got[1].Last.Equal(want) {
		t.Errorf("claude-code last %v, want %v", got[1].Last, want)
	}
}

func TestAgentActivityOfTheSeed(t *testing.T) {
	f := seededDemo(t)
	f.SeedReportAs(1, "codex", "x", fixedNow().Add(-6*time.Minute))
	got, err := f.AgentActivity(fixedNow().Add(-12 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Agent != "claude-code" || got[1].Agent != "codex" || got[1].Since != 1 {
		t.Fatalf("got %+v", got) // the seed's claude-code reports are the latest; its GitHub one is left out
	}
	for _, a := range got {
		if a.Agent == GitHubAgent {
			t.Error("the GitHub poller is listed as an agent")
		}
	}
}
