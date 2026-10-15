package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

type milestoneMode int

const (
	milestoneNew milestoneMode = iota
	milestoneRename
	milestoneDue
)

// milestoneSheet is the sheet that makes a milestone or changes its name or its due date.
type milestoneSheet struct {
	open    bool
	mode    milestoneMode
	project int // for a new milestone
	id      int // for the others
	name    string
	title   string // being typed
	due     time.Time
	hasDue  bool
	dueOpen bool // the calendar popover
}

func (m *Main) newMilestone(project int) {
	m.manage.milestone = milestoneSheet{open: true, mode: milestoneNew, project: project, due: m.today()}
}

func (m *Main) renameMilestone(ms data.Milestone) {
	m.manage.milestone = milestoneSheet{open: true, mode: milestoneRename, id: ms.ID, title: ms.Title, name: ms.Title}
}

func (m *Main) dueMilestone(ms data.Milestone) {
	s := milestoneSheet{open: true, mode: milestoneDue, id: ms.ID, name: ms.Title, due: m.today()}
	if ms.DueAt != nil {
		s.due, s.hasDue = *ms.DueAt, true
	}
	m.manage.milestone = s
}

// today is the start of today, where the calendar opens.
func (m *Main) today() time.Time {
	y, mo, d := m.app.Now().Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, m.app.Now().Location())
}

// milestoneSheet draws the sheet while it is open.
func (m *Main) milestoneSheet(c *ui.Context, t *theme.Tokens) {
	s := &m.manage.milestone
	if !s.open {
		return
	}
	sheet(c, t, &s.open, func() {
		theme.Paragraph(c, theme.Title, t.Text, m.milestoneSheetTitle(), 0)
		if s.mode != milestoneDue {
			labeled(c, t, m.say.FieldTitle(), func() {
				wellInput(c, t, "milestone-title", &s.title, m.say.FieldTitle(), m.say.ManageMilestonePlaceholder(), theme.Body, true, m.milestoneSheetKey)
			})
		}
		if s.mode != milestoneRename {
			labeled(c, t, m.say.PropDue(), func() { m.milestoneDueField(c, t) })
		}
		primary := m.say.ManageSave()
		if s.mode == milestoneNew {
			primary = m.say.ManageCreate()
		}
		cancelled, chosen := sheetButtons(c, m.say.ManageCancel(), primary, s.mode == milestoneDue || strings.TrimSpace(s.title) != "", false)
		if cancelled {
			s.open = false
		}
		if chosen {
			m.saveMilestone()
		}
	})
}

func (m *Main) milestoneSheetTitle() string {
	s := m.manage.milestone
	switch s.mode {
	case milestoneRename:
		return m.say.ManageMilestoneRenameTitle()
	case milestoneDue:
		return m.say.ManageMilestoneDueTitle(s.name)
	}
	return m.say.ManageMilestoneNewTitle()
}

// milestoneSheetKey is the keyboard of the title field: Enter saves.
func (m *Main) milestoneSheetKey(k ui.TextKey) bool {
	if k.Mods != 0 || k.Key != ui.KeyEnter {
		return false
	}
	m.saveMilestone()
	return true
}

// milestoneDueField is the day, or "No due date", in a well; it opens a calendar.
func (m *Main) milestoneDueField(c *ui.Context, t *theme.Tokens) {
	s := &m.manage.milestone
	text, ink := m.say.ManageMilestoneNoDue(), t.TextTertiary
	if s.hasDue {
		text, ink = m.clock.Date(s.due, m.app.Now()), t.Text
	}
	btn := ui.ButtonBase(c.Key("milestone-due")).Height(30).Padding(0, 10).Radius(theme.RadiusControl).Background(t.BgSubtle).
		AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorPointer).Label(m.say.A11yDate(m.say.PropDue(), text))
	btn.Children(func() { theme.Text(c, theme.Body, ink, text) })
	if btn.Clicked() {
		s.dueOpen = !s.dueOpen
	}
	ui.Popover(c, btn, &s.dueOpen, func() {
		ui.Column(c).Padding(8).Gap(8).Children(func() {
			if ui.Calendar(c, &s.due).Label(m.say.PickerDay()).Changed() {
				s.hasDue = true
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, m.say.PickerClear()).Clicked() {
					s.hasDue, s.dueOpen = false, false
				}
				if ui.PrimaryButton(c, m.say.PickerDone()).Clicked() {
					s.hasDue, s.dueOpen = true, false
				}
			})
		})
	})
}

// saveMilestone writes what the sheet holds and closes it. A due date is a day, at midnight.
func (m *Main) saveMilestone() {
	s := &m.manage.milestone
	title := strings.TrimSpace(s.title)
	if s.mode != milestoneDue && title == "" {
		return
	}
	y, mo, d := s.due.Date()
	day := time.Date(y, mo, d, 0, 0, 0, 0, m.app.Now().Location())
	var err error
	switch s.mode {
	case milestoneNew:
		var due *time.Time
		if s.hasDue {
			due = &day
		}
		err = m.app.CreateMilestone(s.project, title, due)
	case milestoneRename:
		err = m.app.UpdateMilestone(s.id, data.MilestonePatch{Title: &title})
	case milestoneDue:
		patch := data.MilestonePatch{DueAt: data.TimePtr(time.Time{})}
		if s.hasDue {
			patch.DueAt = &day
		}
		err = m.app.UpdateMilestone(s.id, patch)
	}
	_ = err // a failure is in the banner
	s.open = false
}
