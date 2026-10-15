package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
)

// The words of the interface that depend on the data: a status, a priority, a sentence about a
// task. Fixed texts are read straight from the catalog (m.say).

func statusLabel(say locales.Messages, s data.Status) string {
	switch s {
	case data.Backlog:
		return say.StatusBacklog()
	case data.Todo:
		return say.StatusTodo()
	case data.Doing:
		return say.StatusDoing()
	case data.Blocked:
		return say.StatusBlocked()
	case data.Review:
		return say.StatusReview()
	case data.Done:
		return say.StatusDone()
	case data.Cancelled:
		return say.StatusCancelled()
	}
	return string(s)
}

func priorityName(say locales.Messages, p data.Priority) string {
	switch p {
	case data.P0:
		return say.PriorityUrgent()
	case data.P1:
		return say.PriorityHigh()
	case data.P2:
		return say.PriorityNormal()
	case data.P3:
		return say.PriorityLow()
	}
	return string(p)
}

// priorityLabel is "P1  High".
func priorityLabel(say locales.Messages, p data.Priority) string {
	return string(p) + "  " + priorityName(say, p)
}

// providerName is a provider's name: its product's, except for Custom.
func providerName(say locales.Messages, p data.Provider) string {
	if p.ID == "custom" {
		return say.ProviderCustom()
	}
	return p.Name
}

// providerNote is the line under a provider's name. The presets write theirs in English; the
// ones the app knows are said in the user's language.
func providerNote(say locales.Messages, p data.Provider) string {
	switch p.ID {
	case "magpie":
		return say.ProviderNoteMagpie()
	case "ollama":
		return say.ProviderNoteOllama()
	case "custom":
		return say.ProviderNoteCustom()
	}
	return p.Note
}

// problemText words a failure of the model. Errors that are not a known kind keep their own text.
func problemText(say locales.Messages, err error) string {
	switch data.ProblemOf(err) {
	case data.ProblemNotConfigured:
		return say.ProblemNotConfigured()
	case data.ProblemNoKey:
		return say.ProblemNoKey()
	case data.ProblemCancelled:
		return say.ProblemCancelled()
	case data.ProblemTimeout:
		return say.ProblemTimeout()
	case data.ProblemUnreachable:
		return say.ProblemUnreachable()
	case data.ProblemKeyRejected:
		return say.ProblemKeyRejected()
	case data.ProblemUnknownModel:
		return say.ProblemUnknownModel()
	case data.ProblemRateLimited:
		return say.ProblemRateLimited()
	case data.ProblemBusy:
		return say.ProblemBusy()
	}
	return err.Error()
}

// detailText is the quiet line of a receipt, worded from what the agent changed. What the words
// made of a new task shows too, so it can be checked: the reminder and the deadline it set, and a
// priority that is not the default.
func (m *Main) detailText(d data.Detail) string {
	text := m.detailMain(d)
	if !d.Due.IsZero() {
		text += " · " + m.say.ChangeDue(m.clock.When(d.Due, m.app.Now()))
	}
	if d.Priority != "" {
		text += " · " + d.Priority
	}
	return text
}

func (m *Main) detailMain(d data.Detail) string {
	now := m.app.Now()
	switch d.Kind {
	case data.DetailReminder:
		return m.say.DetailReminder(m.clock.When(d.At, now))
	case data.DetailDue:
		return m.say.DetailDue(m.clock.When(d.At, now))
	case data.DetailDone:
		return m.say.DetailDone()
	case data.DetailUpdated:
		parts := make([]string, 0, len(d.Changes))
		for _, c := range d.Changes {
			parts = append(parts, m.changeText(c))
		}
		return strings.Join(parts, " · ")
	}
	return m.say.DetailAdded(m.inboxOr(d.Project))
}

// changeText words one change of an update the way a new task's detail reads: "Tomorrow 9:00 AM ·
// reminder", "Fri · due", "Marked done", "Moved to adhd-todo".
func (m *Main) changeText(c data.Change) string {
	now := m.app.Now()
	switch c.Kind {
	case data.ChangeReminder:
		if c.At.IsZero() {
			return m.say.ChangeReminderCleared()
		}
		return m.say.DetailReminder(m.clock.When(c.At, now))
	case data.ChangeDue:
		if c.At.IsZero() {
			return m.say.ChangeDueCleared()
		}
		return m.say.DetailDue(m.clock.When(c.At, now))
	case data.ChangeStatus:
		if data.Status(c.Value) == data.Done {
			return m.say.DetailDone()
		}
		return m.say.ChangeStatus(statusLabel(m.say, data.Status(c.Value)))
	case data.ChangeProject:
		return m.say.ChangeProject(m.inboxOr(c.Value))
	case data.ChangeRenamed:
		return m.say.ChangeRenamed()
	case data.ChangeNotes:
		return m.say.ChangeNotes()
	}
	return c.Value // a priority: P1
}

// inboxOr names the Inbox for a task without a project.
func (m *Main) inboxOr(project string) string {
	if project == "" {
		return m.say.NavInbox()
	}
	return project
}

// needSummary is the second line of a Needs you row: why it is there.
func (m *Main) needSummary(it data.NowItem) string {
	now := m.app.Now()
	switch it.Kind {
	case data.NeedBlocked:
		switch {
		case it.Source != data.SourceAgent:
			return m.say.NeedBlockedByMe()
		case it.Words == "":
			return m.say.NeedAgentWaiting()
		}
		return m.say.NeedAgentWaitingFor(it.Words)
	case data.NeedReminder:
		return m.clock.Reminder(*it.RemindAt, now)
	case data.NeedPRClosed:
		return m.say.NeedPRClosed(itoa(it.PRNumber))
	case data.NeedPRFailing:
		return m.say.NeedPRChecksFailing(itoa(it.PRNumber))
	case data.NeedPRChanges:
		return m.say.NeedPRChangesRequested(itoa(it.PRNumber))
	case data.NeedReview:
		if it.PRNumber > 0 {
			return m.say.NeedReviewPR(itoa(it.PRNumber))
		}
		return m.say.NeedReview()
	}
	return m.say.NeedNoUpdate(m.clock.Span(it.Hours))
}

// agentSummary is the second line of an agent row: what it said, and what it does next.
func (m *Main) agentSummary(it data.NowItem) string {
	if it.NextStep == "" {
		return it.Summary
	}
	return m.say.AgentSummaryNext(it.Summary, m.sentenceCase(it.NextStep))
}

// sentenceCase lowers the first letter of a step that follows "next:", keeping acronyms and names
// ("MCP tools"). Chinese has no case to lower.
func (m *Main) sentenceCase(s string) string {
	if s == "" || strings.ToUpper(s[:1]) != s[:1] {
		return s
	}
	if len(s) > 1 && strings.ToUpper(s[1:2]) == s[1:2] && strings.ToLower(s[1:2]) != s[1:2] {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
