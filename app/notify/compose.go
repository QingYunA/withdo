// Package notify words and paces the app's system notifications: what each event says, when a
// burst of them becomes one summary, and when none is shown. macOS draws them; this package only
// chooses their text, sound, stack and ID.
package notify

import (
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
)

// SummaryID is the ID of the summary notification: each update of a burst replaces the last.
const SummaryID = "summary"

// TaskID is the ID of the notification about a task; a click opens the task.
func TaskID(taskID int) string { return fmt.Sprintf("task/%d", taskID) }

// ParseTaskID reverses TaskID.
func ParseTaskID(id string) (int, bool) {
	var n int
	if _, err := fmt.Sscanf(id, "task/%d", &n); err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// stack is the Group of a project's notifications: they pile up together in Notification Center.
// The Inbox is one stack, whatever the language.
func stack(project string) string {
	if project == "" {
		return "inbox"
	}
	return "project/" + project
}

func projectLabel(say locales.Messages, project string) string {
	if project == "" {
		return say.NavInbox()
	}
	return project
}

// Event words a notification about an event.
func Event(say locales.Messages, e data.Event) mygo.NotificationOptions {
	o := mygo.NotificationOptions{
		ID: TaskID(e.TaskID), Subtitle: projectLabel(say, e.ProjectName), Group: stack(e.ProjectName),
	}
	agent := views.AgentName(e.Agent)
	switch e.Kind {
	case data.EventHandedBack:
		o.Title = say.NotifyHandedBack(agent, e.Title)
		o.Body = e.Detail
		if e.PRNumber > 0 {
			o.Body = say.NotifyPRReady(e.PRNumber)
		}
	case data.EventStuck:
		o.Title, o.Body = say.NotifyStuck(agent, e.Title), e.Detail
	case data.EventChecksFailing:
		o.Title, o.Body = say.NotifyChecksFailing(e.PRNumber, e.Title), say.NotifyChecksBody()
	case data.EventMerged:
		// Good news that wants nothing from the user: no sound.
		o.Title, o.Body, o.Silent = say.NotifyMerged(e.PRNumber, e.Title), say.NotifyMergedBody(), true
	}
	return o
}

// Reminder words the notification of a due reminder: its project, or the Inbox, in the body.
func Reminder(say locales.Messages, r data.Reminder) mygo.NotificationOptions {
	return mygo.NotificationOptions{ID: TaskID(r.TaskID), Title: say.NotifyReminder(r.Title), Body: projectLabel(say, r.ProjectName)}
}

// Summary words the one notification a burst becomes. It is silent only when every event in it
// was a merge.
func Summary(say locales.Messages, c Counts) mygo.NotificationOptions {
	var parts []string
	add := func(n int, word func(int) string) {
		if n > 0 {
			parts = append(parts, word(n))
		}
	}
	add(c.HandedBack, say.NotifyCountFinished)
	add(c.Stuck, say.NotifyCountStuck)
	add(c.ChecksFailing, say.NotifyCountChecks)
	add(c.Merged, say.NotifyCountMerged)
	return mygo.NotificationOptions{
		ID:     SummaryID,
		Title:  say.NotifySummaryTitle(c.Total()),
		Body:   say.NotifySummaryBody(strings.Join(parts, say.NotifyCountSeparator())),
		Silent: c.Merged == c.Total(),
	}
}
