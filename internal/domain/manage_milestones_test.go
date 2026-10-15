package domain_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestUpdateMilestone(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1", DueAt: ptr("2026-12-01T00:00:00.000Z")}))

	renamed := must[*domain.Milestone](t)(c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{Title: ptr("v1.0")}))
	eq(t, "title", renamed.Title, "v1.0")
	eqPtr(t, "due kept", renamed.DueAt, ptr("2026-12-01T00:00:00.000Z"))
	eq(t, "status kept", renamed.Status, domain.MilestoneOpen)

	moved := must[*domain.Milestone](t)(c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{DueAt: domain.Set("2027-01-15T00:00:00.000Z")}))
	eqPtr(t, "due", moved.DueAt, ptr("2027-01-15T00:00:00.000Z"))
	cleared := must[*domain.Milestone](t)(c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{DueAt: domain.Clear[string]()}))
	if cleared.DueAt != nil {
		t.Errorf("due not cleared: %v", *cleared.DueAt)
	}

	closed := must[*domain.Milestone](t)(c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{Status: ptr(domain.MilestoneClosed)}))
	eq(t, "closed", closed.Status, domain.MilestoneClosed)
	reopened := must[*domain.Milestone](t)(c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{Status: ptr(domain.MilestoneOpen)}))
	eq(t, "reopened", reopened.Status, domain.MilestoneOpen)
	eq(t, "title survives", reopened.Title, "v1.0")
}

func TestUpdateMilestoneRejectsWhatIsWrong(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	_, err := c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{Title: ptr(" ")})
	wantFieldError(t, err, "title")
	_, err = c.UpdateMilestone(ctx, m.ID, domain.MilestonePatch{Status: ptr("paused")})
	wantFieldError(t, err, "status")
	_, err = c.UpdateMilestone(ctx, 999, domain.MilestonePatch{Title: ptr("x")})
	if !domain.IsNotFound(err) {
		t.Errorf("a missing milestone is not found: %v", err)
	}
	eq(t, "unchanged", must[[]domain.Milestone](t)(c.ListMilestones(ctx, p.ID))[0].Title, "v1")
}
