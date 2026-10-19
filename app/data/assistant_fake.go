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
	// SummaryScript answers a request for the paragraph of a period. Nil answers with one line.
	SummaryScript func(SummaryRequest) (SummaryText, error)
	// SummaryBlock, when non-nil, holds every Summarize until it is closed or the request is cancelled.
	SummaryBlock chan struct{}
	// RecentScript answers a request for the sentences of a repository's items; nil answers with the
	// titles (FallbackNote). RecentLinesScript and RecentExplainScript answer the other two; nil
	// answers with a fixed text. RecentBlock, when non-nil, holds all three until it is closed or the
	// request is cancelled.
	RecentScript        func(RecentDescribeRequest) (map[int]RecentNote, error)
	RecentLinesScript   func(RecentLinesRequest) (string, error)
	RecentExplainScript func(RecentExplainRequest) (RecentExplanation, error)
	RecentBlock         chan struct{}

	mu        sync.Mutex
	asked     []string
	places    []Place
	pics      [][]Attachment
	undone    []int
	resets    int
	summaries []SummaryRequest
	described []RecentDescribeRequest
	lined     []RecentLinesRequest
	explained []RecentExplainRequest
}

var _ Assistant = (*FakeAssistant)(nil)

// Configured implements Assistant.
func (f *FakeAssistant) Configured() bool { return f.Ready }

// Ask implements Assistant.
func (f *FakeAssistant) Ask(ctx context.Context, text string, place Place, pictures ...Attachment) (Reply, error) {
	f.mu.Lock()
	f.asked, f.places = append(f.asked, text), append(f.places, place)
	f.pics = append(f.pics, append([]Attachment(nil), pictures...))
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

// Summarize implements Assistant: SummaryScript answers it, or a fixed one-line paragraph when nil.
func (f *FakeAssistant) Summarize(ctx context.Context, req SummaryRequest) (SummaryText, error) {
	f.mu.Lock()
	f.summaries = append(f.summaries, req)
	script, block := f.SummaryScript, f.SummaryBlock
	f.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return SummaryText{}, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return SummaryText{}, err
	}
	if script == nil {
		return SummaryText{Paragraph: "Done.", Model: "fake-model"}, nil
	}
	return script(req)
}

// Summaries is every request for a paragraph the UI made, in order.
func (f *FakeAssistant) Summaries() []SummaryRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]SummaryRequest(nil), f.summaries...)
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

// Pictures is the pictures sent with every text the UI sent, in order: an empty list for a text
// sent alone.
func (f *FakeAssistant) Pictures() [][]Attachment {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]Attachment(nil), f.pics...)
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
