package domain

import (
	"fmt"
	"strings"
	"time"
)

// FormatTaskLine is the one-line task summary, e.g.
// `#12 [doing] P2 feat/login — Add login (PR #3)`.
func FormatTaskLine(t Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "#%d [%s] %s", t.ID, t.Status, t.Priority)
	if t.Branch != nil && *t.Branch != "" {
		b.WriteString(" " + *t.Branch)
	}
	fmt.Fprintf(&b, " — %s", t.Title)
	switch {
	case t.PRNumber != nil && *t.PRNumber != 0:
		fmt.Fprintf(&b, " (PR #%d)", *t.PRNumber)
	case t.PRURL != nil && *t.PRURL != "":
		fmt.Fprintf(&b, " (%s)", *t.PRURL)
	}
	return b.String()
}

// FormatOverview renders the overview as text for the get_overview tool, with times as stored (UTC).
func FormatOverview(o *Overview) string { return FormatOverviewIn(o, nil) }

// FormatOverviewIn is FormatOverview with report times shown in loc; nil keeps them as stored.
func FormatOverviewIn(o *Overview, loc *time.Location) string {
	if len(o.Projects) == 0 {
		return "No active tasks (nothing in doing, blocked or review)."
	}
	var lines []string
	for _, g := range o.Projects {
		lines = append(lines, groupTitle(g))
		for _, t := range g.Tasks {
			lines = append(lines, formatOverviewTask(t, loc)...)
		}
		lines = append(lines, "")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n ")
}

func groupTitle(g OverviewGroup) string {
	if g.Project == nil {
		return "Inbox"
	}
	if g.Project.RepoSlug != nil && *g.Project.RepoSlug != "" {
		return fmt.Sprintf("%s (%s)", g.Project.Name, *g.Project.RepoSlug)
	}
	return g.Project.Name
}

func formatOverviewTask(t OverviewTask, loc *time.Location) []string {
	flag := ""
	if t.Stale {
		flag = fmt.Sprintf("  STALE (%dh)", t.HoursSinceActivity)
	}
	lines := []string{"  " + FormatTaskLine(t.Task) + flag}
	if r := t.LatestReport; r != nil {
		lines = append(lines, fmt.Sprintf("      last: %s (%s, %s)", r.Summary, r.Agent, inZone(r.CreatedAt, loc)))
		if r.NextStep != nil && *r.NextStep != "" {
			lines = append(lines, "      next: "+*r.NextStep)
		}
	}
	return lines
}

// inZone shows a stored timestamp in loc, to the second; it comes back as stored when loc is nil or
// the timestamp does not parse.
func inZone(stored string, loc *time.Location) string {
	if loc == nil {
		return stored
	}
	t, err := time.Parse(timeLayout, stored)
	if err != nil {
		return stored
	}
	return t.In(loc).Format(time.RFC3339)
}
