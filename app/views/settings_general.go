package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// repoURL is the project on GitHub, the About row's link.
const repoURL = "https://github.com/QingYunA/adhd-todo"

// archivedRows is how many archived projects the pane shows before the list scrolls.
const archivedRows = 5

// settingsURLNotifications opens the Notifications pane of System Settings.
const settingsURLNotifications = "x-apple.systempreferences:com.apple.Notifications-Settings.extension"

// generalPane is the language, the notifications, the updates, the categories, the archived projects, and where the data lives.
func (s *Settings) generalPane(c *ui.Context, t *theme.Tokens) {
	group(c, t, "language", func() {
		settingsRow(c, t, "language", s.say.SettingsLanguage(), "", true, func() { s.languageControl(c, t) })
	})
	groupLabel(c, t, s.say.SettingsNotifications())
	group(c, t, "notifications", func() {
		s.notificationsRow(c, t)
	})
	if s.app.Act.CanUpdate() {
		group(c, t, "updates", func() { s.updatesRow(c, t) })
	}
	s.categoriesGroup(c, t)
	groupLabel(c, t, s.say.ManageArchivedTitle())
	group(c, t, "archived", func() { s.archivedRowsView(c, t) })
	group(c, t, "about", func() {
		settingsRow(c, t, "folder", s.say.SettingsDataFolder(), withTilde(s.app.Config.DataFolder, s.app.Config.Home), true, func() {
			if label := showInFolderLabel(s.say); settingsLink(c, t, "show-folder", label, label).Clicked() {
				s.app.Act.ShowInFinder(s.app.Config.DataFolder)
			}
		})
		settingsRow(c, t, "about", s.say.SettingsAboutName(s.app.Config.Version), s.say.SettingsAboutLicense(), false, func() {
			if settingsOutLink(c, t, "github", s.say.SettingsAboutLink(), s.say.A11yAboutGitHub()).Clicked() {
				s.app.Act.OpenURL(repoURL)
			}
		})
	})
}

// advancedPane is the model the user brings (the Assistant pane that was), then the Recent changes
// page switch, the end-of-day reminder and the sound effects.
func (s *Settings) advancedPane(c *ui.Context, t *theme.Tokens) {
	groupLabel(c, t, s.say.SettingsOwnModel())
	s.assistantPane(c, t)
	group(c, t, "pages", func() { s.recentChangesRow(c, t) })
	group(c, t, "end-of-day", func() { s.endOfDayRows(c, t) })
	group(c, t, "sound", func() { s.soundRow(c, t) })
}

// notificationsRow is the switch for every system notification. When macOS refused the last one the
// quiet line says so in the warning color, with a link to where it is allowed again.
func (s *Settings) notificationsRow(c *ui.Context, t *theme.Tokens) {
	on := s.app.Notifications()
	meta, ink := s.say.SettingsNotifyMeta(), t.TextSecondary
	denied := s.app.Act.NotificationsDenied()
	if denied {
		meta, ink = s.say.SettingsNotifyDenied(), t.Stale
	}
	settingsRowInk(c, t, "notify", s.say.SettingsNotifyTitle(), meta, ink, true, func() {
		ui.Row(c.Key("notify-controls")).Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
			if denied {
				link := s.say.SettingsNotifyOpenSettings()
				if settingsOutLink(c, t, "notify-open-settings", link, link).Clicked() {
					s.app.Act.OpenURL(settingsURLNotifications)
				}
			}
			ui.Switch(c.Key("notify-switch"), &on).Label(s.say.SettingsNotifySwitch())
		})
	}, nil)
	if on != s.app.Notifications() {
		s.app.SetNotifications(on)
	}
}

// endOfDayRows are the end-of-day reminder's switch and the time it comes. With the reminder off, or
// the notifications, the time is dimmed and out of reach.
func (s *Settings) endOfDayRows(c *ui.Context, t *theme.Tokens) {
	on := s.app.EndOfDayOn()
	settingsRow(c, t, "end-of-day", s.say.SettingsEndOfDay(), s.say.SettingsEndOfDayMeta(), true, func() {
		ui.Switch(c.Key("end-of-day-switch"), &on).Label(s.say.SettingsEndOfDaySwitch())
	})
	if on != s.app.EndOfDayOn() {
		s.app.SetEndOfDay(on)
	}
	active := on && s.app.Notifications()
	settingsRowLead(c, t, "end-of-day-time", s.say.SettingsEndOfDayTime(), "", false, func() {
		ui.Box(c.Key("end-of-day-time-pop")).Shrink(0).Opacity(dimmed(active)).Disabled(!active).Children(func() {
			if at, ok := popUp(c, t, "end-of-day-time", s.app.EndOfDayAt(), s.endOfDayTimes()); ok {
				s.app.SetEndOfDayAt(at)
			}
		})
	}, func() { ui.Box(c).Width(14).Shrink(0) }) // indented under the switch
}

// soundRow is the switch for the quiet sounds of the Summary scene.
func (s *Settings) soundRow(c *ui.Context, t *theme.Tokens) {
	on := s.app.SoundEffects()
	settingsRow(c, t, "sound", s.say.SettingsSoundEffects(), s.say.SettingsSoundEffectsMeta(), true, func() {
		ui.Switch(c.Key("sound-switch"), &on).Label(s.say.SettingsSoundEffectsSwitch())
	})
	if on != s.app.SoundEffects() {
		s.app.SetSoundEffects(on)
	}
}

// updatesRow is the switch for checking for updates in the background; the update plugin keeps it.
func (s *Settings) updatesRow(c *ui.Context, t *theme.Tokens) {
	on := s.app.Act.AutomaticUpdates()
	settingsRow(c, t, "updates", s.say.SettingsUpdates(), s.say.SettingsUpdatesMeta(), true, func() {
		ui.Switch(c.Key("updates-switch"), &on).Label(s.say.SettingsUpdatesSwitch())
	})
	if on != s.app.Act.AutomaticUpdates() {
		s.app.Act.SetAutomaticUpdates(on)
	}
}

// recentChangesRow is the switch of the Recent changes page, which is off until the user turns it on.
// Switched on with gh missing or signed out, its second line says what to run; the switch still turns.
func (s *Settings) recentChangesRow(c *ui.Context, t *theme.Tokens) {
	on := s.app.RecentChangesOn()
	meta, ink := s.say.SettingsRecentChangesMeta(), t.TextSecondary
	if on && s.app.GitHub != nil {
		switch s.app.GitHub.Status().State {
		case data.GitHubMissing:
			meta, ink = s.say.SettingsRecentNoGH(ghInstallCommand()), t.Stale
		case data.GitHubSignedOut:
			meta, ink = s.say.SettingsRecentSignedOut(), t.Stale
		}
	}
	settingsRowInk(c, t, "recent-changes", s.say.SettingsRecentChanges(), meta, ink, true, func() {
		ui.Switch(c.Key("recent-changes-switch"), &on).Label(s.say.SettingsRecentChangesSwitch())
	}, nil)
	if on != s.app.RecentChangesOn() {
		s.app.SetRecentChangesOn(on)
	}
}

// dimmed is the opacity of a control that is out of reach.
func dimmed(active bool) float32 {
	if active {
		return 1
	}
	return 0.45
}

// endOfDayTimes are the times the reminder can come: every half hour from four to nine in the
// evening, and the one picked when it is none of them.
func (s *Settings) endOfDayTimes() []popOption {
	now := s.app.Now()
	y, m, d := now.Date()
	var opts []popOption
	add := func(h, min int) {
		at := time.Date(y, m, d, h, min, 0, 0, now.Location())
		opts = append(opts, popOption{at.Format("15:04"), s.clock.Time(at)})
	}
	for h := 16; h <= 21; h++ {
		add(h, 0)
		if h < 21 {
			add(h, 30)
		}
	}
	picked := s.app.EndOfDayAt()
	for _, o := range opts {
		if o.value == picked {
			return opts
		}
	}
	if h, min, ok := data.ClockOf(picked); ok {
		add(h, min)
	}
	return opts
}

// languageControl chooses the language of the interface: the system's, or one the app is written
// in, each named in itself.
func (s *Settings) languageControl(c *ui.Context, t *theme.Tokens) {
	opts := []popOption{{"", s.say.LanguageSystem()}}
	for _, tag := range s.app.Act.Languages() {
		opts = append(opts, popOption{tag, i18n.Autonym(tag)})
	}
	if tag, ok := popUp(c, t, "language", s.app.Act.Language(), opts); ok {
		s.app.Act.SetLanguage(tag)
	}
}

// archivedRowsView lists the archived projects with a Restore link each, or says there are none.
func (s *Settings) archivedRowsView(c *ui.Context, t *theme.Tokens) {
	archived := s.app.snap.archived
	if len(archived) == 0 {
		settingsRow(c, t, "none", s.say.ManageArchivedNone(), "", true, nil)
		return
	}
	rows := func() {
		for i, p := range archived {
			settingsRow(c, t, "archived-"+itoa(p.ID), p.Name, p.RepoSlug, i == 0, func() {
				if settingsLink(c, t, "restore-"+itoa(p.ID), s.say.ManageRestore(), s.say.ManageA11yRestore(p.Name)).Clicked() {
					_ = s.app.RestoreProject(p.ID)
				}
			})
		}
	}
	if len(archived) <= archivedRows {
		rows()
		return
	}
	ui.Scroll(c.Key("archived")).MaxHeight(archivedRows * theme.SettingsRowMin).Children(rows)
}
