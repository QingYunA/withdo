package data

// SummaryCounts are how many tasks were finished in a period and how many of them went in as a
// merged pull request: what the end-of-day reminder says.
type SummaryCounts struct {
	Finished, Merged int
}

// CountSummary counts what was finished in a period, from the store, so that it can be asked from any
// goroutine. Who finished each task does not matter here, only whether it was done and merged.
func CountSummary(store Store, p Period, agent func(string) string) (SummaryCounts, error) {
	tasks, err := store.ListTasks(Filter{Status: []Status{Done}})
	if err != nil {
		return SummaryCounts{}, err
	}
	// The reports of a task are read only for the tasks of the period: BuildSummary asks for them
	// then.
	reports := func(id int, github bool) *Report {
		detail, err := store.GetTask(id)
		if err != nil {
			return nil
		}
		for i := range detail.Reports { // newest first
			if (detail.Reports[i].Agent == GitHubAgent) == github {
				return &detail.Reports[i]
			}
		}
		return nil
	}
	s := BuildSummary(SummaryInput{
		Period: p, Tasks: tasks, Agent: agent,
		Latest: func(id int) *Report { return reports(id, false) },
		Event:  func(id int) *Report { return reports(id, true) },
	})
	return SummaryCounts{Finished: s.Total(), Merged: s.Merged()}, nil
}
