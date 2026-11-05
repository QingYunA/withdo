package views

import (
	"context"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The official model in Settings > Model (M15, v44): when the sync server offers it, a card of its
// own above the providers. In use it is the big first card (the assistant's sprite, what it is, how
// many turns are left today on a quota bar, Test); while a provider of one's own is in use it is one
// row under that provider's card, with Use this. Signed out the card says it comes with signing in,
// and Sign in opens Account.

const (
	officialRowSprite float32 = 24 // the sprite in the one-row group
	quotaBarH         float32 = 4
	officialTestID            = "official"
)

// officialStatus is the status line of the official model, in the parts the card colors: what it is,
// then whether it is there. The one-row group under a provider of one's own says only the turns.
func (s *Settings) officialStatus(c *ui.Context, t *theme.Tokens, info data.OfficialInfo, bill data.BillingInfo, row bool) {
	if info.Phase == data.OfficialSignedOut {
		theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsOfficialSignedOut(), 2)
		return
	}
	ui.Row(c).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
		switch {
		case row:
		case bill.Offered:
			s.planPill(c, t, bill)
		default:
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsOfficialProvided()+" ·")
		}
		switch {
		case info.Phase == data.OfficialUsedUp:
			theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsOfficialUsedUp()+" · "+s.say.SettingsOfficialBack())
		case row:
			if info.Left >= 0 {
				theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsOfficialLeft(itoa(info.Left)))
			}
		default:
			statusDot(c, t.Status(data.Done))
			theme.Text(c, theme.Meta, t.Status(data.Done), s.say.SettingsOfficialAvailable())
			if info.Left >= 0 {
				theme.Text(c, theme.Meta, t.TextSecondary, "· "+s.say.SettingsOfficialLeft(leftOf(info, bill)))
			}
			if bill.Offered && bill.Known && !bill.Pro {
				theme.Text(c, theme.Meta, t.TextSecondary, "·")
				if s.upgradeLink(c, t).Clicked() {
					s.ShowUpgrade()
				}
			}
		}
	})
}

// upgradeLink is the accent Upgrade to Pro after a Free account's quota: a link, not a button. The
// plan is sold in Account, so it opens the sheet there.
func (s *Settings) upgradeLink(c *ui.Context, t *theme.Tokens) ui.Element {
	text := s.say.SettingsProUpgrade()
	btn := ui.ButtonBase(c.Key("official-upgrade")).Padding(0, 2).Radius(theme.RadiusControl).Shrink(0).Label(text).Cursor(ui.CursorPointer)
	ink := t.AccentInk
	if btn.Hovered() {
		ink = ink.Mix(t.Text, 0.2)
	}
	btn.Children(func() { theme.Text(c, theme.Meta, ink, text) })
	return btn
}

// leftOf is the turns left today: "86", or against the plan's limit ("14 / 20") where plans are sold.
func leftOf(info data.OfficialInfo, bill data.BillingInfo) string {
	if bill.Offered && info.Limit > 0 {
		return itoa(info.Left) + " / " + itoa(info.Limit)
	}
	return itoa(info.Left)
}

// quotaBar is the day's turns: a 4 pt track with the part that is left filled in the done green.
func quotaBar(c *ui.Context, t *theme.Tokens, info data.OfficialInfo) {
	if info.Limit <= 0 || info.Left < 0 {
		return
	}
	left, used := float32(min(info.Left, info.Limit)), float32(max(0, info.Limit-info.Left))
	ui.Row(c.Key("quota")).FillWidth().Height(quotaBarH).Shrink(0).Margin(6, 0, 0).Background(t.Border.Alpha(0.5)).Children(func() {
		ui.Box(c.Key("quota-left")).Grow(left).Basis(0).Height(quotaBarH).Background(t.Status(data.Done))
		ui.Box(c.Key("quota-used")).Grow(used).Basis(0).Height(quotaBarH)
	})
}

// officialCard is the big first card: the official model in use (or, signed out or used up, the one
// that would be).
func (s *Settings) officialCard(c *ui.Context, t *theme.Tokens, info data.OfficialInfo) {
	res := s.assist.test
	if res.id != officialTestID {
		res = testResult{}
	}
	bill := s.billing()
	group(c, t, "official", func() {
		ui.Row(c).Shrink(0).Padding(14, 16).Gap(14).AlignItems(ui.Center).Children(func() {
			s.officialSprite(c, info, theme.LogoCard)
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				theme.Text(c, theme.BodyStrong, t.Text, s.say.SettingsOfficialName()).Ellipsis("…")
				s.officialStatus(c, t, info, bill, false)
				quotaBar(c, t, info)
				s.officialTestLine(c, t, res)
			})
			s.officialButton(c, t, info, false)
		})
	})
}

// officialRow is the one-row group under a provider of one's own in use: the official model as the
// thing to switch to.
func (s *Settings) officialRow(c *ui.Context, t *theme.Tokens, info data.OfficialInfo) {
	res := s.assist.test
	if res.id != officialTestID {
		res = testResult{}
	}
	group(c, t, "official", func() {
		ui.Row(c).Shrink(0).MinHeight(theme.SettingsRowMin).Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(func() {
			s.officialSprite(c, info, officialRowSprite)
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				theme.Text(c, theme.Body, t.Text, s.say.SettingsOfficialName()).Ellipsis("…")
				s.officialStatus(c, t, info, data.BillingInfo{}, true)
				s.officialTestLine(c, t, res)
			})
			s.officialButton(c, t, info, true)
		})
	})
}

// officialSprite is the assistant's sprite; signed out it is faded, the model not there yet.
func (s *Settings) officialSprite(c *ui.Context, info data.OfficialInfo, size float32) {
	icon := theme.AvatarIcon(c.Key("official-sprite"), theme.Assistant, size).Shrink(0)
	if info.Phase == data.OfficialSignedOut {
		icon.Opacity(0.5)
	}
}

// officialButton is the control at the right: Sign in (accent) signed out; else Test on the card,
// Use this on the row.
func (s *Settings) officialButton(c *ui.Context, t *theme.Tokens, info data.OfficialInfo, row bool) {
	switch {
	case info.Phase == data.OfficialSignedOut:
		if settingsButton(c, t, "official-sign-in", s.say.SettingsOfficialSignIn(), true).Clicked() {
			s.openPane(PaneAccount)
		}
	case row:
		if settingsButton(c, t, "official-use", s.say.SettingsOfficialUseThis(), false).Clicked() {
			_ = s.app.Models.UseOfficial()
			s.assist.test = testResult{}
		}
	default:
		if settingsButton(c, t, "official-test", s.say.ModelsTest(), false).Clicked() {
			s.runOfficialTest()
		}
	}
}

// officialTestLine is how the last test went: the time it took, or why it failed.
func (s *Settings) officialTestLine(c *ui.Context, t *theme.Tokens, res testResult) {
	switch {
	case res.running:
		theme.Text(c, theme.Meta, t.TextSecondary, s.say.ModelsTesting())
	case res.ok:
		ui.Row(c).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
			statusDot(c, t.Status(data.Done))
			theme.Text(c, theme.Meta, t.Status(data.Done), res.text)
		})
	case res.text != "":
		theme.Paragraph(c, theme.Meta, t.Blocked(), res.text, 2)
	}
}

// runOfficialTest checks the session and the server off the UI thread and shows how it went.
func (s *Settings) runOfficialTest() {
	if s.assist.test.running && s.assist.test.id == officialTestID {
		return
	}
	s.assist.test = testResult{id: officialTestID, running: true}
	models, post := s.app.Models, s.app.Act.Post
	go func() {
		d, err := models.TestOfficial(context.Background())
		post(func() { s.assist.test = s.outcomeOf(officialTestID, d, err) })
	}()
}

// officialAsked asks the server for today's turns again once per opening of the pane, so the card
// reads a fresh count.
func (s *Settings) officialAsked() {
	if s.assist.asked || s.app.Models == nil {
		return
	}
	s.assist.asked = true
	s.app.Models.RefreshOfficial()
	s.app.Models.RefreshBilling()
}
