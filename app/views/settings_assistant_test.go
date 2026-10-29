package views

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// setUp stores a key for each provider and makes the first the one in use, as if the user had set
// them up before.
func (r *settingsRig) setUp(t *testing.T, ids ...string) {
	t.Helper()
	for _, id := range ids {
		mustNil(t, r.fm.Save(id, data.ProviderEdit{Key: "key-" + id}))
	}
	mustNil(t, r.fm.SetActive(ids[0]))
	r.tt.Frame()
}

// waitSaved lets Save and test finish and shows what it said.
func (r *settingsRig) waitSaved(t *testing.T) {
	t.Helper()
	for i := 0; i < 5 && r.s.assist.saving; i++ {
		r.rec.drain(t)
	}
	r.tt.Frame()
}

// waitModels runs what is posted until the provider's model list has come back; a list asked
// for another provider may come back first.
func (r *settingsRig) waitModels(t *testing.T, id string) {
	t.Helper()
	for l := r.s.assist.lists[id]; l != nil && l.running; {
		r.rec.drain(t)
	}
	r.tt.Frame()
}

func (r *settingsRig) assistant(t *testing.T) {
	t.Helper()
	r.pane(t, "Model")
}

func TestSwitchingPanesNeverChangesTheWindowSize(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	w, h := r.s.ContentSize()
	if w != 800 || h != 580 {
		t.Fatalf("content size %v x %v, want 800 x 580", w, h)
	}
	rows := func() map[string]float32 { // the sidebar rows, by the name they open
		out := map[string]float32{}
		for _, name := range []string{"Account", "General", "Categories", "Agents", "Model", "About"} {
			if rect, ok := r.tt.Find(r.s.say.A11yPane(name)); ok {
				out[name] = rect.Y
			}
		}
		return out
	}
	before := rows()
	for _, name := range []string{"Model", "Agents", "Categories", "About", "Account", "General", "Model"} {
		r.pane(t, name)
		if gw, gh := r.s.ContentSize(); gw != w || gh != h {
			t.Errorf("after %q the content size is %v x %v", name, gw, gh)
		}
		after := rows()
		for row, y := range before {
			if after[row] != y {
				t.Errorf("after %q the sidebar row %q moved from %v to %v", name, row, y, after[row])
			}
		}
	}
	mustNil(t, r.tt.Click("Set up Kimi")) // a tile opens its form under the grid
	r.tt.Frame()
	after := rows()
	for row, y := range before {
		if after[row] != y {
			t.Errorf("the provider grid moved the sidebar row %q to %v", row, after[row])
		}
	}
}

func TestAssistantAtRestShowsTheProviderInUseAndTheGrid(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	mustNil(t, r.fm.Save("ollama", data.ProviderEdit{Model: "qwen3:8b"}))
	r.assistant(t)
	hasFold(t, r.tt, "Switch to")
	has(t, r.tt, "Anthropic", "claude-haiku-5-5", "Not tested yet", "Test", "Switch to",
		"OpenAI", "DeepSeek", "Kimi", "Ollama", "Magpie", "Custom")
	hasNot(t, r.tt, "In use", "Also set up", "API key", "Save and test")
}

func TestTheTestResultShowsOnTheCardAsConnectedOrTheFriendlyError(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic")
	r.assistant(t)
	mustNil(t, r.tt.Click("Test"))
	has(t, r.tt, "· Testing…")
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Connected · 640 ms")
	hasNot(t, r.tt, "Not tested yet", "Test again")

	r.fm.TestFn = func(string, data.ProviderEdit) (time.Duration, error) {
		return 0, errors.New("The provider didn't accept the API key.")
	}
	r.s.assist.test = testResult{}
	r.tt.Frame()
	mustNil(t, r.tt.Click("Test"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "The provider didn't accept the API key.", "Test")
	hasNot(t, r.tt, "Test again", "Connected · 640 ms")
}

func TestChoosingASetUpProviderMakesItTheOneInUseAndTestsIt(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	mustNil(t, r.tt.Click("Use DeepSeek"))
	if activeID(r.fm) != "deepseek" {
		t.Fatalf("active = %q", activeID(r.fm))
	}
	if r.s.assist.pick != "" {
		t.Errorf("choosing a set-up provider opened the form for %q", r.s.assist.pick)
	}
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Connected · 640 ms", "deepseek-chat")
	hasNot(t, r.tt, "API key", "Save and test")
	if r.s.assist.test.id != "deepseek" {
		t.Errorf("the test belongs to %q", r.s.assist.test.id)
	}
}

func TestTheTestResultBelongsToOneProvider(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	mustNil(t, r.tt.Click("Test"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Connected · 640 ms")
	r.s.assist.test.id = "deepseek"
	r.tt.Frame()
	has(t, r.tt, "Not tested yet")
	hasNot(t, r.tt, "Connected · 640 ms")
}

func TestTheGridAlwaysShowsTwelveProviders(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	has(t, r.tt, "Anthropic", "OpenAI", "DeepSeek", "Kimi", "智谱 GLM", "通义千问",
		"Gemini", "OpenRouter", "硅基流动", "Ollama", "Magpie", "Custom")
	hasNot(t, r.tt, "Pick a provider", "Done")
	// The tile of the one in use opens its form: that is where its key and model are changed.
	mustNil(t, r.tt.Click("Change Anthropic"))
	has(t, r.tt, "API key", "Save and test")
}

func TestTilesAreInTheOrderOfTheMockup(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	var ids []string
	for _, p := range r.fm.Providers() {
		ids = append(ids, p.ID)
	}
	want := []string{"anthropic", "openai", "deepseek", "kimi", "zhipu", "qwen", "gemini", "openrouter", "siliconflow", "ollama", "magpie", "custom"}
	if !slices.Equal(ids, want) {
		t.Errorf("providers %v", ids)
	}
	first, _ := r.tt.Find("Anthropic")
	kimi, _ := r.tt.Find("Kimi")
	gemini, _ := r.tt.Find("Gemini")
	if kimi.Y != first.Y || kimi.X <= first.X {
		t.Errorf("the first row is not Anthropic, OpenAI, DeepSeek, Kimi: %v and %v", first, kimi)
	}
	if gemini.Y <= first.Y {
		t.Errorf("Gemini at %v is not under Anthropic at %v", gemini.Y, first.Y)
	}
}

func TestNothingSetUpShowsTheOneLineAndTheGrid(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	has(t, r.tt, "Not set up: dates and reminders are still read on this computer.")
	hasFold(t, r.tt, "Switch to")
	has(t, r.tt, "Anthropic", "OpenAI", "DeepSeek", "Kimi", "智谱 GLM", "通义千问", "Gemini", "OpenRouter", "硅基流动", "Ollama", "Magpie", "Custom")
	hasNot(t, r.tt, "Done", "In use", "API key") // nothing is picked yet
}

func TestAPickedTileShowsKeyModelAndLinkToGetAKey(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	mustNil(t, r.tt.Click("Set up Kimi"))
	has(t, r.tt, "API key", "Paste your Kimi key", "Get a key", "Model", "kimi-k2-0905-preview", "Save and test")
	hasNot(t, r.tt, "Base URL", "Protocol")
	mustNil(t, r.tt.Click("Get a key"))
	if !slices.Equal(r.rec.urls, []string{"https://platform.moonshot.cn/console/api-keys"}) {
		t.Errorf("opened %v", r.rec.urls)
	}
	mustNil(t, r.tt.Click("Choose a model"))
	r.tt.Frame()
	has(t, r.tt, "kimi-k2-turbo-preview", "Other…")
}

func TestMagpieAndOllamaHaveNoKeyField(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	for _, name := range []string{"Magpie", "Ollama"} {
		mustNil(t, r.tt.Click("Set up "+name))
		has(t, r.tt, "Model", "Save and test")
		hasNot(t, r.tt, "API key", "Get a key")
	}
}

func TestSavingAndTestingAKeyMakesTheProviderTheOneInUse(t *testing.T) {
	r := newSettingsRig(t)
	var got data.ProviderEdit
	r.fm.TestFn = func(_ string, e data.ProviderEdit) (time.Duration, error) {
		got = e
		return 312 * time.Millisecond, nil
	}
	r.setUp(t, "anthropic")
	r.assistant(t)
	mustNil(t, r.tt.Click("Set up Kimi"))
	mustNil(t, r.tt.Click("Paste your Kimi key"))
	r.tt.Type("sk-secret-value")
	hasNot(t, r.tt, "sk-secret-value") // a secure field
	mustNil(t, r.tt.Click("Save and test"))
	has(t, r.tt, "Testing…")
	r.waitSaved(t)
	if activeID(r.fm) != "kimi" || !r.fm.KeySaved("kimi") {
		t.Fatalf("active %q, key saved %v", activeID(r.fm), r.fm.KeySaved("kimi"))
	}
	if got.Model != "kimi-k2-0905-preview" || got.BaseURL != "https://api.moonshot.cn/v1" {
		t.Errorf("the test was sent %+v", got)
	}
	hasFold(t, r.tt, "Switch to")
	has(t, r.tt, "Kimi", "Connected · 312 ms")
	hasNot(t, r.tt, "sk-secret-value")
}

func TestEnterInTheKeyFieldSavesAndTests(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	mustNil(t, r.tt.Click("Set up DeepSeek"))
	mustNil(t, r.tt.Click("Paste your DeepSeek key"))
	r.tt.Type("sk-1")
	r.tt.Key(0, ui.KeyEnter)
	r.waitSaved(t)
	if activeID(r.fm) != "deepseek" {
		t.Errorf("active = %q", activeID(r.fm))
	}
}

func TestAFailedTestStaysInTheFormInPlainWordsAndKeepsTheKey(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.TestFn = func(string, data.ProviderEdit) (time.Duration, error) {
		return 0, errors.New("The provider didn't accept the API key.")
	}
	r.assistant(t)
	mustNil(t, r.tt.Click("Set up OpenAI"))
	mustNil(t, r.tt.Click("Paste your OpenAI key"))
	r.tt.Type("sk-bad")
	mustNil(t, r.tt.Click("Save and test"))
	r.waitSaved(t)
	has(t, r.tt, "The provider didn't accept the API key.", "Save and test")
	if activeID(r.fm) == "openai" {
		t.Error("a provider that failed its test is in use")
	}
	hasNot(t, r.tt, "sk-bad")
	// The fix needs only a click: the key is stored.
	r.fm.TestFn = nil
	mustNil(t, r.tt.Click("Save and test"))
	r.waitSaved(t)
	if activeID(r.fm) != "openai" {
		t.Errorf("active = %q after the retry", activeID(r.fm))
	}
}

func TestSaveAndTestAsksForWhatIsMissing(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	mustNil(t, r.tt.Click("Set up OpenAI"))
	mustNil(t, r.tt.Click("Save and test"))
	has(t, r.tt, "Needs an API key.")
	if r.s.assist.saving {
		t.Error("it tested with no key")
	}
	mustNil(t, r.tt.Click("Set up Custom"))
	mustNil(t, r.tt.Click("Save and test"))
	has(t, r.tt, "Enter the address of the endpoint.")
}

func TestCustomAddsTheAddressAndTheProtocol(t *testing.T) {
	r := newSettingsRig(t)
	var got data.ProviderEdit
	r.fm.TestFn = func(_ string, e data.ProviderEdit) (time.Duration, error) {
		got = e
		return time.Millisecond, nil
	}
	r.assistant(t)
	mustNil(t, r.tt.Click("Set up Custom"))
	has(t, r.tt, "Base URL", "Protocol", "OpenAI-compatible", "Anthropic API", "API key", "Model")
	mustNil(t, r.tt.Click("https://…"))
	r.tt.Type("http://127.0.0.1:9000/v1")
	mustNil(t, r.tt.Click("Anthropic API"))
	r.tt.Frame()
	mustNil(t, r.tt.Click("Save and test"))
	has(t, r.tt, "Choose or type a model.")
	save, _ := r.tt.Find("Save and test")
	r.tt.ClickAt(save.X-80, save.Y+save.H/2) // the empty model field beside the button
	r.tt.Type("my-model")
	r.tt.Key(0, ui.KeyEnter)
	r.waitSaved(t)
	if got.BaseURL != "http://127.0.0.1:9000/v1" || got.Model != "my-model" || !got.Anthropic {
		t.Errorf("the test was sent %+v", got)
	}
	if activeID(r.fm) != "custom" {
		t.Errorf("active = %q", activeID(r.fm))
	}
}

func TestOtherTurnsTheModelIntoATextFieldThatSaves(t *testing.T) {
	r := newSettingsRig(t)
	var got data.ProviderEdit
	r.fm.TestFn = func(_ string, e data.ProviderEdit) (time.Duration, error) {
		got = e
		return time.Millisecond, nil
	}
	r.setUp(t, "anthropic")
	r.assistant(t)
	mustNil(t, r.tt.Click("Change Anthropic")) // the one in use opens its form
	mustNil(t, r.tt.Click("Choose a model"))
	r.tt.Frame()
	mustNil(t, r.tt.Click("Other…"))
	r.tt.Frame()
	r.tt.Key(ui.Cmd, ui.KeyA)
	r.tt.Type("claude-fable-9")
	r.tt.Key(0, ui.KeyEnter)
	r.waitSaved(t)
	if got.Model != "claude-fable-9" {
		t.Errorf("the test was sent %+v", got)
	}
}

func TestTheModelListIsFetchedForTheStoredKeyAndOffersWhatTheProviderListed(t *testing.T) {
	r := newSettingsRig(t)
	var asked []data.ProviderEdit
	r.fm.ListFn = func(id string, e data.ProviderEdit) ([]string, error) {
		if id == "deepseek" {
			asked = append(asked, e)
		}
		return []string{"deepseek-chat", "deepseek-reasoner"}, nil
	}
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	mustNil(t, r.tt.Click("Use DeepSeek")) // a set-up provider switches and is tested
	r.tt.Frame()
	mustNil(t, r.tt.Click("Change DeepSeek")) // its tile, now in use, opens its form
	r.waitModels(t, "deepseek")
	if len(asked) != 1 || asked[0].Key != "" {
		t.Fatalf("asked %+v, want one request with the stored key", asked)
	}
	mustNil(t, r.tt.Click("Choose a model"))
	r.tt.Frame()
	has(t, r.tt, "deepseek-reasoner", "Other…")
	mustNil(t, r.tt.Click("deepseek-reasoner"))
	if got := r.s.draftOf("deepseek").model; got != "deepseek-reasoner" {
		t.Errorf("model = %q", got)
	}
}

func TestATypedKeyFetchesTheModelListWhenTheFieldIsLeft(t *testing.T) {
	r := newSettingsRig(t)
	var asked []string
	r.fm.ListFn = func(_ string, e data.ProviderEdit) ([]string, error) {
		asked = append(asked, e.Key)
		return []string{"kimi-k2-0905-preview", "kimi-latest"}, nil
	}
	r.assistant(t)
	mustNil(t, r.tt.Click("Set up Kimi"))
	if len(asked) != 0 {
		t.Fatalf("asked %v before there was a key", asked)
	}
	mustNil(t, r.tt.Click("Paste your Kimi key"))
	r.tt.Type("sk-typed")
	r.tt.Key(0, ui.KeyTab) // leave the field
	r.tt.Frame()
	r.rec.drain(t)
	r.tt.Frame()
	if !slices.Equal(asked, []string{"sk-typed"}) {
		t.Fatalf("asked with keys %v", asked)
	}
	mustNil(t, r.tt.Click("Choose a model"))
	r.tt.Frame()
	has(t, r.tt, "kimi-latest")
}

func TestAFailedModelListFallsBackToTheSuggestionsWithoutAFuss(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.ListFn = func(string, data.ProviderEdit) ([]string, error) { return nil, errors.New("401") }
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	mustNil(t, r.tt.Click("Use DeepSeek")) // a set-up provider switches and is tested
	r.tt.Frame()
	mustNil(t, r.tt.Click("Change DeepSeek")) // its tile, now in use, opens its form
	r.waitModels(t, "deepseek")
	has(t, r.tt, "deepseek-chat")
	if r.s.assist.err != "" {
		t.Errorf("the list failed out loud: %q", r.s.assist.err)
	}
}

func TestModelChoicesAreTheSuggestionsThenTheListedModelsWithoutRepeatsAndBounded(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	p := data.Provider{ID: "kimi", Suggestions: []string{"a", "b"}}
	var listed []string
	for i := 0; i < 100; i++ {
		listed = append(listed, "m"+itoa(i))
	}
	r.s.assist.lists["kimi"] = &modelList{names: append([]string{"b"}, listed...)}
	got := r.s.modelChoices(p, "mine")
	if got[0] != "mine" || got[1] != "a" || got[2] != "b" || got[3] != "m0" {
		t.Errorf("order %v", got[:4])
	}
	if len(got) != maxModelChoices+1 { // the 24 plus the current model, which is always offered
		t.Errorf("%d choices", len(got))
	}
	if n := strings.Count(strings.Join(got, ","), ",b,"); n != 1 {
		t.Errorf("b appears %d times", n)
	}
	// With nothing to offer there is no pop-up: the model is typed.
	if got := r.s.modelChoices(data.Provider{ID: "custom"}, "x"); len(got) != 0 {
		t.Errorf("choices %v for a provider with none", got)
	}
}

func TestInChineseTheAssistantPaneSaysItInChinese(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.setUp(t, "anthropic", "deepseek")
	r.s.Show(PaneModel)
	r.tt.Frame()
	has(t, r.tt, "换一个", "测试")
	hasFold(t, r.tt, "换一个")
	mustNil(t, r.tt.Click("设置 Kimi"))
	has(t, r.tt, "智谱 GLM", "通义千问", "硅基流动", "自定义", "保存并测试")
}

func TestAssistantMemoryRowOpensTheFileOrTheFolder(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "assistant-memory.md")
	r := newSettingsRig(t, func(a *App) { a.Config.AssistantMemory = file })
	r.setUp(t, "anthropic")
	r.assistant(t)
	has(t, r.tt, "Assistant memory", "Open file")
	if err := r.tt.Click("Open file"); err != nil {
		t.Fatal(err)
	}
	if len(r.rec.files) != 0 || !slices.Equal(r.rec.folders, []string{dir}) { // nothing written yet: the folder shows
		t.Errorf("files %v, folders %v", r.rec.files, r.rec.folders)
	}
	mustNil(t, os.WriteFile(file, []byte("# 助手记忆\n"), 0o644))
	r.tt.Frame()
	if err := r.tt.Click("Open file"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rec.files, []string{file}) {
		t.Errorf("files %v", r.rec.files)
	}
}

func TestAssistantMemoryRowIsHiddenWithoutAFile(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic")
	r.assistant(t)
	hasNot(t, r.tt, "Assistant memory")
}
