package data

import (
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

// parseTime reads a stored timestamp. A malformed one maps to the zero time rather than an error:
// the database is shared with agents, and one bad row must not blank the board.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.Local()
}

func parseTimePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t := parseTime(*s)
	return &t
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtr(v *int64) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}

// nonZero is the id as a pointer for the domain layer, nil for 0.
func nonZero(v int) *int64 {
	if v == 0 {
		return nil
	}
	n := int64(v)
	return &n
}

func int64Ptr(v *int) *int64 {
	if v == nil {
		return nil
	}
	n := int64(*v)
	return &n
}

func fromProject(p domain.Project) Project {
	return Project{
		ID: int(p.ID), Name: p.Name, RepoSlug: str(p.RepoSlug), RepoPath: str(p.RepoPath),
		CreatedAt: parseTime(p.CreatedAt), UpdatedAt: parseTime(p.UpdatedAt), ArchivedAt: parseTimePtr(p.ArchivedAt),
		ParentID: intPtr(p.ParentID),
	}
}

func fromMilestone(m domain.Milestone) Milestone {
	return Milestone{
		ID: int(m.ID), ProjectID: int(m.ProjectID), Title: m.Title, DueAt: parseTimePtr(m.DueAt),
		Open: m.Status == "open", CreatedAt: parseTime(m.CreatedAt),
	}
}

func fromTask(t domain.Task) Task {
	out := Task{
		ID: int(t.ID), ParentID: intPtr(t.ParentID), ProjectID: intPtr(t.ProjectID), MilestoneID: intPtr(t.MilestoneID),
		Title: t.Title, Notes: str(t.Notes), Status: Status(t.Status), Priority: Priority(t.Priority),
		CategoryID: intOf(t.CategoryID), DueAt: parseTimePtr(t.DueAt), RemindAt: parseTimePtr(t.RemindAt), RemindedAt: parseTimePtr(t.RemindedAt),
		Branch: str(t.Branch), PRURL: str(t.PRURL), Source: Source(t.Source),
		CreatedAt: parseTime(t.CreatedAt), UpdatedAt: parseTime(t.UpdatedAt), CompletedAt: parseTimePtr(t.CompletedAt),
		HandedTo: str(t.HandedTo), HandedAt: parseTimePtr(t.HandedAt),
	}
	if t.PRNumber != nil {
		out.PRNumber = int(*t.PRNumber)
	}
	out.PRState, out.PRChecks, out.PRReview = pullRequestOf(t)
	out.PRCheckedAt, out.PRClosedSeenAt = parseTimePtr(t.PRCheckedAt), parseTimePtr(t.PRClosedSeenAt)
	return out
}

func intOf(v *int64) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func fromCategory(c domain.Category) Category {
	icon, _ := pixmap.ParseGrid(c.Icon) // a malformed row draws nothing rather than blanking the sidebar
	return Category{ID: int(c.ID), Builtin: c.Builtin, Name: c.Name, Icon: icon, Color: c.Color, Position: c.Position}
}

func fromReport(r domain.Report) Report {
	out := Report{
		ID: int(r.ID), TaskID: int(r.TaskID), Agent: r.Agent, Summary: r.Summary,
		NextStep: str(r.NextStep), CreatedAt: parseTime(r.CreatedAt), Cwd: r.Cwd,
	}
	if r.Status != nil {
		out.Status = Status(*r.Status)
	}
	for _, l := range r.Links {
		out.Links = append(out.Links, Link{URL: l.URL, Title: l.Title})
	}
	return out
}

func fromOverview(o *domain.Overview) Overview {
	out := Overview{GeneratedAt: parseTime(o.GeneratedAt), StaleHours: int(o.StaleHours)}
	for _, g := range o.Projects {
		og := OverviewGroup{}
		if g.Project != nil {
			og.Project = &Project{ID: int(g.Project.ID), Name: g.Project.Name, RepoSlug: str(g.Project.RepoSlug)}
		}
		for _, t := range g.Tasks {
			ot := OverviewTask{Task: fromTask(t.Task), Stale: t.Stale, HoursSinceActivity: int(t.HoursSinceActivity)}
			if t.LatestReport != nil {
				r := fromReport(*t.LatestReport)
				ot.LatestReport = &r
			}
			og.Tasks = append(og.Tasks, ot)
		}
		out.Groups = append(out.Groups, og)
	}
	return out
}

func toStatuses(in []Status) []domain.Status {
	out := make([]domain.Status, len(in))
	for i, s := range in {
		out[i] = domain.Status(s)
	}
	return out
}

// nullableText maps the UI's "empty clears" convention onto a domain patch value.
func nullableText(s *string) domain.Nullable[string] {
	switch {
	case s == nil:
		return domain.Nullable[string]{}
	case *s == "":
		return domain.Clear[string]()
	}
	return domain.Set(*s)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := domain.FormatTime(*t)
	return &s
}

func nullableTime(t *time.Time) domain.Nullable[string] {
	switch {
	case t == nil:
		return domain.Nullable[string]{}
	case t.IsZero():
		return domain.Clear[string]()
	}
	return domain.Set(domain.FormatTime(*t))
}

func toPatch(p Patch) domain.TaskPatch {
	out := domain.TaskPatch{
		Title: p.Title, Notes: nullableText(p.Notes), Branch: nullableText(p.Branch),
		DueAt: nullableTime(p.DueAt), RemindAt: nullableTime(p.RemindAt),
	}
	if p.Status != nil {
		s := domain.Status(*p.Status)
		out.Status = &s
	}
	if p.Priority != nil {
		pr := domain.Priority(*p.Priority)
		out.Priority = &pr
	}
	if p.Category != nil {
		out.CategoryID = domain.Clear[int64]()
		if *p.Category != 0 {
			out.CategoryID = domain.Set(int64(*p.Category))
		}
	}
	if p.ProjectID != nil {
		out.ProjectID = domain.Clear[int64]()
		if *p.ProjectID != 0 {
			out.ProjectID = domain.Set(int64(*p.ProjectID))
		}
		// A milestone belongs to one project, so a move leaves it behind.
		out.MilestoneID = domain.Clear[int64]()
	}
	if p.MilestoneID != nil {
		if *p.MilestoneID == 0 {
			out.MilestoneID = domain.Clear[int64]()
		} else {
			out.MilestoneID = domain.Set(int64(*p.MilestoneID))
		}
	}
	return out
}

func pullRequestOf(t domain.Task) (PRState, PRChecks, PRReview) {
	snap := t.PullRequest()
	return PRState(snap.State), PRChecks(snap.Checks), PRReview(snap.Review)
}
