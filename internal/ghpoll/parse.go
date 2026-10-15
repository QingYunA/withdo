package ghpoll

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// PRViewFields are the fields asked of gh pr view.
const PRViewFields = "state,mergedAt,closedAt,isDraft,reviewDecision,statusCheckRollup"

// prView is the part of gh pr view --json that is read.
type prView struct {
	State          string       `json:"state"`
	MergedAt       *string      `json:"mergedAt"`
	IsDraft        bool         `json:"isDraft"`
	ReviewDecision string       `json:"reviewDecision"`
	Rollup         []checkEntry `json:"statusCheckRollup"`
}

// checkEntry is a CheckRun (status, conclusion) or a StatusContext (state) of the rollup.
type checkEntry struct {
	Type       string `json:"__typename"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"`
}

// ParsePRView reads the output of gh pr view --json PRViewFields.
func ParsePRView(raw []byte) (domain.PRSnapshot, error) {
	var v prView
	if err := json.Unmarshal(raw, &v); err != nil {
		return domain.PRSnapshot{}, fmt.Errorf("read gh pr view output: %w", err)
	}
	state, err := v.prState()
	if err != nil {
		return domain.PRSnapshot{}, err
	}
	return domain.PRSnapshot{State: state, Checks: summarize(v.Rollup), Review: reviewOf(v.ReviewDecision)}, nil
}

func (v prView) prState() (domain.PRState, error) {
	switch {
	case v.State == "MERGED" || v.MergedAt != nil && *v.MergedAt != "":
		return domain.PRMerged, nil
	case v.State == "CLOSED":
		return domain.PRClosed, nil
	case v.State == "OPEN" && v.IsDraft:
		return domain.PRDraft, nil
	case v.State == "OPEN":
		return domain.PROpen, nil
	}
	return "", fmt.Errorf("unknown pull request state %q", v.State)
}

func reviewOf(decision string) domain.PRReview {
	switch decision {
	case "APPROVED":
		return domain.PRReviewApproved
	case "CHANGES_REQUESTED":
		return domain.PRReviewChangesRequested
	case "REVIEW_REQUIRED":
		return domain.PRReviewRequired
	}
	return domain.PRReviewNone
}

// summarize folds the rollup into one word: failing if any check failed, else pending if any is
// still running or waiting, else passing; none without checks. Skipped, neutral and cancelled
// checks count as passing: they are not what the user waits on.
func summarize(rollup []checkEntry) domain.PRChecks {
	if len(rollup) == 0 {
		return domain.PRChecksNone
	}
	pending := false
	for _, e := range rollup {
		switch e.outcome() {
		case domain.PRChecksFailing:
			return domain.PRChecksFailing
		case domain.PRChecksPending:
			pending = true
		}
	}
	if pending {
		return domain.PRChecksPending
	}
	return domain.PRChecksPassing
}

// outcome is passing, failing or pending for one entry.
func (e checkEntry) outcome() domain.PRChecks {
	if e.Type == "StatusContext" || e.State != "" {
		switch strings.ToUpper(e.State) {
		case "FAILURE", "ERROR":
			return domain.PRChecksFailing
		case "PENDING", "EXPECTED":
			return domain.PRChecksPending
		}
		return domain.PRChecksPassing
	}
	if status := strings.ToUpper(e.Status); status != "COMPLETED" {
		return domain.PRChecksPending
	}
	switch strings.ToUpper(e.Conclusion) {
	case "FAILURE", "TIMED_OUT", "STARTUP_FAILURE":
		return domain.PRChecksFailing
	case "ACTION_REQUIRED":
		return domain.PRChecksPending
	}
	return domain.PRChecksPassing
}
