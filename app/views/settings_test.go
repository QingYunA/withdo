package views

import (
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
	mac *agentRig // this Mac, for the Agents pane

	title string
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
	r := &settingsRig{app: app, rec: rec, fm: fm, mac: newAgentRig(t)}
	app.Config.AgentSetup = r.mac.env
	r.s = NewSettings(app)
	r.s.OnTitle = func(title string) { r.title = title }
	w, h := r.s.ContentSize()
	r.tt = ui.NewTester(r.s.View, int(w), int(h))
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

func TestSettingsToolbarHasFivePanesAndTheTitleIsTheOpenOne(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "General", "Assistant", "Agents", "GitHub", "Sync")
	if r.title != "General" {
		t.Errorf("title %q, want the open pane's name", r.title)
	}
	for _, want := range []struct {
		click, title string
		pane         Pane
	}{{"Assistant", "Assistant", PaneAssistant}, {"Agents", "Agents", PaneAgents}, {"GitHub", "GitHub", PaneGitHub}, {"Sync", "Sync", PaneSync}, {"General", "General", PaneGeneral}} {
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
	has(t, r.tt, "Pick a provider") // nothing is set up: the grid shows
}

func TestDataFolderIsShownRelativeToHomeButRevealedInFull(t *testing.T) {
	r := newSettingsRig(t)
	r.app.Config.DataFolder = "/Users/mac/Library/Application Support/adhd-todo"
	r.app.Config.Home = "/Users/mac"
	r.tt.Frame()
	r.tt.Scroll(340, 300, 0, 3000) // the data folder is below the notifications and sound groups
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
	has(t, r.tt, "Language", "System")
	hasNot(t, r.tt, "Install MCP")
	r.tt.Scroll(340, 300, 0, 3000) // the notifications and sound groups made the pane taller than the window
	r.tt.Frame()
	has(t, r.tt, "No archived projects. Archive one from its menu in the sidebar.",
		"Data folder", r.app.Config.DataFolder, "Show in Finder", "adhd-todo 0.2.0", "Open source, MIT license", "GitHub")
	hasFold(t, r.tt, "Archived projects")
	if err := r.tt.Click("Show in Finder"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rec.folders, []string{r.app.Config.DataFolder}) {
		t.Errorf("revealed %v", r.rec.folders)
	}
	if err := r.tt.Click("adhd-todo on GitHub"); err != nil {
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
	r.tt.Scroll(340, 300, 0, 3000) // the archived projects are below the notifications and sound groups
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
	has(t, r.tt, "Or run this once in a terminal, for Claude Code", r.app.Config.ConnectCommand, "Copy")
	hasNot(t, r.tt, "Connect Claude Code")
	r.tt.Scroll(340, 300, 0, 3000)
	r.tt.Frame()
	if err := r.tt.Click("Copy command"); err != nil {
		t.Fatal(err)
	}
	if r.tt.Clipboard() != r.app.Config.ConnectCommand {
		t.Errorf("clipboard %q", r.tt.Clipboard())
	}
	has(t, r.tt, "Copied")
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
	has(t, r.tt, "General", "Assistant", "Agents", "GitHub", "Sync")
	r.pane(t, "GitHub")
	has(t, r.tt, "Following pull requests isn't available here.")
	r.pane(t, "Assistant")
	has(t, r.tt, "The assistant isn't available here.")
	r.pane(t, "Sync")
	has(t, r.tt, "Sync isn't available here.")
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
