package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
)

// demoPhone opens the demo database as a phone core at demoNow.
func demoPhone(t *testing.T) *Core {
	t.Helper()
	phone, err := open(demoDatabase(t, demoNow), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open the phone core: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	return phone
}

// completedIDs flattens a Completed page to its task ids, newest first, and checks the days.
func completedIDs(t *testing.T, raw string) (ids []int, page completedPageJSONForTest) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatalf("decode Completed: %v", err)
	}
	for _, day := range page.Days {
		for _, task := range day.Tasks {
			if !strings.HasPrefix(task.CompletedAt.Format("2006-01-02"), day.Day) {
				t.Errorf("task %d finished %v is under day %s", task.ID, task.CompletedAt, day.Day)
			}
			ids = append(ids, task.ID)
		}
	}
	return ids, page
}

type completedPageJSONForTest struct {
	Days []struct {
		Day   string
		Tasks []struct {
			ID          int
			CompletedAt time.Time
			Project     string
			By          string
			ByName      string
			Summary     string
		}
	}
	More bool
}

// TestCompletedIsTheDesktopsHistory pins Completed against data.LogbookOf: the same tasks in the
// same order, grouped by day, each with who finished it by the desktop's rule.
func TestCompletedIsTheDesktopsHistory(t *testing.T) {
	phone := demoPhone(t)
	raw, err := phone.Completed(0, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	ids, page := completedIDs(t, raw)
	want, err := phone.real.LogbookOf(0, false, defaultCompleted+1)
	if err != nil {
		t.Fatal(err)
	}
	var wantIDs []int
	for _, task := range want {
		if !task.IsSubtask() {
			wantIDs = append(wantIDs, task.ID)
		}
	}
	if len(ids) == 0 || fmtInts(ids) != fmtInts(wantIDs) {
		t.Errorf("Completed ids = %v, want %v", ids, wantIDs)
	}
	if page.More {
		t.Error("More on a page with room")
	}
	checkGolden(t, "completed.golden.json", raw)

	// Who finished it: the agent whose report closed the task, else you.
	by := map[int]string{}
	for _, day := range page.Days {
		for _, task := range day.Tasks {
			by[task.ID] = task.By + "|" + task.ByName
		}
	}
	if by[20] != "claude-code|Claude Code" {
		t.Errorf("task 20 finished by %q, want the agent", by[20])
	}
	if by[21] != "|you" {
		t.Errorf("task 21 finished by %q, want you", by[21])
	}
}

func fmtInts(ids []int) string {
	b, _ := json.Marshal(ids)
	return string(b)
}

// TestCompletedScopesAndPages: a project narrows it, agents keeps what agents wrote, and limit
// pages it with More.
func TestCompletedScopesAndPages(t *testing.T) {
	phone := demoPhone(t)
	all, _ := mustCompleted(t, phone, 0, false, 0)

	inProject, _ := mustCompleted(t, phone, 1, false, 0)
	if len(inProject) == 0 || len(inProject) >= len(all) {
		t.Errorf("project 1 has %d of %d tasks", len(inProject), len(all))
	}
	for _, id := range inProject {
		task := mustTask(t, phone, int64(id))
		if task.ProjectID == nil || *task.ProjectID != 1 {
			t.Errorf("task %d is not in project 1", id)
		}
	}

	byAgents, _ := mustCompleted(t, phone, 0, true, 0)
	for _, id := range byAgents {
		if mustTask(t, phone, int64(id)).Source != data.SourceAgent {
			t.Errorf("task %d was not written by an agent", id)
		}
	}

	// The limit counts loaded tasks, subtasks included, as on the desktop: a short page says More.
	first, page := mustCompleted(t, phone, 0, false, 2)
	if !page.More || len(first) > 2 || len(first) == 0 || first[0] != all[0] {
		t.Errorf("limit 2 = %v More=%v, want a prefix of %v and More", first, page.More, all)
	}
}

func mustCompleted(t *testing.T, phone *Core, project int64, agents bool, limit int64) ([]int, completedPageJSONForTest) {
	t.Helper()
	raw, err := phone.Completed(project, agents, limit)
	if err != nil {
		t.Fatal(err)
	}
	return completedIDs(t, raw)
}

func mustTask(t *testing.T, phone *Core, id int64) data.Task {
	t.Helper()
	task, err := phone.dom.GetTask(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := phone.real.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range snap.Tasks {
		if int64(candidate.ID) == task.ID {
			return candidate
		}
	}
	t.Fatalf("task %d not in the snapshot", id)
	return data.Task{}
}

// summaryOnPage is the part of Summary's JSON the tests read.
type summaryOnPage struct {
	Week   bool   `json:"week"`
	From   string `json:"from"`
	To     string `json:"to"`
	Over   bool   `json:"over"`
	Total  int    `json:"total"`
	Merged int    `json:"merged"`
	Ledger []struct {
		Agent  string `json:"agent"`
		Name   string `json:"name"`
		You    bool   `json:"you"`
		Count  int    `json:"count"`
		Merged int    `json:"merged"`
		Tasks  []struct {
			ID int64 `json:"id"`
		} `json:"tasks"`
	} `json:"ledger"`
	Paragraph string `json:"paragraph"`
	Goods     []struct {
		Actor   int    `json:"actor"`
		Kind    string `json:"kind"`
		Starred bool   `json:"starred"`
		Day     int    `json:"day"`
		TaskID  int64  `json:"task_id"`
	} `json:"goods"`
}

// TestSummaryIsTheDesktopsLedger: the counts are data.CountSummary's, the people are you first then
// the agents, and the goods are one per finished task in the order they were finished.
func TestSummaryIsTheDesktopsLedger(t *testing.T) {
	phone := demoPhone(t)
	snap, err := phone.real.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, week := range []bool{false, true} {
		raw, err := phone.Summary(week, 0)
		if err != nil {
			t.Fatal(err)
		}
		var got summaryOnPage
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("decode Summary: %v", err)
		}
		period := data.DayPeriod(demoNow, 0)
		if week {
			period = data.WeekPeriod(demoNow, 0, time.Monday)
		}
		counts := data.CountSummary(snap, period, summaryAgentKey)
		if got.Total != counts.Finished || got.Merged != counts.Merged || got.Total == 0 {
			t.Errorf("week=%v: total %d merged %d, want %+v", week, got.Total, got.Merged, counts)
		}
		if got.Week != week || got.Over {
			t.Errorf("week=%v: Week=%v Over=%v", week, got.Week, got.Over)
		}
		if len(got.Ledger) == 0 || !got.Ledger[0].You || got.Ledger[0].Name != "you" || got.Ledger[0].Agent != "" {
			t.Errorf("the ledger does not start with you: %+v", got.Ledger)
		}
		if len(got.Goods) != got.Total {
			t.Errorf("%d goods for %d tasks", len(got.Goods), got.Total)
		}
		for _, good := range got.Goods {
			found := false
			for _, task := range got.Ledger[good.Actor].Tasks {
				found = found || task.ID == good.TaskID
			}
			if !found || good.Kind == "" {
				t.Errorf("good %+v is not thrown by its ledger row", good)
			}
		}
		name := "summary-day.golden.json"
		if week {
			name = "summary-week.golden.json"
			if got.From != "2026-10-19" || got.To != "2026-10-25" {
				t.Errorf("week = %s to %s, want Monday 2026-10-19 to 2026-10-25", got.From, got.To)
			}
			if got.Goods[len(got.Goods)-1].Day != 1 {
				t.Errorf("the last good is on day %d, want Tuesday (1)", got.Goods[len(got.Goods)-1].Day)
			}
		} else if got.From != "2026-10-20" || got.To != "2026-10-20" {
			t.Errorf("day = %s to %s", got.From, got.To)
		}
		checkGolden(t, name, raw)
	}

	// Yesterday is over, and nothing from today leaks into it.
	raw, err := phone.Summary(false, 1)
	if err != nil {
		t.Fatal(err)
	}
	var yesterday summaryOnPage
	if err := json.Unmarshal([]byte(raw), &yesterday); err != nil {
		t.Fatal(err)
	}
	if !yesterday.Over || yesterday.From != "2026-10-19" || yesterday.Total != 1 {
		t.Errorf("yesterday = %+v", yesterday)
	}
}

// TestSummaryParagraphAndMarkdown: a stored paragraph shows in Summary and leads the Markdown, which
// is the desktop's export in the app's language.
func TestSummaryParagraphAndMarkdown(t *testing.T) {
	phone := demoPhone(t)
	raw, err := phone.Summary(false, 0)
	if err != nil {
		t.Fatal(err)
	}
	var before summaryOnPage
	_ = json.Unmarshal([]byte(raw), &before)
	if before.Paragraph != "" {
		t.Fatalf("paragraph before any summary row = %q", before.Paragraph)
	}

	period := data.DayPeriod(demoNow, 0)
	if _, err := phone.real.PutSummary(period, data.StoredSummary{Paragraph: "A good day.", Model: "test", Kinds: map[int]string{21: "book"}}); err != nil {
		t.Fatal(err)
	}
	raw, err = phone.Summary(false, 0)
	if err != nil {
		t.Fatal(err)
	}
	var after summaryOnPage
	_ = json.Unmarshal([]byte(raw), &after)
	if after.Paragraph != "A good day." {
		t.Errorf("paragraph = %q", after.Paragraph)
	}
	var picked string
	for _, good := range after.Goods {
		if good.TaskID == 21 {
			picked = good.Kind
		}
	}
	if picked != "book" {
		t.Errorf("task 21's good = %q, want the assistant's pick", picked)
	}

	md, err := phone.SummaryMarkdown(false, 0, "en")
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := phone.real.Snapshot()
	sum := data.BuildSummary(data.SummaryInput{Snapshot: snap, Now: demoNow, Period: period, Agent: summaryAgentKey, Kinds: map[int]string{21: "book"}})
	links := func(id int) []data.Link { return data.TaskLinks(snap.Reports[id]) }
	if want := sum.Markdown(locales.In(""), "A good day.", links); md != want {
		t.Errorf("Markdown differs from the desktop's export\n%s\nwant\n%s", md, want)
	}
	for _, line := range []string{"# Daily summary, 2026-10-20", "A good day.", "## You", "## Claude Code", "- Port the MCP server (withdo, PR #14): Eight tools ported with their tests"} {
		if !strings.Contains(md, line) {
			t.Errorf("Markdown lacks %q:\n%s", line, md)
		}
	}
	zh, err := phone.SummaryMarkdown(true, 0, "zh-Hans")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(zh, "周报") || !strings.Contains(zh, "## 你") {
		t.Errorf("the Chinese Markdown is not Chinese:\n%s", zh)
	}
}

// TestNewPagesHaveNoNullOnANewDatabase: the new reads that take arguments, on a database with
// nothing in it, are arrays and objects without a null, as TestJSONOfANewDatabaseHasNoNull wants of
// the others.
func TestNewPagesHaveNoNullOnANewDatabase(t *testing.T) {
	phone, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	reads := map[string]func() (string, error){
		"Completed":      func() (string, error) { return phone.Completed(0, false, 0) },
		"Summary":        func() (string, error) { return phone.Summary(false, 0) },
		"SummaryWeek":    func() (string, error) { return phone.Summary(true, 3) },
		"ParseQuick":     func() (string, error) { return phone.ParseQuick("") },
		"CategoryIcons":  func() (string, error) { return phone.CategoryIcons("") },
		"CategoryIconsQ": func() (string, error) { return phone.CategoryIcons("nothing like this") },
		"DoTodayMany":    func() (string, error) { return phone.DoTodayMany("[1]") },
	}
	for name, read := range reads {
		raw, err := read()
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Errorf("%s = %q, not JSON: %v", name, raw, err)
		} else if path := findNull(decoded, name); path != "" {
			t.Errorf("%s has a null at %s: %s", name, path, raw)
		}
	}
	if md, err := phone.SummaryMarkdown(false, 0, "en"); err != nil || !strings.HasPrefix(md, "# Daily summary, ") {
		t.Errorf("Markdown of an empty day = %q (%v)", md, err)
	}
}
