package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
)

func mkTask(id int, s data.Status, src data.Source) data.Task {
	return data.Task{ID: id, Status: s, Source: src, Priority: data.P2}
}

func actionKindsOf(acts []taskAction) []actionKind {
	var out []actionKind
	for _, a := range acts {
		out = append(out, a.Kind)
	}
	return out
}

func TestActionsForListsTheActionsInOrderAndNoneForNoTasks(t *testing.T) {
	if got := actionsFor(nil); got != nil {
		t.Errorf("actions for no tasks = %v", got)
	}
	acts := actionsFor([]data.Task{mkTask(1, data.Todo, data.SourceUser)})
	want := []actionKind{actDone, actCancel, actMove, actPriority, actHandOff, actDelete}
	if got := actionKindsOf(acts); !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	last := acts[len(acts)-1]
	if !last.Danger {
		t.Error("Delete is not marked as danger")
	}
	for _, a := range acts[:len(acts)-1] {
		if a.Danger {
			t.Errorf("%s is marked as danger", a.Name)
		}
	}
}

func TestActionsApplyOnlyToTheTasksTheyChange(t *testing.T) {
	parent := 1
	tasks := []data.Task{
		mkTask(1, data.Todo, data.SourceUser),
		mkTask(2, data.Done, data.SourceUser),
		mkTask(3, data.Todo, data.SourceAgent),
		mkTask(4, data.Cancelled, data.SourceUser),
		{ID: 5, Status: data.Todo, Source: data.SourceUser, ParentID: &parent},
	}
	ids := func(k actionKind) []int { return actionOf(tasks, k).Ids }
	cases := []struct {
		kind actionKind
		want []int
	}{
		{actDone, []int{1, 3, 4, 5}},
		{actCancel, []int{1, 2, 3, 5}},
		{actMove, []int{1, 2, 3, 4}}, // the subtask stays with its parent
		{actPriority, []int{1, 2, 3, 4, 5}},
		{actHandOff, []int{1, 5}}, // the user's unfinished ones
		{actDelete, []int{1, 2, 3, 4, 5}},
	}
	for _, c := range cases {
		if got := ids(c.kind); !slices.Equal(got, c.want) {
			t.Errorf("action %d applies to %v, want %v", c.kind, got, c.want)
		}
	}
}

func TestAnActionWithNothingToDoIsNotEnabled(t *testing.T) {
	agents := []data.Task{mkTask(1, data.Todo, data.SourceAgent), mkTask(2, data.Review, data.SourceAgent)}
	if actionOf(agents, actHandOff).Enabled() {
		t.Error("Hand off is enabled for agent tasks")
	}
	done := []data.Task{mkTask(1, data.Done, data.SourceUser)}
	if actionOf(done, actDone).Enabled() {
		t.Error("Done is enabled for a done task")
	}
	if !actionOf(done, actDelete).Enabled() {
		t.Error("Delete is not enabled")
	}
	if (taskAction{}).Enabled() {
		t.Error("the zero action is enabled")
	}
}

func TestOnlyCancelIsMissingFromTheBar(t *testing.T) {
	var bar []actionKind
	for _, a := range actionsFor([]data.Task{mkTask(1, data.Todo, data.SourceUser)}) {
		if a.OnBar() {
			bar = append(bar, a.Kind)
		}
	}
	if want := []actionKind{actDone, actMove, actPriority, actHandOff, actDelete}; !slices.Equal(bar, want) {
		t.Errorf("bar = %v, want %v", bar, want)
	}
}

func TestMovableSkipsTasksAlreadyThereAndSubtasks(t *testing.T) {
	p1, p2, parent := 1, 2, 9
	tasks := []data.Task{
		{ID: 1, ProjectID: &p1},
		{ID: 2, ProjectID: &p2},
		{ID: 3},
		{ID: 4, ProjectID: &p2, ParentID: &parent},
	}
	cases := []struct {
		project int
		want    []int
	}{{1, []int{2, 3}}, {2, []int{1, 3}}, {0, []int{1, 2}}}
	for _, c := range cases {
		if got := idsOf(movable(tasks, c.project)); !slices.Equal(got, c.want) {
			t.Errorf("movable to %d = %v, want %v", c.project, got, c.want)
		}
	}
	if !allIn(tasks[:1], 1) || allIn(tasks[:2], 1) || allIn(nil, 0) {
		t.Error("allIn is wrong")
	}
}

func TestStatusAndPriorityTargetsSkipTasksThatHaveItAlready(t *testing.T) {
	tasks := []data.Task{mkTask(1, data.Todo, data.SourceUser), mkTask(2, data.Done, data.SourceUser)}
	tasks[1].Priority = data.P0
	if got := idsOf(statusTargets(tasks, data.Done)); !slices.Equal(got, []int{1}) {
		t.Errorf("done targets = %v", got)
	}
	if got := idsOf(repriced(tasks, data.P0)); !slices.Equal(got, []int{1}) {
		t.Errorf("P0 targets = %v", got)
	}
	if !allPriority(tasks[1:], data.P0) || allPriority(tasks, data.P0) || allPriority(nil, data.P0) {
		t.Error("allPriority is wrong")
	}
}

func TestHandOffAndTakeBackAreForOpenTasksOfTheRightOwner(t *testing.T) {
	cases := []struct {
		task          data.Task
		hand, takeBck bool
	}{
		{mkTask(1, data.Todo, data.SourceUser), true, false},
		{mkTask(1, data.Done, data.SourceUser), false, false},
		{mkTask(1, data.Doing, data.SourceAgent), false, true},
		{mkTask(1, data.Cancelled, data.SourceAgent), false, false},
	}
	for _, c := range cases {
		if canHandOff(c.task) != c.hand || canTakeBack(c.task) != c.takeBck {
			t.Errorf("%s %s: hand %v, take back %v", c.task.Source, c.task.Status, canHandOff(c.task), canTakeBack(c.task))
		}
	}
}

func TestEveryKeyedActionShowsItsOwnKeyAsItsHint(t *testing.T) {
	want := map[actionKind]string{actDone: "⌘↵", actMove: "M", actPriority: "P", actHandOff: "⌘⇧A", actDelete: "⌘⌫", actCancel: ""}
	seen := map[binding]actionKind{}
	for _, a := range actionsFor([]data.Task{mkTask(1, data.Todo, data.SourceUser)}) {
		if a.Key.hint != want[a.Kind] {
			t.Errorf("%s hints %q, want %q", a.Name, a.Key.hint, want[a.Kind])
		}
		if a.Key == (binding{}) {
			continue
		}
		if other, dup := seen[a.Key]; dup {
			t.Errorf("%s and %d share a key", a.Name, other)
		}
		seen[a.Key] = a.Kind
	}
}

func TestPressedActionFindsTheActionOfTheKeyDownAmongTheKindsAsked(t *testing.T) {
	acts := actionsFor([]data.Task{mkTask(1, data.Todo, data.SourceUser)})
	only := func(want binding) func(binding) bool { return func(b binding) bool { return b == want } }
	if a, ok := pressedAction(acts, only(keyHandOff)); !ok || a.Kind != actHandOff {
		t.Errorf("hand-off key found %v %v", a.Kind, ok)
	}
	if _, ok := pressedAction(acts, only(keyHandOff), actDone, actMove); ok {
		t.Error("found an action outside the kinds asked for")
	}
	if _, ok := pressedAction(acts, only(binding{})); ok {
		t.Error("the zero binding found an action")
	}
	if _, ok := pressedAction(acts, func(binding) bool { return false }); ok {
		t.Error("found an action with no key down")
	}
}
