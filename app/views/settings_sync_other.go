package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The two choices of the "This Mac was linked to another account" sheet.
const (
	syncOtherOld  = 0
	syncOtherMove = 1
)

// syncOtherSheet asks what to do when a sign-in succeeded as an account that does not own this
// Mac: sign in with the old account again (the recommended way), or move this Mac to the new
// account. Nothing syncs while it is open; Cancel signs out of the new account.
func (s *Settings) syncOtherSheet(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot) {
	other := snap.OtherAccount
	if s.pane != PaneAccount || s.app.Sync == nil || other == nil {
		s.syncHadOther = false
		return
	}
	if !s.syncHadOther {
		s.syncOtherChoice, s.syncHadOther = syncOtherOld, true
	}
	known := other.PreviousEmail != ""
	body := s.say.SettingsSyncOtherBodyUnknown(other.AccountEmail)
	oldTitle := s.say.SettingsSyncOtherOldAnother()
	moveBody := s.say.SettingsSyncOtherMoveBodyUnknown(other.AccountEmail)
	if known {
		body = s.say.SettingsSyncOtherBody(other.PreviousEmail, other.AccountEmail)
		oldTitle = s.say.SettingsSyncOtherOld(other.PreviousEmail)
		moveBody = s.say.SettingsSyncOtherMoveBody(other.AccountEmail, other.PreviousEmail)
	}
	cancelled := false
	ui.Overlay(c.Key("sync-other-sheet"), func() {
		back := ui.Box(c.Key("sync-other-scrim")).Absolute().Left(0).Top(settingsBandHeight).Right(0).Bottom(0).
			Background(ui.RGBA(0, 0, 0, 0.18)).Modal()
		if back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape) {
			cancelled = true
		}
		panel := ui.Column(c.Key("sync-other-panel")).Absolute().Left(syncSheetLeft).Top(syncSheetTop).
			Width(syncSheetWidth).Padding(20, 22).Radius(theme.RadiusPanel).Background(t.SettingsGroup()).
			Border(1, t.Border).Shadow(0, 18, 40, 0, t.Shadow).Role(ui.RoleDialog).Modal()
		panel.Children(func() {
			title := theme.Input
			title.Weight = 600
			theme.Text(c, title, t.Text, s.say.SettingsSyncOtherTitle())
			theme.Paragraph(c, theme.Meta, t.TextSecondary, body, 0).Margin(6, 0, 0)
			if syncChoiceCard(c, t, "sync-other-old", oldTitle, s.say.SettingsSyncOtherOldBody(other.AccountEmail),
				s.syncOtherChoice == syncOtherOld, s.say.SettingsSyncRecommended()) {
				s.syncOtherChoice = syncOtherOld
			}
			if syncChoiceCard(c, t, "sync-other-move", s.say.SettingsSyncOtherMove(other.AccountEmail), moveBody,
				s.syncOtherChoice == syncOtherMove, "") {
				s.syncOtherChoice = syncOtherMove
			}
			ui.Row(c).Justify(ui.End).Gap(8).Margin(18, 0, 0).Children(func() {
				if settingsButton(c, t, "sync-other-cancel", s.say.SettingsSyncCancel(), false).Clicked() {
					cancelled = true
				}
				if settingsButton(c, t, "sync-other-continue", s.say.SettingsSyncContinue(), true).Clicked() {
					if s.syncOtherChoice == syncOtherMove {
						s.app.Sync.SyncMoveToAccount()
					} else {
						// The email card opens with the old address filled in.
						s.syncLogin, s.syncFocusAccount = other.PreviousEmail, true
						s.app.Sync.SyncUseOldAccount()
					}
				}
			})
		})
	})
	if cancelled {
		s.syncHadOther = false
		s.app.Sync.SyncCancel()
	}
}

// syncWrongAccountRow is This Mac's row when the signed-in account does not own this Mac: what is
// wrong and what to do, with Sign out right there. Nothing retries.
func (s *Settings) syncWrongAccountRow(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	syncRow(c, t, "this-mac", false, func() {
		theme.Text(c, theme.Body, t.Text, s.say.SettingsSyncThisMac())
		theme.Paragraph(c, theme.Meta, t.Blocked(), s.say.SettingsSyncWrongAccount(), 2)
		if snap.Pending > 0 {
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncWaiting(itoa(snap.Pending)), 2)
		}
	}, func() {
		if settingsButton(c, t, "sync-wrong-signout", s.say.SettingsSyncSignOut(), true).Clicked() {
			svc.SyncSignOut()
		}
	})
}

// syncGoogleRow is the Google row of a signed-in account: Link when it has no Google identity,
// Linked when it has, the wait for the browser, and the line of a failed link.
func (s *Settings) syncGoogleRow(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	if snap.GoogleLink == data.GoogleLinkHidden || snap.Phase == data.SyncWrongAccount {
		return
	}
	syncRow(c, t, "google", false, func() {
		ui.Row(c.Key("sync-google-label")).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Image(c, theme.IconGoogle).Size(syncGoogleMark, syncGoogleMark).Shrink(0)
			theme.Text(c, theme.Body, t.Text, s.say.SettingsSyncGoogleRow())
		})
		switch {
		case snap.GoogleLink == data.GoogleLinkLinked:
			theme.Text(c, theme.Meta, t.Status(data.Done), s.say.SettingsSyncGoogleLinked())
		case snap.GoogleLink == data.GoogleLinkWaiting:
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncGoogleLinkWaiting())
		case snap.GoogleLinkFailure != data.GoogleLinkFailureNone:
			theme.Paragraph(c, theme.Meta, t.Blocked(), s.syncGoogleLinkFailureText(snap.GoogleLinkFailure), 2)
		}
	}, func() {
		switch snap.GoogleLink {
		case data.GoogleLinkAvailable:
			if settingsButton(c, t, "sync-google-link", s.say.SettingsSyncGoogleLink(), false).Clicked() {
				svc.SyncLinkGoogle(s.app.Act.OpenURL)
			}
		case data.GoogleLinkWaiting:
			if settingsLink(c, t, "sync-google-link-cancel", s.say.SettingsSyncCancel(), s.say.SettingsSyncCancel()).Clicked() {
				svc.SyncCancelGoogleLink()
			}
		}
	})
}

// syncGoogleLinkFailureText words why linking Google failed.
func (s *Settings) syncGoogleLinkFailureText(failure data.GoogleLinkFailure) string {
	switch failure {
	case data.GoogleLinkFailureUsed:
		return s.say.SettingsSyncGoogleLinkUsed()
	case data.GoogleLinkFailureNetwork:
		return s.say.SettingsSyncGoogleLinkNetwork()
	}
	return s.say.SettingsSyncGoogleLinkFailed()
}
