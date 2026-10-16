package views

import (
	"log"

	"github.com/QingYunA/adhd-todo/app/data"
)

// dismissedSet is what the user put aside of the handed-over rows, loaded from the store on first use.
func (a *App) dismissedSet() data.Seen {
	if !a.dismissedLoaded {
		a.dismissedLoaded, a.dismissed = true, data.Seen{}
		if a.Dismissed != nil {
			if s, err := a.Dismissed.LoadSeen(); err != nil {
				log.Printf("dismissed: %v", err) // every handed-over row shows again once
			} else {
				a.dismissed = s
			}
		}
	}
	return a.dismissed
}

// dismiss puts a handed-over row aside until its task is handed over again, and forgets tasks that
// are gone. The task itself is not changed.
func (a *App) dismiss(it data.Item) {
	alive := func(id int) bool {
		_, ok := a.Task(id)
		return ok || !a.snap.loaded
	}
	next, changed := a.dismissedSet().MarkSeen([]data.Item{it}, alive)
	if !changed {
		return
	}
	a.dismissed = next
	if a.Dismissed != nil {
		if err := a.Dismissed.SaveSeen(next); err != nil {
			log.Printf("dismissed: %v", err)
		}
	}
}
