package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// markFor is how long a value the assistant set carries its ✦ in the inspector.
const markFor = 10 * time.Second

// markReceipts notes the fields of a task the assistant just set, from what its receipts say, so
// the inspector can mark them for a while.
func (m *Main) markReceipts(receipts []data.Receipt) {
	now := m.app.Now()
	for _, r := range receipts {
		fields := markFields(r.Detail)
		if len(fields) == 0 {
			continue
		}
		if m.marks == nil {
			m.marks = map[int]map[string]time.Time{}
		}
		if m.marks[r.TaskID] == nil {
			m.marks[r.TaskID] = map[string]time.Time{}
		}
		for _, f := range fields {
			m.marks[r.TaskID][f] = now
		}
	}
}

// unmark forgets the marks of a task whose change was undone.
func (m *Main) unmark(task int) { delete(m.marks, task) }

// markFields are the inspector rows (by key) a receipt's change set. A field that was cleared is
// not marked: there is no value to put a mark after.
func markFields(d data.Detail) []string {
	var out []string
	switch d.Kind {
	case data.DetailReminder:
		out = append(out, "remind")
		if !d.Due.IsZero() {
			out = append(out, "due")
		}
	case data.DetailDue:
		out = append(out, "due")
	case data.DetailDone:
		out = append(out, "status")
	case data.DetailUpdated:
		for _, c := range d.Changes {
			if f := changeField(c); f != "" {
				out = append(out, f)
			}
		}
	}
	if d.Priority != "" {
		out = append(out, "priority")
	}
	if d.Category != "" {
		out = append(out, "category")
	}
	return out
}

func changeField(c data.Change) string {
	switch c.Kind {
	case data.ChangeReminder:
		if !c.At.IsZero() {
			return "remind"
		}
	case data.ChangeDue:
		if !c.At.IsZero() {
			return "due"
		}
	case data.ChangeStatus:
		return "status"
	case data.ChangePriority:
		return "priority"
	case data.ChangeCategory:
		if c.Value != "" {
			return "category"
		}
	case data.ChangeProject:
		return "project"
	}
	return ""
}

// assistMark is the small accent ✦ after a value of the open task that the assistant set in the
// last 10 s.
func (m *Main) assistMark(c *ui.Context, t *theme.Tokens, key string) {
	at, ok := m.marks[m.inspector.lastOpen][key]
	if !ok {
		return
	}
	age := m.app.Now().Sub(at)
	if age >= markFor {
		return
	}
	c.After(markFor - age)
	label := m.say.MarkAssistant()
	ui.Row(c.Key("mark-"+key)).Shrink(0).Padding(0, 5).Radius(5).Background(t.BgSelected).AlignItems(ui.Center).
		Label(label).Tooltip(label).Children(func() {
		theme.Text(c, theme.MetaStrong, t.Accent, "✦")
	})
}
