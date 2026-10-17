package views

import (
	"log"

	"github.com/QingYunA/adhd-todo/app/data"
)

// loadPrefs reads the saved prefs the first time; a file that cannot be read leaves the defaults.
// a.prefsMu is held.
func (a *App) loadPrefs() {
	if a.prefsLoaded {
		return
	}
	a.prefsLoaded = true
	if a.Prefs == nil {
		return
	}
	if p, err := a.Prefs.LoadPrefs(); err != nil {
		log.Printf("prefs: %v", err) // the defaults apply
	} else {
		a.prefs = p
	}
}

// savePrefs changes the prefs with set and keeps them. It loads first, so that nothing else in the
// file is lost.
func (a *App) savePrefs(set func(*data.Prefs)) {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.loadPrefs()
	set(&a.prefs)
	if a.Prefs != nil {
		if err := a.Prefs.SavePrefs(a.prefs); err != nil {
			log.Printf("prefs: %v", err)
		}
	}
}

// Notifications says whether system notifications are on (Settings > General), which they are
// until the user switches them off. The notifier asks from its own goroutine.
func (a *App) Notifications() bool {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.loadPrefs()
	return !a.prefs.NotificationsOff
}

// SetNotifications switches the system notifications, and keeps the choice.
func (a *App) SetNotifications(on bool) {
	a.savePrefs(func(p *data.Prefs) { p.NotificationsOff = !on })
}
