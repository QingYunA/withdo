package data

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/ghpoll"
)

// The fixtures in testdata are what gh printed for QingYunA/adhd-todo on 2026-10-19; in
// gh-issues.json issue 25 was edited to be closed, by pull request 41, since none was.

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func recentDay(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC) }

func TestParseRecentReadsAMergedPullRequestFromGHsJSON(t *testing.T) {
	items, err := ParseRecent(RecentMerged, "QingYunA/adhd-todo", fixture(t, "gh-pr-merged.json"), recentDay(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].Number != 41 || items[1].Number != 36 || items[2].Number != 35 {
		t.Fatalf("items %+v, want 41, 36, 35 newest first", items)
	}
	pr := items[0]
	if pr.Kind != RecentMerged || pr.Repo != "QingYunA/adhd-todo" || pr.Author != "QingYunA" || pr.Bot || pr.Draft ||
		pr.URL != "https://github.com/QingYunA/adhd-todo/pull/41" || pr.Branch != "claude/ledger-count-width" ||
		!strings.HasPrefix(pr.Title, "fix(summary): give a two-digit ledger count room") || !strings.Contains(pr.Body, "ledgerCount") {
		t.Errorf("pr %+v", pr)
	}
	if want := []string{"app/views/summary_ledger.go", "app/views/summary_ledger_width_test.go"}; !slices.Equal(pr.Files, want) {
		t.Errorf("files %v, want the names gh lists", pr.Files)
	}
	if !pr.ClosedAt.Equal(time.Date(2026, 10, 19, 3, 4, 29, 0, time.UTC)) || !pr.At().Equal(pr.ClosedAt) {
		t.Errorf("a merged pull request is dated by when it merged: %v", pr.ClosedAt)
	}
}

func TestParseRecentLeavesOutWhatMergedBeforeThePeriod(t *testing.T) {
	items, err := ParseRecent(RecentMerged, "r", fixture(t, "gh-pr-merged.json"), time.Date(2026, 10, 19, 3, 0, 0, 0, time.UTC))
	if err != nil || len(items) != 1 || items[0].Number != 41 {
		t.Errorf("%+v, %v: only 41 merged after 03:00", items, err)
	}
}

func TestParseRecentReadsOpenPullRequestsAndKeepsThemWheneverTheyWereOpened(t *testing.T) {
	items, err := ParseRecent(RecentOpen, "r", fixture(t, "gh-pr-open.json"), recentDay(18, 0))
	if err != nil || len(items) != 2 {
		t.Fatalf("%+v, %v", items, err)
	}
	if items[0].Kind != RecentOpen || !items[0].ClosedAt.IsZero() || !items[0].At().Equal(items[0].CreatedAt) || len(items[0].Files) == 0 {
		t.Errorf("open pull request %+v", items[0])
	}
}

func TestParseRecentReadsIssuesOpenedOrClosedInThePeriodAndWhichPullRequestFixedOne(t *testing.T) {
	items, err := ParseRecent(RecentIssue, "r", fixture(t, "gh-issues.json"), time.Date(2026, 10, 18, 8, 59, 58, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := map[int]RecentItem{}
	for _, i := range items {
		got[i.Number] = i
	}
	if len(got) != 3 {
		t.Fatalf("issues %v: 37 and 28 were opened in the period, 25 was closed in it; 26 and 27 were opened a moment before it", got)
	}
	if bot := got[37]; !bot.Bot || bot.Author != "devin-ai-integration" || bot.Solved() {
		t.Errorf("37 is a bot's open issue, named without the app/ of gh: %+v", bot)
	}
	if closed := got[25]; !closed.Solved() || closed.FixedBy != 41 || !closed.At().Equal(time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("25 is solved by 41, dated by its closing: %+v", closed)
	}
	if got[37].FixedBy != 38 {
		t.Errorf("an open issue a pull request will close names it too: %d", got[37].FixedBy)
	}
	if items[0].Number != 37 {
		t.Errorf("newest first, got %d", items[0].Number)
	}
}

func TestParseRecentRefusesAnswerThatIsNotJSON(t *testing.T) {
	if _, err := ParseRecent(RecentMerged, "r", []byte("not json"), time.Time{}); err == nil {
		t.Error("garbage was accepted")
	}
}

func TestFallbackNoteStripsTheConventionalPrefixAndSortsByIt(t *testing.T) {
	for _, c := range []struct {
		title, sentence string
		cat             RecentCategory
	}{
		{"feat(settings): copy the setup text", "copy the setup text", RecentNew},
		{"fix(summary)!: two-digit counts", "two-digit counts", RecentFixed},
		{"docs: readme", "readme", RecentOther},
		{"chore(deps): bump mygo", "bump mygo", RecentOther},
		{"Add a Windows build", "Add a Windows build", RecentOther},
		{"feat:", "feat:", RecentNew},
	} {
		got := FallbackNote(RecentItem{Kind: RecentMerged, Title: c.title})
		if got.Sentence != c.sentence || got.Category != c.cat {
			t.Errorf("%q: %+v, want %q in %q", c.title, got, c.sentence, c.cat)
		}
	}
	if n := FallbackNote(RecentItem{Kind: RecentOpen, Title: "feat(x): y"}); n.Category != "" || n.Sentence != "y" {
		t.Errorf("only a merged pull request has a category: %+v", n)
	}
}

func demoSection(t *testing.T, note func(RecentItem) (RecentNote, bool)) RecentSection {
	t.Helper()
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local)
	ch, err := DemoRecent(now).Fetch(context.Background(), "QingYunA/adhd-todo", RecentSince(now, 7))
	if err != nil {
		t.Fatal(err)
	}
	return BuildRecent(ch, note)
}

func numbers(ls []RecentLine) []int {
	var out []int
	for _, l := range ls {
		out = append(out, l.Item.Number)
	}
	return out
}

func TestBuildRecentGroupsByTheTitlesWhenTheAssistantSaidNothing(t *testing.T) {
	s := demoSection(t, func(RecentItem) (RecentNote, bool) { return RecentNote{}, false })
	if got := numbers(s.New); !slices.Equal(got, []int{14, 34, 35, 29, 12}) {
		t.Errorf("new %v, newest first", got)
	}
	if got := numbers(s.Fixed); !slices.Equal(got, []int{41}) {
		t.Errorf("fixed %v: 42 is a test, so it is Other", got)
	}
	if got := numbers(s.Other); !slices.Equal(got, []int{42, 43, 44, 45}) {
		t.Errorf("other %v", got)
	}
	if len(s.Open) != 2 || len(s.Issues) != 2 || s.Merged() != 10 || s.Count() != 14 {
		t.Errorf("open %d issues %d merged %d count %d", len(s.Open), len(s.Issues), s.Merged(), s.Count())
	}
	if l := s.Fixed[0]; l.Written || l.Note.Sentence != "give a two-digit ledger count room, so 48 is not cut to an ellipsis" {
		t.Errorf("without a model a line is the title without its prefix: %+v", l)
	}
	if topics := s.OtherTopics(); len(topics) != 0 {
		t.Errorf("without a model there are no topics: %v", topics)
	}
}

func TestBuildRecentUsesWhatTheAssistantWroteAndItsCategoryAndTopics(t *testing.T) {
	notes := map[int]RecentNote{
		41: {Sentence: "Two-digit totals fit", Category: RecentFixed},
		42: {Sentence: "A test stops failing", Category: RecentFixed},
		43: {Sentence: "The page is specified", Category: RecentOther, Topic: "docs"},
		44: {Sentence: "Version is 0.4.0", Topic: "version"}, // no category: the title's
	}
	s := demoSection(t, func(i RecentItem) (RecentNote, bool) { n, ok := notes[i.Number]; return n, ok })
	if got := numbers(s.Fixed); !slices.Equal(got, []int{41, 42}) {
		t.Errorf("fixed %v", got)
	}
	if l := s.Fixed[0]; !l.Written || l.Note.Sentence != "Two-digit totals fit" {
		t.Errorf("%+v", l)
	}
	if got := numbers(s.Other); !slices.Equal(got, []int{43, 44, 45}) {
		t.Errorf("other %v", got)
	}
	if topics := s.OtherTopics(); !slices.Equal(topics, []string{"docs", "version"}) {
		t.Errorf("topics %v", topics)
	}
}

func TestSectionsAreOrderedByTheirNewestItem(t *testing.T) {
	a := RecentSection{Repo: "a", Last: recentDay(10, 0)}
	b := RecentSection{Repo: "b", Last: recentDay(12, 0)}
	c := RecentSection{Repo: "c", Last: recentDay(11, 0)}
	all := []RecentSection{a, b, c}
	RecentOrder(all)
	if all[0].Repo != "b" || all[1].Repo != "c" || all[2].Repo != "a" {
		t.Errorf("%v", all)
	}
}

func TestRecentSinceIsLocalMidnightOfTheFirstOfNDaysEndingToday(t *testing.T) {
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local)
	if got, want := RecentSince(now, 7), time.Date(2026, 10, 8, 0, 0, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("7 days: %v, want %v", got, want)
	}
	if got, want := RecentSince(now, 30), time.Date(2026, 9, 15, 0, 0, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("30 days: %v, want %v", got, want)
	}
}

func TestFakeRecentCutsToThePeriodAndFetchReposAsksEach(t *testing.T) {
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local)
	f := DemoRecent(now)
	f.Fail("bad/repo", &RecentError{Problem: RecentNoAccess, Repo: "bad/repo", Text: "gone"})
	got := FetchRepos(context.Background(), f, []string{"QingYunA/adhd-todo", "QingYunA/picpocket", "bad/repo"}, RecentSince(now, 3))
	if n := len(got["QingYunA/adhd-todo"].Changes.Merged); n != 8 {
		t.Errorf("3 days: %d merged of adhd-todo, want 8 (all but the 4 days old)", n)
	}
	if n := len(got["QingYunA/picpocket"].Changes.Merged); n != 1 {
		t.Errorf("picpocket merged %d in 3 days, want 1", n)
	}
	var re *RecentError
	if err := got["bad/repo"].Err; !errors.As(err, &re) || re.Problem != RecentNoAccess {
		t.Errorf("err %v", err)
	}
	if len(f.Fetched()) != 3 {
		t.Errorf("fetched %v", f.Fetched())
	}
}

// ghRunner answers each gh call by what it lists.
type ghRunner struct {
	calls [][]string
	fail  error
}

func (r *ghRunner) Run(_ context.Context, args ...string) (ghpoll.Output, error) {
	r.calls = append(r.calls, args)
	if r.fail != nil {
		return ghpoll.Output{}, r.fail
	}
	file := "gh-issues.json"
	switch {
	case args[0] == "pr" && slices.Contains(args, "merged"):
		file = "gh-pr-merged.json"
	case args[0] == "pr":
		file = "gh-pr-open.json"
	}
	raw, err := os.ReadFile(filepath.Join("testdata", file))
	return ghpoll.Output{Stdout: raw}, err
}

func TestGHRecentAsksGHForTheThreeListsOfARepository(t *testing.T) {
	r := &ghRunner{}
	since := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	ch, err := GHRecent{Runner: r}.Fetch(context.Background(), "QingYunA/adhd-todo", since)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 3 {
		t.Fatalf("calls %v", r.calls)
	}
	line := func(i int) string { return strings.Join(r.calls[i], " ") }
	for i, want := range []string{
		"pr list --state merged --search merged:>=2026-10-13 --json ",
		"pr list --state open --json ",
		"issue list --state all --search updated:>=2026-10-13 --json ",
	} {
		if !strings.HasPrefix(line(i), want) || !strings.Contains(line(i), "--repo QingYunA/adhd-todo --limit 100") {
			t.Errorf("call %d: %s", i, line(i))
		}
	}
	if !strings.Contains(line(0), "mergedAt") || !strings.Contains(line(0), "files") || !strings.Contains(line(2), "closedByPullRequestsReferences") {
		t.Errorf("the fields: %s / %s", line(0), line(2))
	}
	if len(ch.Merged) != 3 || ch.Merged[0].Number != 41 || len(ch.Open) != 2 || len(ch.Issues) != 5 {
		t.Errorf("changes: %d merged, %d open, %d issues", len(ch.Merged), len(ch.Open), len(ch.Issues))
	}
}

func TestGHRecentSaysWhyGHFailed(t *testing.T) {
	for _, c := range []struct {
		err  error
		want RecentProblem
	}{
		{ghpoll.ErrNotInstalled, RecentNoGH},
		{&ghpoll.ExitError{Code: 4, Stderr: []byte("To get started with GitHub CLI, please run:  gh auth login")}, RecentSignedOut},
		{&ghpoll.ExitError{Code: 1, Stderr: []byte("GraphQL: Could not resolve to a Repository with the name 'a/b'. (repository)")}, RecentNoAccess},
		{&ghpoll.ExitError{Code: 1, Stderr: []byte(`error connecting to api.github.com`)}, RecentOffline},
		{&ghpoll.ExitError{Code: 1, Stderr: []byte("boom")}, RecentOtherProblem},
	} {
		_, err := GHRecent{Runner: &ghRunner{fail: c.err}}.Fetch(context.Background(), "a/b", time.Now())
		var re *RecentError
		if !errors.As(err, &re) || re.Problem != c.want || re.Repo != "a/b" {
			t.Errorf("%v: got %v, want problem %d", c.err, err, c.want)
		}
	}
}

func TestTheCacheKeepsASentenceByURLAndUpdatedAtAndInTheFileForTheNextRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recent-changes.json")
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC)
	item := RecentItem{URL: "https://github.com/a/b/pull/1", UpdatedAt: now.Add(-time.Hour)}
	c := NewRecentCache(path)
	if _, ok := c.Note("en", item); ok {
		t.Fatal("an empty cache knew a sentence")
	}
	c.PutNotes("en", now, []RecentNoteFor{{Item: item, Note: RecentNote{Sentence: "It fits", Category: RecentFixed}}})
	if n, ok := c.Note("en", item); !ok || n.Sentence != "It fits" || n.Category != RecentFixed {
		t.Fatalf("note %+v %v", n, ok)
	}
	if _, ok := c.Note("zh-Hans", item); ok {
		t.Error("a sentence in English was given for a window in Chinese")
	}
	if again, ok := NewRecentCache(path).Note("en", item); !ok || again.Sentence != "It fits" {
		t.Errorf("the next run reads the file: %+v %v", again, ok)
	}
	changed := item
	changed.UpdatedAt = now
	if _, ok := c.Note("en", changed); ok {
		t.Error("a pull request updated since was not written again")
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Error("the temporary file was left behind")
	}
}

func TestTheCacheKeepsAnExplanationAndDropsItWhenTheItemChanges(t *testing.T) {
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC)
	item := RecentItem{URL: "u", UpdatedAt: now}
	c := NewRecentCache("")
	c.PutExplanation("en", now, item, RecentExplanation{What: "w", Why: "y", Where: "p"})
	if x, ok := c.Explanation("en", item); !ok || x.What != "w" || x.Where != "p" {
		t.Fatalf("%+v %v", x, ok)
	}
	if _, ok := c.Note("en", item); ok {
		t.Error("an explanation alone is not a sentence")
	}
	c.PutNotes("en", now, []RecentNoteFor{{Item: item, Note: RecentNote{Sentence: "s"}}})
	if _, ok := c.Explanation("en", item); ok {
		t.Error("writing the sentence again forgot nothing: it replaced the entry, explanation included")
	}
}

func TestTheCacheKeepsLinesByRepositoryPeriodAndLanguageAndForgetsOldItems(t *testing.T) {
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC)
	c := NewRecentCache("")
	c.PutLines("a/b", 7, "en", StoredLines{Text: "Two went in.", WrittenAt: now})
	if l, ok := c.Lines("a/b", 7, "en"); !ok || l.Text != "Two went in." || !l.WrittenAt.Equal(now) {
		t.Errorf("%+v %v", l, ok)
	}
	for _, miss := range []struct {
		repo string
		days int
		lang string
	}{{"a/c", 7, "en"}, {"a/b", 30, "en"}, {"a/b", 7, "zh-Hans"}} {
		if _, ok := c.Lines(miss.repo, miss.days, miss.lang); ok {
			t.Errorf("lines of %+v were given", miss)
		}
	}
	old := RecentItem{URL: "old", UpdatedAt: now.Add(-100 * 24 * time.Hour)}
	c.PutNotes("en", now.Add(-99*24*time.Hour), []RecentNoteFor{{Item: old, Note: RecentNote{Sentence: "old"}}})
	c.PutNotes("en", now, nil)
	if _, ok := c.Note("en", old); ok {
		t.Error("a sentence for an item not updated in 90 days was kept")
	}
}

func TestAnUnreadableCacheFileStartsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recent-changes.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := NewRecentCache(path)
	c.PutLines("a/b", 7, "en", StoredLines{Text: "x"})
	if _, ok := c.Lines("a/b", 7, "en"); !ok {
		t.Error("the cache did not start empty")
	}
}
