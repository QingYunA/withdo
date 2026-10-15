package ghpoll

import (
	"reflect"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func snap(state domain.PRState, checks domain.PRChecks, review domain.PRReview) domain.PRSnapshot {
	return domain.PRSnapshot{State: state, Checks: checks, Review: review}
}

var (
	never    = domain.PRSnapshot{}
	openOK   = snap(domain.PROpen, domain.PRChecksPassing, domain.PRReviewNone)
	openFail = snap(domain.PROpen, domain.PRChecksFailing, domain.PRReviewNone)
	openPend = snap(domain.PROpen, domain.PRChecksPending, domain.PRReviewNone)
	openChg  = snap(domain.PROpen, domain.PRChecksPassing, domain.PRReviewChangesRequested)
	draftBad = snap(domain.PRDraft, domain.PRChecksFailing, domain.PRReviewNone)
	merged   = snap(domain.PRMerged, domain.PRChecksPassing, domain.PRReviewApproved)
	closed   = snap(domain.PRClosed, domain.PRChecksNone, domain.PRReviewNone)
)

const (
	repMerged  = "PR #12 merged"
	repClosed  = "PR #12 was closed without merging"
	repFailing = "Checks are failing on PR #12"
	repChanges = "Changes were requested on PR #12"
)

// The effects of a check are a function of what was stored and what was seen: each report fires
// when its condition becomes true, never while it stays true.
func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		old, new domain.PRSnapshot
		done     bool
		reports  []string
	}{
		{"first look, nothing special", never, openOK, false, nil},
		{"open to merged", openOK, merged, true, []string{repMerged}},
		{"first look finds it merged", never, merged, true, []string{repMerged}},
		{"draft to merged", draftBad, merged, true, []string{repMerged}},
		{"merged stays merged", merged, merged, false, nil},
		{"merged with an old failing state files only the merge", openFail, snap(domain.PRMerged, domain.PRChecksFailing, domain.PRReviewChangesRequested), true, []string{repMerged}},

		{"open to closed", openOK, closed, false, []string{repClosed}},
		{"draft to closed", draftBad, closed, false, []string{repClosed}},
		{"first look finds it closed", never, closed, false, []string{repClosed}},
		{"closed stays closed", closed, closed, false, nil},
		{"reopened", closed, openOK, false, nil},
		{"reopened and closed again", openOK, closed, false, []string{repClosed}},

		{"checks start failing", openPend, openFail, false, []string{repFailing}},
		{"first look finds checks failing", never, openFail, false, []string{repFailing}},
		{"checks keep failing", openFail, openFail, false, nil},
		{"checks recover", openFail, openOK, false, nil},
		{"checks fail again after a rerun", openPend, openFail, false, []string{repFailing}},
		{"failing checks of a draft say nothing", openOK, draftBad, false, nil},
		{"draft marked ready while failing", draftBad, openFail, false, []string{repFailing}},
		{"failing checks of a closed pull request say nothing", openFail, snap(domain.PRClosed, domain.PRChecksFailing, domain.PRReviewNone), false, []string{repClosed}},

		{"changes requested", openOK, openChg, false, []string{repChanges}},
		{"first look finds changes requested", never, openChg, false, []string{repChanges}},
		{"changes still requested", openChg, openChg, false, nil},
		{"changes addressed", openChg, openOK, false, nil},
		{"changes requested again after approval", snap(domain.PROpen, domain.PRChecksPassing, domain.PRReviewApproved), openChg, false, []string{repChanges}},

		{"failing and changes together, failing first", openOK, snap(domain.PROpen, domain.PRChecksFailing, domain.PRReviewChangesRequested), false, []string{repFailing, repChanges}},
		{"one of two persists", openFail, snap(domain.PROpen, domain.PRChecksFailing, domain.PRReviewChangesRequested), false, []string{repChanges}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.old, tt.new, 12)
			if got.Snapshot != tt.new {
				t.Errorf("snapshot = %+v, want %+v", got.Snapshot, tt.new)
			}
			if got.Done != tt.done {
				t.Errorf("done = %v, want %v", got.Done, tt.done)
			}
			if !reflect.DeepEqual(got.Reports, tt.reports) {
				t.Errorf("reports = %q, want %q", got.Reports, tt.reports)
			}
		})
	}
}
