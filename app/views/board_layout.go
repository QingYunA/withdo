package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
)

// columnPlan is how one board column is drawn: in full at width, or as a narrow strip.
type columnPlan struct {
	expanded bool
	width    float32
}

// collapseOrder is the order in which columns give way when the board is too narrow: what the
// user parks or has not started goes first, what needs attention stays.
var collapseOrder = []data.Status{data.Backlog, data.Todo, data.Review, data.Doing, data.Blocked}

// planColumns decides which of the board columns are drawn in full. A column is a strip when it is
// empty, or is Done, unless it is forced open (the user opened it, a task of it is open in the
// inspector, or a task is being added to it). When the full columns do not fit in avail points
// they narrow, down to theme.MinColumnWidth, and then give way in collapseOrder. Forced columns
// never give way; if they alone overflow, the board scrolls.
func planColumns(avail float32, counts map[data.Status]int, forced map[data.Status]bool) map[data.Status]columnPlan {
	open := map[data.Status]bool{}
	for _, s := range data.BoardStatuses {
		open[s] = forced[s] || (s != data.Done && counts[s] > 0)
	}
	room, n := boardRoom(avail, open)
	for n > 0 && room < float32(n)*theme.MinColumnWidth && collapseOne(open, forced) {
		room, n = boardRoom(avail, open)
	}
	width := theme.ColumnWidth
	if n > 0 {
		width = min(max(room/float32(n), theme.MinColumnWidth), theme.ColumnWidth)
	}
	plans := make(map[data.Status]columnPlan, len(open))
	for s, isOpen := range open {
		plans[s] = columnPlan{expanded: isOpen, width: width}
	}
	return plans
}

// boardRoom is the width left for the n open columns once padding, gaps and strips are paid for.
func boardRoom(avail float32, open map[data.Status]bool) (room float32, n int) {
	for _, s := range data.BoardStatuses {
		if open[s] {
			n++
		}
	}
	strips := float32(len(data.BoardStatuses) - n)
	gaps := float32(len(data.BoardStatuses)-1) * theme.ColumnGap
	return avail - 2*theme.BoardPadding - gaps - strips*theme.StripWidth, n
}

// collapseOne turns the least important open column that is not forced into a strip.
func collapseOne(open, forced map[data.Status]bool) bool {
	for _, s := range collapseOrder {
		if open[s] && !forced[s] {
			open[s] = false
			return true
		}
	}
	return false
}

// boardWidth is the room the board has: the window less the sidebar and, when a task is open, the
// inspector, each with its hairline.
func boardWidth(window float32, inspectorOpen bool) float32 {
	w := window - theme.SidebarWidth - 1
	if inspectorOpen {
		w -= theme.InspectorWidth + 1
	}
	return w
}
