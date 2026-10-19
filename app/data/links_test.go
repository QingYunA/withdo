package data

import (
	"testing"
	"time"
)

// The pages an agent attached to a report reach the app's Report.
func TestAReportCarriesItsLinks(t *testing.T) {
	withStore(t, func(t *testing.T, s *Real, clock *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan", Source: SourceAgent}))
		want := []Link{{URL: "https://example.com/a", Title: "A"}, {URL: "file:///tmp/b.html"}}
		s.SeedReport(Report{TaskID: task.ID, Agent: "claude-code", Summary: "with pages", CreatedAt: *clock, Links: want})
		s.SeedReport(Report{TaskID: task.ID, Agent: "claude-code", Summary: "without", CreatedAt: clock.Add(time.Minute)})
		reports := must[TaskDetail](t)(s.GetTask(task.ID)).Reports // newest first
		if len(reports) != 2 {
			t.Fatalf("reports = %d", len(reports))
		}
		if len(reports[0].Links) != 0 {
			t.Errorf("a report without pages has %v", reports[0].Links)
		}
		if got := reports[1].Links; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("links = %v, want %v", got, want)
		}
	})
}
