package data

import (
	"context"
	"errors"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
)

// PlaceKind says where a request to the assistant was typed.
type PlaceKind int

const (
	PlaceNow      PlaceKind = iota // the Now page: no place of its own
	PlaceProject                   // a project's page
	PlaceInbox                     // the Inbox page
	PlaceTask                      // the inspector of one task
	PlaceCalendar                  // the Calendar page, with a day selected
)

// Place is where a request was typed. The assistant puts new tasks there and takes "this" and
// "it" to mean the task of a task place.
type Place struct {
	Kind      PlaceKind
	ProjectID int // PlaceProject
	TaskID    int // PlaceTask
	// Day is the selected day of the Calendar page as "2006-01-02" (PlaceCalendar).
	Day string
}

// Assistant is the built-in agent behind the composer, as the UI sees it.
type Assistant interface {
	// Configured reports whether a model is ready. Until it is, the composer adds plain tasks.
	Configured() bool
	// Ask runs one turn of the current conversation, typed in place. It blocks, so the UI calls
	// it off its thread; cancelling ctx stops it. On error the Reply still lists the changes made
	// before the turn stopped. The error's text is fit to show.
	Ask(ctx context.Context, text string, place Place) (Reply, error)
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
	Detail Detail // what changed, as data: the views word it
	Status Status
	Bell   bool // the change set a reminder
}

// Detail is what a receipt says about a change: a reminder at a time, a due time, the fields an
// update changed. The agent reports it as data and the views word it in the user's language.
type Detail = agent.Detail

// The kinds of Detail and of the changes an update lists, as the agent names them.
type (
	DetailKind = agent.DetailKind
	Change     = agent.Change
	ChangeKind = agent.ChangeKind
)

const (
	DetailAdded    = agent.DetailAdded
	DetailReminder = agent.DetailReminder
	DetailDue      = agent.DetailDue
	DetailUpdated  = agent.DetailUpdated
	DetailDone     = agent.DetailDone

	ChangeReminder = agent.ChangeReminder
	ChangeDue      = agent.ChangeDue
	ChangeStatus   = agent.ChangeStatus
	ChangePriority = agent.ChangePriority
	ChangeProject  = agent.ChangeProject
	ChangeRenamed  = agent.ChangeRenamed
	ChangeNotes    = agent.ChangeNotes
)

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
	KeyRequired    bool   // false for presets that work without a key (Ollama, Magpie, Custom)
	Configured     bool   // in use, or with a key (or, for a preset that needs none, saved once)
	KeyURL         string // the page that hands out API keys; empty when there is no key field
	Monogram       string // the letters of the logo tile; the tile's color is the theme's
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
	// ListModels asks the provider which models it offers, with the stored key unless e has one.
	// A failure is not shown to the user: the preset's suggestions stand in.
	ListModels(ctx context.Context, id string, e ProviderEdit) ([]string, error)
}

// Problem says why a request to the model failed, so the views can word it in the user's
// language.
type Problem int

const (
	ProblemOther         Problem = iota // anything else: the text is the provider's own
	ProblemNotConfigured                // no model is connected
	ProblemNoKey                        // the provider needs an API key
	ProblemCancelled
	ProblemTimeout
	ProblemUnreachable // connection refused or an unknown host
	ProblemKeyRejected
	ProblemUnknownModel
	ProblemRateLimited
	ProblemBusy // a second request while one runs
)

// ProblemError is a failure the views word themselves. Error is the English text; Err is the
// failure it describes.
type ProblemError struct {
	Problem Problem
	Text    string
	Err     error
}

func (e *ProblemError) Error() string { return e.Text }
func (e *ProblemError) Unwrap() error { return e.Err }

// ProblemOf says what kind of failure err is; ProblemOther for any error that is not a
// ProblemError.
func ProblemOf(err error) Problem {
	var p *ProblemError
	if errors.As(err, &p) {
		return p.Problem
	}
	return ProblemOther
}
