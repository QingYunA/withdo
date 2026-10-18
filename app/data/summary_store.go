package data

import (
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

var (
	_ Summaries = (*Real)(nil)
	_ Summaries = (*Fake)(nil)
)

// GetSummary implements Summaries.
func (r *Real) GetSummary(p Period) (*StoredSummary, error) {
	s, err := r.core.GetSummary(r.ctx(), p.Kind(), p.FirstDate())
	if err != nil || s == nil {
		return nil, err
	}
	out := fromSummary(*s)
	return &out, nil
}

// PutSummary implements Summaries.
func (r *Real) PutSummary(p Period, s StoredSummary) (StoredSummary, error) {
	kinds := make(map[int64]string, len(s.Kinds))
	for id, kind := range s.Kinds {
		kinds[int64(id)] = kind
	}
	put, err := r.core.PutSummary(r.ctx(), domain.Summary{
		Period: p.Kind(), FirstDate: p.FirstDate(), Paragraph: s.Paragraph, Model: s.Model, Kinds: kinds,
	})
	if err != nil {
		return StoredSummary{}, err
	}
	return fromSummary(*put), nil
}

func fromSummary(s domain.Summary) StoredSummary {
	kinds := make(map[int]string, len(s.Kinds))
	for id, kind := range s.Kinds {
		kinds[int(id)] = kind
	}
	return StoredSummary{Paragraph: s.Paragraph, Model: s.Model, Kinds: kinds, CreatedAt: parseTime(s.CreatedAt)}
}

// fakeSummaries is what the Fake keeps of summaries.
type fakeSummaries map[string]StoredSummary

func summaryKey(p Period) string { return p.Kind() + " " + p.FirstDate() }

// GetSummary implements Summaries.
func (f *Fake) GetSummary(p Period) (*StoredSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.summaries[summaryKey(p)]
	if !ok {
		return nil, nil
	}
	return &s, nil
}

// PutSummary implements Summaries.
func (f *Fake) PutSummary(p Period, s StoredSummary) (StoredSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.summaries == nil {
		f.summaries = fakeSummaries{}
	}
	kinds := make(map[int]string, len(s.Kinds))
	for id, kind := range s.Kinds {
		if domain.IsGoodKind(kind) {
			kinds[id] = kind
		}
	}
	s.Kinds, s.CreatedAt = kinds, f.Now()
	f.summaries[summaryKey(p)] = s
	return s, nil
}

// CompleteAt marks a task done at a moment, for demos and tests that want a day with things finished
// at different hours.
func (f *Fake) CompleteAt(id int, at time.Time) {
	f.mu.Lock()
	if t, ok := f.find(id); ok {
		t.Status, t.CompletedAt, t.UpdatedAt = Done, &at, at
	}
	f.mu.Unlock()
	f.notify()
}
