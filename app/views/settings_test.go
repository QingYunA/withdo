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
	r.tt = newTester(r.s.View, int(w), int(h))
	r.tt.SetTitleBar(ui.TitleBar{Height: SettingsTitleBarHeight, Left: 78})
	still(r.tt)
	r.tt.Frame()
	r.tt.Frame()
	return r
}

// pane opens a pane by its name from the sidebar.
func (r *settingsRig) pane(t *testing.T, name string) {
	t.Helper()
	if err := r.tt.Click(r.s.say.A11yPane(name)); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
}

func TestSettingsSidebarHasEveryPaneAndTheTitleIsTheOpenOne(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "Account", "General", "Categories", "Agents", "Model", "About Withdo")
	hasNot(t, r.tt, "Advanced", "Assistant", "Sync")
	if r.title != "General" {
		t.Errorf("title %q, want the open pane's name", r.title)
	}
	for _, want := range []struct {
		name, title string
		pane        Pane
	}{
		{"Model", "Model", PaneModel},
		{"Agents", "Agents", PaneAgents},
		{"Categories", "Categories", PaneCategories},
		{"About", "About", PaneAbout},
		{"Account", "Account", PaneAccount},
		{"General", "General", PaneGeneral},
	} {
		r.pane(t, want.name)
		if r.s.pane != want.pane || r.title != want.title {
			t.Errorf("after %q: pane %v, title %q", want.name, r.s.pane, r.title)
		}
		has(t, r.tt, want.title)
	}
}

// The sidebar moves with Up and Down while it has the keyboard: from the row clicked, through the
// rows, and back.
func TestSettingsSidebarArrowsMoveBetweenPanes(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "General")
	for _, want := range []Pane{PaneCategories, PaneAgents, PaneModel, PaneAbout} {
		r.tt.Key(0, ui.KeyDown)
		r.tt.Frame()
		if r.s.pane != want {
			t.Fatalf("Down went to %v, want %v", r.s.pane, want)
		}
	}
	r.tt.Key(0, ui.KeyDown) // the last row: it stays
	if r.s.pane != PaneAbout {
		t.Fatalf("Down from About went to %v", r.s.pane)
	}
	for _, want := range []Pane{PaneModel, PaneAgents, PaneCategories, PaneGeneral, PaneAccount} {
		r.tt.Key(0, ui.KeyUp)
		r.tt.Frame()
		if r.s.pane != want {
			t.Fatalf("Up went to %v, want %v", r.s.pane, want)
		}
	}
	if got := r.title; got != "Account" {
		t.Errorf("title %q, want Account", got)
	}
}

func TestSettingsShowSelectsAPane(t *testing.T) {
	r := newSettingsRig(t)
	r.s.Show(PaneModel)
	r.tt.Frame()
	if r.title != "Model" {
		t.Errorf("title %q", r.title)
	}
	hasFold(t, r.tt, "Switch to") // nothing is set up: the line and the grid show
	has(t, r.tt, "Not set up: dates and reminders are still read on this computer.")
}

func TestDataFolderIsShownRelativeToHomeButRevealedInFull(t *testing.T) {
	r := newSettingsRig(t)
	r.app.Config.DataFolder = "/Users/mac/Library/Application Support/withdo"
	r.app.Config.Home = "/Users/mac"
	r.tt.Frame()
	has(t, r.tt, "~/Library/Application Support/withdo")
	hasNot(t, r.tt, "/Users/mac/Library/Application Support/withdo")
	if err := r.tt.Click("Show in Finder"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rec.folders, []string{"/Users/mac/Library/Application Support/withdo"}) {
		t.Errorf("revealed %v", r.rec.folders)
	}
	if got := withTilde("/Users/macro/x", "/Users/mac"); got != "/Users/macro/x" {
		t.Errorf("a sibling of the home folder became %q", got)
	}
}

func TestGeneralPaneHasLanguageNotificationsAndTheDataFolder(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "Language", "Follow system", "English", "简体中文")
	hasNot(t, r.tt, "Install MCP", "Archived projects", "Privacy policy")
	hasFold(t, r.tt, "Notifications and sound")
	has(t, r.tt, "System notifications", "End-of-day reminder", "Sound effects")
	hasFold(t, r.tt, "Updates and data")
	has(t, r.tt, "Data folder", r.app.Config.DataFolder, "Show in Finder")
	if err := r.tt.Click("Show in Finder"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rec.folders, []string{r.app.Config.DataFolder}) {
		t.Errorf("revealed %v", r.rec.folders)
	}
}

func TestCategoriesPaneHasTheCategoriesAndTheArchivedProjects(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Categories")
	has(t, r.tt, "New category…", "Archived projects", "No archived projects. Archive one from its menu in the sidebar.")
	hasNot(t, r.tt, "Language", "End-of-day reminder")
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

func TestAboutPaneSaysAGPLAndLinksToTheApp(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "About")
	has(t, r.tt, "Withdo 0.2.0", "Open source, AGPL-3.0", "GitHub", "Privacy policy", "Website")
	hasNot(t, r.tt, "MIT")
	for _, want := range []struct{ click, url string }{
		{"Withdo on GitHub", repoURL},
		{"Privacy policy", privacyURL},
		{"Website", websiteURL},
	} {
		if err := r.tt.Click(want.click); err != nil {
			t.Fatal(err)
		}
		if len(r.rec.urls) == 0 || r.rec.urls[len(r.rec.urls)-1] != want.url {
			t.Errorf("%q opened %v, want %s", want.click, r.rec.urls, want.url)
		}
	}
}

func TestLanguageSegmentedControlChoosesALanguageOrTheSystemOne(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "Follow system", "English", "简体中文")
	if err := r.tt.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	if r.rec.language != "zh-Hans" {
		t.Errorf("chose %q", r.rec.language)
	}
	r.rec.language = "zh-Hans"
	r.tt.Frame()
	if err := r.tt.Click("English"); err != nil {
		t.Fatal(err)
	}
	if r.rec.language != "en" {
		t.Errorf("chose %q", r.rec.language)
	}
	if err := r.tt.Click("Follow system"); err != nil {
		t.Fatal(err)
	}
	if r.rec.language != "" {
		t.Errorf("chose %q after Follow system", r.rec.language)
	}
}

// The segmented control is a radio group: Left and Right choose among its parts while it has the
// keyboard, without a pop-up.
func TestLanguageSegmentedControlMovesWithTheArrowKeys(t *testing.T) {
	r := newSettingsRig(t)
	if err := r.tt.Click("English"); err != nil { // a click gives the control the keyboard
		t.Fatal(err)
	}
	if r.rec.language != "en" {
		t.Fatalf("the click chose %q", r.rec.language)
	}
	r.tt.Key(0, ui.KeyRight)
	r.tt.Frame()
	if r.rec.language != "zh-Hans" {
		t.Errorf("Right chose %q, want 简体中文", r.rec.language)
	}
	r.tt.Key(0, ui.KeyLeft)
	r.tt.Frame()
	if r.rec.language != "en" {
		t.Errorf("Left chose %q, want English", r.rec.language)
	}
	r.tt.Key(0, ui.KeyLeft)
	r.tt.Frame()
	if r.rec.language != "" {
		t.Errorf("Left again chose %q, want Follow system", r.rec.language)
	}
}

func TestAgentsPaneCopiesTheCommand(t *testing.T) {
	r := newSettingsRig(t)
	r.pane(t, "Agents")
	has(t, r.tt, "Connect by hand")
	hasNot(t, r.tt, "Or run this once in a terminal, for Claude Code", "Connect Claude Code")
	mustNil(t, r.tt.Click("Connect by hand"))
	r.tt.Frame()
	has(t, r.tt, "Or run this once in a terminal, for Claude Code", r.app.Config.ConnectCommand, "Copy")
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
	r.pane(t, "Agents")
	r.tt.Scroll(340, 300, 0, 9000)
	r.tt.Frame()
	has(t, r.tt, "Connected as QingYunA · checked 2m ago", "Check now")
	hasNot(t, r.tt, "brew install gh", "gh auth login")
	mustNil(t, r.tt.Click("Check now"))
	if gh := r.app.GitHub.(*data.FakeGitHub); gh.Checks != 1 {
		t.Errorf("CheckNow called %d times", gh.Checks)
	}
}

func TestSettingsWithoutPollerOrModelsStillHasEveryPane(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.GitHub, a.Models = nil, nil })
	has(t, r.tt, "Account", "General", "Categories", "Agents", "Model", "About Withdo")
	r.pane(t, "Agents")
	r.tt.Scroll(340, 300, 0, 9000)
	r.tt.Frame()
	has(t, r.tt, "Following pull requests isn't available here.", "Recent changes")
	r.pane(t, "Model")
	has(t, r.tt, "The assistant isn't available here.")
	r.pane(t, "Account")
	has(t, r.tt, "Sync isn't available here.")
	r.pane(t, "About")
	has(t, r.tt, "Open source, AGPL-3.0")
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

// The switches that left Advanced for General, and the ones that stayed behind with the model.
func TestTheReminderAndSoundMovedToGeneralAndRecentChangesToTheGitHubGroup(t *testing.T) {
	r := newSettingsRig(t)
	has(t, r.tt, "End-of-day reminder", "Sound effects")
	hasNot(t, r.tt, "Recent changes")
	r.pane(t, "Model")
	hasFold(t, r.tt, "Switch to") // a group label is set in the section style
	hasNot(t, r.tt, "Recent changes", "End-of-day reminder", "Sound effects")
	r.pane(t, "Agents")
	r.tt.Scroll(340, 300, 0, 9000)
	r.tt.Frame()
	has(t, r.tt, "Recent changes")
}
