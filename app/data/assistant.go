package data

import (
	"context"
	"time"
)

// Assistant is the built-in agent behind the ask field, as the UI sees it.
type Assistant interface {
	// Configured reports whether a model is ready. Until it is, the ask field adds Inbox tasks.
	Configured() bool
	// Ask runs one turn of the current conversation. It blocks, so the UI calls it off its
	// thread; cancelling ctx stops it. On error the Reply still lists the changes made before the
	// turn stopped. The error's text is fit to show.
	Ask(ctx context.Context, text string) (Reply, error)
	// Undo reverts the one change a receipt describes.
	Undo(receiptID int) error
	// Reset forgets the conversation.
	Reset()
}

// Reply is what one turn produced.
type Reply struct {
	Text     string
	Receipts []Receipt
	Question *Question
}

// Receipt is one change the assistant made.
type Receipt struct {
	ID     int // for Undo
	TaskID int
	Title  string
	Detail string // "Tomorrow 9:00 AM · reminder"
	Status Status
	Bell   bool // the change set a reminder
}

// Question is a short question with 2 to 4 answers to pick from; a typed answer works too.
type Question struct {
	Text    string
	Options []string
}

// Provider is one row of Settings > Models.
type Provider struct {
	ID, Name, Note string
	BaseURL        string   // what a request would use now
	DefaultBaseURL string   // what the preset starts with, shown as the placeholder
	Model          string   // what a request would use now
	Suggestions    []string // models to offer under the field
	HasKeyField    bool     // false for presets with a fixed key (Magpie, Ollama)
	PickProtocol   bool     // Custom: OpenAI-compatible or Anthropic
	Anthropic      bool     // the protocol in use
	Active         bool
}

// ProviderEdit is what is typed into a provider's fields. Key is empty when unchanged.
type ProviderEdit struct {
	BaseURL, Model, Key string
	Anthropic           bool
}

// ModelSettings is what Settings > Models reads and writes.
type ModelSettings interface {
	Providers() []Provider
	// KeySaved reports whether a key is stored for the provider, without revealing it.
	KeySaved(id string) bool
	// Save stores what was typed. A non-empty Key goes to the keychain.
	Save(id string, e ProviderEdit) error
	// SetActive makes a provider the one the assistant uses.
	SetActive(id string) error
	// Test sends a one-token request with what is typed, saved or not.
	Test(ctx context.Context, id string, e ProviderEdit) (time.Duration, error)
}
