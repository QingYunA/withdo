package data

import (
	"context"
	"sync"
	"time"
)

// FakeAssistant is a scripted Assistant for tests and the design review. It answers from Script
// without any model; with Block set it waits there, so the "thinking" state can be drawn.
type FakeAssistant struct {
	// Ready is what Configured reports.
	Ready bool
	// Script answers a turn. Nil answers with an empty reply.
	Script func(text string) (Reply, error)
	// Block, when non-nil, holds every Ask until it is closed or the request is cancelled.
	Block chan struct{}

	mu     sync.Mutex
	asked  []string
	places []Place
	undone []int
	resets int
}

var _ Assistant = (*FakeAssistant)(nil)

// Configured implements Assistant.
func (f *FakeAssistant) Configured() bool { return f.Ready }

// Ask implements Assistant.
func (f *FakeAssistant) Ask(ctx context.Context, text string, place Place) (Reply, error) {
	f.mu.Lock()
	f.asked, f.places = append(f.asked, text), append(f.places, place)
	block, script := f.Block, f.Script
	f.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return Reply{}, ctx.Err()
		}
	}
	if script == nil {
		return Reply{}, nil
	}
	return script(text)
}

// Undo implements Assistant.
func (f *FakeAssistant) Undo(id int) error {
	f.mu.Lock()
	f.undone = append(f.undone, id)
	f.mu.Unlock()
	return nil
}

// Reset implements Assistant.
func (f *FakeAssistant) Reset() {
	f.mu.Lock()
	f.resets++
	f.mu.Unlock()
}

// Asked is every text the UI sent, in order.
func (f *FakeAssistant) Asked() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.asked...)
}

// Places is the place of every text the UI sent, in order.
func (f *FakeAssistant) Places() []Place {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Place(nil), f.places...)
}

// Undone is every receipt id the UI asked to undo.
func (f *FakeAssistant) Undone() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.undone...)
}

// Resets is how many times the conversation was reset.
func (f *FakeAssistant) Resets() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resets
}

// FakeModels is ProviderSettings whose Test answers from TestFn instead of the network.
type FakeModels struct {
	ProviderSettings
	// TestFn answers the Test button; nil works in 640 ms.
	TestFn func(id string, e ProviderEdit) (time.Duration, error)
	// ListFn answers the model list; nil has no list, so the suggestions stand in.
	ListFn func(id string, e ProviderEdit) ([]string, error)
}

var _ ModelSettings = (*FakeModels)(nil)

// Test implements ModelSettings.
func (f *FakeModels) Test(_ context.Context, id string, e ProviderEdit) (time.Duration, error) {
	if f.TestFn == nil {
		return 640 * time.Millisecond, nil
	}
	return f.TestFn(id, e)
}

// ListModels implements ModelSettings.
func (f *FakeModels) ListModels(_ context.Context, id string, e ProviderEdit) ([]string, error) {
	if f.ListFn == nil {
		return nil, nil
	}
	return f.ListFn(id, e)
}
