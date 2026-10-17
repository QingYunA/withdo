package agent

import (
	"fmt"
	"slices"
)

// checkImages refuses a picture number the request does not have, so the model can read why and
// try again.
func (tb *toolbox) checkImages(given []int) error {
	for _, i := range given {
		if i < 0 || i >= tb.images {
			if tb.images == 0 {
				return fmt.Errorf("images: this request has no pictures")
			}
			return fmt.Errorf("images: picture %d does not exist; this request has pictures 0 to %d", i, tb.images-1)
		}
	}
	return nil
}

// keep records that a task keeps pictures of this turn: the numbers given, or every picture when
// given is nil. A task named twice keeps the union.
func (tb *toolbox) keep(taskID int64, given []int) {
	if tb.images == 0 {
		return
	}
	if given == nil {
		given = make([]int, tb.images)
		for i := range given {
			given[i] = i
		}
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	at := slices.IndexFunc(tb.keeps, func(k ImageKeep) bool { return k.TaskID == taskID })
	if at < 0 {
		tb.keeps = append(tb.keeps, ImageKeep{TaskID: taskID})
		at = len(tb.keeps) - 1
	}
	all := append(slices.Clone(tb.keeps[at].Indexes), given...)
	slices.Sort(all)
	tb.keeps[at].Indexes = slices.Compact(all)
}

// kept is what each task keeps, in the order the tasks were first touched; a task that keeps no
// picture (an empty list) is left out.
func (tb *toolbox) kept() []ImageKeep {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	var out []ImageKeep
	for _, k := range tb.keeps {
		if len(k.Indexes) > 0 {
			out = append(out, ImageKeep{TaskID: k.TaskID, Indexes: slices.Clone(k.Indexes)})
		}
	}
	return out
}
