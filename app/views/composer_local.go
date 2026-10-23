package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/quickadd"
)

// The no-model path of the composer: with no assistant to read the sentence, local rules read the
// dates, the reminder, the #category and the ! in it (internal/quickadd) and split it at the
// explicit separators. Each task they find is added at once; the receipt says what was added and
// Undo (also Cmd+Z) takes all of them back as one change.

// quickCategories are the names a #tag may name: the categories as the app words them, and the
// stored English names of Work and Life too. ids finds the category by the lowercase name.
func (m *Main) quickCategories() (names []string, ids map[string]int) {
	ids = map[string]int{}
	add := func(name string, id int) {
		if key := strings.ToLower(name); name != "" && ids[key] == 0 {
			names, ids[key] = append(names, name), id
		}
	}
	for _, c := range m.app.snap.categories {
		add(categoryTitle(m.say, c), c.ID)
		add(c.Name, c.ID)
	}
	return names, ids
}

// addPlain reads the text with the local rules and adds what it names in the place of the page: its
// project, else the Inbox. The tasks the page would get anyway (the day of the calendar, the
// category of the page) are the defaults of the words that do not say otherwise.
func (m *Main) addPlain(text string) {
	cs, place := &m.cmp[surfacePage], m.placeOf(surfacePage)
	names, ids := m.quickCategories()
	now := m.app.Now()
	drafts := quickadd.Parse(text, now, names)
	if len(drafts) == 0 { // only separators: keep what was typed as the title
		drafts = []quickadd.Draft{{Title: strings.Join(strings.Fields(text), " ")}}
	}
	project := ""
	inputs := make([]data.CreateTaskInput, len(drafts))
	for i, d := range drafts {
		in := data.CreateTaskInput{Title: d.Title, Source: data.SourceUser, Category: place.Category, DueAt: d.Due, RemindAt: d.RemindAt}
		if d.Category != "" {
			in.Category = ids[strings.ToLower(d.Category)]
		}
		if d.Important {
			in.Priority = data.P1
		}
		if place.Kind == data.PlaceProject {
			in.ProjectID, project = data.IntPtr(place.ProjectID), m.app.projectName(&place.ProjectID)
		}
		inputs[i] = in
	}
	made, err := m.app.AddTasks(inputs)
	if err != nil {
		return
	}
	cs.buf, cs.noticeAt, cs.noticeText, cs.noticeLen = "", now, m.receipt(drafts, project), undoWithin
	if plain(drafts) {
		cs.noticeLen = noticeFor
	}
	cs.undoTicket = m.undos.offer(undoEntry{
		title: cs.noticeText, until: now.Add(cs.noticeLen),
		undo: func() { _, _ = m.app.DeleteTasks(made) },
	})
}

// plain reports whether the words gave one task and nothing but its title.
func plain(drafts []quickadd.Draft) bool {
	d := drafts[0]
	return len(drafts) == 1 && d.Due == nil && d.RemindAt == nil && d.Category == "" && !d.Important
}

// receipt is the line that says what was added: "Added to Inbox" for a plain task, else what the
// words were taken to mean, "Added 2: 体检 (Mon 9:00 AM reminder), 交房租 (due Fri)".
func (m *Main) receipt(drafts []quickadd.Draft, project string) string {
	if plain(drafts) {
		return m.say.DetailAdded(m.inboxOr(project))
	}
	now := m.app.Now()
	var items []string
	for _, d := range drafts {
		var details []string
		if d.RemindAt != nil {
			details = append(details, m.say.QuickAddReminder(m.clock.When(*d.RemindAt, now)))
		}
		if d.Due != nil {
			details = append(details, m.say.QuickAddDue(m.receiptDue(*d.Due, now)))
		}
		if d.Category != "" {
			details = append(details, d.Category)
		}
		if d.Important {
			details = append(details, m.say.QuickAddImportant())
		}
		items = append(items, m.receiptItem(d.Title, details))
	}
	return m.say.QuickAddReceipt(len(drafts), m.receiptJoin(items))
}

// receiptDue is a due time in a receipt: the day alone for the end of it, else the day and the time.
func (m *Main) receiptDue(due, now time.Time) string {
	if due.Hour() == data.DefaultDueHour && due.Minute() == 0 {
		return m.clock.DayShort(due, now)
	}
	return m.clock.When(due, now)
}

// receiptItem is a title with the details of what was read from the words, if any.
func (m *Main) receiptItem(title string, details []string) string {
	if len(details) == 0 {
		return title
	}
	return m.say.QuickAddItem(title, m.receiptJoin(details))
}

// receiptJoin lists the parts the way the language does.
func (m *Main) receiptJoin(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out = m.say.QuickAddJoin(out, p)
	}
	return out
}
