package views

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestTheSwitchIsOffByDefaultAndWhileOffNothingOfThePageExistsOrIsRead(t *testing.T) {
	app, _, rec := newTestApp(t) // a fresh install
	_ = rec
	read := data.DemoRecent(fixedNow())
	app.Recent, app.Assistant = read, recentAssistant(true)
	if app.RecentChangesOn() {
		t.Fatal("the page is on before the user said so")
	}
	tt, m, _ := newTesterFor(app, 1100)
	tt.Frame()
	if tt.HasText("Recent changes") {
		t.Errorf("a sidebar row while the page is off: %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.Key4)
	tt.Frame()
	if m.nav.screen == screenRecent {
		t.Error("Cmd+4 opened a page that is off")
	}
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	if tt.HasText("Recent changes") {
		t.Error("a link on the project page while the page is off")
	}
	if got := read.Fetched(); len(got) != 0 {
		t.Errorf("read %v while the page is off", got)
	}
}

func TestSwitchedOnTheRowSitsUnderCalendarAndCmd4OpensThePage(t *testing.T) {
	r := newRecentRig(t, true)
	r.tt.Frame()
	if !above(t, r.tt, "Calendar", "Recent changes") || !above(t, r.tt, "Recent changes", "Projects") {
		t.Errorf("sidebar order: %q", r.tt.Texts())
	}
	r.tt.Key(ui.Cmd, ui.Key4)
	r.tt.Frame()
	if r.m.nav.screen != screenRecent {
		t.Fatalf("Cmd+4 went to %+v", r.m.nav)
	}
	r.settle(t)
	r.has(t, "All repositories", "7 days", "30 days", "Updated just now")
	r.app.SetRecentChangesOn(false)
	r.tt.Frame()
	if r.m.nav.screen == screenRecent || r.tt.HasText("Updated just now") {
		t.Errorf("the page stays open after it is switched off: %+v", r.m.nav)
	}
}

func TestEachProjectIsASectionWithItsGroupsInOrderAndTheOneChangedLastFirst(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	r.has(t, "withdo", "QingYunA/withdo", "10 merged · 2 open · 2 issues", "picpocket", "QingYunA/picpocket", "2 merged",
		"These 7 days · written by the assistant", "Rewrite", "New", "Fixed", "Open", "Issues",
		"3 more: docs, version, screenshots", "The MCP server runs on the Go core", "Two-digit ledger totals fit",
		"The MCP server moves to 0.5", "Solved · fixed by #41 · today · #56", "New · you · yesterday · #58")
	r.lacks(t, "answer-me-with-html"+"QingYunA", "The recent changes page is specified") // a project with nothing; what is folded
	if !above(t, r.tt, "New", "Fixed") || !above(t, r.tt, "Fixed", "3 more: docs, version, screenshots") ||
		!above(t, r.tt, "3 more: docs, version, screenshots", "Open") || !above(t, r.tt, "Open", "Issues") {
		t.Errorf("groups out of order: %q", r.tt.Texts())
	}
	if !above(t, r.tt, "QingYunA/withdo", "QingYunA/picpocket") {
		t.Error("the project changed last is not first")
	}
	r.has(t, "Claude Code · today · #14", "You · yesterday · #34", "cursor-agent · open for 20 minutes · #40", "You · open for 2 hours · #39")
}

func TestAMergedPullRequestIsByTheAgentWhoseTaskHoldsItsURLElseByItsAuthor(t *testing.T) {
	r := newRecentRig(t, true, func(r *recentRig) {
		r.app.GitHub = data.NewFakeGitHub(data.GitHubStatus{State: data.GitHubReady, Account: "someone-else", CheckedAt: fixedNow()})
	})
	r.open(t)
	r.has(t, "Claude Code · today · #14", "QingYunA · yesterday · #34")
}

func TestWithoutAModelTheTitlesStandInAndNothingIsAsked(t *testing.T) {
	r := newRecentRig(t, false)
	r.open(t)
	r.has(t, "give a two-digit ledger count room, so 48 is not cut to an ellipsis", "native UI with mygo", "New", "Fixed", "4 more")
	r.lacks(t, "Rewrite", "These 7 days · written by the assistant", "3 more: docs, version, screenshots")
	if n := len(r.asst.RecentDescribed()) + len(r.asst.RecentLined()); n != 0 {
		t.Errorf("%d requests to an assistant with no model", n)
	}
	got := r.m.recentSections()[0]
	if len(got.Fixed) != 1 || len(got.New) != 5 || got.Fixed[0].Written {
		t.Errorf("without a model fix goes under Fixed by its prefix: %d fixed, %d new", len(got.Fixed), len(got.New))
	}
}

func askedNumbers(reqs []data.RecentDescribeRequest) map[string][]int {
	out := map[string][]int{}
	for _, r := range reqs {
		for _, i := range r.Items {
			out[r.Repo] = append(out[r.Repo], i.Number)
		}
	}
	return out
}

func TestTheAssistantWritesEachItemOnceAndTheLinesOncePerProjectAndPeriod(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	asked := askedNumbers(r.asst.RecentDescribed())
	if len(asked["QingYunA/withdo"]) != 14 || len(asked["QingYunA/picpocket"]) != 2 || len(asked) != 2 {
		t.Errorf("asked about %v: every item of the two projects shown, once", asked)
	}
	for _, req := range r.asst.RecentDescribed() {
		if len(req.Items) > recentChunk || req.Language != "English" {
			t.Errorf("request %d items in %q", len(req.Items), req.Language)
		}
	}
	if len(r.asst.RecentLined()) != 2 {
		t.Fatalf("lines asked %d times, want once for each project", len(r.asst.RecentLined()))
	}
	lines := r.asst.RecentLined()[0]
	if lines.Period != "the last 7 days" || len(lines.Lines) == 0 || lines.Lines[0].Note.Sentence == "" {
		t.Errorf("lines request %+v is told the sentences", lines)
	}
	r.click(t, "Update now") // read again: everything is kept
	r.settle(t)
	r.m.recentPick("")
	if n := len(r.asst.RecentDescribed()); n != 3 {
		t.Errorf("%d requests after reading again, want the 3 of the first time", n)
	}
	if len(r.asst.RecentLined()) != 2 {
		t.Errorf("lines written again: %d", len(r.asst.RecentLined()))
	}
}

func TestAnItemUpdatedSinceIsWrittenAgainAndOnlyThat(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	before := len(r.asst.RecentDescribed())
	ch, _ := r.read.Fetch(t.Context(), "QingYunA/withdo", time.Time{})
	ch.Merged[0].UpdatedAt = ch.Merged[0].UpdatedAt.Add(time.Minute) // #14, whose description changed
	ch.Merged[0].Title = "feat(mcp): port the MCP server, now with tests"
	r.read.Set(ch)
	r.click(t, "Update now")
	r.settle(t)
	got := r.asst.RecentDescribed()
	if len(got) != before+1 || len(got[before].Items) != 1 || got[before].Items[0].Number != ch.Merged[0].Number {
		t.Errorf("after the update: %d requests, last %+v", len(got), got[len(got)-1].Items)
	}
}

func TestTheRepositoryPopUpListsEachRepositoryWithItsCountAndFilters(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	r.click(t, "Repository: All repositories")
	r.has(t, "14", "0") // withdo, and answer-me-with-html with nothing
	opts := r.m.repoOptions(r.m.recentSections())
	want := []repoOption{{"", "All repositories", -1}, {"QingYunA/answer-me-with-html", "answer-me-with-html", 0}, {"QingYunA/picpocket", "picpocket", 2}, {"QingYunA/withdo", "withdo", 14}}
	if !slices.Equal(opts, want) {
		t.Errorf("options %v, want %v", opts, want)
	}
	if err := r.tt.Click("answer-me-with-html, 0"); err != nil { // dimmed, but it can be picked
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.m.recent.repo != "QingYunA/answer-me-with-html" {
		t.Fatalf("picked %q", r.m.recent.repo)
	}
	r.has(t, "Nothing changed in the last 7 days")
	r.lacks(t, "QingYunA/withdo", "Rewrite")
	r.m.recentPick("QingYunA/picpocket")
	r.tt.Frame()
	r.has(t, "QingYunA/picpocket", "2 merged")
	r.lacks(t, "QingYunA/withdo", "Two-digit ledger totals fit")
	stored, _ := r.app.Prefs.LoadPrefs()
	if stored.RecentRepo != "QingYunA/picpocket" {
		t.Errorf("the choice is not kept: %+v", stored)
	}
}

func TestAProjectNamePicksItsRepositoryAndTheChoiceSurvivesANewWindow(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	r.tt.Scroll(700, 400, 0, 3000) // the section is under the fold
	r.tt.Frame()
	r.click(t, "Show only QingYunA/picpocket")
	if r.tt.HasText("QingYunA/withdo") || !r.tt.HasText("QingYunA/picpocket") {
		t.Errorf("a click on the name did not pick it: %q", r.tt.Texts())
	}
	m := NewMain(r.app)
	m.recentLoad()
	if m.recent.repo != "QingYunA/picpocket" || m.recent.days != 7 {
		t.Errorf("remembered %q for %d days", m.recent.repo, m.recent.days)
	}
}

func TestThirtyDaysReadsAgainForTheLongerPeriodAndIsRemembered(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	before := len(r.read.Fetched())
	r.click(t, "30 days")
	r.settle(t)
	if len(r.read.Fetched()) <= before {
		t.Error("GitHub was not read for the other period")
	}
	r.has(t, "These 30 days · written by the assistant")
	if days := r.asst.RecentLined(); days[len(days)-1].Period != "the last 30 days" {
		t.Errorf("lines for %q", days[len(days)-1].Period)
	}
	if _, days := r.app.recentChoice(); days != 30 {
		t.Errorf("days %d", days)
	}
	r.lacks(t, "Nothing changed in the last 30 days")
}

func TestUnfoldingAnItemAsksForItsExplanationOnceAndKeepsIt(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	item := "Show what it does: The MCP server runs on the Go core"
	r.click(t, item)
	r.settle(t)
	r.has(t, "Read the description and the changed files", "What", "Why", "Where",
		"The MCP server's tools now run on the Go core, and Claude Code's MCP client connects as before.",
		"Open on GitHub", "Task #20")
	r.click(t, item) // fold
	r.lacks(t, "Open on GitHub")
	r.click(t, item) // unfold again
	r.settle(t)
	r.has(t, "Open on GitHub")
	got := r.asst.RecentExplained()
	if len(got) != 1 || got[0].Item.Number != 14 || got[0].Language != "English" {
		t.Errorf("asked %+v, want one request for PR 14", got)
	}
	r.click(t, "Open on GitHub")
	if len(r.rec.urls) != 1 || r.rec.urls[0] != "https://github.com/QingYunA/withdo/pull/14" {
		t.Errorf("opened %v", r.rec.urls)
	}
	r.click(t, "Task #20")
	if r.m.openID != 20 {
		t.Errorf("the inspector shows task %d", r.m.openID)
	}
}

func TestAnItemWithNoTaskOffersOnlyGitHubAndWithNoModelNoExplanation(t *testing.T) {
	r := newRecentRig(t, false)
	r.open(t)
	r.click(t, "Show what it does: copy the setup text for any agent")
	r.has(t, "Open on GitHub")
	r.lacks(t, "What", "Read the description and the changed files", "Task #29")
	if len(r.asst.RecentExplained()) != 0 {
		t.Error("asked for an explanation with no model")
	}
}

func TestAnIssueUnfoldsTooWithoutAWhere(t *testing.T) {
	r := newRecentRig(t, true)
	r.asst.RecentExplainScript = func(data.RecentExplainRequest) (data.RecentExplanation, error) {
		return data.RecentExplanation{What: "Export stops.", Why: "Pictures break it."}, nil
	}
	r.open(t)
	r.click(t, "Show what it does: Export fails for tasks with pictures")
	r.settle(t)
	r.has(t, "Read the description", "Export stops.", "Pictures break it.", "Open on GitHub")
	r.lacks(t, "Where", "Read the description and the changed files")
	if req := r.asst.RecentExplained(); len(req) != 1 || req[0].Item.Kind != data.RecentIssue {
		t.Errorf("asked %+v", req)
	}
}

func TestAFailedExplanationSaysSoAndRetryAsksAgain(t *testing.T) {
	r := newRecentRig(t, true)
	fail := true
	r.asst.RecentExplainScript = func(data.RecentExplainRequest) (data.RecentExplanation, error) {
		if fail {
			return data.RecentExplanation{}, &data.ProblemError{Problem: data.ProblemTimeout, Text: "timeout"}
		}
		return data.RecentExplanation{What: "It works.", Why: "Because."}, nil
	}
	r.open(t)
	r.click(t, "Show what it does: The MCP server runs on the Go core")
	r.settle(t)
	r.has(t, "Couldn't write the explanation.", "Retry")
	fail = false
	r.click(t, "Retry")
	r.settle(t)
	r.has(t, "It works.")
	if n := len(r.asst.RecentExplained()); n != 2 {
		t.Errorf("asked %d times", n)
	}
}

func TestOtherUnfoldsIntoOneLineEach(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	r.lacks(t, "The recent changes page is specified")
	r.click(t, "Show the other changes")
	r.has(t, "The recent changes page is specified", "The version is 0.4.0", "Summary screenshots are fresh")
}

func TestRewriteWritesTheLinesAgainAndAFailureSaysSo(t *testing.T) {
	r := newRecentRig(t, true, func(r *recentRig) { r.app.setRecentChoice("QingYunA/picpocket", 7) })
	r.open(t)
	r.has(t, "It now watches new branches in worktrees, and the tray no longer misses focus at launch.")
	r.asst.RecentLinesScript = func(data.RecentLinesRequest) (string, error) { return "A second try.", nil }
	r.click(t, "Rewrite")
	r.settle(t)
	r.has(t, "A second try.")
	r.lacks(t, "It now watches new branches")
	r.asst.RecentLinesScript = func(data.RecentLinesRequest) (string, error) {
		return "", &data.ProblemError{Problem: data.ProblemTimeout, Text: "timeout"}
	}
	r.click(t, "Rewrite")
	r.settle(t)
	r.has(t, "A second try.") // the old ones stay until new ones arrive
}

func TestTheDotBesideRewriteSaysMoreMergedSinceTheLinesWereWritten(t *testing.T) {
	r := newRecentRig(t, true)
	r.open(t)
	s := r.m.recentSections()[0]
	stored, ok := r.app.RecentCache.Lines(s.project.RepoSlug, 7, "English")
	if !ok || s.newSince(stored.WrittenAt) {
		t.Fatalf("a fresh paragraph already says more was merged: %v %v", ok, stored)
	}
	if !s.newSince(s.Last.Add(-time.Hour)) {
		t.Error("something merged after the lines were written, and there is no dot")
	}
}

func TestNothingInThePeriodSaysSoAndGHMissingOrSignedOutSaysWhatToRun(t *testing.T) {
	r := newRecentRig(t, true, func(r *recentRig) { r.app.Recent = data.NewFakeRecent() })
	r.open(t)
	r.has(t, "Nothing changed in the last 7 days")
	r.lacks(t, "Rewrite")

	for _, c := range []struct {
		problem        data.RecentProblem
		title, command string
	}{
		{data.RecentNoGH, "Install the GitHub CLI first", "brew install gh"},
		{data.RecentSignedOut, "Sign in to the GitHub CLI first", "gh auth login"},
	} {
		r := newRecentRig(t, true, func(r *recentRig) {
			for _, p := range r.app.snap.projects {
				r.read.Fail(p.RepoSlug, &data.RecentError{Problem: c.problem, Repo: p.RepoSlug, Text: "x"})
			}
		})
		r.open(t)
		r.has(t, c.title, c.command, "Copy")
		r.lacks(t, "withdo"+"QingYunA/withdo")
	}
}

func TestOneRepositoryThatCannotBeReadSaysSoInItsSectionAndTheOthersShow(t *testing.T) {
	r := newRecentRig(t, true, func(r *recentRig) {
		r.read.Fail("QingYunA/picpocket", &data.RecentError{Problem: data.RecentNoAccess, Repo: "QingYunA/picpocket", Text: "gone"})
	})
	r.open(t)
	r.has(t, "Two-digit ledger totals fit", "QingYunA/picpocket", "Couldn't read this repository. Check that gh can open it:", "gh repo view QingYunA/picpocket", "Copy")
	if !above(t, r.tt, "Two-digit ledger totals fit", "Couldn't read this repository. Check that gh can open it:") {
		t.Error("the failed project is not after the others")
	}
}

func TestGitHubIsReadWhenThePageOpensAgainOnClickAndEveryTenMinutesWhileTheWindowIsKey(t *testing.T) {
	key := true
	r := newRecentRig(t, true, func(r *recentRig) { r.app.WindowKey = func() bool { return key } })
	clock := fixedNow()
	r.app.Now = func() time.Time { return clock }
	r.open(t)
	reads := func() int { return len(r.read.Fetched()) }
	first := reads()
	if first != 3 {
		t.Fatalf("%d repositories read when the page opened, want the 3 of the projects", first)
	}
	r.tt.Frame()
	if reads() != first {
		t.Error("read again with nothing asked")
	}
	clock = clock.Add(9 * time.Minute)
	r.tt.Frame()
	r.settle(t)
	if reads() != first {
		t.Error("read again before ten minutes")
	}
	r.has(t, "Updated 9 min ago")
	key = false
	clock = clock.Add(2 * time.Minute)
	r.tt.Frame()
	r.settle(t)
	if reads() != first {
		t.Error("read while the window is not key")
	}
	key = true
	r.tt.Frame()
	r.settle(t)
	if reads() != 2*first {
		t.Errorf("%d reads after the window came back, want %d", reads(), 2*first)
	}
	r.click(t, "Update now")
	r.settle(t)
	if reads() != 3*first {
		t.Errorf("%d reads after the click, want %d", reads(), 3*first)
	}
	r.m.goTo(nav{})
	r.tt.Frame()
	r.m.goTo(nav{screen: screenRecent})
	r.tt.Frame()
	r.settle(t)
	if reads() != 4*first {
		t.Errorf("%d reads after the page opened again, want %d", reads(), 4*first)
	}
}

func TestAProjectPageLinksToTheRecentChangesOfItsRepository(t *testing.T) {
	r := newRecentRig(t, true)
	r.m.goTo(nav{screen: screenProject, projectID: 3})
	r.tt.Frame()
	r.has(t, "Recent changes")
	if err := r.tt.Click("See what changed lately in this repository"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	r.settle(t)
	if r.m.nav.screen != screenRecent || r.m.recent.repo != "QingYunA/picpocket" {
		t.Fatalf("went to %+v with %q picked", r.m.nav, r.m.recent.repo)
	}
	r.lacks(t, "Two-digit ledger totals fit")
	r.has(t, "QingYunA/picpocket")
}

func TestARepositoryThatIsNoLongerAProjectsFallsBackToAll(t *testing.T) {
	r := newRecentRig(t, true, func(r *recentRig) { r.app.setRecentChoice("gone/repo", 7) })
	r.open(t)
	r.has(t, "QingYunA/withdo", "QingYunA/picpocket")
}

func TestTheSettingsRowTurnsThePageOnAndKeepsTheChoiceBesideTheOtherPrefs(t *testing.T) {
	prefs := &data.MemoryPrefs{}
	r := newSettingsRig(t, func(a *App) { a.Prefs = prefs })
	r.app.SetSoundEffects(false)
	r.pane(t, "Agents")
	r.tt.Scroll(340, 300, 0, 9000)
	r.tt.Frame()
	has(t, r.tt, "Recent changes", "The latest merged pull requests and issues, by project · gh signed in")
	if r.app.RecentChangesOn() {
		t.Fatal("on by default")
	}
	if err := r.tt.Click("Show the Recent changes page"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if !r.app.RecentChangesOn() {
		t.Fatal("clicking the switch left it off")
	}
	stored, _ := prefs.LoadPrefs()
	if !stored.RecentChanges || !stored.SoundOff {
		t.Errorf("stored %+v", stored)
	}
	if again := (&App{Prefs: prefs}); !again.RecentChangesOn() {
		t.Error("the choice was not kept")
	}
}

func TestSwitchedOnWithGHSignedOutTheRowSaysWhatIsMissing(t *testing.T) {
	r := newSettingsRig(t, func(a *App) {
		a.GitHub = data.NewFakeGitHub(data.GitHubStatus{State: data.GitHubSignedOut})
		a.SetRecentChangesOn(true)
	})
	r.pane(t, "Agents")
	r.tt.Scroll(340, 300, 0, 9000)
	r.tt.Frame()
	has(t, r.tt, "gh not signed in", "gh auth login")
	if !r.app.RecentChangesOn() {
		t.Error("the switch does not turn on without gh")
	}
}

func TestTheChinesePageSaysItInChinese(t *testing.T) {
	r := newRecentRig(t, true)
	inChinese(r.tt)
	r.open(t)
	r.has(t, "最近变化", "全部仓库", "7 天", "30 天", "合并 10 · 在做 2 · 问题 2", "这 7 天 · 助手写的", "重写", "新功能", "修复", "在做", "问题",
		"其他 3 项：文档、版本号、截图", "周报总数是两位数时，不再被截成省略号", "你 · 昨天 · #34", "已解决 · 由 #41 修好 · 今天 · #56", "新问题 · 你 · 昨天 · #58",
		"开了 2 小时")
	if got := r.asst.RecentLined(); len(got) == 0 || got[0].Language != "Simplified Chinese" {
		t.Errorf("lines asked in %+v", got)
	}
	r.lacks(t, "Recent changes")
	if !strings.Contains(strings.Join(r.tt.Texts(), "\n"), "3 天前") {
		t.Error("days ago is not in Chinese")
	}
}
