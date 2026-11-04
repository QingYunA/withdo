package views

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// Withdo Pro in Settings > Model (M16, v45): the plan pill and the footer row of the official card,
// and the upgrade sheet the footer's Upgrade to Pro (and the composer's line) opens. The sheet asks
// the server for a payment page, opens it in the browser, and waits: the plan changes on the
// server, the app hears of it (the "billing" event, or the window coming back), and a Pro account
// closes the sheet by itself.

const (
	proSheetWidth   float32 = 380
	proSheetTop     float32 = 84
	proSprite       float32 = 36
	proPriceLine            = "pro-price"
	proCheckoutTO           = 20 * time.Second
	proPayPalHeight float32 = 32
)

// PayPal's brand (M17, v46): the button is its gold with the wordmark in its two blues, in both
// appearances, so these are not theme tokens.
var (
	paypalGold  = ui.RGB(0xFF, 0xC4, 0x39)
	paypalBlue  = ui.RGB(0x00, 0x30, 0x87)
	paypalLight = ui.RGB(0x00, 0x9C, 0xDE)
)

// proState is the upgrade sheet and the footer's Manage subscription.
type proState struct {
	open    bool
	monthly bool   // the Monthly segment is chosen; the sheet opens on Yearly
	waiting bool   // the payment page is open in the browser
	paypal  bool   // ... and it is PayPal's: the sheet words the wait for PayPal
	url     string // that page, for Reopen
	busy    bool   // a payment page is being asked for
	err     string // why the last try failed, in the sheet
	// managing is a portal link being asked for; manageErr is why the last one failed, under the footer.
	managing  bool
	manageErr string
}

// ShowUpgrade opens the upgrade sheet over Settings > Model, as the composer's line asks. It does
// nothing for a window that has no plan to sell.
func (s *Settings) ShowUpgrade() {
	s.Show(PaneModel)
	s.openUpgrade()
}

// Focused is told when the Settings window gets the keyboard back: after paying in the browser, the
// plan is asked for again.
func (s *Settings) Focused() {
	if s.app.Models != nil {
		s.app.Models.RefreshBilling()
	}
}

func (s *Settings) openUpgrade() {
	s.pro = proState{open: true}
}

// billing is the plan as the server last said it, the zero value for a build with no models.
func (s *Settings) billing() data.BillingInfo {
	if s.app.Models == nil {
		return data.BillingInfo{}
	}
	return s.app.Models.Billing()
}

// planPill is the small word before the official card's status: Free, or Pro in the accent's wash.
func (s *Settings) planPill(c *ui.Context, t *theme.Tokens, bill data.BillingInfo) {
	text, ink, ground := s.say.SettingsPlanFree(), t.TextSecondary, t.BgSubtle
	if bill.Pro {
		text, ink, ground = s.say.SettingsPlanPro(), t.AccentInk, t.Accent.Alpha(0.14)
	}
	ui.Row(c.Key("plan-pill")).Shrink(0).Padding(0, 6).Radius(4).Background(ground).Children(func() {
		theme.Text(c, theme.Meta, ink, text)
	})
}

// officialFooter is the row under the official card's hairline: what Pro is (or which plan this
// is) and the button that goes with it. It waits for the server's answer, which has the trial and
// the dates.
func (s *Settings) officialFooter(c *ui.Context, t *theme.Tokens, bill data.BillingInfo) {
	if !bill.Offered || !bill.Known {
		return
	}
	ui.Box(c.Key("official-footer-rule")).Height(theme.Hairline).Shrink(0).Margin(0, 16).Background(t.Border)
	ui.Row(c.Key("official-footer")).Shrink(0).Padding(9, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.planLine(bill), 2)
			if s.pro.manageErr != "" {
				theme.Paragraph(c, theme.Meta, t.Blocked(), s.pro.manageErr, 2)
			}
		})
		if bill.Subscribed() {
			if settingsButton(c, t, "pro-manage", s.say.SettingsProManage(), false).Clicked() {
				s.manageSubscription()
			}
			return
		}
		if bill.Pro {
			return
		}
		if settingsButton(c, t, "pro-upgrade", s.say.SettingsProUpgrade(), true).Clicked() {
			s.openUpgrade()
		}
	})
}

// planLine is what the footer says: what Pro gives while the account is Free, and for a Pro one the
// plan with the day it ends its trial, renews or ends.
func (s *Settings) planLine(bill data.BillingInfo) string {
	if !bill.Pro {
		if bill.TrialUsed {
			return s.say.SettingsProLine()
		}
		return s.say.SettingsProLineTrial()
	}
	via := data.ProviderWaffo
	if bill.ViaPayPal() {
		via = data.ProviderPayPal
	}
	date := ""
	if !bill.PeriodEnd.IsZero() {
		date = s.clock.Date(bill.PeriodEnd, s.app.Now())
	}
	if date != "" && bill.Interval != "" {
		switch {
		case bill.Trialing():
			return s.say.SettingsProIntervalTrialUntil(via, bill.Interval, date)
		case bill.Canceling():
			return s.say.SettingsProIntervalEnds(via, bill.Interval, date)
		}
		return s.say.SettingsProIntervalRenews(via, bill.Interval, date)
	}
	switch {
	case date == "":
		return s.say.SettingsPlanPro()
	case bill.Trialing():
		return s.say.SettingsProTrialUntil(via, date)
	case bill.Canceling():
		return s.say.SettingsProEnds(via, date)
	}
	return s.say.SettingsProRenews(via, date)
}

// manageSubscription asks for a fresh portal link and opens it. The link lives 15 minutes, so it is
// never kept.
func (s *Settings) manageSubscription() {
	if s.pro.managing {
		return
	}
	s.pro.managing, s.pro.manageErr = true, ""
	models, act := s.app.Models, s.app.Act
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), proCheckoutTO)
		defer cancel()
		url, err := models.ManageURL(ctx)
		act.Post(func() {
			s.pro.managing = false
			if err != nil {
				s.pro.manageErr = s.say.SettingsProManageFailed()
				return
			}
			act.OpenURL(url)
		})
	}()
}

// upgradeSheet is the sheet over the pane while it is open.
func (s *Settings) upgradeSheet(c *ui.Context, t *theme.Tokens) {
	if !s.pro.open {
		return
	}
	bill := s.billing()
	if bill.Pro || !bill.Offered { // it arrived, or there is nothing to sell: the sheet has no more to do
		s.pro = proState{}
		return
	}
	dismiss := false
	ui.Overlay(c.Key("pro-sheet"), func() {
		back := ui.Box(c.Key("pro-scrim")).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(ui.RGBA(0, 0, 0, 0.18)).Modal()
		if back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape) {
			dismiss = true
		}
		panel := ui.Column(c.Key("pro-sheet-panel")).Absolute().Left((theme.SettingsWidth-proSheetWidth)/2).Top(proSheetTop).
			Width(proSheetWidth).Padding(22).Radius(theme.RadiusPanel).Background(t.SettingsGroup()).
			Border(1, t.Border).Shadow(0, 18, 40, 0, t.Shadow).Role(ui.RoleDialog).Modal()
		panel.Children(func() {
			if s.pro.waiting {
				dismiss = s.waitingBody(c, t)
				return
			}
			s.upgradeBody(c, t, bill)
		})
	})
	if dismiss {
		s.pro = proState{}
	}
}

// upgradeBody is the sheet before the payment: the plan, the price, what Pro gives, the button.
func (s *Settings) upgradeBody(c *ui.Context, t *theme.Tokens, bill data.BillingInfo) {
	title := theme.Input
	title.Weight = 600
	theme.Text(c, title, t.Text, s.say.ProSheetTitle())
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.ProSheetLine(), 0).Margin(2, 0, 0)

	ui.Box(c.Key("pro-line-gap")).Height(12).Shrink(0)
	var notes []string
	if pct := data.SavingPercent(bill.Monthly, bill.Yearly); pct > 0 {
		notes = []string{"", s.say.ProSave(itoa(pct))}
	}
	sel := 1
	if s.pro.monthly {
		sel = 0
	}
	labels := []string{s.say.ProMonthly(), s.say.ProYearly()}
	if i := segmentedIn(c, t, "pro-plan", s.say.ProSheetTitle(), labels, notes, sel, true); i >= 0 {
		s.pro.monthly = i == 0
	}
	ui.Box(c.Key("pro-seg-gap")).Height(14).Shrink(0)

	ui.Row(c.Key(proPriceLine)).Shrink(0).Gap(6).AlignItems(ui.End).Children(func() {
		price, suffix := bill.Yearly, ""
		if per := data.PerMonth(bill.Yearly); per != "" {
			suffix = s.say.ProPriceYearly(per)
		}
		if s.pro.monthly {
			price, suffix = bill.Monthly, s.say.ProPriceMonthly()
		}
		theme.Text(c, theme.Title, t.Text, price)
		theme.Text(c, theme.Meta, t.TextSecondary, suffix).Margin(0, 0, 5)
	})

	ui.Column(c.Key("pro-ticks")).Shrink(0).Gap(5).Margin(12, 0, 0).Children(func() {
		for i, text := range []string{s.say.ProTickModel(), s.say.ProTickDevices(), s.say.ProTickSync(), s.say.ProTickCancel()} {
			ui.Row(c.Key("pro-tick-" + itoa(i))).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, theme.IconCheck).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Status(data.Done)).Shrink(0)
				theme.Text(c, theme.Meta, t.Text, text)
			})
		}
	})

	label, fine := s.say.ProStartTrial(), s.say.ProFinePrintTrial()
	if bill.TrialUsed {
		label, fine = s.say.ProSubscribe(), s.say.ProFinePrint()
	}
	if bill.PayPal {
		fine = s.say.ProFinePrintTrialPayPal()
		if bill.TrialUsed {
			fine = s.say.ProFinePrintPayPal()
		}
	}
	btn := settingsButton(c, t, "pro-start", label, true).FillWidth().Margin(16, 0, 0)
	if s.pro.busy {
		btn.Opacity(0.6)
	}
	if btn.Clicked() && !s.pro.busy {
		s.startCheckout(data.ProviderWaffo)
	}
	if bill.PayPal {
		s.paypalButton(c, t)
	}
	if s.pro.err != "" {
		theme.Paragraph(c, theme.Meta, t.Blocked(), s.pro.err, 0).Margin(8, 0, 0)
	}
	theme.Paragraph(c, theme.Meta, t.TextTertiary, fine, 0).Margin(8, 0, 0)
}

// paypalButton is the hairline with "or" and PayPal's button under the main one. It checks out the
// plan chosen above, at PayPal. The label is one message with PayPal in it, so each language orders
// it as it should; the word PayPal is drawn as the wordmark (PayPal's blues, in italics).
func (s *Settings) paypalButton(c *ui.Context, t *theme.Tokens) {
	ui.Row(c.Key("pro-or")).FillWidth().Shrink(0).Gap(8).AlignItems(ui.Center).Margin(10, 0, 0).Children(func() {
		ui.Box(c.Key("pro-or-left")).Grow(1).Height(theme.Hairline).Background(t.Border)
		theme.Text(c, theme.Meta, t.TextTertiary, s.say.ProOr())
		ui.Box(c.Key("pro-or-right")).Grow(1).Height(theme.Hairline).Background(t.Border)
	})
	label := s.say.ProPayPal()
	btn := ui.ButtonBase(c.Key("pro-paypal")).FillWidth().Height(proPayPalHeight).Shrink(0).Radius(7).Margin(8, 0, 0).
		Background(paypalGold).Label(label).Cursor(ui.CursorPointer).Center()
	if btn.Hovered() {
		btn.Background(paypalGold.Mix(ui.RGB(0, 0, 0), 0.06))
	}
	if s.pro.busy {
		btn.Opacity(0.6)
	}
	btn.Children(func() { paypalLabel(c, label) })
	if btn.Clicked() && !s.pro.busy {
		s.startCheckout(data.ProviderPayPal)
	}
}

// paypalLabel draws a message with PayPal in it: the words in PayPal's dark blue, PayPal as its
// wordmark, "Pay" in the dark blue and "Pal" in the light one, both bold italic.
func paypalLabel(c *ui.Context, label string) {
	words := theme.MetaStrong
	words.Weight = 600
	mark := words
	mark.Weight = 800
	before, after, found := strings.Cut(label, "PayPal")
	ui.Row(c.Key("pro-paypal-label")).Gap(4).AlignItems(ui.Center).Children(func() {
		if !found {
			theme.Text(c, words, paypalBlue, label)
			return
		}
		if before = strings.TrimSpace(before); before != "" {
			theme.Text(c, words, paypalBlue, before)
		}
		ui.Row(c.Key("pro-paypal-mark")).Gap(0).Children(func() {
			theme.Text(c.Key("pro-paypal-pay"), mark, paypalBlue, "Pay").Italic()
			theme.Text(c.Key("pro-paypal-pal"), mark, paypalLight, "Pal").Italic()
		})
		if after = strings.TrimSpace(after); after != "" {
			theme.Text(c, words, paypalBlue, after)
		}
	})
}

// startCheckout asks the server for the payment page of the chosen plan at provider, off the UI thread, and
// opens it in the browser; the sheet then waits.
func (s *Settings) startCheckout(provider string) {
	plan := data.PlanYearly
	if s.pro.monthly {
		plan = data.PlanMonthly
	}
	s.pro.busy, s.pro.err = true, ""
	models, act, lang := s.app.Models, s.app.Act, s.lang
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), proCheckoutTO)
		defer cancel()
		url, err := models.StartCheckout(ctx, plan, provider, lang)
		act.Post(func() {
			s.pro.busy = false
			switch {
			case errors.Is(err, data.ErrAlreadySubscribed):
				s.pro = proState{} // another device bought it: the plan is being read again
			case errors.Is(err, data.ErrCheckoutThrottled):
				s.pro.err = s.say.ProCheckoutThrottled()
			case err != nil:
				s.pro.err = s.say.ProCheckoutFailed()
			default:
				s.pro.waiting, s.pro.url, s.pro.paypal = true, url, provider == data.ProviderPayPal
				act.OpenURL(url)
			}
		})
	}()
}

// waitingBody is the sheet while the payment is made in the browser. It returns true when Cancel
// was pressed.
func (s *Settings) waitingBody(c *ui.Context, t *theme.Tokens) (cancelled bool) {
	ui.Column(c.Key("pro-waiting")).FillWidth().AlignItems(ui.Center).Gap(6).Children(func() {
		theme.AvatarIcon(c.Key("pro-sprite"), theme.Assistant, proSprite).Shrink(0).Margin(0, 0, 6)
		title := theme.Input
		title.Weight = 600
		titleText, line, reopen := s.say.ProWaitingTitle(), s.say.ProWaitingLine(), s.say.ProReopen()
		if s.pro.paypal {
			titleText, line, reopen = s.say.ProWaitingTitlePayPal(), s.say.ProWaitingLinePayPal(), s.say.ProReopenPayPal()
		}
		theme.Text(c, title, t.Text, titleText)
		theme.Paragraph(c, theme.Meta, t.TextSecondary, line, 0).TextAlign(ui.Center)
		if settingsButton(c, t, "pro-reopen", reopen, false).FillWidth().Margin(10, 0, 0).Clicked() {
			s.app.Act.OpenURL(s.pro.url)
		}
		l := settingsLink(c, t, "pro-cancel", s.say.ManageCancel(), s.say.ManageCancel())
		if l.Clicked() {
			cancelled = true
		}
	})
	return cancelled
}
