package views

import (
	"strconv"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// filedFor is how long the quiet toast of an auto-filed task shows. It is shorter than undoWithin:
// it only tells, and Undo is there for the rare miss.
const filedFor = 4 * time.Second

// noteFiled remembers the new tasks whose category the assistant chose itself, for offerFiled to
// say so once the window has a context to put a toast in.
func (m *Main) noteFiled(receipts []data.Receipt) {
	for _, r := range receipts {
		if r.Detail.AutoFiled && r.Detail.Category != "" {
			m.filed = append(m.filed, r)
		}
	}
}

// offerFiled shows the toast of each task noted by noteFiled: "Filed under Life", with Undo, which
// clears the category so the task is unsorted again.
func (m *Main) offerFiled(c *ui.Context) {
	for _, r := range m.filed {
		id := r.TaskID
		c.AddToast(ui.Toast{
			ID:    "filed-" + strconv.Itoa(id),
			Title: m.say.TaskFiled(categoryLabel(m.say, r.Detail.Category)), Action: m.say.MenuUndo(), Timeout: filedFor,
			OnAction: func() { _ = m.app.Update(id, data.Patch{Category: data.IntPtr(0)}) },
		})
	}
	m.filed = nil
}
