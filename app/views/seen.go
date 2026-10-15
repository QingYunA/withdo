package views

import (
	"log"

	"github.com/QingYunA/adhd-todo/app/data"
)

// seenSet is what the user has seen of the handed-over rows, loaded from the store on first use.
func (a *App) seenSet() data.Seen {
	if !a.seenLoaded {
		a.seenLoaded, a.seen = true, data.Seen{}
		if a.Seen != nil {
			if s, err := a.Seen.LoadSeen(); err != nil {
				log.Printf("seen: %v", err) // every handed-over row counts as new once
			} else {
				a.seen = s
			}
		}
	}
	return a.seen
}

// markSeen records that the user saw these handed-over rows, and forgets tasks that are gone. It
// writes only when something changed.
func (a *App) markSeen(items []data.Item) {
	alive := func(id int) bool {
		_, ok := a.Task(id)
		return ok || !a.snap.loaded
	}
	next, changed := a.seenSet().MarkSeen(items, alive)
	if !changed {
		return
	}
	a.seen = next
	if a.Seen != nil {
		if err := a.Seen.SaveSeen(next); err != nil {
			log.Printf("seen: %v", err)
		}
	}
}
