package domain_test

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func sum(period, first, text string, kinds map[int64]string) domain.Summary {
	return domain.Summary{Period: period, FirstDate: first, Paragraph: text, Model: "m-1", Kinds: kinds}
}

func TestAPeriodWithoutASummaryHasNone(t *testing.T) {
	c, _ := newCore(t)
	got, err := c.GetSummary(ctx, domain.PeriodDay, "2026-10-17")
	if err != nil || got != nil {
		t.Fatalf("got %+v, %v; want nil", got, err)
	}
}

func TestPutSummaryKeepsTheParagraphTheModelAndTheKinds(t *testing.T) {
	c, _ := newCore(t)
	put, err := c.PutSummary(ctx, sum(domain.PeriodDay, "2026-10-17", "Seven done.", map[int64]string{12: "gem", 14: "jar"}))
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "created", put.CreatedAt, domain.FormatTime(t0))
	got, err := c.GetSummary(ctx, domain.PeriodDay, "2026-10-17")
	if err != nil {
		t.Fatal(err)
	}
	same(t, "summary", *got, *put)
	same(t, "kinds", got.Kinds, map[int64]string{12: "gem", 14: "jar"})
}

func TestWritingAPeriodAgainReplacesItsRow(t *testing.T) {
	c, clk := newCore(t)
	must[*domain.Summary](t)(c.PutSummary(ctx, sum(domain.PeriodWeek, "2026-10-13", "First.", map[int64]string{1: "gem"})))
	clk.advance(time.Hour)
	again := must[*domain.Summary](t)(c.PutSummary(ctx, sum(domain.PeriodWeek, "2026-10-13", "Second.", map[int64]string{2: "book"})))
	got := must[*domain.Summary](t)(c.GetSummary(ctx, domain.PeriodWeek, "2026-10-13"))
	eq(t, "paragraph", got.Paragraph, "Second.")
	eq(t, "created", got.CreatedAt, again.CreatedAt)
	same(t, "kinds", got.Kinds, map[int64]string{2: "book"})
}

func TestADayAndAWeekOfTheSameDateAreTwoSummaries(t *testing.T) {
	c, _ := newCore(t)
	must[*domain.Summary](t)(c.PutSummary(ctx, sum(domain.PeriodDay, "2026-10-13", "Day.", nil)))
	must[*domain.Summary](t)(c.PutSummary(ctx, sum(domain.PeriodWeek, "2026-10-13", "Week.", nil)))
	eq(t, "day", must[*domain.Summary](t)(c.GetSummary(ctx, domain.PeriodDay, "2026-10-13")).Paragraph, "Day.")
	eq(t, "week", must[*domain.Summary](t)(c.GetSummary(ctx, domain.PeriodWeek, "2026-10-13")).Paragraph, "Week.")
}

func TestPutSummaryDropsKindsItDoesNotKnow(t *testing.T) {
	c, _ := newCore(t)
	put := must[*domain.Summary](t)(c.PutSummary(ctx, sum(domain.PeriodDay, "2026-10-17", "x", map[int64]string{1: "gem", 2: "dragon"})))
	same(t, "returned", put.Kinds, map[int64]string{1: "gem"})
	same(t, "stored", must[*domain.Summary](t)(c.GetSummary(ctx, domain.PeriodDay, "2026-10-17")).Kinds, map[int64]string{1: "gem"})
}

func TestPutSummaryRefusesABadPeriodOrNoDate(t *testing.T) {
	c, _ := newCore(t)
	if _, err := c.PutSummary(ctx, sum("month", "2026-10-01", "x", nil)); err == nil {
		t.Error("a month is not a period")
	}
	if _, err := c.PutSummary(ctx, sum(domain.PeriodDay, "", "x", nil)); err == nil {
		t.Error("a summary needs its date")
	}
}

func TestNoKindsAreStoredAsAnEmptyObject(t *testing.T) {
	c, _ := newCore(t)
	must[*domain.Summary](t)(c.PutSummary(ctx, sum(domain.PeriodDay, "2026-10-17", "x", nil)))
	got := must[*domain.Summary](t)(c.GetSummary(ctx, domain.PeriodDay, "2026-10-17"))
	if got.Kinds == nil || len(got.Kinds) != 0 {
		t.Errorf("kinds = %#v, want an empty map", got.Kinds)
	}
}
