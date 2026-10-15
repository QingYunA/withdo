package views

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// settingsRig is the Settings window's view in a tester, as the shell shows it.
type settingsRig struct {
	tt  *ui.Tester
	s   *Settings
	app *App
	rec *recorder
	fm  *data.FakeModels

	title  string
	height float32
}

func newSettingsRig(t *testing.T, mod ...func(*App)) *settingsRig {
	t.Helper()
	app, _, rec := newTestApp(t)
	fm := newFakeModels(t)
	app.Models = fm
	app.GitHub = data.NewFakeGitHub(readyAt(2 * time.Minute))
	for _, f := range mod {
		f(app)
	}
	r := &settingsRig{app: app, rec: rec, fm: fm}
	r.s = NewSettings(app)
	r.s.OnLayout = func(title string, h float32) { r.title, r.height = title, h }
	r.tt = ui.NewTester(r.s.View, int(settingsWidth), 560)
	r.tt.SetTitleBar(ui.TitleBar{Height: SettingsTitleBarHeight, Left: 78})
	still(r.tt)
	r.tt.Frame()
	r.tt.Frame()
	return r
}

func (r *settingsRig) pane(t *testing.T, name string) {
	t.Helper()
	if err := r.tt.Click(name); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
}

func TestSettingsToolbarHasFourPanesAndTheTitleIsTheOpenOne(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "General", "Assistant", "Agents", "GitHub")
	if r.title != "General" {
		t.Errorf("title %q, want the open pane's name", r.title)
	}
	for _, want := range []struct {
		click, title string
		pane         Pane
	}{{"Assistant", "Assistant", PaneAssistant}, {"Agents", "Agents", PaneAgents}, {"GitHub", "GitHub", PaneGitHub}, {"General", "General", PaneGeneral}} {
		r.pane(t, want.click)
		if r.s.pane != want.pane || r.title != want.title {
			t.Errorf("after %q: pane %v, title %q", want.click, r.s.pane, r.title)
		}
	}
}

func TestSettingsShowSelectsAPane(t *testing.T) {
	r := newSettingsRig(t)
	r.s.Show(PaneAssistant)
	r.tt.Frame()
	if r.title != "Assistant" {
		t.Errorf("title %q", r.title)
	}
	has(t, r.tt, "Provider", "Model")
}

func TestSettingsReportsItsHeightToFitItsContent(t *testing.T) {
	r := newSettingsRig(t)
	general := r.height
	if general < 200 {
		t.Fatalf("height %v: not measured", general)
	}
	r.pane(t, "Assistant")
	if r.height == general {
		t.Errorf("the Assistant pane is as tall as General (%v)", general)
	}
}

func TestDataFolderIsShownRelativeToHomeButRevealedInFull(t *testing.T) {
	r := newSettingsRig(t)
	r.app.Config.DataFolder = "/Users/mac/Library/Application Support/adhd-todo"
	r.app.Config.Home = "/Users/mac"
	r.tt.Frame()
	has(t, r.tt, "~/Library/Application Support/adhd-todo")
	hasNot(t, r.tt, "/Users/mac/Library/Application Support/adhd-todo")
	if err := r.tt.Click("Show in Finder"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rec.folders, []string{"/Users/mac/Library/Application Support/adhd-todo"}) {
		t.Errorf("revealed %v", r.rec.folders)
	}
	if got := withTilde("/Users/macro/x", "/Users/mac"); got != "/Users/macro/x" {
		t.Errorf("a sibling of the home folder became %q", got)
	}
}

func TestGeneralPaneHasLanguageArchivedDataFolderAndAbout(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "Language", "System", "No archived projects. Archive one from its menu in the sidebar.",
		"Data folder", r.app.Config.DataFolder, "Show in Finder", "adhd-todo 0.2.0", "Open source, MIT license", "GitHub ↗")
	hasFold(t, r.tt, "Archived projects")
	hasNot(t, r.tt, "Connect Claude Code")
	if err := r.tt.Click("Show in Finder"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rec.folders, []string{r.app.Config.DataFolder}) {
		t.Errorf("revealed %v", r.rec.folders)
	}
	if err := r.tt.Click("GitHub ↗"); err != nil {
		t.Fatal(err)
	}
	if len(r.rec.urls) != 1 || r.rec.urls[0] != "https://github.com/QingYunA/adhd-todo" {
		t.Errorf("opened %v", r.rec.urls)
	}
}

func TestLanguagePopUpChoosesALanguageOrTheSystemOne(t *testing.T) {
	r := newSettingsRig(t)
	if err := r.tt.Click("System"); err != nil {
		t.Fatal(err)
	}
	has(t, r.tt, "English", "简体中文")
	if err := r.tt.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	if r.rec.language != "zh-Hans" {
		t.Errorf("chose %q", r.rec.language)
	}
	r.rec.language = "zh-Hans"
	r.tt.Frame()
	if err := r.tt.Click("简体中文"); err != nil { // the pop-up now shows the choice
		t.Fatal(err)
	}
	if err := r.tt.Click("System"); err != nil {
		t.Fatal(err)
	}
	if r.rec.language != "" {
		t.Errorf("chose %q after System", r.rec.language)
	}
}

func TestArchivedProjectsListWithRestore(t *testing.T) {
	r := newSettingsRig(t)
	if err := r.app.ArchiveProject(3); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	name := r.app.snap.archived[0].Name
	has(t, r.tt, name, "Restore")
	if err := r.tt.Click("Restore " + name); err != nil {
		t.Fatal(err)
	}
	if len(r.app.snap.archived) != 0 {
		t.Errorf("still archived: %v", r.app.snap.archived)
	}
	r.tt.Frame()
	has(t, r.tt, "No archived projects. Archive one from its menu in the sidebar.")
}

func TestAgentsPaneCopiesTheCommand(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Agents")
	has(t, r.tt, "Connect Claude Code", "Run this once in a terminal. Agents then report their progress here.", r.app.Config.ConnectCommand, "Copy")
	if err := r.tt.Click("Copy command"); err != nil {
		t.Fatal(err)
	}
	if r.tt.Clipboard() != r.app.Config.ConnectCommand {
		t.Errorf("clipboard %q", r.tt.Clipboard())
	}
	has(t, r.tt, "Copied")
}

func TestAssistantPaneShowsProviderModelKeyStatusAndAdvanced(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Assistant")
	has(t, r.tt, "Provider", "Anthropic", "Model", "claude-haiku-5-5", "API key", "Kept in your Keychain", "Not tested yet", "Test", "Base URL")
	hasFold(t, r.tt, "Advanced")
	hasNot(t, r.tt, "OpenAI", "DeepSeek")
}

func TestChoosingAProviderMakesItTheActiveOne(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Assistant")
	if err := r.tt.Click("Anthropic"); err != nil {
		t.Fatal(err)
	}
	has(t, r.tt, "Magpie", "OpenAI", "DeepSeek", "Kimi", "OpenRouter", "Ollama", "Custom")
	if err := r.tt.Click("OpenAI"); err != nil {
		t.Fatal(err)
	}
	if activeID(r.fm) != "openai" {
		t.Errorf("active = %q", activeID(r.fm))
	}
	r.tt.Frame()
	has(t, r.tt, "gpt-4.1-mini")
}

func TestMagpieAndOllamaHaveNoKeyRow(t *testing.T) {
	r := newSettingsRig(t)
	mustNil(t, r.fm.SetActive("magpie"))
	r.pane(t, "Assistant")
	has(t, r.tt, "Uses your Magpie gateway at 127.0.0.1:3425")
	hasNot(t, r.tt, "API key", "Kept in your Keychain")
}

func TestModelPopUpOffersTheSuggestionsAndOther(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Assistant")
	if err := r.tt.Click("claude-haiku-5-5"); err != nil {
		t.Fatal(err)
	}
	has(t, r.tt, "claude-sonnet-5-5", "claude-opus-5-5", "Other…")
	if err := r.tt.Click("claude-sonnet-5-5"); err != nil {
		t.Fatal(err)
	}
	if got := modelOf(r.fm, "anthropic"); got != "claude-sonnet-5-5" {
		t.Errorf("model = %q", got)
	}
}

func TestOtherTurnsTheModelRowIntoATextField(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Assistant")
	if err := r.tt.Click("claude-haiku-5-5"); err != nil {
		t.Fatal(err)
	}
	if err := r.tt.Click("Other…"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if !r.s.assist.fs.get("anthropic/model").editing {
		t.Fatal("the field did not take the keyboard")
	}
	r.tt.Key(ui.Cmd, ui.KeyA)
	r.tt.Type("claude-fable-9")
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if got := modelOf(r.fm, "anthropic"); got != "claude-fable-9" {
		t.Errorf("model = %q", got)
	}
}

func TestAProviderWithNoSuggestionsAsksForTheModelInAField(t *testing.T) {
	r := newSettingsRig(t)
	mustNil(t, r.fm.SetActive("custom"))
	r.pane(t, "Assistant")
	has(t, r.tt, "Protocol", "OpenAI-compatible", "Anthropic API")
	if err := r.tt.Click("Anthropic API"); err != nil {
		t.Fatal(err)
	}
	for _, p := range r.fm.Providers() {
		if p.ID == "custom" && !p.Anthropic {
			t.Error("the protocol did not change")
		}
	}
}

func TestSavingAKeyTestsItAtOnceAndNeverShowsIt(t *testing.T) {
	r := newSettingsRig(t)
	var got data.ProviderEdit
	r.fm.TestFn = func(_ string, e data.ProviderEdit) (time.Duration, error) {
		got = e
		return 640 * time.Millisecond, nil
	}
	r.pane(t, "Assistant")
	if err := r.tt.Click("Paste your API key"); err != nil {
		t.Fatal(err)
	}
	r.tt.Type("sk-secret-value")
	hasNot(t, r.tt, "sk-secret-value") // a secure field
	r.tt.Key(0, ui.KeyEnter)
	r.tt.Frame()
	if !r.fm.KeySaved("anthropic") {
		t.Fatal("the key was not stored")
	}
	has(t, r.tt, "Testing…")
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Works · 640 ms", "Test again")
	if got.Model != "claude-haiku-5-5" {
		t.Errorf("the test was sent %+v", got)
	}
	hasNot(t, r.tt, "Paste your API key", "sk-secret-value")
}

func TestTestLinkReportsTheFriendlyError(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.TestFn = func(string, data.ProviderEdit) (time.Duration, error) {
		return 0, errors.New("The provider didn't accept the API key.")
	}
	r.pane(t, "Assistant")
	if err := r.tt.Click("Test"); err != nil {
		t.Fatal(err)
	}
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "The provider didn't accept the API key.", "Test again")
	hasNot(t, r.tt, "Not tested yet")
}

func TestTheTestResultBelongsToOneProvider(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Assistant")
	mustNil(t, r.tt.Click("Test"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Works · 640 ms")
	mustNil(t, r.tt.Click("Anthropic"))
	mustNil(t, r.tt.Click("OpenAI"))
	r.tt.Frame()
	hasNot(t, r.tt, "Works · 640 ms")
	has(t, r.tt, "Not tested yet")
}

func TestBaseURLIsAnAdvancedFieldThatSavesOnEnter(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Assistant")
	has(t, r.tt, "Change only for a gateway such as Magpie", "https://api.anthropic.com")
	mustNil(t, r.tt.Click("https://api.anthropic.com")) // the field's placeholder
	r.tt.Type("https://gateway.example/v1")
	r.tt.Key(0, ui.KeyEnter)
	for _, p := range r.fm.Providers() {
		if p.ID == "anthropic" && p.BaseURL != "https://gateway.example/v1" {
			t.Errorf("base url = %q", p.BaseURL)
		}
	}
}

func TestGitHubPaneShowsTheStatusAndChecksNow(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "GitHub")
	has(t, r.tt, "Connected as QingYunA · checked 2m ago", "Check now")
	hasNot(t, r.tt, "brew install gh", "gh auth login")
	mustNil(t, r.tt.Click("Check now"))
	if gh := r.app.GitHub.(*data.FakeGitHub); gh.Checks != 1 {
		t.Errorf("CheckNow called %d times", gh.Checks)
	}
}

func TestSettingsWithoutPollerOrModelsStillHasEveryPane(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.GitHub, a.Models = nil, nil })
	has(t, r.tt, "General", "Assistant", "Agents", "GitHub")
	r.pane(t, "GitHub")
	has(t, r.tt, "Following pull requests isn't available here.")
	r.pane(t, "Assistant")
	hasNot(t, r.tt, "Provider")
}

func modelOf(fm *data.FakeModels, id string) string {
	for _, p := range fm.Providers() {
		if p.ID == id {
			return p.Model
		}
	}
	return ""
}

func activeID(fm *data.FakeModels) string {
	for _, p := range fm.Providers() {
		if p.Active {
			return p.ID
		}
	}
	return ""
}

// hasFold checks that a text is shown, whatever its case: group labels are set in the section style.
func hasFold(t *testing.T, tt *ui.Tester, want string) {
	t.Helper()
	for _, s := range tt.Texts() {
		if strings.EqualFold(s, want) {
			return
		}
	}
	t.Errorf("no %q (in any case) in %q", want, tt.Texts())
}

// The top of the window is one band: the pane's name centered in the row of the traffic lights,
// and the pane icons right under it.
func TestSettingsTitleAndPanesAreOneBandUnderTheTrafficLights(t *testing.T) {
	r := newSettingsRig(t)
	title, ok := r.tt.Find("General")
	if !ok {
		t.Fatalf("no title in %q", r.tt.Texts())
	}
	if title.Y+title.H > SettingsTitleBarHeight {
		t.Errorf("the title ends at %.0f, below the traffic lights' row (%.0f)", title.Y+title.H, SettingsTitleBarHeight)
	}
	if mid := title.X + title.W/2; mid < settingsWidth/2-1 || mid > settingsWidth/2+1 {
		t.Errorf("the title is centered at %.0f, not in the middle of the window (%.0f)", mid, settingsWidth/2)
	}
	pane, _ := r.tt.Find("Assistant")
	if pane.Y < SettingsTitleBarHeight {
		t.Errorf("the pane icons start at %.0f, in the title's row", pane.Y)
	}
}
