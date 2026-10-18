package views

import (
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// Pages an agent attached to a report: the chips in the inspector, the Open page button of a task in
// review, and the Summary export.

const (
	pageURL    = "https://example.com/design/draft.html"
	pageFile   = "file:///Users/demo/reports/crash-report.html"
	goneFile   = "file:///Users/demo/reports/import-report.html"
	pullURL    = "https://github.com/QingYunA/adhd-todo/pull/19"
	pageHome   = "/Users/demo"
	pageReport = "Opened the design draft"
)

// linksApp is the seeded fake where the review task 2 ("Native UI with mygo", a pull request) has a
// newer report with pages attached, and the file the import report names is gone.
func linksApp(t *testing.T) (*App, *data.Fake, *recorder) {
	t.Helper()
	app, store, rec := newTestApp(t)
	app.Config.Home = pageHome
	app.FileExists = func(p string) bool { return p != "/Users/demo/reports/import-report.html" }
	store.AddReport(data.Report{
		TaskID: 2, Agent: "claude-code", Status: data.Review, Summary: pageReport, CreatedAt: fixedNow().Add(-10 * time.Minute),
		Links: []data.Link{
			{URL: pageURL, Title: "Design draft"}, {URL: pageFile}, {URL: pullURL}, {URL: goneFile, Title: "Import report"},
		},
	})
	app.Refresh()
	return app, store, rec
}

func TestAChipSaysTheTitleThenTheFileNameThenTheHostAndTheEndOfThePath(t *testing.T) {
	for _, c := range []struct{ link, want string }{
		{"https://example.com/a/b", "Given"},
		{"file:///Users/demo/reports/crash-report.html", "crash-report.html"},
		{"https://github.com/QingYunA/adhd-todo/pull/19", "github.com › pull/19"},
		{"https://example.com/report.html", "example.com › report.html"},
		{"https://example.com/docs/report.html", "example.com › docs/report.html"},
		{"https://example.com", "example.com"},
		{"https://example.com/", "example.com"},
	} {
		l := data.Link{URL: c.link}
		if c.want == "Given" {
			l.Title = "Given"
		}
		if got := linkLabel(l); got != c.want {
			t.Errorf("linkLabel(%q) = %q, want %q", c.link, got, c.want)
		}
	}
}

func TestTheInspectorShowsAReportsPagesAsChips(t *testing.T) {
	app, _, _ := linksApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.OpenTask(2)
	tt.Frame()
	has(t, tt, "Design draft", "crash-report.html", "github.com › pull/19", "Import report")
}

func TestClickingAChipOpensTheWebPageInTheBrowserAndAFileWithItsApp(t *testing.T) {
	app, _, rec := linksApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.OpenTask(2)
	tt.Frame()
	if err := tt.Click("Design draft"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("crash-report.html"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || rec.urls[0] != pageURL {
		t.Errorf("urls = %v", rec.urls)
	}
	if len(rec.files) != 1 || rec.files[0] != "/Users/demo/reports/crash-report.html" {
		t.Errorf("files = %v", rec.files)
	}
}

func TestAFileThatIsGoneSaysSoOnClickAndOpensNothing(t *testing.T) {
	app, _, rec := linksApp(t)
	cur := fixedNow()
	app.Now = func() time.Time { return cur }
	tt, m, _ := newTesterFor(app, 1100)
	m.OpenTask(2)
	tt.Frame()
	tt.Scroll(950, 400, 0, 3000) // the chip and its tip at the end of Activity, on any font's line heights
	tt.Frame()
	hasNot(t, tt, "File not found")
	if err := tt.Click("Import report"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	has(t, tt, "File not found")
	if len(rec.files) != 0 || len(rec.urls) != 0 {
		t.Errorf("opened %v %v", rec.files, rec.urls)
	}
	cur = cur.Add(linkTipFor + time.Second)
	tt.Frame()
	hasNot(t, tt, "File not found")
}

func TestAFileLinkShowsItsPathFromTheHomeFolder(t *testing.T) {
	app, _, _ := linksApp(t)
	m := NewMain(app)
	if got := m.linkAddress(data.Link{URL: pageFile}); got != "~/reports/crash-report.html" {
		t.Errorf("address = %q", got)
	}
	if got := m.linkAddress(data.Link{URL: "file:///tmp/x.html"}); got != "/tmp/x.html" {
		t.Errorf("address = %q", got)
	}
	if got := m.linkAddress(data.Link{URL: pageURL}); got != pageURL {
		t.Errorf("address = %q", got)
	}
}

func TestAReviewRowOffersOpenPageBeforeOpenPRAndAccept(t *testing.T) {
	app, _, rec := linksApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	has(t, tt, "Open page", "Open PR", "Accept")
	x := func(text string) float32 {
		r, ok := tt.Find(text)
		if !ok {
			t.Fatalf("no %q", text)
		}
		return r.X
	}
	if !(x("Open page") < x("Open PR") && x("Open PR") < x("Accept")) {
		t.Errorf("order: Open page %v, Open PR %v, Accept %v", x("Open page"), x("Open PR"), x("Accept"))
	}
	if err := tt.Click("Open page: Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	if len(rec.urls) != 1 || rec.urls[0] != pageURL {
		t.Errorf("the button opens the first link of the latest report: %v", rec.urls)
	}
	if m.openID != 0 {
		t.Error("the button also opened the inspector")
	}
}

func TestAReviewRowWithoutAPullRequestOffersOpenPageAndAcceptInsteadOfReview(t *testing.T) {
	app, store, _ := newTestApp(t)
	task, err := store.CreateTask(data.CreateTaskInput{Title: "Tidy the docs", Source: data.SourceAgent, Status: data.Review, ProjectID: data.IntPtr(1)})
	if err != nil {
		t.Fatal(err)
	}
	store.AddReport(data.Report{TaskID: task.ID, Agent: "claude-code", Summary: "Docs page", CreatedAt: fixedNow(), Links: []data.Link{{URL: pageURL}}})
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, "Open page", "Accept")
	hasNot(t, tt, "Review")
}

func TestOnlyATaskInReviewGetsOpenPage(t *testing.T) {
	app, store, _ := newTestApp(t)
	// Task 3 is blocked: it waits for an answer, not for a look.
	store.AddReport(data.Report{TaskID: 3, Agent: "claude-code", Status: data.Blocked, Summary: "Need a call", CreatedAt: fixedNow(), Links: []data.Link{{URL: pageURL}}})
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	hasNot(t, tt, "Open page")
	has(t, tt, "Answer")
}

func TestAnAgentsRowAndACardShowOpenPageWhenHovered(t *testing.T) {
	app, _, _ := linksApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	hasNot(t, tt, "Open page")
	hover(t, tt, "Native UI with mygo")
	has(t, tt, "Open page", "Open PR", "Accept")
	m.board[1] = true
	tt.Move(2, 700)
	tt.Frame()
	tt.Frame()
	hasNot(t, tt, "Open page")
	hover(t, tt, "Native UI with mygo")
	has(t, tt, "Open page", "Open PR", "Accept")
}

func TestTheSummaryExportEndsATaskLineWithItsPagesOnce(t *testing.T) {
	app, store, _ := summaryApp(t)
	store.AddReport(data.Report{
		TaskID: 2, Agent: "claude-code", Summary: "Draft", CreatedAt: fixedNow().Add(-time.Hour),
		Links: []data.Link{{URL: pageURL, Title: "Design [v2] draft"}, {URL: "file:///tmp/a b (1).html"}},
	})
	store.AddReport(data.Report{
		TaskID: 2, Agent: "claude-code", Summary: "Again", CreatedAt: fixedNow().Add(-30 * time.Minute),
		Links: []data.Link{{URL: pageURL, Title: "Design [v2] draft"}},
	})
	app.Refresh()
	_, m := summaryMain(t, app, false)
	md := m.summaryMarkdown(m.summaryFor(app.Now()))
	want := " [Design \\[v2\\] draft](" + pageURL + ") [a b (1).html](file:///tmp/a%20b%20%281%29.html)\n"
	if !strings.Contains(md, want) {
		t.Errorf("the Markdown lacks %q:\n%s", want, md)
	}
	if strings.Count(md, pageURL) != 1 {
		t.Errorf("a page attached twice shows once:\n%s", md)
	}
}

func TestLinkChipsReadInChinese(t *testing.T) {
	app, _, _ := linksApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	inChinese(tt)
	m.OpenTask(2)
	tt.Frame()
	has(t, tt, "Design draft", "打开页面")
}
