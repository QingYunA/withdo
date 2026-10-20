package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The task actions: what can be done to a task, or to the tasks selected together. The bulk bar, the
// task menus and the keys all read the same list from actionsFor, so which actions there are, in
// what order, which tasks each one applies to and which key does it are decided here and nowhere
// else. This file is pure (no Main, no Context): the views draw the list and run its actions.

// binding is a key and the hint that shows it, so the two cannot drift apart.
type binding struct {
	mods ui.Modifiers
	key  ui.Key
	hint string
}

// down says whether the key is pressed in the window.
func (b binding) down(c *ui.Context) bool { return c.Shortcut(b.mods, b.key) }

// inMenu says whether the key is pressed while the menu's panel has the keyboard.
func (b binding) inMenu(panel ui.Element) bool { return panel.OverlayShortcut(b.mods, b.key) }

var (
	keyStatus   = binding{0, ui.KeyS, "S"}
	keyPriority = binding{0, ui.KeyP, "P"}
	keyMove     = binding{0, ui.KeyM, "M"}
	keyRemind   = binding{0, ui.KeyR, "R"}
	keyAsk      = binding{ui.Cmd, ui.KeyJ, keyHint("⌘J")}
	keyDone     = binding{ui.Cmd, ui.KeyEnter, keyHint("⌘↵")}
	keyHandOff  = binding{ui.Cmd | ui.Shift, ui.KeyA, keyHint("⌘⇧A")}
	keyDelete   = binding{ui.Cmd, ui.KeyBackspace, keyHint("⌘⌫")}
)

// actionKind names an action.
type actionKind int

const (
	actDone actionKind = iota
	actCancel
	actMove
	actPriority
	actHandOff
	actDelete
)

// taskAction is one action offered for a set of tasks.
type taskAction struct {
	Kind   actionKind
	Name   string // what keys and tests call it
	Icon   *ui.SVG
	Key    binding  // the zero binding when the action has no key
	Page   menuPage // the menu page the action opens, pageMain when it runs at once
	Group  int      // actions of one group sit together; the menu rules off one group from the next
	Danger bool
	Ids    []int // the tasks it applies to, in order; none means it would do nothing
}

// Enabled says whether the action applies to any of the tasks.
func (a taskAction) Enabled() bool { return len(a.Ids) > 0 }

// OnBar says whether the bulk bar has a button for the action (Cancel is only in the menu).
func (a taskAction) OnBar() bool { return a.Kind != actCancel }

// actionsFor lists the actions for tasks in the order they are offered. Move and Priority name the
// tasks they could change; which of those a given project or priority changes is movable and
// repriced. There are none for no tasks.
func actionsFor(tasks []data.Task) []taskAction {
	if len(tasks) == 0 {
		return nil
	}
	return []taskAction{
		{Kind: actDone, Name: "done", Icon: theme.IconCheck, Key: keyDone, Ids: idsOf(statusTargets(tasks, data.Done))},
		{Kind: actCancel, Name: "cancel", Icon: theme.IconClose, Ids: idsOf(statusTargets(tasks, data.Cancelled))},
		{Kind: actMove, Name: "move", Icon: theme.IconMove, Key: keyMove, Page: pageMove, Group: 1, Ids: idsOf(only(tasks, func(t data.Task) bool { return !t.IsSubtask() }))},
		{Kind: actPriority, Name: "priority", Icon: theme.IconFlag, Key: keyPriority, Page: pagePriority, Group: 1, Ids: idsOf(tasks)},
		{Kind: actHandOff, Name: "hand", Icon: theme.IconAgent, Key: keyHandOff, Group: 1, Ids: idsOf(only(tasks, canHandOff))},
		{Kind: actDelete, Name: "delete", Icon: theme.IconTrash, Key: keyDelete, Group: 2, Danger: true, Ids: idsOf(tasks)},
	}
}

// actionOf is the action of the kind for tasks; the zero action, which applies to nothing, for none.
func actionOf(tasks []data.Task, kind actionKind) taskAction {
	for _, a := range actionsFor(tasks) {
		if a.Kind == kind {
			return a
		}
	}
	return taskAction{}
}

// pressedAction is the action among acts of the given kinds (any, with none named) whose key is
// down, as down judges it.
func pressedAction(acts []taskAction, down func(binding) bool, kinds ...actionKind) (taskAction, bool) {
	for _, a := range acts {
		if a.Key == (binding{}) || (len(kinds) > 0 && !containsKind(kinds, a.Kind)) {
			continue
		}
		if down(a.Key) {
			return a, true
		}
	}
	return taskAction{}, false
}

func containsKind(kinds []actionKind, k actionKind) bool {
	for _, x := range kinds {
		if x == k {
			return true
		}
	}
	return false
}

// ownOpen says whether the task is the user's own and unfinished: the one kind of task that can be
// marked done by its glyph and handed to an agent.
func ownOpen(t data.Task) bool { return t.Source == data.SourceUser && !t.Status.Terminal() }

// canHandOff says whether the task can be handed to an agent.
func canHandOff(t data.Task) bool { return ownOpen(t) }

// canTakeBack says whether the task is an agent's unfinished one, which the user can take back.
func canTakeBack(t data.Task) bool { return t.Source == data.SourceAgent && !t.Status.Terminal() }

// statusTargets are the tasks not in the status yet.
func statusTargets(tasks []data.Task, s data.Status) []data.Task {
	return only(tasks, func(t data.Task) bool { return t.Status != s })
}

// movable are the tasks that moving to the project (or the Inbox, for project 0) changes: a subtask
// stays with its parent, and a task that is there already is left alone.
func movable(tasks []data.Task, project int) []data.Task {
	return only(tasks, func(t data.Task) bool { return !t.IsSubtask() && !inProject(t, project) })
}

// repriced are the tasks that do not have the priority yet.
func repriced(tasks []data.Task, p data.Priority) []data.Task {
	return only(tasks, func(t data.Task) bool { return t.Priority != p })
}

// allIn says whether there are tasks and every one is in the project, or in the Inbox for project 0.
func allIn(tasks []data.Task, project int) bool {
	return len(tasks) > 0 && len(only(tasks, func(t data.Task) bool { return !inProject(t, project) })) == 0
}

// allPriority says whether there are tasks and every one has the priority.
func allPriority(tasks []data.Task, p data.Priority) bool {
	return len(tasks) > 0 && len(repriced(tasks, p)) == 0
}

// inProject says whether the task is in the project, or in the Inbox for project 0.
func inProject(t data.Task, project int) bool {
	return (t.ProjectID == nil && project == 0) || (t.ProjectID != nil && *t.ProjectID == project)
}

// only keeps the tasks keep accepts.
func only(tasks []data.Task, keep func(data.Task) bool) []data.Task {
	var out []data.Task
	for _, t := range tasks {
		if keep(t) {
			out = append(out, t)
		}
	}
	return out
}

// idsOf are the ids of tasks, in order.
func idsOf(tasks []data.Task) []int {
	out := make([]int, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
	}
	return out
}
