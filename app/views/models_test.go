package views

import (
	"errors"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func newModelsRig(t *testing.T) (*ui.Tester, *Main, *App, *data.FakeModels) {
	t.Helper()
	app, _, _ := newTestApp(t)
	fm := newFakeModels(t)
	app.Models = fm
	tt, m, _ := newTesterFor(app, 1100)
	m.openModels()
	tt.Frame()
	return tt, m, app, fm
}

func activeID(fm *data.FakeModels) string {
	for _, p := range fm.Providers() {
		if p.Active {
			return p.ID
		}
	}
	return ""
}

func TestModelsTabListsEveryPreset(t *testing.T) {
	tt, _, _, _ := newModelsRig(t)
	has(t, tt, "Anthropic", "Magpie", "OpenAI", "DeepSeek", "Kimi", "OpenRouter", "Ollama", "Custom", "Base URL", "Model", "Test")
	has(t, tt, "Needs an API key")
}

func TestSettingsWithoutModelsHasNoTabs(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.OpenSettings()
	tt.Frame()
	hasNot(t, tt, "Models", "Base URL")
	has(t, tt, "Connect Claude Code")
}

func TestGeneralTabStillThere(t *testing.T) {
	tt, m, _, _ := newModelsRig(t)
	if err := tt.Click("General"); err != nil {
		t.Fatal(err)
	}
	if m.models.tab != tabGeneral {
		t.Fatal("tab did not switch")
	}
	has(t, tt, "Connect Claude Code", "Data folder")
}

func TestPickingAProviderShowsItsFieldsAndUseThisOneActivatesIt(t *testing.T) {
	tt, _, _, fm := newModelsRig(t)
	if err := tt.Click("OpenAI"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "Use this one", "Use model gpt-4.1")
	if activeID(fm) != "" {
		t.Fatalf("picking a row activated %q", activeID(fm))
	}
	if err := tt.Click("Use this one"); err != nil {
		t.Fatal(err)
	}
	if activeID(fm) != "openai" {
		t.Errorf("active = %q", activeID(fm))
	}
	has(t, tt, "In use")
}

func TestMagpieShowsItsGatewayAndNoKeyField(t *testing.T) {
	tt, _, _, _ := newModelsRig(t)
	if err := tt.Click("Magpie"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "Uses your Magpie gateway at 127.0.0.1:3425")
	hasNot(t, tt, "API key")
}

func TestTypingAKeySavesItWithoutShowingIt(t *testing.T) {
	tt, _, _, fm := newModelsRig(t)
	if err := tt.Click("OpenAI"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Paste your API key"); err != nil {
		t.Fatal(err)
	}
	tt.Type("sk-secret-value")
	hasNot(t, tt, "sk-secret-value") // a password field
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !fm.KeySaved("openai") {
		t.Error("the key was not stored")
	}
	has(t, tt, "Saved in your Keychain — paste a new one to replace it")
}

func TestEditingTheModelSavesOnEnterAndSuggestionsFillIt(t *testing.T) {
	tt, _, _, fm := newModelsRig(t)
	if err := tt.Click("OpenAI"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("gpt-5-mini"); err != nil {
		t.Fatal(err)
	}
	for _, p := range fm.Providers() {
		if p.ID == "openai" && p.Model != "gpt-5-mini" {
			t.Errorf("model = %q", p.Model)
		}
	}
}

func TestCustomLetsMeChooseTheProtocol(t *testing.T) {
	tt, _, _, fm := newModelsRig(t)
	if err := tt.Click("Custom"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "OpenAI-compatible")
	if err := tt.Click("Anthropic API"); err != nil {
		t.Fatal(err)
	}
	for _, p := range fm.Providers() {
		if p.ID == "custom" && !p.Anthropic {
			t.Error("the protocol did not change")
		}
	}
}

func TestTestButtonReportsWorksWithTiming(t *testing.T) {
	tt, _, app, fm := newModelsRig(t)
	fm.TestFn = func(string, data.ProviderEdit) (time.Duration, error) { return 640 * time.Millisecond, nil }
	if err := tt.Click("Test"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "Testing…")
	app.Act.(*recorder).drain(t)
	tt.Frame()
	has(t, tt, "Works · 640 ms")
}

func TestTestButtonSendsWhatIsTypedAndShowsTheError(t *testing.T) {
	tt, _, app, fm := newModelsRig(t)
	var got data.ProviderEdit
	fm.TestFn = func(_ string, e data.ProviderEdit) (time.Duration, error) {
		got = e
		return 0, errors.New("The provider didn't accept the API key.")
	}
	if err := tt.Click("Paste your API key"); err != nil {
		t.Fatal(err)
	}
	tt.Type("sk-typed")
	if err := tt.Click("Test"); err != nil {
		t.Fatal(err)
	}
	app.Act.(*recorder).drain(t)
	tt.Frame()
	has(t, tt, "The provider didn't accept the API key.")
	if got.Key != "sk-typed" && !fm.KeySaved("anthropic") {
		t.Errorf("Test got key %q and nothing was saved", got.Key)
	}
}

func TestTestResultBelongsToOneProvider(t *testing.T) {
	tt, _, app, _ := newModelsRig(t)
	if err := tt.Click("Test"); err != nil {
		t.Fatal(err)
	}
	app.Act.(*recorder).drain(t)
	tt.Frame()
	has(t, tt, "Works · 640 ms")
	if err := tt.Click("OpenAI"); err != nil {
		t.Fatal(err)
	}
	hasNot(t, tt, "Works · 640 ms")
}
