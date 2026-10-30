package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// syncGoogleLabel is the 13 pt 500 text of the Google button.
var syncGoogleLabel = theme.Style{Size: 13, Line: 18, Weight: 500}

// syncOrText is the 12 pt text.tertiary "or use your email" between the hairlines.
var syncOrText = theme.Style{Size: 12, Line: 16, Weight: 400}

// The Google button and the waiting card (ui-spec, Settings > Account > Google).
const (
	syncGoogleMark    float32 = 14
	syncGoogleMarkBig float32 = 36
	syncGoogleButtonH float32 = 32
)

// syncGoogleButton is "Continue with Google": a full-width 32 pt outlined button with Google's G,
// and under it the "or use your email" rule that leads to the fields. The browser is opened through
// the shell's own OpenURL.
func (s *Settings) syncGoogleButton(c *ui.Context, t *theme.Tokens, svc SyncService, busy bool) {
	text := s.say.SettingsSyncContinueGoogle()
	btn := ui.ButtonBase(c.Key("sync-google")).FillWidth().Height(syncGoogleButtonH).Shrink(0).Margin(14, 0, 0).
		Radius(7).Label(text).Cursor(ui.CursorPointer).Center().
		Background(t.SettingsGroup()).Border(theme.Hairline, t.Border) // the hairline reads on the white card
	if btn.Hovered() {
		btn.Background(t.SettingsGroup().Mix(t.Text, 0.06))
	}
	btn.Children(func() {
		ui.Row(c.Key("sync-google-row")).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Image(c, theme.IconGoogle).Size(syncGoogleMark, syncGoogleMark).Shrink(0)
			theme.Text(c, syncGoogleLabel, t.Text, text)
		})
	})
	if busy {
		btn.Disabled(true).Opacity(0.45)
	} else if btn.Clicked() {
		svc.SyncSignInWithGoogle(s.app.Act.OpenURL)
	}
	ui.Row(c.Key("sync-or")).FillWidth().Shrink(0).Gap(10).AlignItems(ui.Center).Margin(12, 0, 0).Children(func() {
		ui.Box(c.Key("sync-or-left")).Grow(1).Height(theme.Hairline).Background(t.Border)
		theme.Text(c, syncOrText, t.TextTertiary, s.say.SettingsSyncOrEmail())
		ui.Box(c.Key("sync-or-right")).Grow(1).Height(theme.Hairline).Background(t.Border)
	})
}

// syncGoogleStep is the card while the browser shows Google's consent page: the G, "Finish in your
// browser", the line that the window picks the answer up by itself, and Cancel.
func (s *Settings) syncGoogleStep(c *ui.Context, t *theme.Tokens, svc SyncService) {
	ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Image(c, theme.IconGoogle).Size(syncGoogleMarkBig, syncGoogleMarkBig).Shrink(0)
		hero := theme.Input
		hero.Weight = 600
		theme.Paragraph(c, hero, t.Text, s.say.SettingsSyncGoogleTitle(), 0).TextAlign(ui.Center).Margin(12, 0, 0)
		theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncGoogleBody(), 0).
			TextAlign(ui.Center).Margin(6, 0, 0)
		if settingsButton(c, t, "sync-google-cancel", s.say.SettingsSyncCancel(), false).Margin(16, 0, 0).Clicked() {
			svc.SyncCancel()
		}
	})
}
