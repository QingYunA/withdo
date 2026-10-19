package views

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// Dragging a stack (docs/design/ui-spec.md, Selecting several tasks): dragging one of the rows
// selected together takes all of them, in the order the page draws them. The row in the hand carries
// at most three cards fanned behind it and a count at its top right, the others fade where they
// were, and dropped on the tray they are handed over as one change. Dragging a row that is not
// selected takes only that row and clears the selection.

// stackFade is how faint the other rows of a stack are while it is in the air.
const stackFade float32 = 0.5

// stackKey writes the rows of a stack into a drag's value, which must stay comparable.
func stackKey(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

// stackIDs reads the rows of a stack back from its key.
func stackIDs(key string) []int {
	var ids []int
	for _, part := range strings.Split(key, ",") {
		if id, err := strconv.Atoi(part); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// stackOf is the rows a drag of this task takes to the tray: when it is one of several selected, the
// user's unfinished ones among them, in page order; else only itself.
func (m *Main) stackOf(task data.Task) []int {
	if !m.multi.has(task.ID) || m.multi.count() < 2 {
		return []int{task.ID}
	}
	var ids []int
	for _, id := range m.selectedRows() {
		if t, ok := m.app.Task(id); ok && canHandOff(t) {
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 || !slices.Contains(ids, task.ID) {
		return []int{task.ID}
	}
	return ids
}

// startedDrag is called by the row that is being dragged: it says which rows are in the air, and
// drops the selection when the row dragged was not in it.
func (m *Main) startedDrag(id int, stack []int) {
	m.ho.stack, m.ho.stackSeen = stack, true
	if !m.multi.has(id) && m.multi.count() > 0 {
		m.clearMulti()
	}
}

// fadesInStack says whether the row is one of a stack in the air that is not the one in the hand.
func (m *Main) fadesInStack(id int) bool {
	return len(m.ho.stack) > 1 && slices.Contains(m.ho.stack, id)
}

// stackBadge is the count at the top right of the stack in the hand: accent, with white badge text.
// It is a child of the carried row, so it exists only while it is dragged.
func (m *Main) stackBadge(c *ui.Context, t *theme.Tokens) {
	n := len(m.ho.stack)
	if n < 2 {
		return
	}
	ui.Row(c.Key("stack-badge")).Absolute().Right(-6).Top(-9).Height(18).MinWidth(18).Padding(0, 5).Radius(9).
		Background(t.Accent).Center().Children(func() {
		theme.Text(c, theme.Badge, t.AccentText, strconv.Itoa(n)).Shrink(0)
	})
}

// stackPaint draws the cards of a stack of n in r, from the farthest back: up to two fanned behind,
// then the one in the hand. The hand's own card is turned 2 degrees by its lift, so the others are
// turned that much less than -4 and -2 degrees; with Reduce Motion there is no lift.
func stackPaint(p *ui.Painter, r ui.Rect, t *theme.Tokens, n int, radius float32, lifted bool) {
	angles := []float64{-4, -2}
	if n == 2 {
		angles = []float64{-2}
	}
	for i, deg := range angles {
		if lifted {
			deg -= 2
		}
		// Each card further back also sits a little lower, so the stack reads as a stack.
		below := r
		below.Y += 3 * float32(len(angles)-i)
		path := chamfered(below, deg, 4)
		p.FillPath(path, t.Bg)
		p.StrokePath(path, 1, t.TextTertiary)
	}
	p.Fill(r, t.Bg, radius)
	p.Stroke(r, t.TextTertiary, radius, 1)
}

// chamfered is r turned deg degrees about its middle, its corners cut by k.
func chamfered(r ui.Rect, deg float64, k float32) *ui.Path {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	hw, hh := r.W/2, r.H/2
	pts := [][2]float32{
		{-hw + k, -hh}, {hw - k, -hh}, {hw, -hh + k}, {hw, hh - k},
		{hw - k, hh}, {-hw + k, hh}, {-hw, hh - k}, {-hw, -hh + k},
	}
	sin, cos := math.Sincos(deg * math.Pi / 180)
	s, c := float32(sin), float32(cos)
	path := &ui.Path{}
	for i, pt := range pts {
		x, y := cx+pt[0]*c-pt[1]*s, cy+pt[0]*s+pt[1]*c
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	return path.Close()
}
