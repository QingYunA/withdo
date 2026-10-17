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

var (
	repMerged  = domain.PRReport{Kind: domain.KindPRMerged, Summary: "PR #12 merged"}
	repClosed  = domain.PRReport{Kind: domain.KindPRClosed, Summary: "PR #12 was closed without merging"}
	repFailing = domain.PRReport{Kind: domain.KindChecksFailing, Summary: "Checks are failing on PR #12"}
	repChanges = domain.PRReport{Kind: domain.KindChangesRequested, Summary: "Changes were requested on PR #12"}
)

// The effects of a check are a function of what was stored and what was seen: each report fires
// when its condition becomes true, never while it stays true.
func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		old, new domain.PRSnapshot
		done     bool
		reports  []domain.PRReport
	}{
		{"first look, nothing special", never, openOK, false, nil},
		{"open to merged", openOK, merged, true, []domain.PRReport{repMerged}},
		{"first look finds it merged", never, merged, true, []domain.PRReport{repMerged}},
		{"draft to merged", draftBad, merged, true, []domain.PRReport{repMerged}},
		{"merged stays merged", merged, merged, false, nil},
		{"merged with an old failing state files only the merge", openFail, snap(domain.PRMerged, domain.PRChecksFailing, domain.PRReviewChangesRequested), true, []domain.PRReport{repMerged}},

		{"open to closed", openOK, closed, false, []domain.PRReport{repClosed}},
		{"draft to closed", draftBad, closed, false, []domain.PRReport{repClosed}},
		{"first look finds it closed", never, closed, false, []domain.PRReport{repClosed}},
		{"closed stays closed", closed, closed, false, nil},
		{"reopened", closed, openOK, false, nil},
		{"reopened and closed again", openOK, closed, false, []domain.PRReport{repClosed}},

		{"checks start failing", openPend, openFail, false, []domain.PRReport{repFailing}},
		{"first look finds checks failing", never, openFail, false, []domain.PRReport{repFailing}},
		{"checks keep failing", openFail, openFail, false, nil},
		{"checks recover", openFail, openOK, false, nil},
		{"checks fail again after a rerun", openPend, openFail, false, []domain.PRReport{repFailing}},
		{"failing checks of a draft say nothing", openOK, draftBad, false, nil},
		{"draft marked ready while failing", draftBad, openFail, false, []domain.PRReport{repFailing}},
		{"failing checks of a closed pull request say nothing", openFail, snap(domain.PRClosed, domain.PRChecksFailing, domain.PRReviewNone), false, []domain.PRReport{repClosed}},

		{"changes requested", openOK, openChg, false, []domain.PRReport{repChanges}},
		{"first look finds changes requested", never, openChg, false, []domain.PRReport{repChanges}},
		{"changes still requested", openChg, openChg, false, nil},
		{"changes addressed", openChg, openOK, false, nil},
		{"changes requested again after approval", snap(domain.PROpen, domain.PRChecksPassing, domain.PRReviewApproved), openChg, false, []domain.PRReport{repChanges}},

		{"failing and changes together, failing first", openOK, snap(domain.PROpen, domain.PRChecksFailing, domain.PRReviewChangesRequested), false, []domain.PRReport{repFailing, repChanges}},
		{"one of two persists", openFail, snap(domain.PROpen, domain.PRChecksFailing, domain.PRReviewChangesRequested), false, []domain.PRReport{repChanges}},
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
				t.Errorf("reports = %+v, want %+v", got.Reports, tt.reports)
			}
		})
	}
}
