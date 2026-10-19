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
	// Category is the id of the category page of Mine (PlaceNow), 0 for Mine as a whole: a new
	// task is in that category.
	Category int
}

// Assistant is the built-in agent behind the composer, as the UI sees it.
type Assistant interface {
	// Configured reports whether a model is ready. Until it is, the composer adds plain tasks.
	Configured() bool
	// Ask runs one turn of the current conversation, typed in place. It blocks, so the UI calls
	// it off its thread; cancelling ctx stops it. On error the Reply still lists the changes made
	// before the turn stopped. The error's text is fit to show.
	// Pictures ride along with the text; most calls send none.
	Ask(ctx context.Context, text string, place Place, pictures ...Attachment) (Reply, error)
	// Undo reverts the one change a receipt describes.
	Undo(receiptID int) error
	// Reset forgets the conversation.
	Reset()
	// Summarize writes the paragraph of a period and picks the goods of its tasks, in one call to
	// the model of the composer; the conversation is not touched. It blocks, so the UI calls it off
	// its thread. The error's text is fit to show.
	Summarize(ctx context.Context, req SummaryRequest) (SummaryText, error)
	// DescribeRecent writes a sentence, and for a merged pull request a category, for each item of
	// a repository on the Recent changes page, keyed by item number. Like the two below it makes one
	// request to the model of the composer, blocks, and leaves the conversation alone.
	DescribeRecent(ctx context.Context, req RecentDescribeRequest) (map[int]RecentNote, error)
	// WriteRecentLines writes the two or three sentences at the head of a repository's section.
	WriteRecentLines(ctx context.Context, req RecentLinesRequest) (string, error)
	// ExplainRecent says what an item did, why and where, from its description and file names.
	ExplainRecent(ctx context.Context, req RecentExplainRequest) (RecentExplanation, error)
}

// Attachment is a picture sent with a request, already prepared by PrepareImage. It lives in
// memory until the turn is over; the tasks the turn creates or changes keep it (see Images).
type Attachment struct {
	Name string // the file's name, for the thumbnail's label
	MIME string // image/jpeg or image/png: the prepared copy the model gets
	Data []byte
	// Original is the file as it was read, when that is up to 20 MB and not the prepared copy
	// already; nil otherwise. OriginalMIME is its type: image/jpeg, png, gif or webp.
	Original     []byte
	OriginalMIME string
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
	Images int  // how many of the turn's pictures the task kept
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
	ChangeCategory = agent.ChangeCategory
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
	ProblemBusy     // a second request while one runs
	ProblemNoImages // pictures were sent to a model that cannot read them
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
