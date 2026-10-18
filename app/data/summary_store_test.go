package data

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// summaryStores are the two Summaries: the real one over a database and the fake.
func summaryStores(t *testing.T) map[string]Summaries {
	t.Helper()
	now := func() time.Time { return friday }
	real, err := OpenReal(bg, filepath.Join(t.TempDir(), "t.db"), RealOptions{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = real.Close() })
	return map[string]Summaries{"real": real, "fake": NewEmptyFake(now)}
}

func TestSummariesAreKeptPerPeriodAndReadBack(t *testing.T) {
	for name, s := range summaryStores(t) {
		day, week := DayPeriod(friday, 0), WeekPeriod(friday, 0, time.Monday)
		if got, err := s.GetSummary(day); err != nil || got != nil {
			t.Fatalf("%s: before writing: %+v, %v", name, got, err)
		}
		put, err := s.PutSummary(day, StoredSummary{Paragraph: "Seven done.", Model: "m-1", Kinds: map[int]string{12: "gem", 14: "dragon"}})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !put.CreatedAt.Equal(friday) || !reflect.DeepEqual(put.Kinds, map[int]string{12: "gem"}) {
			t.Errorf("%s: put %+v: stamped with the clock, with only the kinds that are goods", name, put)
		}
		got, err := s.GetSummary(day)
		if err != nil || got == nil || got.Paragraph != "Seven done." || got.Model != "m-1" || !reflect.DeepEqual(got.Kinds, map[int]string{12: "gem"}) {
			t.Errorf("%s: day %+v, %v", name, got, err)
		}
		if other, _ := s.GetSummary(week); other != nil {
			t.Errorf("%s: the week of the same date has the day's summary", name)
		}
		if past, _ := s.GetSummary(DayPeriod(friday, 1)); past != nil {
			t.Errorf("%s: yesterday has today's summary", name)
		}
	}
}

func TestWritingAPeriodAgainReplacesItsSummary(t *testing.T) {
	for name, s := range summaryStores(t) {
		p := DayPeriod(friday, 0)
		_, _ = s.PutSummary(p, StoredSummary{Paragraph: "First."})
		_, _ = s.PutSummary(p, StoredSummary{Paragraph: "Second.", Kinds: map[int]string{1: "jar"}})
		got, _ := s.GetSummary(p)
		if got.Paragraph != "Second." || len(got.Kinds) != 1 {
			t.Errorf("%s: %+v", name, got)
		}
	}
}
