package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// A task marked done keeps its old place for a moment, then folds out of the active list while it
// opens at the top of Done. The store already says done; the App keeps a snapshot of the task as it
// was so lists can show it until the animation ends.
//
// Accept (see theme.AcceptFoldAt): the button stamps a pixel check, the row holds, then folds. Any
// other way (the status glyph, Space, the task menu) fills the glyph first, holds the same beat,
// then folds the same way. All of it is a function of the time since the click.
const (
	completeFill = theme.MotionSpring // glyph fill and check
	completeHold = theme.StampHold
)

// completionKind is how a task was marked done.
type completionKind int

const (
	byGlyph  completionKind = iota // the glyph, Space or the task menu
	byAccept                       // the Accept button of a row or a card
)

// foldAt is when the row begins to fold, counted from the click.
func (k completionKind) foldAt() time.Duration {
	if k == byAccept {
		return theme.AcceptFoldAt
	}
	return completeFill + completeHold
}

// total is from the click to the end of the fold.
func (k completionKind) total() time.Duration { return k.foldAt() + theme.FoldDuration }

type completion struct {
	task data.Task // as it was before it was marked done
	at   time.Time
	kind completionKind
}

func (a *App) startCompletion(t data.Task, kind completionKind) {
	a.completing[t.ID] = completion{task: t, at: a.Now(), kind: kind}
}

// pruneCompletions forgets animations that have finished.
func (a *App) pruneCompletions() {
	now := a.Now()
	for id, c := range a.completing {
		if now.Sub(c.at) >= c.kind.total() {
			delete(a.completing, id)
		}
	}
}

// phase is where a row stands in its task's completion. The zero value is not a phase: take it from
// completionPhase, whose rest state is a whole, opaque row.
type phase struct {
	fill    float32 // the glyph filling, 0 to 1
	stamp   int     // the frame of the Accept stamp, -1 while the task is not being accepted
	opacity float32
	height  float32 // the share of the row's height left: 1 whole, 0 folded away
	moving  bool    // the task is in a completion, as the row that leaves or the row that opens
	done    bool    // the row shows the task done: it is the one that opens
}

// completionPhase says where task's completion is for the row that shows it as it is. The row that
// shows the task as it was leaves, folding after its stamp or fill; the row that shows it done, in
// the list it joins, opens once the first begins to fold.
func (a *App) completionPhase(c *ui.Context, task data.Task) phase {
	rest := phase{stamp: -1, opacity: 1, height: 1, done: task.Status == data.Done}
	comp, found := a.completing[task.ID]
	if !found {
		return rest
	}
	age, foldAt := a.Now().Sub(comp.at), comp.kind.foldAt()
	if age >= comp.kind.total() || (rest.done && age < foldAt) {
		return rest
	}
	reduce := c.Preferences().ReduceMotion
	c.After(16 * time.Millisecond)
	ph := rest
	ph.moving = true
	if rest.done {
		ph.fill = 1
		ph.height, ph.opacity = theme.OpenAt(age-foldAt, reduce)
		return ph
	}
	ph.fill = float32(age) / float32(completeFill)
	if ph.fill > 1 || reduce {
		ph.fill = 1
	}
	if comp.kind == byAccept {
		ph.stamp = theme.StampFrameAt(age, reduce)
	}
	if age >= foldAt {
		ph.height, ph.opacity = theme.FoldAt(age-foldAt, reduce)
	}
	return ph
}

// withCompleting returns tasks as lists should show them: a task that is completing stays in the
// place it had, as it was, until its fold ends; from the beginning of the fold the task as it is now
// is there too, for the row that opens in Done.
func (a *App) withCompleting(tasks []data.Task, keep func(data.Task) bool) []data.Task {
	return a.shownWithCompleting(tasks, keep, false)
}

// withCompletingCards is withCompleting for the board, whose cards leave and arrive with the
// lanes' own motion (cardMotion). A card that is accepted stays as it was until its stamp has had
// its beat; any other card is moved at once, for it has no glyph to fill.
func (a *App) withCompletingCards(tasks []data.Task, keep func(data.Task) bool) []data.Task {
	return a.shownWithCompleting(tasks, keep, true)
}

func (a *App) shownWithCompleting(tasks []data.Task, keep func(data.Task) bool, cards bool) []data.Task {
	a.pruneCompletions()
	if len(a.completing) == 0 {
		return tasks
	}
	now := a.Now()
	out := make([]data.Task, 0, len(tasks)+len(a.completing))
	seen := map[int]bool{}
	for _, t := range tasks {
		seen[t.ID] = true
		comp, ok := a.completing[t.ID]
		if !ok || !keep(comp.task) {
			out = append(out, t)
			continue
		}
		leaves, opens := comp.shows(now, cards)
		if leaves {
			out = append(out, comp.task)
		}
		if opens {
			out = append(out, t)
		}
	}
	for id, comp := range a.completing {
		if leaves, _ := comp.shows(now, cards); !seen[id] && leaves && keep(comp.task) {
			out = append(out, comp.task)
		}
	}
	return out
}

// shows says which rows of the completion a list draws now: the task as it was, leaving, and the
// task as it is, opening in Done.
func (comp completion) shows(now time.Time, cards bool) (leaves, opens bool) {
	age, foldAt := now.Sub(comp.at), comp.kind.foldAt()
	if cards {
		leaves = comp.kind == byAccept && age < foldAt
		return leaves, !leaves
	}
	return age < comp.kind.total(), age >= foldAt
}
