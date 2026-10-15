package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// repoURL is the project on GitHub, the About row's link.
const repoURL = "https://github.com/QingYunA/adhd-todo"

// archivedRows is how many archived projects the pane shows before the list scrolls.
const archivedRows = 5

// generalPane is the language, the archived projects, and where the data lives.
func (s *Settings) generalPane(c *ui.Context, t *theme.Tokens) {
	group(c, t, "language", func() {
		settingsRow(c, t, "language", s.say.SettingsLanguage(), "", true, func() { s.languageControl(c, t) })
	})
	groupLabel(c, t, s.say.ManageArchivedTitle())
	group(c, t, "archived", func() { s.archivedRowsView(c, t) })
	group(c, t, "about", func() {
		settingsRow(c, t, "folder", s.say.SettingsDataFolder(), withTilde(s.app.Config.DataFolder, s.app.Config.Home), true, func() {
			if settingsLink(c, t, "show-folder", s.say.SettingsShowInFinder(), s.say.SettingsShowInFinder()).Clicked() {
				s.app.Act.ShowInFinder(s.app.Config.DataFolder)
			}
		})
		settingsRow(c, t, "about", s.say.SettingsAboutName(s.app.Config.Version), s.say.SettingsAboutLicense(), false, func() {
			if settingsLink(c, t, "github", s.say.SettingsAboutLink(), s.say.SettingsAboutLink()).Clicked() {
				s.app.Act.OpenURL(repoURL)
			}
		})
	})
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
