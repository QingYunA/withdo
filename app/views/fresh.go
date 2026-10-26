package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The row a task the input just made has stays tinted (accent.soft) for freshFor, then fades out in
// freshFade (Craft rule 7: ease-out, 250 ms).
const (
	freshFor  = 2 * time.Second
	freshFade = 250 * time.Millisecond
)

// freshen notes the tasks the input just made, so their rows show it.
func (m *Main) freshen(ids []int) {
	if m.fresh == nil {
		m.fresh = map[int]time.Time{}
	}
	for _, id := range ids {
		m.fresh[id] = m.app.Now()
	}
}

// freshReceipts is freshen for what the assistant added; when it added a task with no date and no
// project, the No date fold opens so the row shows.
func (m *Main) freshReceipts(receipts []data.Receipt) {
	for _, r := range receipts {
		if r.Detail.Kind != data.DetailAdded {
			continue
		}
		m.freshen([]int{r.TaskID})
		if task, ok := m.app.Task(r.TaskID); ok && mine(task) && task.ProjectID == nil && task.DueAt == nil && task.RemindAt == nil {
			m.app.noDateOpen = true
		}
	}
	m.app.Refresh()
}

// freshShare is how much of the tint a row has now, 1 down to 0, and whether the row needs a frame
// at the time the tint ends. A task not made by the input has none.
func (m *Main) freshShare(id int) (share float32, until time.Duration) {
	at, ok := m.fresh[id]
	if !ok {
		return 0, 0
	}
	age := m.app.Now().Sub(at)
	switch {
	case age < freshFor:
		return 1, freshFor - age
	case age < freshFor+freshFade:
		left := float32(freshFor+freshFade-age) / float32(freshFade)
		return left * left, freshFor + freshFade - age // ease-out: quick at first, slow at the end
	}
	delete(m.fresh, id)
	return 0, 0
}
