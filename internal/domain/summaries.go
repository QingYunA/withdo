package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// The periods a summary is written for.
const (
	PeriodDay  = "day"
	PeriodWeek = "week"
)

// GoodKinds are the kinds of goods a finished task becomes in the Summary scene: a feature, a bug
// fix, a refactor or migration, docs or design, tests, a reply, money or a bill, a plan or research,
// and anything else.
var GoodKinds = []string{"gem", "jar", "gear", "book", "flask", "mail", "coin", "scroll", "crate"}

// GoodCrate is the kind of anything else.
const GoodCrate = "crate"

// IsGoodKind reports whether kind is one of GoodKinds.
func IsGoodKind(kind string) bool {
	for _, k := range GoodKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// Summary is what the assistant wrote about a day or a week: the paragraph, the model that wrote it,
// and the kind of good each finished task became. A period has one; writing it again replaces it.
type Summary struct {
	Period    string           `json:"period"`     // PeriodDay or PeriodWeek
	FirstDate string           `json:"first_date"` // the local date the period starts on, YYYY-MM-DD
	Paragraph string           `json:"paragraph"`
	Model     string           `json:"model"`
	Kinds     map[int64]string `json:"kinds"` // by task id
	CreatedAt string           `json:"created_at"`
}

// GetSummary reads the summary of a period, or nil when none was written.
func (c *Core) GetSummary(ctx context.Context, period, firstDate string) (*Summary, error) {
	var s Summary
	var kinds string
	err := c.db.QueryRowContext(ctx, `SELECT period, first_date, paragraph, model, kinds, created_at FROM summaries
		WHERE period = ? AND first_date = ?`, period, firstDate).Scan(&s.Period, &s.FirstDate, &s.Paragraph, &s.Model, &kinds, &s.CreatedAt)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}
	s.Kinds = parseKinds(kinds)
	return &s, nil
}

// PutSummary stores the summary of a period, replacing the one it has, and returns it as stored: its
// CreatedAt is the time of the call, and a kind that is not one of GoodKinds is left out.
func (c *Core) PutSummary(ctx context.Context, s Summary) (*Summary, error) {
	if s.Period != PeriodDay && s.Period != PeriodWeek {
		return nil, invalid("period", "must be %q or %q", PeriodDay, PeriodWeek)
	}
	if s.FirstDate == "" {
		return nil, invalid("first_date", "must not be empty")
	}
	kinds := map[string]string{}
	stored := map[int64]string{}
	for id, kind := range s.Kinds {
		if IsGoodKind(kind) {
			kinds[strconv.FormatInt(id, 10)], stored[id] = kind, kind
		}
	}
	raw, err := json.Marshal(kinds)
	if err != nil {
		return nil, fmt.Errorf("encode kinds: %w", err)
	}
	s.CreatedAt, s.Kinds = c.stamp(), stored
	err = c.write(ctx, func(q store.Querier) error {
		_, err := q.ExecContext(ctx, `INSERT INTO summaries (period, first_date, paragraph, model, kinds, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (period, first_date) DO UPDATE SET paragraph = excluded.paragraph, model = excluded.model,
				kinds = excluded.kinds, created_at = excluded.created_at`,
			s.Period, s.FirstDate, s.Paragraph, s.Model, string(raw), s.CreatedAt)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("put summary: %w", err)
	}
	return &s, nil
}

// parseKinds reads the stored JSON; what cannot be read is no kinds at all, since a task without a
// stored kind falls back to its words.
func parseKinds(raw string) map[int64]string {
	var by map[string]string
	if json.Unmarshal([]byte(raw), &by) != nil {
		return map[int64]string{}
	}
	out := make(map[int64]string, len(by))
	for key, kind := range by {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil && IsGoodKind(kind) {
			out[id] = kind
		}
	}
	return out
}
