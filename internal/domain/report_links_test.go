package domain_test

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestCleanLinksKeepsWebAndFileLinks(t *testing.T) {
	in := []domain.ReportLink{
		{URL: " https://github.com/acme/widgets/pull/19 ", Title: " PR #19 "},
		{URL: "http://localhost:8080/report.html"},
		{URL: "file:///Users/me/report.html", Title: "Report"},
		{URL: "file://localhost/tmp/a b.html"},
	}
	kept, dropped := domain.CleanLinks(in)
	if len(dropped) != 0 {
		t.Fatalf("dropped = %+v", dropped)
	}
	eq(t, "kept", len(kept), 4)
	eq(t, "trimmed url", kept[0].URL, "https://github.com/acme/widgets/pull/19")
	eq(t, "trimmed title", kept[0].Title, "PR #19")
}

func TestCleanLinksDropsWhatItCannotKeepAndSaysWhy(t *testing.T) {
	cases := []struct {
		name string
		link domain.ReportLink
		want string
	}{
		{"ftp", domain.ReportLink{URL: "ftp://example.com/x"}, "only https, http and file URLs"},
		{"javascript", domain.ReportLink{URL: "javascript:alert(1)"}, "only https, http and file URLs"},
		{"no scheme", domain.ReportLink{URL: "example.com/x"}, "only https, http and file URLs"},
		{"no host", domain.ReportLink{URL: "https:///x"}, "needs a host"},
		{"relative file", domain.ReportLink{URL: "file://report.html"}, "absolute path"},
		{"file on another host", domain.ReportLink{URL: "file://nas/share/a.html"}, "absolute path"},
		{"empty", domain.ReportLink{URL: "  "}, "empty"},
		{"unparsable", domain.ReportLink{URL: "https://exa mple.com/%zz"}, "not a valid URL"},
		{"long title", domain.ReportLink{URL: "https://example.com", Title: strings.Repeat("x", 81)}, "longer than 80"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kept, dropped := domain.CleanLinks([]domain.ReportLink{c.link})
			eq(t, "kept", len(kept), 0)
			eq(t, "dropped", len(dropped), 1)
			if !strings.Contains(dropped[0].Reason, c.want) {
				t.Errorf("reason = %q, want it to mention %q", dropped[0].Reason, c.want)
			}
		})
	}
}

func TestCleanLinksCountsTitleInCharactersNotBytes(t *testing.T) {
	kept, dropped := domain.CleanLinks([]domain.ReportLink{{URL: "https://example.com", Title: strings.Repeat("页", 80)}})
	eq(t, "kept", len(kept), 1)
	eq(t, "dropped", len(dropped), 0)
}

func TestCleanLinksKeepsFiveAndDropsTheRest(t *testing.T) {
	var in []domain.ReportLink
	for i := 0; i < 7; i++ {
		in = append(in, domain.ReportLink{URL: "https://example.com/" + string(rune('a'+i))})
	}
	kept, dropped := domain.CleanLinks(in)
	eq(t, "kept", len(kept), 5)
	eq(t, "dropped", len(dropped), 2)
	eq(t, "first dropped", dropped[0].URL, "https://example.com/f")
}

func TestCleanLinksOfNothingIsAnEmptyList(t *testing.T) {
	kept, dropped := domain.CleanLinks(nil)
	if kept == nil || len(kept) != 0 || len(dropped) != 0 {
		t.Errorf("kept = %#v, dropped = %#v", kept, dropped)
	}
}

func TestReportProgressStoresLinksOnTheReport(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	first := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: dir, Links: []domain.ReportLink{
		{URL: "https://example.com/a", Title: "A"}, {URL: "file:///tmp/b.html"},
	}})
	eq(t, "result links", len(first.Report.Links), 2)
	eq(t, "nothing dropped", len(first.Dropped), 0)
	progress(t, c, domain.ProgressInput{Summary: "two", Cwd: dir})

	got := must[[]domain.Report](t)(c.ListReports(ctx, first.Task.ID))
	eq(t, "reports", len(got), 2)
	eq(t, "stored links", len(got[0].Links), 2)
	eq(t, "stored url", got[0].Links[1].URL, "file:///tmp/b.html")
	eq(t, "stored title", got[0].Links[0].Title, "A")
	if got[1].Links == nil || len(got[1].Links) != 0 {
		t.Errorf("a report without links should read as an empty list, got %#v", got[1].Links)
	}
}

// A report whose links are all bad is still filed, and the agent is told what was dropped.
func TestReportProgressFilesAReportWhoseLinksAreAllBad(t *testing.T) {
	c, _ := newCore(t)
	r := progress(t, c, domain.ProgressInput{Summary: "done", Cwd: widgetsRepo(t, "feat/login"), Links: []domain.ReportLink{
		{URL: "ftp://example.com/x"}, {URL: "file://relative.html"},
	}})
	eq(t, "task created", r.Created, true)
	eq(t, "report links", len(r.Report.Links), 0)
	eq(t, "dropped", len(r.Dropped), 2)
	eq(t, "dropped url", r.Dropped[0].URL, "ftp://example.com/x")
	got := must[[]domain.Report](t)(c.ListReports(ctx, r.Task.ID))
	eq(t, "reports", len(got), 1)
}

func TestRestoreTaskPutsTheReportLinksBack(t *testing.T) {
	c, _ := newCore(t)
	r := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: widgetsRepo(t, "feat/login"), Links: []domain.ReportLink{{URL: "https://example.com/a", Title: "A"}}})
	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, r.Task.ID))
	noErr(t, c.DeleteTask(ctx, r.Task.ID))
	must[*domain.Task](t)(c.RestoreTask(ctx, *snap))
	got := must[[]domain.Report](t)(c.ListReports(ctx, r.Task.ID))
	eq(t, "links", len(got[0].Links), 1)
	eq(t, "title", got[0].Links[0].Title, "A")
}
