package store_test

import "testing"

// Migration 010 adds summaries: a day or a week, once, and a rewrite takes the same row.
func TestMigration010AddsSummariesKeyedByPeriodAndFirstDate(t *testing.T) {
	db := openCopyOf(t, "testdata/ts-v2.db")
	insert := `INSERT INTO summaries (period, first_date, paragraph, model, kinds, created_at) VALUES (?, ?, 'p', 'm', '{}', 'x')`
	if _, err := db.ExecContext(ctx, insert, "day", "2026-10-17"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, insert, "week", "2026-10-17"); err != nil {
		t.Errorf("a week and a day may start on the same date: %v", err)
	}
	if _, err := db.ExecContext(ctx, insert, "day", "2026-10-17"); err == nil {
		t.Error("a period has one summary")
	}
	if _, err := db.ExecContext(ctx, insert, "month", "2026-10-01"); err == nil {
		t.Error("a period is a day or a week")
	}
}
