package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// A task marked done keeps its old place for a moment while its glyph fills and checks, then
// fades out of the active list. The store already says done; the App keeps a snapshot of the
// task as it was so lists can show it until the animation ends.
const (
	completeFill  = theme.MotionSpring // glyph fill and check
	completeHold  = 120 * time.Millisecond
	completeFade  = 180 * time.Millisecond
	completeTotal = completeFill + completeHold + completeFade
)

type completion struct {
	task data.Task // as it was before it was marked done
	at   time.Time
}

func (a *App) startCompletion(t data.Task) {
	a.completing[t.ID] = completion{task: t, at: a.Now()}
}

// pruneCompletions forgets animations that have finished.
func (a *App) pruneCompletions() {
	now := a.Now()
	for id, c := range a.completing {
		if now.Sub(c.at) >= completeTotal {
			delete(a.completing, id)
		}
	}
}

// completionPhase says how far a task's completion animation is: fill is 0..1 for the glyph, fade
// is the opacity the row should have. ok is false when the task is not completing.
func (a *App) completionPhase(c *ui.Context, id int) (fill, opacity float32, ok bool) {
	comp, found := a.completing[id]
	if !found {
		return 0, 1, false
	}
	age := a.Now().Sub(comp.at)
	if age >= completeTotal {
		return 1, 0, false
	}
	reduce := c.Preferences().ReduceMotion
	fill = float32(age) / float32(completeFill)
	if fill > 1 || reduce {
		fill = 1
	}
	opacity = 1
	if fadeAge := age - completeFill - completeHold; fadeAge > 0 {
		opacity = 1 - float32(fadeAge)/float32(completeFade)
	}
	c.After(16 * time.Millisecond)
	return fill, max(opacity, 0), true
}

// withCompleting returns tasks as lists should show them: tasks that are completing stay in the
// place they had, as they were.
func (a *App) withCompleting(tasks []data.Task, keep func(data.Task) bool) []data.Task {
	a.pruneCompletions()
	if len(a.completing) == 0 {
		return tasks
	}
	out := make([]data.Task, 0, len(tasks)+len(a.completing))
	seen := map[int]bool{}
	for _, t := range tasks {
		if comp, ok := a.completing[t.ID]; ok && keep(comp.task) {
			t = comp.task
		}
		seen[t.ID] = true
		out = append(out, t)
	}
	for id, comp := range a.completing {
		if !seen[id] && keep(comp.task) {
			out = append(out, comp.task)
		}
	}
	return out
}
