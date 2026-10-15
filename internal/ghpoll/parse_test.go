package ghpoll

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The samples in testdata are shaped like the output of
// gh pr view <url> --json state,mergedAt,closedAt,isDraft,reviewDecision,statusCheckRollup.
func TestParsePRView(t *testing.T) {
	tests := []struct {
		file string
		want domain.PRSnapshot
	}{
		{"open-passing-approved.json", domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksPassing, Review: domain.PRReviewApproved}},
		{"open-failing.json", domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksFailing, Review: domain.PRReviewRequired}},
		{"open-pending.json", domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksPending, Review: domain.PRReviewNone}},
		{"status-context-failing.json", domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksFailing, Review: domain.PRReviewNone}},
		{"open-no-checks.json", domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksNone, Review: domain.PRReviewNone}},
		{"open-null-rollup.json", domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksNone, Review: domain.PRReviewNone}},
		{"draft.json", domain.PRSnapshot{State: domain.PRDraft, Checks: domain.PRChecksFailing, Review: domain.PRReviewNone}},
		{"merged.json", domain.PRSnapshot{State: domain.PRMerged, Checks: domain.PRChecksPassing, Review: domain.PRReviewApproved}},
		{"closed.json", domain.PRSnapshot{State: domain.PRClosed, Checks: domain.PRChecksNone, Review: domain.PRReviewNone}},
		{"changes-requested.json", domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksPassing, Review: domain.PRReviewChangesRequested}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			got, err := ParsePRView(raw)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParsePRViewMergedWinsOverClosedAt(t *testing.T) {
	// A merged pull request also has closedAt; an old gh may say CLOSED with mergedAt set.
	got, err := ParsePRView([]byte(`{"state":"CLOSED","mergedAt":"2026-10-14T09:02:13Z","closedAt":"2026-10-14T09:02:13Z","isDraft":false,"reviewDecision":"","statusCheckRollup":[]}`))
	if err != nil || got.State != domain.PRMerged {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestParsePRViewErrors(t *testing.T) {
	for name, in := range map[string]string{
		"empty":         "",
		"not json":      "no pull requests found",
		"unknown state": `{"state":"WEIRD"}`,
		"no state":      `{"isDraft":false}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePRView([]byte(in)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestChecksOfOneEntry(t *testing.T) {
	tests := []struct {
		entry string
		want  domain.PRChecks
	}{
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"SUCCESS"}`, domain.PRChecksPassing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"NEUTRAL"}`, domain.PRChecksPassing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"SKIPPED"}`, domain.PRChecksPassing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"CANCELLED"}`, domain.PRChecksPassing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"STALE"}`, domain.PRChecksPassing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"FAILURE"}`, domain.PRChecksFailing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"TIMED_OUT"}`, domain.PRChecksFailing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"STARTUP_FAILURE"}`, domain.PRChecksFailing},
		{`{"__typename":"CheckRun","status":"COMPLETED","conclusion":"ACTION_REQUIRED"}`, domain.PRChecksPending},
		{`{"__typename":"CheckRun","status":"IN_PROGRESS","conclusion":""}`, domain.PRChecksPending},
		{`{"__typename":"CheckRun","status":"QUEUED"}`, domain.PRChecksPending},
		{`{"__typename":"CheckRun","status":"WAITING"}`, domain.PRChecksPending},
		{`{"__typename":"StatusContext","state":"SUCCESS"}`, domain.PRChecksPassing},
		{`{"__typename":"StatusContext","state":"FAILURE"}`, domain.PRChecksFailing},
		{`{"__typename":"StatusContext","state":"ERROR"}`, domain.PRChecksFailing},
		{`{"__typename":"StatusContext","state":"PENDING"}`, domain.PRChecksPending},
		{`{"__typename":"StatusContext","state":"EXPECTED"}`, domain.PRChecksPending},
	}
	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.entry, `"`, ""), func(t *testing.T) {
			raw := `{"state":"OPEN","statusCheckRollup":[` + tt.entry + `]}`
			got, err := ParsePRView([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			if got.Checks != tt.want {
				t.Errorf("checks = %s, want %s", got.Checks, tt.want)
			}
		})
	}
}
