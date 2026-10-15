package providers

// Protocol is the wire format a provider speaks. Two cover nearly every provider and gateway.
type Protocol string

const (
	Anthropic Protocol = "anthropic" // the Anthropic Messages API
	OpenAI    Protocol = "openai"    // OpenAI-compatible chat completions
)

// Preset describes a provider the user can pick in Settings.
type Preset struct {
	ID       string
	Name     string
	Protocol Protocol
	BaseURL  string   // the default; empty means the SDK's own (api.anthropic.com)
	Models   []string // suggestions; the first is the default
	// FixedKey is sent as the API key when the provider needs none or a well-known one
	// ("magpie", "ollama"); such a preset has no key field.
	FixedKey string
	// KeyOptional lets a preset work with no key at all (a local server of your own).
	KeyOptional bool
	// Editable protocol: the Custom preset lets the user choose.
	PickProtocol bool
	Note         string
}

// Display is the base URL as shown: the default for a preset that leaves it to the SDK.
func (p Preset) Display() string {
	if p.BaseURL == "" && p.Protocol == Anthropic {
		return "https://api.anthropic.com"
	}
	return p.BaseURL
}

// IDs of the presets that need a mention elsewhere.
const (
	IDAnthropic = "anthropic"
	IDMagpie    = "magpie"
	IDCustom    = "custom"
)

// Presets lists every provider in the order Settings shows them.
var Presets = []Preset{
	{ID: IDAnthropic, Name: "Anthropic", Protocol: Anthropic,
		Models: []string{"claude-haiku-5-5", "claude-sonnet-5-5", "claude-opus-5-5"}},
	{ID: IDMagpie, Name: "Magpie", Protocol: Anthropic, BaseURL: "http://127.0.0.1:3425", FixedKey: "magpie",
		Models: []string{"claude-haiku-5-5", "claude-sonnet-5-5", "claude-opus-5-5"},
		Note:   "Uses your Magpie gateway at 127.0.0.1:3425"},
	{ID: "openai", Name: "OpenAI", Protocol: OpenAI, BaseURL: "https://api.openai.com/v1",
		Models: []string{"gpt-4.1-mini", "gpt-4.1", "gpt-5-mini"}},
	{ID: "deepseek", Name: "DeepSeek", Protocol: OpenAI, BaseURL: "https://api.deepseek.com",
		Models: []string{"deepseek-chat"}},
	{ID: "kimi", Name: "Kimi", Protocol: OpenAI, BaseURL: "https://api.moonshot.cn/v1",
		Models: []string{"kimi-k2-0905-preview", "kimi-k2-turbo-preview"}},
	{ID: "openrouter", Name: "OpenRouter", Protocol: OpenAI, BaseURL: "https://openrouter.ai/api/v1",
		Models: []string{"anthropic/claude-sonnet-4.5", "openai/gpt-4.1-mini", "google/gemini-2.5-flash"}},
	{ID: "ollama", Name: "Ollama", Protocol: OpenAI, BaseURL: "http://127.0.0.1:11434/v1", FixedKey: "ollama",
		Models: []string{"qwen3:8b", "llama3.1:8b"}, Note: "Runs on this Mac. No key needed."},
	{ID: IDCustom, Name: "Custom", Protocol: OpenAI, KeyOptional: true, PickProtocol: true,
		Note: "Any OpenAI-compatible or Anthropic-compatible endpoint."},
}

// PresetByID finds a preset.
func PresetByID(id string) (Preset, bool) {
	for _, p := range Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// NeedsKey reports whether Settings shows a key field and the preset cannot run without one.
func (p Preset) NeedsKey() bool { return p.FixedKey == "" && !p.KeyOptional }

// HasKeyField reports whether Settings shows a key field at all.
func (p Preset) HasKeyField() bool { return p.FixedKey == "" }
