package data

// SummaryCounts are how many tasks were finished in a period and how many of them went in as a
// merged pull request: what the end-of-day reminder says.
type SummaryCounts struct {
	Finished, Merged int
}

// Counts are the numbers of a summary.
func (s Summary) Counts() SummaryCounts {
	return SummaryCounts{Finished: s.Total(), Merged: s.Merged()}
}

// CountSummary counts what was finished in a period in a snapshot. Who finished each task does not
// matter here, only whether it was done and merged. The Summary page counts its own with Counts, from
// the same BuildSummary.
func CountSummary(s Snapshot, p Period, agent func(string) string) SummaryCounts {
	return BuildSummary(SummaryInput{Snapshot: s, Period: p, Agent: agent}).Counts()
}
