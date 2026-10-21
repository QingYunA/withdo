package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// showsPriorityMark says whether a row or a card draws something for the task's priority: the star
// of a task of the user's own that is important, the P0 or P1 badge of an agent's.
func showsPriorityMark(task data.Task) bool {
	if mine(task) {
		return starred(task)
	}
	return theme.ShowsPriority(task.Priority)
}

// priorityMark draws it: a 12 pt star in status.review for the user's own task, instead of a badge
// (v31), or the badge. Nothing for a priority that is not shown.
func priorityMark(c *ui.Context, t *theme.Tokens, task data.Task) {
	switch {
	case !showsPriorityMark(task):
	case mine(task):
		theme.Star(c, t, true, theme.IconSmall)
	default:
		theme.PriorityBadge(c, t, task.Priority)
	}
}
