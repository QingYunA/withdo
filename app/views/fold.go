package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Folding is layout motion driven by hand, from the app clock, rather than by mygo's Exit
// transition: the fold has to follow the clock the tests and the screenshots set, keep the rows
// under it sliding in step with no Transition of their own, and let a section close with its last
// row. A box clips its content to a share of the height the content laid out at the frame before;
// the content itself is never laid out at another size, so nothing reflows while it closes.

// foldKey names the box around a row. A task leaving a list and the same task opening in Done can
// share a parent, so the two take keys of their own.
func foldKey(id int, done bool) string {
	if done {
		return "fold-done-" + itoa(id)
	}
	return "fold-" + itoa(id)
}

// closeTo clips box to share of the height inner has, where share is 1 for the whole of it. Before
// inner has been laid out its height is not known: a box that opens stays shut, one that closes is
// left whole for a frame.
func closeTo(box, inner ui.Element, share float32) {
	if share >= 1 {
		return
	}
	switch natural := inner.Bounds().H; {
	case natural > 0:
		box.MaxHeight(natural * share).ClipY()
	case share <= 0:
		box.MaxHeight(0).ClipY()
	}
}

// foldBox draws body in a box that is share of its height.
func foldBox(c *ui.Context, key string, share float32, body func()) {
	box := ui.Column(c.Key(key)).Shrink(0)
	var inner ui.Element
	box.Children(func() { inner = ui.Column(c).Shrink(0).Children(body) })
	closeTo(box, inner, share)
}

// rowPhase is completionPhase for a row drawn now. A row inside a section that folds as one keeps
// its height: the section closes, the rows only fade.
func (m *Main) rowPhase(c *ui.Context, task data.Task) phase {
	ph := m.app.completionPhase(c, task)
	if m.owned[foldKey(task.ID, ph.done)] {
		ph.height = 1
	}
	return ph
}

// wholeShare says how much of a section is left when every one of its tasks is in a completion
// (all leaving, or all opening): the least of their shares, so the header and the box close with the
// last row. A section with any other row folds row by row, and its share is 1.
func (m *Main) wholeShare(c *ui.Context, tasks []data.Task) (share float32, whole bool) {
	share = 1
	for _, task := range tasks {
		ph := m.app.completionPhase(c, task)
		if !ph.moving {
			return 1, false
		}
		share = min(share, ph.height)
	}
	return share, len(tasks) > 0
}

// section draws body, a section's header and rows for tasks, in a box that closes when every one of
// the tasks is leaving and opens when every one is opening. key names the box.
func (m *Main) section(c *ui.Context, key string, tasks []data.Task, body func()) {
	share, whole := m.wholeShare(c, tasks)
	if whole {
		if m.owned == nil {
			m.owned = map[string]bool{}
		}
		for _, task := range tasks {
			m.owned[foldKey(task.ID, task.Status == data.Done)] = true
		}
		defer clear(m.owned)
	}
	foldBox(c, key, share, body)
}

// itemTasks are the tasks of a section's items.
func itemTasks(items []data.Item) []data.Task {
	out := make([]data.Task, len(items))
	for i, it := range items {
		out[i] = it.Task
	}
	return out
}

// ruleShares says how much of the hairline above each row is left: it closes with the row it
// is above, and, when every row above it is leaving, with them, so no line is left standing at the
// top of the box.
func ruleShares(phs []phase) []float32 {
	out := make([]float32, len(phs))
	lead, leadShare := true, float32(1)
	for i, ph := range phs {
		out[i] = 1
		switch {
		case ph.moving:
			out[i] = ph.height
			leadShare = min(leadShare, ph.height)
		case lead:
			out[i] = leadShare
			lead = false
		default:
			lead = false
		}
	}
	return out
}
