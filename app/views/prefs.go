package views

import (
	"log"
	"time"

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

// SoundEffects says whether the Summary scene makes sounds (Settings > General), which it does until
// the user switches them off. The cues ask from timer goroutines.
func (a *App) SoundEffects() bool {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.loadPrefs()
	return !a.prefs.SoundOff
}

// SetSoundEffects switches the Summary scene's sounds, and keeps the choice.
func (a *App) SetSoundEffects(on bool) {
	a.savePrefs(func(p *data.Prefs) { p.SoundOff = !on })
}

// EndOfDayOn says whether the end-of-day reminder is on (Settings > General > Notifications), which
// it is until the user switches it off. The reminder asks from its own goroutine.
func (a *App) EndOfDayOn() bool {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.loadPrefs()
	return !a.prefs.EndOfDayOff
}

// SetEndOfDay switches the end-of-day reminder, and keeps the choice.
func (a *App) SetEndOfDay(on bool) {
	a.savePrefs(func(p *data.Prefs) { p.EndOfDayOff = !on })
}

// EndOfDayEnabled says whether the reminder may come: it is on, and so are the notifications.
func (a *App) EndOfDayEnabled() bool { return a.Notifications() && a.EndOfDayOn() }

// EndOfDayAt is when the reminder comes, as "15:04": the user's pick, else 18:30.
func (a *App) EndOfDayAt() string {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.loadPrefs()
	if _, _, ok := data.ClockOf(a.prefs.EndOfDayAt); ok {
		return a.prefs.EndOfDayAt
	}
	return data.DefaultEndOfDay
}

// SetEndOfDayAt picks when the reminder comes, as "15:04", and keeps the choice.
func (a *App) SetEndOfDayAt(at string) {
	if _, _, ok := data.ClockOf(at); ok {
		a.savePrefs(func(p *data.Prefs) { p.EndOfDayAt = at })
	}
}

// EndOfDayLast is the date the reminder last came or was left out.
func (a *App) EndOfDayLast() string {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.loadPrefs()
	return a.prefs.EndOfDayLast
}

// SetEndOfDayLast keeps the date the reminder last came or was left out.
func (a *App) SetEndOfDayLast(date string) {
	a.savePrefs(func(p *data.Prefs) { p.EndOfDayLast = date })
}

// WeekStart is the day the weeks start on, as the window last drew them; Sunday before it has drawn.
// The reminder asks from its own goroutine.
func (a *App) WeekStart() time.Weekday { return time.Weekday(a.weekStart.Load()) }

func (a *App) setWeekStart(d time.Weekday) { a.weekStart.Store(int32(d)) }
