package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// ScopeKind says where in the app a request was typed.
type ScopeKind int

const (
	ScopeNow      ScopeKind = iota // the Now page: no place of its own
	ScopeProject                   // a project's page
	ScopeInbox                     // the Inbox page
	ScopeTask                      // the inspector of one task
	ScopeCalendar                  // the Calendar page, with a day selected
)

// Scope is the place a request comes from. It decides where a new task goes and what "this" and
// "it" mean.
type Scope struct {
	Kind      ScopeKind
	ProjectID int64 // ScopeProject
	TaskID    int64 // ScopeTask
	// Day is the selected day of the Calendar page, "2006-01-02" in the user's zone (ScopeCalendar).
	Day string
	// CategoryID is the category page of Mine (ScopeNow), 0 for Mine as a whole: a new task is in
	// that category unless the words say otherwise.
	CategoryID int64
}

// scopeSection is the part of the prompt that states the scope. A project that is gone or
// archived, which is what a stale page may name, reads as the Now page: new tasks go to the Inbox.
// So does a category that was deleted.
func scopeSection(now time.Time, projects []domain.Project, categories []domain.Category, sc Scope, focus *domain.Task) string {
	switch {
	case sc.Kind == ScopeTask && focus != nil:
		return "Where the request comes from: the inspector of one task. “this”, “it”, “the task” mean the focused task, " +
			"so change it with update_task and its id; do not look it up with find_tasks. " +
			"A bare time such as “friday afternoon” or “push it to tomorrow” changes the date the task already uses: " +
			"the reminder if it has one and no due date, otherwise the due date.\n" + focusedTask(now, projects, *focus)
	case sc.Kind == ScopeProject:
		if p, ok := activeProject(projects, sc.ProjectID); ok {
			return fmt.Sprintf("Where the request comes from: the page of the project “%s” (id %d). A new task goes in that "+
				"project unless the words name another project (then use that one) or the Inbox (then pass project_id 0).\n", p.Name, p.ID)
		}
	case sc.Kind == ScopeCalendar:
		if day, ok := selectedDay(now, sc.Day); ok {
			return calendarSection(day)
		}
	case sc.Kind == ScopeInbox:
		return "Where the request comes from: the Inbox page. A new task goes to the Inbox unless the words name a project.\n"
	}
	if sc.Kind == ScopeNow {
		for _, c := range categories {
			if c.ID == sc.CategoryID {
				return fmt.Sprintf("Where the request comes from: the %s page of Mine. A new task goes to the Inbox unless the words "+
					"name a project, and its category is %s unless the words name another category.\n", c.Name, c.Name)
			}
		}
	}
	return "Where the request comes from: the Now page. A new task goes to the Inbox unless the words name a project.\n"
}

func activeProject(projects []domain.Project, id int64) (domain.Project, bool) {
	for _, p := range projects {
		if p.ID == id && p.ArchivedAt == nil {
			return p, true
		}
	}
	return domain.Project{}, false
}

// focusedTask spells out the task the inspector shows, times in the user's zone.
func focusedTask(now time.Time, projects []domain.Project, t domain.Task) string {
	project := "Inbox"
	if t.ProjectID != nil {
		project = fmt.Sprintf("project %d", *t.ProjectID)
		for _, p := range projects {
			if p.ID == *t.ProjectID {
				project = "project " + p.Name
			}
		}
	}
	parts := []string{
		fmt.Sprintf("Focused task: id %d, “%s”", t.ID, t.Title),
		"status " + string(t.Status), "priority " + string(t.Priority), project,
	}
	for _, f := range []struct {
		name   string
		stored *string
	}{{"due", t.DueAt}, {"remind", t.RemindAt}} {
		if s := localRFC3339(f.stored, now.Location()); s != "" {
			parts = append(parts, f.name+" "+s)
		}
	}
	line := strings.Join(parts, ", ") + ".\n"
	if t.Notes != nil && strings.TrimSpace(*t.Notes) != "" {
		line += "Its notes: " + strings.TrimSpace(*t.Notes) + "\n"
	}
	return line
}

// selectedDay reads the day of a calendar scope in the user's zone; a day that does not parse
// leaves the scope reading as the Now page.
func selectedDay(now time.Time, s string) (time.Time, bool) {
	day, err := time.ParseInLocation("2006-01-02", s, now.Location())
	return day, err == nil
}

// calendarSection states the Calendar page and its selected day: a task with no time words is due
// that day at 18:00, written out in full so the model does not work out the offset.
func calendarSection(day time.Time) string {
	due := time.Date(day.Year(), day.Month(), day.Day(), 18, 0, 0, 0, day.Location())
	return fmt.Sprintf("Where the request comes from: the Calendar page, with %s selected. A new task goes to the Inbox "+
		"unless the words name a project. The selected day is the default day: a task with no time words (“buy milk”) is due "+
		"that day at 18:00 local, due_at %s, and a time of day without a day (“at 3pm”) is on that day. "+
		"If the words name another day (“tomorrow”, “next Monday”), that day wins.\n",
		day.Format("Monday, 2006-01-02"), due.Format(time.RFC3339))
}
