package ghpoll

import (
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// Plan is what a check changes: the snapshot to store, whether the task is finished, and the
// reports GitHub files against it.
type Plan struct {
	Snapshot domain.PRSnapshot
	Done     bool
	Reports  []string
}

// Decide compares what was stored (the zero snapshot before the first check) with what was seen.
// Each effect fires when its condition becomes true, not while it stays true:
//
//   - merged: the task is done, with the report "PR #12 merged";
//   - closed without merging: the report "PR #12 was closed without merging" (the Needs you row
//     itself comes from the stored state);
//   - checks failing, or changes requested, on an open pull request (not a draft): a report each.
//
// The summaries are English text stored in the database, like any agent's report.
func Decide(old, seen domain.PRSnapshot, number int64) Plan {
	plan := Plan{Snapshot: seen}
	if seen.State == domain.PRMerged && old.State != domain.PRMerged {
		plan.Done = true
		plan.Reports = append(plan.Reports, fmt.Sprintf("PR #%d merged", number))
	}
	if seen.State == domain.PRClosed && old.State != domain.PRClosed {
		plan.Reports = append(plan.Reports, fmt.Sprintf("PR #%d was closed without merging", number))
	}
	if checksFailing(seen) && !checksFailing(old) {
		plan.Reports = append(plan.Reports, fmt.Sprintf("Checks are failing on PR #%d", number))
	}
	if changesRequested(seen) && !changesRequested(old) {
		plan.Reports = append(plan.Reports, fmt.Sprintf("Changes were requested on PR #%d", number))
	}
	return plan
}

func checksFailing(s domain.PRSnapshot) bool {
	return s.State == domain.PROpen && s.Checks == domain.PRChecksFailing
}

func changesRequested(s domain.PRSnapshot) bool {
	return s.State == domain.PROpen && s.Review == domain.PRReviewChangesRequested
}
