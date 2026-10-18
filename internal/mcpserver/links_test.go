package mcpserver_test

import (
	"strings"
	"testing"
)

func linksOf(res result) []map[string]any {
	var out []map[string]any
	for _, r := range res.list("reports") {
		out = append(out, r.(map[string]any))
	}
	return out
}

func TestReportProgressTakesLinksAndGetTaskReturnsThem(t *testing.T) {
	r := newRig(t)
	a := r.ok("report_progress", map[string]any{"summary": "report page ready", "links": []map[string]any{
		{"url": "file:///tmp/report.html", "title": "Report"},
		{"url": "https://github.com/acme/widgets/pull/3"},
	}})
	if a.Data["dropped_links"] != nil {
		t.Errorf("dropped_links = %v, want none", a.Data["dropped_links"])
	}
	r.ok("report_progress", map[string]any{"summary": "no links here"})

	g := r.ok("get_task", map[string]any{"task_id": a.Data["task_id"]})
	reports := linksOf(g)
	if len(reports) != 2 {
		t.Fatalf("reports = %d", len(reports))
	}
	first, _ := reports[0]["links"].([]any)
	if len(first) != 2 {
		t.Fatalf("links of the first report = %v", reports[0]["links"])
	}
	l0 := first[0].(map[string]any)
	if l0["url"] != "file:///tmp/report.html" || l0["title"] != "Report" {
		t.Errorf("link = %v", l0)
	}
	if second, ok := reports[1]["links"].([]any); !ok || len(second) != 0 {
		t.Errorf("links of a report with none = %#v, want an empty list", reports[1]["links"])
	}
}

func TestReportProgressDropsBadLinksButFilesTheReport(t *testing.T) {
	r := newRig(t)
	a := r.ok("report_progress", map[string]any{"summary": "draft ready", "links": []map[string]any{
		{"url": "ftp://example.com/x"},
		{"url": "https://example.com/ok", "title": "Fine"},
	}})
	dropped := a.list("dropped_links")
	if len(dropped) != 1 {
		t.Fatalf("dropped_links = %v", a.Data["dropped_links"])
	}
	d := dropped[0].(map[string]any)
	if d["url"] != "ftp://example.com/x" || !strings.Contains(d["reason"].(string), "https, http and file") {
		t.Errorf("dropped = %v", d)
	}
	if !strings.Contains(a.Text, "ftp://example.com/x") {
		t.Errorf("text = %q, want it to name the dropped link", a.Text)
	}
	g := r.ok("get_task", map[string]any{"task_id": a.Data["task_id"]})
	if kept, _ := linksOf(g)[0]["links"].([]any); len(kept) != 1 {
		t.Errorf("kept = %v", linksOf(g)[0]["links"])
	}
}

func TestReportProgressFilesAReportWithOnlyBadLinks(t *testing.T) {
	r := newRig(t)
	a := r.ok("report_progress", map[string]any{"summary": "done", "links": []map[string]any{{"url": "file://relative.html"}}})
	if a.Data["created"] != true || len(a.list("dropped_links")) != 1 {
		t.Errorf("result = %+v", a.Data)
	}
}

func TestInstructionsAskForLinksToWhatCanBeOpened(t *testing.T) {
	r := newRig(t)
	if got := r.session.InitializeResult().Instructions; !strings.Contains(got, "attach it to the report with links") {
		t.Errorf("instructions lack the links sentence: %q", got)
	}
}
