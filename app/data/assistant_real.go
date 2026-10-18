package data

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/providers"
)

// AgentAssistant is the Assistant backed by the Eino agent and the configured provider.
type AgentAssistant struct {
	Agent    *agent.Agent
	Provider *providers.Manager
	// Images keeps the pictures of a turn with the tasks it created or changed; nil keeps none.
	Images Images
}

var _ Assistant = (*AgentAssistant)(nil)

// Configured implements Assistant.
func (a *AgentAssistant) Configured() bool { return a.Provider.Configured() }

// Ask implements Assistant. The error is already in plain words.
func (a *AgentAssistant) Ask(ctx context.Context, text string, place Place, pictures ...Attachment) (Reply, error) {
	images := make([]agent.Image, len(pictures))
	for i, p := range pictures {
		images[i] = agent.Image{MIME: p.MIME, Data: p.Data}
	}
	res, err := a.Agent.Ask(ctx, text, agent.Scope{Kind: agent.ScopeKind(place.Kind), ProjectID: int64(place.ProjectID), TaskID: int64(place.TaskID), Day: place.Day, Category: domain.Category(place.Category)}, images...)
	out := Reply{Text: res.Reply}
	for _, r := range res.Receipts {
		out.Receipts = append(out.Receipts, Receipt{
			ID: r.ID, TaskID: int(r.TaskID), Title: r.Title, Detail: r.Detail, Status: Status(r.Status), Bell: r.Bell,
		})
	}
	if q := res.Question; q != nil {
		out.Question = &Question{Text: q.Text, Options: q.Options}
	}
	if kerr := a.keepPictures(res.Images, pictures, out.Receipts); kerr != nil && err == nil {
		return out, kerr
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return out, context.Canceled
		}
		return out, problemFor(err)
	}
	return out, nil
}

// errPicturesNotKept is the answer when the tasks were changed but their pictures could not be saved.
var errPicturesNotKept = errors.New("The pictures could not be saved with the tasks.")

// keepPictures attaches the pictures of the turn to the tasks that keep them, and marks the
// receipts of those tasks with how many they got. The first failure stops it.
func (a *AgentAssistant) keepPictures(keeps []agent.ImageKeep, pictures []Attachment, receipts []Receipt) error {
	if a.Images == nil {
		return nil
	}
	for _, k := range keeps {
		pics := make([]Attachment, 0, len(k.Indexes))
		for _, i := range k.Indexes {
			if i >= 0 && i < len(pictures) {
				pics = append(pics, pictures[i])
			}
		}
		links, err := a.Images.AttachImages(int(k.TaskID), pics)
		if err != nil {
			return errPicturesNotKept
		}
		for i := range receipts {
			if receipts[i].TaskID == int(k.TaskID) {
				receipts[i].Images = len(links)
			}
		}
	}
	return nil
}

// problemFor turns a failure of the model into one the views can word, by the kind
// providers.Classify gives it; the English text is providers.Friendly's.
func problemFor(err error) error {
	if errors.Is(err, agent.ErrBusy) {
		return &ProblemError{Problem: ProblemBusy, Text: err.Error(), Err: err}
	}
	text := providers.Friendly(err)
	if p, ok := problemKinds[providers.Classify(err)]; ok {
		return &ProblemError{Problem: p, Text: text, Err: err}
	}
	return errors.New(text)
}

var problemKinds = map[providers.Kind]Problem{
	providers.KindNotConfigured: ProblemNotConfigured,
	providers.KindNoKey:         ProblemNoKey,
	providers.KindCancelled:     ProblemCancelled,
	providers.KindTimeout:       ProblemTimeout,
	providers.KindUnreachable:   ProblemUnreachable,
	providers.KindKeyRejected:   ProblemKeyRejected,
	providers.KindUnknownModel:  ProblemUnknownModel,
	providers.KindRateLimited:   ProblemRateLimited,
	providers.KindNoImages:      ProblemNoImages,
}

// Undo implements Assistant.
func (a *AgentAssistant) Undo(id int) error {
	err := a.Agent.Undo(context.Background(), id)
	if err == nil && a.Images != nil {
		a.Images.CollectSoon() // the task the receipt made may be gone with its pictures
	}
	return err
}

// Reset implements Assistant.
func (a *AgentAssistant) Reset() { a.Agent.Reset() }

// ProviderSettings is the ModelSettings backed by the provider file and the keychain.
type ProviderSettings struct {
	Provider *providers.Manager

	mu    sync.Mutex
	saved map[string]bool // key presence, read from the keychain once per provider
}

var _ ModelSettings = (*ProviderSettings)(nil)

// Providers implements ModelSettings.
func (p *ProviderSettings) Providers() []Provider {
	cfg := p.Provider.Config()
	out := make([]Provider, 0, len(providers.Presets))
	for _, pr := range providers.Presets {
		r, _ := cfg.Resolve(pr.ID)
		_, saved := cfg.Providers[pr.ID]
		configured := cfg.Active == pr.ID || (pr.HasKeyField() && p.KeySaved(pr.ID)) || (!pr.NeedsKey() && saved)
		out = append(out, Provider{
			ID: pr.ID, Name: pr.Name, Note: pr.Note, BaseURL: r.BaseURL, DefaultBaseURL: pr.Display(),
			Model: r.Model, Suggestions: pr.Models, HasKeyField: pr.HasKeyField(), PickProtocol: pr.PickProtocol,
			Anthropic: r.Protocol == providers.Anthropic, Active: cfg.Active == pr.ID, KeyURL: pr.KeyURL, Monogram: pr.Monogram,
			KeyRequired: pr.NeedsKey(), Configured: configured,
		})
	}
	return out
}

// KeySaved implements ModelSettings.
func (p *ProviderSettings) KeySaved(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if v, ok := p.saved[id]; ok {
		return v
	}
	if p.saved == nil {
		p.saved = map[string]bool{}
	}
	p.saved[id] = p.Provider.HasKey(id)
	return p.saved[id]
}

func entryOf(e ProviderEdit) providers.Entry {
	out := providers.Entry{BaseURL: e.BaseURL, Model: e.Model}
	if e.Anthropic {
		out.Protocol = providers.Anthropic
	}
	return out
}

// Save implements ModelSettings.
func (p *ProviderSettings) Save(id string, e ProviderEdit) error {
	if e.Key != "" {
		if err := p.Provider.SetKey(id, e.Key); err != nil {
			return err
		}
		p.mu.Lock()
		delete(p.saved, id)
		p.mu.Unlock()
	}
	return p.Provider.Save(p.Provider.Config().With(id, entryOf(e)))
}

// SetActive implements ModelSettings.
func (p *ProviderSettings) SetActive(id string) error {
	return p.Provider.Save(p.Provider.Config().WithActive(id))
}

// Test implements ModelSettings.
func (p *ProviderSettings) Test(ctx context.Context, id string, e ProviderEdit) (time.Duration, error) {
	d, err := p.Provider.Test(ctx, id, entryOf(e), e.Key)
	if err != nil {
		return 0, problemFor(err)
	}
	return d, nil
}

// ListModels implements ModelSettings.
func (p *ProviderSettings) ListModels(ctx context.Context, id string, e ProviderEdit) ([]string, error) {
	return p.Provider.Models(ctx, id, entryOf(e), e.Key)
}
