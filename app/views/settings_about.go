package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The app's places on the web: its source, its privacy policy and its site.
const (
	repoURL    = "https://github.com/QingYunA/withdo"
	privacyURL = "https://qingyuna.github.io/withdo/privacy.html"
	websiteURL = "https://qingyuna.github.io/withdo/"
)

// aboutPane is the app, its version, its license, and its links, under its own mark.
func (s *Settings) aboutPane(c *ui.Context, t *theme.Tokens) {
	paneHeader(c, t, s.say.SettingsPaneAbout(), s.say.SettingsPaneAboutLine())
	var lead func()
	if s.loadAppMark() {
		lead = func() { s.appMark(c) }
	}
	group(c, t, "about", func() {
		settingsRowLead(c, t, "about", s.say.SettingsAboutName(s.app.Config.Version), s.say.SettingsAboutLicense(), true, func() {
			ui.Row(c.Key("about-links")).Shrink(0).Gap(10).AlignItems(ui.Center).Children(func() {
				if settingsOutLink(c, t, "github", s.say.SettingsAboutLink(), s.say.A11yAboutGitHub()).Clicked() {
					s.app.Act.OpenURL(repoURL)
				}
				if settingsOutLink(c, t, "privacy", s.say.SettingsAboutPrivacy(), s.say.SettingsAboutPrivacy()).Clicked() {
					s.app.Act.OpenURL(privacyURL)
				}
				if settingsOutLink(c, t, "website", s.say.SettingsAboutWebsite(), s.say.SettingsAboutWebsite()).Clicked() {
					s.app.Act.OpenURL(websiteURL)
				}
			})
		}, lead)
	})
}

// loadAppMark decodes the app's mark once, from the PNG the app carries.
func (s *Settings) loadAppMark() bool {
	if !s.markTried {
		s.markTried = true
		if bmp, err := ui.DecodeBitmap(s.app.Config.AppIcon); err == nil {
			s.mark = bmp.Pixelated()
		}
	}
	return s.mark != nil
}

// appMark is the app's icon, 48 pt square, drawn as pixel art like the other pictures the app
// ships.
func (s *Settings) appMark(c *ui.Context) ui.Element {
	if !s.loadAppMark() {
		return ui.Element{}
	}
	return ui.Image(c, s.mark).Size(48, 48).Shrink(0)
}
