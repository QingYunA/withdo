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
	r.pane(t, "Assistant")
}

func TestSwitchingPanesNeverChangesTheWindowSize(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	w, h := r.s.ContentSize()
	if w != 680 || h != 520 {
		t.Fatalf("content size %v x %v, want 680 x 520", w, h)
	}
	bandBottom := func() float32 { // the labels of the panes: the lowest of the texts of these names
		var low float32
		for _, name := range []string{"General", "Assistant", "Agents", "GitHub"} {
			if rect, ok := r.tt.Find(name); ok {
				low = max(low, rect.Y+rect.H)
			}
		}
		return low
	}
	toolbar := bandBottom()
	for _, name := range []string{"Assistant", "Agents", "GitHub", "General", "Assistant"} {
		r.pane(t, name)
		if gw, gh := r.s.ContentSize(); gw != w || gh != h {
			t.Errorf("after %q the content size is %v x %v", name, gw, gh)
		}
		if got := bandBottom(); got != toolbar {
			t.Errorf("after %q the toolbar moved from %v to %v", name, toolbar, got)
		}
	}
	mustNil(t, r.tt.Click("Change"))
	r.tt.Frame()
	if got := bandBottom(); got != toolbar {
		t.Errorf("the provider grid moved the toolbar to %v", got)
	}
}

func TestAssistantAtRestShowsTheProviderInUseAndTheOthersSetUp(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	mustNil(t, r.fm.Save("ollama", data.ProviderEdit{Model: "qwen3:8b"}))
	r.assistant(t)
	hasFold(t, r.tt, "In use")
	hasFold(t, r.tt, "Also set up")
	has(t, r.tt, "Anthropic", "claude-haiku-5-5", "Not tested yet", "Test", "Change", "DeepSeek", "deepseek-chat", "Ollama", "Use")
	hasNot(t, r.tt, "Pick a provider", "OpenAI", "Kimi", "Magpie")
	// A provider with no key and no use is not "also set up".
	mustNil(t, r.fm.Save("openai", data.ProviderEdit{Model: "gpt-4.1"}))
	r.tt.Frame()
	hasNot(t, r.tt, "OpenAI")
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
	has(t, r.tt, "The provider didn't accept the API key.", "Test again")
	hasNot(t, r.tt, "Connected · 640 ms")
}

func TestUseMakesAnotherProviderTheOneInUseAndTestsIt(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	mustNil(t, r.tt.Click("Use DeepSeek"))
	if activeID(r.fm) != "deepseek" {
		t.Fatalf("active = %q", activeID(r.fm))
	}
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Connected · 640 ms", "deepseek-chat", "Use Anthropic")
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

func TestChangeOpensTwelveTilesAndDoneGoesBack(t *testing.T) {
	r := newSettingsRig(t)
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	mustNil(t, r.tt.Click("Change"))
	hasFold(t, r.tt, "Pick a provider")
	has(t, r.tt, "Anthropic", "OpenAI", "DeepSeek", "Kimi", "智谱 GLM", "通义千问", "Gemini", "OpenRouter", "硅基流动", "Ollama", "Magpie", "Custom", "Done")
	hasNot(t, r.tt, "In use", "Also set up")
	// The one in use is picked, so its fields are there to change.
	has(t, r.tt, "API key", "Save and test")
	mustNil(t, r.tt.Click("Done"))
	hasFold(t, r.tt, "In use")
	hasNot(t, r.tt, "Pick a provider", "Save and test")
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
	row1, _ := r.tt.Find("Anthropic")
	row2, _ := r.tt.Find("Gemini")
	if row2.Y <= row1.Y {
		t.Errorf("Gemini at %v is not under Anthropic at %v", row2.Y, row1.Y)
	}
}

func TestNothingSetUpShowsTheGridWithoutDone(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	hasFold(t, r.tt, "Pick a provider")
	hasNot(t, r.tt, "Done", "In use", "API key") // nothing is picked yet
}

func TestAPickedTileShowsKeyModelAndLinkToGetAKey(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	mustNil(t, r.tt.Click("Kimi"))
	has(t, r.tt, "API key", "Paste your Kimi key", "Get a key", "Model", "kimi-k2-0905-preview", "Save and test")
	hasNot(t, r.tt, "Base URL", "Protocol")
	mustNil(t, r.tt.Click("Get a key"))
	if !slices.Equal(r.rec.urls, []string{"https://platform.moonshot.cn/console/api-keys"}) {
		t.Errorf("opened %v", r.rec.urls)
	}
	mustNil(t, r.tt.Click("kimi-k2-0905-preview"))
	has(t, r.tt, "kimi-k2-turbo-preview", "Other…")
}

func TestMagpieAndOllamaHaveNoKeyField(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	for _, name := range []string{"Magpie", "Ollama"} {
		mustNil(t, r.tt.Click(name))
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
	mustNil(t, r.tt.Click("Change"))
	mustNil(t, r.tt.Click("Kimi"))
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
	hasFold(t, r.tt, "In use")
	has(t, r.tt, "Kimi", "Connected · 312 ms", "Use Anthropic")
	hasNot(t, r.tt, "Pick a provider", "sk-secret-value")
}

func TestEnterInTheKeyFieldSavesAndTests(t *testing.T) {
	r := newSettingsRig(t)
	r.assistant(t)
	mustNil(t, r.tt.Click("DeepSeek"))
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
	mustNil(t, r.tt.Click("OpenAI"))
	mustNil(t, r.tt.Click("Paste your OpenAI key"))
	r.tt.Type("sk-bad")
	mustNil(t, r.tt.Click("Save and test"))
	r.waitSaved(t)
	has(t, r.tt, "The provider didn't accept the API key.", "Save and test", "Pick a provider")
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
	mustNil(t, r.tt.Click("OpenAI"))
	mustNil(t, r.tt.Click("Save and test"))
	has(t, r.tt, "Needs an API key.")
	if r.s.assist.saving {
		t.Error("it tested with no key")
	}
	mustNil(t, r.tt.Click("Custom"))
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
	mustNil(t, r.tt.Click("Custom"))
	has(t, r.tt, "Base URL", "Protocol", "OpenAI-compatible", "Anthropic API", "API key", "Model")
	mustNil(t, r.tt.Click("https://…"))
	r.tt.Type("http://127.0.0.1:9000/v1")
	mustNil(t, r.tt.Click("Anthropic API"))
	r.tt.Frame()
	mustNil(t, r.tt.Click("Save and test"))
	has(t, r.tt, "Choose or type a model.")
	label, _ := r.tt.Find("Model")
	r.tt.ClickAt(label.X+200, label.Y+label.H/2) // the empty field beside the label
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
	mustNil(t, r.tt.Click("Change"))
	mustNil(t, r.tt.Click("claude-haiku-5-5"))
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
	mustNil(t, r.tt.Click("Change"))
	mustNil(t, r.tt.Click("DeepSeek"))
	r.waitModels(t, "deepseek")
	if len(asked) != 1 || asked[0].Key != "" {
		t.Fatalf("asked %+v, want one request with the stored key", asked)
	}
	mustNil(t, r.tt.Click("deepseek-chat"))
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
	mustNil(t, r.tt.Click("Kimi"))
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
	mustNil(t, r.tt.Click("kimi-k2-0905-preview"))
	has(t, r.tt, "kimi-latest")
}

func TestAFailedModelListFallsBackToTheSuggestionsWithoutAFuss(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.ListFn = func(string, data.ProviderEdit) ([]string, error) { return nil, errors.New("401") }
	r.setUp(t, "anthropic", "deepseek")
	r.assistant(t)
	mustNil(t, r.tt.Click("Change"))
	mustNil(t, r.tt.Click("DeepSeek"))
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
	r.s.Show(PaneAssistant)
	r.tt.Frame()
	has(t, r.tt, "更换", "用这个")
	hasFold(t, r.tt, "当前在用")
	mustNil(t, r.tt.Click("更换"))
	has(t, r.tt, "选一个厂商", "完成", "智谱 GLM", "通义千问", "硅基流动", "自定义", "保存并测试")
}
