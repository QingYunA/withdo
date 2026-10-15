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
	// KeyURL is the page where the user gets an API key; empty for a preset without a key field.
	KeyURL string
	// Monogram is the letters on the provider's logo tile. The tile's color is the theme's.
	Monogram string
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

// Presets lists every provider in the order Settings shows them: the grid reads left to right.
var Presets = []Preset{
	{ID: IDAnthropic, Name: "Anthropic", Protocol: Anthropic, Monogram: "A", KeyURL: "https://console.anthropic.com/settings/keys",
		Models: []string{"claude-haiku-5-5", "claude-sonnet-5-5", "claude-opus-5-5"}},
	{ID: "openai", Name: "OpenAI", Protocol: OpenAI, BaseURL: "https://api.openai.com/v1", Monogram: "◉",
		KeyURL: "https://platform.openai.com/api-keys", Models: []string{"gpt-4.1-mini", "gpt-4.1", "gpt-5-mini"}},
	{ID: "deepseek", Name: "DeepSeek", Protocol: OpenAI, BaseURL: "https://api.deepseek.com", Monogram: "DS",
		KeyURL: "https://platform.deepseek.com/api_keys", Models: []string{"deepseek-chat"}},
	{ID: "kimi", Name: "Kimi", Protocol: OpenAI, BaseURL: "https://api.moonshot.cn/v1", Monogram: "K",
		KeyURL: "https://platform.moonshot.cn/console/api-keys", Models: []string{"kimi-k2-0905-preview", "kimi-k2-turbo-preview"}},
	{ID: "zhipu", Name: "智谱 GLM", Protocol: OpenAI, BaseURL: "https://open.bigmodel.cn/api/paas/v4", Monogram: "Z",
		KeyURL: "https://open.bigmodel.cn/usercenter/apikeys", Models: []string{"glm-4.6", "glm-4.5-air"}},
	{ID: "qwen", Name: "通义千问", Protocol: OpenAI, BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Monogram: "Q",
		KeyURL: "https://bailian.console.aliyun.com/?apiKey=1", Models: []string{"qwen-plus", "qwen-flash", "qwen3-max"}},
	{ID: "gemini", Name: "Gemini", Protocol: OpenAI, BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Monogram: "✦",
		KeyURL: "https://aistudio.google.com/apikey", Models: []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.5-flash-lite"}},
	{ID: "openrouter", Name: "OpenRouter", Protocol: OpenAI, BaseURL: "https://openrouter.ai/api/v1", Monogram: "OR",
		KeyURL: "https://openrouter.ai/keys", Models: []string{"anthropic/claude-sonnet-4.5", "openai/gpt-4.1-mini", "google/gemini-2.5-flash"}},
	{ID: "siliconflow", Name: "硅基流动", Protocol: OpenAI, BaseURL: "https://api.siliconflow.cn/v1", Monogram: "SF",
		KeyURL: "https://cloud.siliconflow.cn/account/ak", Models: []string{"Qwen/Qwen3-8B", "deepseek-ai/DeepSeek-V3", "moonshotai/Kimi-K2-Instruct"}},
	{ID: "ollama", Name: "Ollama", Protocol: OpenAI, BaseURL: "http://127.0.0.1:11434/v1", FixedKey: "ollama", Monogram: "◉",
		Models: []string{"qwen3:8b", "llama3.1:8b"}, Note: "Runs on this Mac. No key needed."},
	{ID: IDMagpie, Name: "Magpie", Protocol: Anthropic, BaseURL: "http://127.0.0.1:3425", FixedKey: "magpie", Monogram: "M",
		Models: []string{"claude-haiku-5-5", "claude-sonnet-5-5", "claude-opus-5-5"},
		Note:   "Uses your Magpie gateway at 127.0.0.1:3425"},
	{ID: IDCustom, Name: "Custom", Protocol: OpenAI, KeyOptional: true, PickProtocol: true, Monogram: "+",
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
