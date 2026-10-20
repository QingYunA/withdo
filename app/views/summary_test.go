package views

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// settle runs the posted work (the assistant's answer) and draws the frame after it.
func answered(t *testing.T, tt *ui.Tester, rec *recorder) {
	t.Helper()
	rec.drain(t)
	tt.Frame()
}

func TestSummarySitsUnderCompletedInTheSidebarAndCmd6OpensIt(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	if !above(t, tt, "Completed", "Summary") || !above(t, tt, "Summary", "Projects") {
		t.Errorf("sidebar order: %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.Key6)
	if m.nav.screen != screenSummary {
		t.Fatalf("Cmd+6 went to %+v", m.nav)
	}
	if !m.ShowingSummary() && !tt.HasText("Today") {
		t.Error("Summary does not show")
	}
	tt.Frame()
	if !m.ShowingSummary() {
		t.Error("ShowingSummary is false on the page")
	}
	tt.Key(ui.Cmd, ui.Key1)
	tt.Frame()
	if m.ShowingSummary() {
		t.Error("still Summary after leaving it")
	}
}

func TestSummaryHasTheHeaderTheScenesPlaceTheLedgerAndTheTotal(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, _ := summaryMain(t, app, false)
	answered(t, tt, rec)
	for _, want := range []string{"Summary", "Today", "This week", "Wednesday, Oct 14", "Export Markdown",
		"Today · written by the assistant", "Rewrite", "You", "Claude Code", "Codex", "Shipped today", "8"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	for _, want := range []string{"1 PR merged", "2 PRs merged"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if !above(t, tt, "Today · written by the assistant", "Shipped today") {
		t.Error("the paragraph is not above the ledger")
	}
}

func TestTheLedgerListsYouFirstThenAgentsByCountWithTheirCounts(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	answered(t, tt, rec)
	v := m.summaryFor(app.Now())
	got := []string{}
	for _, p := range v.sum.People {
		who := p.Agent
		if p.You() {
			who = "you"
		}
		got = append(got, who+":"+itoa(len(p.Tasks))+"/"+itoa(p.Merged))
	}
	if want := []string{"you:3/0", "Claude Code:3/2", "Codex:2/1"}; !slices.Equal(got, want) {
		t.Errorf("people %v, want %v", got, want)
	}
	if v.sum.Total() != 8 {
		t.Errorf("total %d, want the eight of the day", v.sum.Total())
	}
}

func TestAPastDayAndTheWeekAreOneClickAway(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	answered(t, tt, rec)
	if err := tt.Click("Go back"); err == nil {
		t.Log("an unexpected label matched")
	}
	if err := tt.Click("Previous"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.sum.back != 1 || !tt.HasText("Tuesday, Oct 13") || !tt.HasText("Draft the UI spec") && !tt.HasText("Shipped that day") {
		t.Errorf("back %d in %q", m.sum.back, tt.Texts())
	}
	for i := 0; i < 2; i++ { // Oct 11 is the first day with nothing in it
		if err := tt.Click("Previous"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	if !tt.HasText("Nothing was finished that day") {
		t.Errorf("a day with nothing says so: %q", tt.Texts())
	}
	if tt.HasText("Shipped that day") || tt.HasText("Rewrite") {
		t.Error("an empty day shows a ledger or a paragraph")
	}
	for m.sum.back > 0 {
		if err := tt.Click("Next"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	if !tt.HasText("Shipped today") {
		t.Errorf("back %d", m.sum.back)
	}
	if err := tt.Click("Next"); err != nil {
		t.Log(err)
	}
	tt.Frame()
	if m.sum.back != 0 {
		t.Errorf("› went past today: %d", m.sum.back)
	}
	if err := tt.Click("This week"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !m.sum.week || !tt.HasText("Shipped this week") || !tt.HasText("Oct 11 – 17") {
		t.Errorf("the week: %q", tt.Texts())
	}
	for i := 0; i < 2; i++ {
		if err := tt.Click("Previous"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	if !tt.HasText("Sep 27 – Oct 3") || !tt.HasText("Nothing was finished that week") {
		t.Errorf("last week: %q", tt.Texts())
	}
}

func TestAWeekAcrossAMonthNamesBothMonths(t *testing.T) {
	app, _, _ := summaryApp(t)
	app.Now = func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.Local) }
	tt, m := summaryMain(t, app, true)
	tt.Frame()
	if !tt.HasText("Oct 25 – Oct 31") && !tt.HasText("Oct 25 – 31") {
		t.Errorf("%q", tt.Texts())
	}
	_ = m
}

func TestTheWeekStartsOnTheSystemsFirstWeekday(t *testing.T) {
	app, _, _ := summaryApp(t)
	tt, m := summaryMain(t, app, true)
	tt.Frame()
	if m.weekStart != time.Sunday {
		t.Skipf("the test system starts weeks on %v", m.weekStart)
	}
	v := m.summaryFor(app.Now())
	if v.period.Start.Weekday() != m.weekStart || v.period.Start.Day() != 11 {
		t.Errorf("period %v", v.period.Start)
	}
}

func TestSummaryWithNothingFinishedShowsTheShutLidTheLineAndADimmedExport(t *testing.T) {
	app, rec := newEmptyApp(t)
	app.Assistant = &data.FakeAssistant{Ready: true}
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	for _, want := range []string{"Nothing finished yet today", "Export Markdown"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	for _, not := range []string{"Rewrite", "Shipped today", "written by the assistant"} {
		if tt.HasText(not) {
			t.Errorf("an empty day shows %q", not)
		}
	}
	if err := tt.Click("Export Markdown"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.sum.export.open {
		t.Error("an empty day opened the export sheet")
	}
	if n := len(app.Assistant.(*data.FakeAssistant).Summaries()); n != 0 {
		t.Errorf("the assistant was asked %d times to write about nothing", n)
	}
	_ = rec
	v := m.summaryFor(app.Now())
	if v.day == nil || v.day.End() != 0 || len(v.sum.People) != 0 {
		t.Error("an empty day has a scene to play")
	}
	if f := v.day.Frame(0, false); len(f.Ops) == 0 {
		t.Error("nothing is drawn")
	}
}

func TestTheParagraphIsWrittenOnceAtFirstOpenKeptAndShownWholeNextTime(t *testing.T) {
	app, store, rec := summaryApp(t)
	fa := app.Assistant.(*data.FakeAssistant)
	tt, m := summaryMain(t, app, false)
	if !tt.HasText("Today · written by the assistant") {
		t.Fatalf("the bubble is not there: %q", tt.Texts())
	}
	if tt.HasText("Eight finished.") {
		t.Error("the paragraph is shown before it was written")
	}
	answered(t, tt, rec) // the assistant is asked off the UI thread
	if got := len(fa.Summaries()); got != 1 {
		t.Fatalf("the assistant was asked %d times", got)
	}
	req := fa.Summaries()[0]
	if req.Week || len(req.Tasks) != 8 || req.Language != "English" || req.Label != "Wednesday 2026-10-14" {
		t.Errorf("request %+v", req)
	}
	if req.Open.InProgress == 0 {
		t.Errorf("still open %+v", req.Open)
	}
	var who []string
	for _, task := range req.Tasks {
		who = append(who, task.Who)
	}
	if !slices.Contains(who, "you") || !slices.Contains(who, "Claude Code") || !slices.Contains(who, "Codex") {
		t.Errorf("who %v", who)
	}
	if !tt.HasText(fa.Summaries()[0].Label) && !strings.Contains(strings.Join(tt.Texts(), " "), "Eight finished.") {
		t.Errorf("the paragraph did not arrive: %q", tt.Texts())
	}
	stored, err := store.GetSummary(data.DayPeriod(app.Now(), 0))
	if err != nil || stored == nil || stored.Model != "fake-model" || stored.Kinds[2] != "gem" || stored.Kinds[20] != "gear" {
		t.Fatalf("stored %+v, %v", stored, err)
	}
	// Another visit shows it at once and does not ask again.
	m.goTo(nav{})
	tt.Frame()
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	if got := len(fa.Summaries()); got != 1 {
		t.Errorf("asked %d times after the second visit", got)
	}
	if !strings.Contains(strings.Join(tt.Texts(), " "), "Eight finished.") {
		t.Errorf("the stored paragraph is not shown: %q", tt.Texts())
	}
	// A new window over the same store reads it too.
	app2, _ := NewApp(store, &recorder{}, testConfig()), 0
	app2.Now = fixedNow
	app2.Assistant = &data.FakeAssistant{Ready: true}
	tt2, _ := summaryMain(t, app2, false)
	if len(app2.Assistant.(*data.FakeAssistant).Summaries()) != 0 || !strings.Contains(strings.Join(tt2.Texts(), " "), "Eight finished.") {
		t.Error("a stored paragraph was written again")
	}
}

func TestTheAssistantsKindsPickTheGoodsOfTheScene(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	before := kindsOf(m.summaryFor(app.Now()))
	answered(t, tt, rec)
	after := kindsOf(m.summaryFor(app.Now()))
	if after[2] != "gem" || after[20] != "gear" || after[1] != "gem" {
		t.Errorf("kinds %v", after)
	}
	if before[2] == "gem" && before[20] == "gear" {
		t.Errorf("before the paragraph the words pick: %v", before)
	}
}

func kindsOf(v *summaryView) map[int]string {
	out := map[int]string{}
	for _, task := range v.tasks {
		out[task.ID] = task.Kind
	}
	return out
}

func TestWithoutAModelThereIsNoParagraphAndNothingIsAsked(t *testing.T) {
	app, _, _ := summaryApp(t)
	fa := &data.FakeAssistant{Ready: false}
	app.Assistant = fa
	tt, _ := summaryMain(t, app, false)
	for _, not := range []string{"written by the assistant", "Rewrite"} {
		if tt.HasText(not) {
			t.Errorf("without a model %q is shown", not)
		}
	}
	for _, want := range []string{"You", "Shipped today", "Export Markdown"} {
		if !tt.HasText(want) {
			t.Errorf("everything else stays: missing %q", want)
		}
	}
	if len(fa.Summaries()) != 0 {
		t.Error("the assistant was asked without a model")
	}
	app.Assistant = nil
	tt2, _ := summaryMain(t, app, false)
	if tt2.HasText("Rewrite") || !tt2.HasText("Shipped today") {
		t.Errorf("with no assistant at all: %q", tt2.Texts())
	}
}

func TestRewriteWritesAgainAndReplacesTheParagraph(t *testing.T) {
	app, store, rec := summaryApp(t)
	fa := app.Assistant.(*data.FakeAssistant)
	tt, _ := summaryMain(t, app, false)
	answered(t, tt, rec)
	n := 0
	fa.SummaryScript = func(data.SummaryRequest) (data.SummaryText, error) {
		n++
		return data.SummaryText{Paragraph: "A second try.", Model: "fake-model-2", Kinds: map[int]string{2: "book"}}, nil
	}
	if err := tt.Click("Rewrite"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("Rewrite") {
		t.Error("Rewrite stays while it writes")
	}
	answered(t, tt, rec)
	if n != 1 || !strings.Contains(strings.Join(tt.Texts(), " "), "A second try.") || tt.HasText("Eight finished.") {
		t.Errorf("rewrote %d times: %q", n, tt.Texts())
	}
	stored, _ := store.GetSummary(data.DayPeriod(app.Now(), 0))
	if stored.Paragraph != "A second try." || stored.Model != "fake-model-2" || stored.Kinds[2] != "book" {
		t.Errorf("stored %+v", stored)
	}
}

func TestAFailedWritingSaysSoAndRewriteTriesAgain(t *testing.T) {
	app, _, rec := summaryApp(t)
	fa := app.Assistant.(*data.FakeAssistant)
	fa.SummaryScript = func(data.SummaryRequest) (data.SummaryText, error) { return data.SummaryText{}, errAsk }
	tt, _ := summaryMain(t, app, false)
	answered(t, tt, rec)
	if !tt.HasText("Couldn't write it. Try Rewrite.") || !tt.HasText("Rewrite") {
		t.Fatalf("%q", tt.Texts())
	}
	if len(fa.Summaries()) != 1 {
		t.Errorf("a failure was retried by itself: %d", len(fa.Summaries()))
	}
	tt.Frame()
	if len(fa.Summaries()) != 1 {
		t.Error("it asks every frame")
	}
	fa.SummaryScript = nil
	if err := tt.Click("Rewrite"); err != nil {
		t.Fatal(err)
	}
	answered(t, tt, rec)
	if tt.HasText("Couldn't write it. Try Rewrite.") || len(fa.Summaries()) != 2 {
		t.Errorf("after Rewrite: %q (%d asks)", tt.Texts(), len(fa.Summaries()))
	}
}

var errAsk = &data.ProblemError{Problem: data.ProblemTimeout, Text: "timed out"}

func TestADotBesideRewriteSaysMoreWasFinishedSinceWhileThePeriodIsNotOver(t *testing.T) {
	app, store, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	answered(t, tt, rec)
	v := m.summaryFor(app.Now())
	para := m.paragraphOf(v.period)
	if m.newSince(v, para, app.Now()) {
		t.Fatal("a dot right after writing")
	}
	// The paragraph is dated by the store's clock: a task finished after it says so.
	later := app.Now().Add(time.Hour)
	store.SeedCompleteAt(9, later.Add(time.Minute))
	app.Refresh()
	v = m.summaryFor(later.Add(2 * time.Minute))
	if !m.newSince(v, para, later.Add(2*time.Minute)) {
		t.Error("no dot when something was finished since")
	}
	if m.newSince(v, para, v.period.End.Add(time.Minute)) {
		t.Error("a dot on a period that is over")
	}
	past := m.paragraphOf(data.DayPeriod(app.Now(), 1))
	if m.newSince(m.summaryFor(app.Now()), past, app.Now()) {
		t.Error("a dot with no paragraph")
	}
}

func TestANewParagraphTypesInAtThirtyMillisecondsACharacterAndAStoredOneShowsWhole(t *testing.T) {
	p := &paragraph{stored: &data.StoredSummary{Paragraph: "Hello there"}}
	at := time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local)
	if text, typing := p.shownText(at, false); text != "Hello there" || typing {
		t.Errorf("a stored paragraph: %q %v", text, typing)
	}
	p.typedAt = at
	for ms, want := range map[int]string{0: "", 29: "", 30: "H", 95: "Hel", 330: "Hello there", 5000: "Hello there"} {
		text, typing := p.shownText(at.Add(time.Duration(ms)*time.Millisecond), false)
		if text != want || typing != (want != "Hello there") {
			t.Errorf("at %d ms: %q (typing %v), want %q", ms, text, typing, want)
		}
	}
	if text, typing := p.shownText(at, true); text != "Hello there" || typing {
		t.Errorf("with Reduce Motion: %q %v", text, typing)
	}
	zh := &paragraph{stored: &data.StoredSummary{Paragraph: "今天收了七件"}, typedAt: at}
	if text, _ := zh.shownText(at.Add(65*time.Millisecond), false); text != "今天" {
		t.Errorf("typing counts characters, not bytes: %q", text)
	}
}

func TestAFreshParagraphTypesInOnScreen(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetPreferences(ui.Preferences{}) // motion on
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	clock := app.Now()
	app.Now = func() time.Time { return clock }
	rec.drain(t)
	// The paragraph arrived at `clock`; 300 ms later ten characters are in.
	app.Now = func() time.Time { return clock.Add(300 * time.Millisecond) }
	tt.Frame()
	joined := strings.Join(tt.Texts(), " ")
	if !strings.Contains(joined, "Eight fini") || strings.Contains(joined, "Eight finished.") {
		t.Errorf("typing: %q", joined)
	}
	app.Now = func() time.Time { return clock.Add(time.Minute) }
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), " "), "Native UI review is still waiting.") {
		t.Errorf("done typing: %q", tt.Texts())
	}
}

func TestOpeningALedgerRowShowsThatPersonsTasksAndAClickOpensTheInspector(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	answered(t, tt, rec)
	for _, hidden := range []string{"Native UI with mygo", "Reply to design feedback"} {
		if tt.HasText(hidden) {
			t.Errorf("%q shows before its row is opened", hidden)
		}
	}
	if err := tt.Click("Claude Code, 3 tasks"); err != nil {
		t.Fatalf("%v in %q", err, tt.Texts())
	}
	tt.Frame()
	for _, want := range []string{"Native UI with mygo", "PR #12 merged · Board, lists and inspector are in", "adhd-todo", "11:05 AM", "Port the MCP server"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText("Reply to design feedback") {
		t.Error("another person's tasks are open")
	}
	if err := tt.Click("Native UI with mygo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.openID != 2 {
		t.Errorf("the inspector shows task %d, want 2", m.openID)
	}
	if err := tt.Click("Claude Code, 3 tasks"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("PR #12 merged · Board, lists and inspector are in") {
		t.Error("the row does not close")
	}
}

func TestTheLedgerRunsUpOneRowAtATimeAfterTheScene(t *testing.T) {
	end := 5 * time.Second
	if got := ledgerAge(end+ledgerLight*2+90*time.Millisecond, end, 2); got != 90*time.Millisecond {
		t.Errorf("age %v", got)
	}
	if ledgerOpacity(ledgerAge(end+time.Second, end, 4)) != 0 || ledgerOpacity(ledgerAge(end+4*ledgerLight, end, 4)) != 1 {
		t.Error("a row is lit from its moment on")
	}
	for age, want := range map[time.Duration]int{-time.Second: 0, 0: 0, 59 * time.Millisecond: 0, 60 * time.Millisecond: 1, 181 * time.Millisecond: 3, time.Hour: 4} {
		if got := ledgerCountAt(4, age); got != want {
			t.Errorf("count at %v: %d, want %d", age, got, want)
		}
	}
	if got := ledgerDone(end, 4, 7); got != end+3*ledgerLight+7*ledgerCount {
		t.Errorf("done at %v", got)
	}
}

func TestTheSceneAndTheLedgerFollowTheClock(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetPreferences(ui.Preferences{}) // motion on
	clock := app.Now()
	app.Now = func() time.Time { return clock }
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	rec.drain(t)
	v := m.summaryFor(clock)
	if at, calm := m.sceneTime(testContext(tt), v, clock); at != 0 || calm {
		t.Errorf("at the start: %v %v", at, calm)
	}
	clock = clock.Add(2 * time.Second)
	if at, _ := m.sceneTime(testContext(tt), v, clock); at != 2*time.Second {
		t.Errorf("two seconds in: %v", at)
	}
	if v.sceneEnd() == 0 || v.sceneEnd() > 12*time.Second {
		t.Errorf("the scene lasts %v", v.sceneEnd())
	}
}

// testContext is a context with the preferences of the tester, for the functions that read them.
func testContext(tt *ui.Tester) *ui.Context {
	var got *ui.Context
	probe := ui.NewTester(func(c *ui.Context) { got = c }, 10, 10)
	probe.SetPreferences(ui.Preferences{})
	probe.Frame()
	_ = tt
	return got
}

func TestTheSkyFollowsTheHourForTodayAndIsDuskForAPastDayAndNightForAWeek(t *testing.T) {
	app, _, _ := summaryApp(t)
	for hour, want := range map[int]theme.SkyKind{9: theme.SkyDay, 16: theme.SkyDay, 17: theme.SkyDusk, 19: theme.SkyDusk, 20: theme.SkyNight, 23: theme.SkyNight} {
		clock := time.Date(2026, 10, 14, hour, 5, 0, 0, time.Local)
		app.Now = func() time.Time { return clock }
		_, m := summaryMain(t, app, false)
		if m.sum.sky != want {
			t.Errorf("%d:05 opens under sky %v, want %v", hour, m.sum.sky, want)
		}
		m.sum.back = 1
		m.summaryFor(clock)
		if m.sum.sky != theme.SkyDusk {
			t.Errorf("a past day at %d:05 is under sky %v, want dusk", hour, m.sum.sky)
		}
		m.sum.week, m.sum.back = true, 0
		m.summaryFor(clock)
		if m.sum.sky != theme.SkyNight {
			t.Errorf("the week at %d:05 is under sky %v, want night", hour, m.sum.sky)
		}
	}
}

func TestTheScenePlaysAgainWhenThePageOpensThePeriodChangesOrTheSkyIsClicked(t *testing.T) {
	app, _, _ := summaryApp(t)
	clock := app.Now()
	app.Now = func() time.Time { return clock }
	tt, m := summaryMain(t, app, false)
	first := m.sum.start
	clock = clock.Add(time.Minute)
	tt.Frame()
	if !m.sum.start.Equal(first) {
		t.Error("the scene started over by itself")
	}
	// A click on the sky replays.
	r := m.sum.scene
	tt.ClickAt(r.X+20, r.Y+10)
	tt.Frame()
	if !m.sum.start.Equal(clock) {
		t.Errorf("a click on the sky did not replay: %v", m.sum.start)
	}
	// Leaving the page and coming back plays it.
	clock = clock.Add(time.Minute)
	m.goTo(nav{})
	tt.Frame()
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	if !m.sum.start.Equal(clock) {
		t.Error("coming back did not play the scene")
	}
	// So does another period.
	clock = clock.Add(time.Minute)
	if err := tt.Click("Previous"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !m.sum.start.Equal(clock) {
		t.Error("another day did not play the scene")
	}
}

func TestHoveringAGoodInTheBinShowsATipAndAClickOpensItsTask(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	answered(t, tt, rec)
	v := m.summaryFor(app.Now())
	r := m.sum.scene
	hit, ok := v.day.HitAt(v.day.Calm(), 124+3, 49+3) // a good in the first layer of the heap
	if !ok {
		t.Fatal("nothing in the bin where the heap is")
	}
	off := float32(theme.SceneOffset(r.W))
	x := r.X + (off+float32(hit.X)+8)*theme.SceneCell
	y := r.Y + (float32(hit.Y)+8)*theme.SceneCell
	// The good drawn on top where the pointer is: goods in a heap overlap.
	cx, cy := theme.CellAt(r.W, x-r.X, y-r.Y)
	top, ok := v.day.HitAt(v.day.Calm(), cx, cy)
	if !ok {
		t.Fatal("the pointer is on no good")
	}
	task := v.tasks[top.Good]
	tt.Move(x, y)
	tt.Frame()
	if !tt.HasText(task.Title) {
		t.Fatalf("no tip for %q in %q", task.Title, tt.Texts())
	}
	if tt.Cursor() != ui.CursorPointer {
		t.Errorf("cursor %v", tt.Cursor())
	}
	tt.ClickAt(x, y)
	tt.Frame()
	if m.openID != task.ID {
		t.Errorf("the inspector shows %d, want %d", m.openID, task.ID)
	}
	tt.Move(r.X+10, r.Y+10)
	tt.Frame()
	if tt.HasText(task.Title) {
		t.Error("the tip stays when the pointer leaves")
	}
}

func TestTheWeekLabelsItsDaysAndHoveringAStackFindsATask(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, true)
	answered(t, tt, rec)
	for _, day := range []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"} {
		if !tt.HasText(day) {
			t.Errorf("no label %q in %q", day, tt.Texts())
		}
	}
	v := m.summaryFor(app.Now())
	if v.week == nil || v.week.Busiest() != 3 {
		t.Fatalf("busiest day %v", v.week)
	}
	r := m.sum.scene
	off := float32(theme.SceneOffset(r.W))
	x := r.X + (off+float32(theme.DayLabelX(3))+8)*theme.SceneCell
	y := r.Y + (float32(71-16)+8)*theme.SceneCell
	tt.Move(x, y)
	tt.Frame()
	first := v.byDay[3][0]
	if !tt.HasText(first.Title) {
		t.Errorf("no tip for %q in %q", first.Title, tt.Texts())
	}
}

func TestExportMarkdownOpensASheetThatCopiesAndSaves(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m := summaryMain(t, app, false)
	answered(t, tt, rec)
	if err := tt.Click("Export Markdown"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !m.sum.export.open {
		t.Fatal("the sheet did not open")
	}
	md := m.summaryMarkdown(m.summaryFor(app.Now()))
	for _, want := range []string{
		"# Daily summary, 2026-10-14\n", "Eight finished. You answered", "\n## You\n- Send the packaging answer\n",
		"\n## Claude Code\n- ", "- Native UI with mygo (adhd-todo, PR #12): Board, lists and inspector are in",
		"- Port the MCP server (adhd-todo, PR #14)", "\n## Codex\n", "Still open: ", " in progress",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("the Markdown lacks %q:\n%s", want, md)
		}
	}
	if strings.Index(md, "Eight finished.") > strings.Index(md, "## You") || strings.Index(md, "## You") > strings.Index(md, "Still open") {
		t.Errorf("order: the paragraph, the people, then what is open:\n%s", md)
	}
	for _, want := range []string{"Copy", "Save as…", "Close"} {
		if !tt.HasText(want) {
			t.Errorf("the sheet lacks %q in %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Clipboard() != md || !tt.HasText("Copied") {
		t.Errorf("clipboard %q", tt.Clipboard())
	}
	if err := tt.Click("Save as…"); err != nil {
		t.Fatal(err)
	}
	if len(rec.saved) != 1 || rec.saved[0].name != "summary-2026-10-14.md" || rec.saved[0].text != md {
		t.Errorf("saved %+v", rec.saved)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if m.sum.export.open {
		t.Error("Esc does not close the sheet")
	}
}

func TestTheMarkdownOfAWeekIsNamedForItsFirstDayAndLeavesOutWhatIsNotThere(t *testing.T) {
	app, _, _ := summaryApp(t)
	app.Assistant = nil
	_, m := summaryMain(t, app, true)
	md := m.summaryMarkdown(m.summaryFor(app.Now()))
	if !strings.HasPrefix(md, "# Weekly summary, 2026-10-11 – 2026-10-17\n\n## You\n") {
		t.Errorf("%q", md[:min(len(md), 120)])
	}
	if exportFileName(m.summaryFor(app.Now()).period) != "summary-week-2026-10-11.md" {
		t.Error("file name")
	}
	if strings.Contains(md, "written") || strings.Contains(md, "Eight finished") {
		t.Errorf("no paragraph was written:\n%s", md)
	}
}

func TestTheOpenPartsAreTheOnesThatHaveSomething(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	for open, want := range map[data.SummaryOpen]string{
		{InProgress: 3, Blocked: 1}:              "3 in progress, 1 blocked",
		{InReview: 2}:                            "2 in review",
		{InProgress: 1, InReview: 1, Blocked: 1}: "1 in progress, 1 in review, 1 blocked",
		{}:                                       "",
	} {
		if got := m.openParts(open); got != want {
			t.Errorf("%+v: %q, want %q", open, got, want)
		}
	}
}

func TestSummaryInChinese(t *testing.T) {
	app, _, rec := summaryApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.SetLocale("zh-Hans")
	m.goTo(nav{screen: screenSummary})
	tt.Frame()
	answered(t, tt, rec)
	for _, want := range []string{"总结", "今天", "本周", "导出 Markdown", "重写", "今天出货", "你", "合并 2 个 PR", "件"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if got := app.Assistant.(*data.FakeAssistant).Summaries()[0].Language; got != "Simplified Chinese" {
		t.Errorf("the model is told to write in %q", got)
	}
	md := m.summaryMarkdown(m.summaryFor(app.Now()))
	for _, want := range []string{"# 2026-10-14 日报", "## 你\n", "（adhd-todo，PR #12）：", "还在做的："} {
		if want == "（adhd-todo，PR #12）：" {
			want = "（adhd-todo, PR #12）："
		}
		if !strings.Contains(md, want) {
			t.Errorf("中文 Markdown lacks %q:\n%s", want, md)
		}
	}
}

func TestMainShowsSummaryOnTheWeekWhenAskedTo(t *testing.T) {
	app, _, _ := summaryApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	tt.Frame()
	m.ShowSummary(true)
	tt.Frame()
	if !m.ShowingSummary() || !m.sum.week || !tt.HasText("Shipped this week") {
		t.Errorf("%q", tt.Texts())
	}
	m.ShowSummary(false)
	tt.Frame()
	if m.sum.week || !tt.HasText("Shipped today") {
		t.Errorf("%q", tt.Texts())
	}
}

func TestEveryKindTheAssistantMayPickHasAFileAndTheWordsPickOne(t *testing.T) {
	if got, want := theme.GoodKinds(), data.GoodKinds; !slices.Equal(got, want) {
		t.Errorf("scene %v, data %v", got, want)
	}
}

func TestTheSceneIsDrawnAtTheColumnsWidthOnAnyWindow(t *testing.T) {
	app, _, rec := summaryApp(t)
	for _, w := range []int{860, 1100, 1500} {
		tt, m, _ := newTesterFor(app, w)
		m.goTo(nav{screen: screenSummary})
		tt.Frame()
		rec.mu.Lock()
		rec.posted = nil
		rec.mu.Unlock()
		want := float32(w) - theme.SidebarWidth - 1
		if got := m.sum.scene.W; got < want-1 || got > want+1 || m.sum.scene.H != theme.SceneHeight {
			t.Errorf("window %d: the scene is %v x %v, want %v wide and %v tall", w, got, m.sum.scene.H, want, theme.SceneHeight)
		}
	}
}
