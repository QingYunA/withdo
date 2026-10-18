package data

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The pages an agent attached to a report reach the app's Report, from the real store and the fake.
func TestAReportCarriesItsLinksFromEitherStore(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store, _ *time.Time) {
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "Write the plan", Source: SourceAgent}))
		id := int64(task.ID)
		want := []Link{{URL: "https://example.com/a", Title: "A"}, {URL: "file:///tmp/b.html"}}
		switch s := s.(type) {
		case *Fake:
			s.AddReport(Report{TaskID: task.ID, Agent: "claude-code", Summary: "with pages", CreatedAt: s.Now(), Links: want})
			s.AddReport(Report{TaskID: task.ID, Agent: "claude-code", Summary: "without", CreatedAt: s.Now().Add(time.Minute)})
		case *Real:
			must[*domain.ProgressResult](t)(s.core.ReportProgress(bg, domain.ProgressInput{
				TaskID: &id, Agent: "claude-code", Summary: "with pages",
				Links: []domain.ReportLink{{URL: want[0].URL, Title: "A"}, {URL: want[1].URL}},
			}))
			must[*domain.ProgressResult](t)(s.core.ReportProgress(bg, domain.ProgressInput{TaskID: &id, Agent: "claude-code", Summary: "without"}))
		}
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
