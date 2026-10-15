package data

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/providers"
)

// AgentAssistant is the Assistant backed by the Eino agent and the configured provider.
type AgentAssistant struct {
	Agent    *agent.Agent
	Provider *providers.Manager
}

var _ Assistant = (*AgentAssistant)(nil)

// Configured implements Assistant.
func (a *AgentAssistant) Configured() bool { return a.Provider.Configured() }

// Ask implements Assistant. The error is already in plain words.
func (a *AgentAssistant) Ask(ctx context.Context, text string, place Place) (Reply, error) {
	res, err := a.Agent.Ask(ctx, text, agent.Scope{Kind: agent.ScopeKind(place.Kind), ProjectID: int64(place.ProjectID), TaskID: int64(place.TaskID), Day: place.Day})
	out := Reply{Text: res.Reply}
	for _, r := range res.Receipts {
		out.Receipts = append(out.Receipts, Receipt{
			ID: r.ID, TaskID: int(r.TaskID), Title: r.Title, Detail: r.Detail, Status: Status(r.Status), Bell: r.Bell,
		})
	}
	if q := res.Question; q != nil {
		out.Question = &Question{Text: q.Text, Options: q.Options}
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return out, context.Canceled
		}
		return out, problemFor(err)
	}
	return out, nil
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
}

// Undo implements Assistant.
func (a *AgentAssistant) Undo(id int) error { return a.Agent.Undo(context.Background(), id) }

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
		out = append(out, Provider{
			ID: pr.ID, Name: pr.Name, Note: pr.Note, BaseURL: r.BaseURL, DefaultBaseURL: pr.Display(),
			Model: r.Model, Suggestions: pr.Models, HasKeyField: pr.HasKeyField(), PickProtocol: pr.PickProtocol,
			Anthropic: r.Protocol == providers.Anthropic, Active: cfg.Active == pr.ID,
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
