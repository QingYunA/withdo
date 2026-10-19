package data

import "github.com/QingYunA/adhd-todo/internal/domain"

var _ Summaries = (*Real)(nil)

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
