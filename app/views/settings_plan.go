package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The plan in Settings > Account (v47): the last row of the account group. It says which plan this is, with a tag for the subscription's status, one line of what it costs and when it
// changes, and the button that goes with it: Upgrade to Pro (the sheet in settings_pro.go) on Free,
// Manage subscription on Pro. It waits for the server's answer, which has the trial and the dates,
// and is not there signed out or on a server that sells no plan.

// planTagWash is the alpha of a tag's wash.
const planTagWash = 0.14

// planRow is the last row of the account group. It waits for the server's answer and is not there
// without a plan to sell.
func (s *Settings) planRow(c *ui.Context, t *theme.Tokens) {
	bill := s.billing()
	if !bill.Offered || !bill.Known {
		return
	}
	syncRow(c, t, "plan", false, func() {
		ui.Row(c.Key("plan-name")).Gap(8).AlignItems(ui.Center).Children(func() {
			name := s.say.SettingsPlanNameFree()
			if bill.Pro {
				name = s.say.SettingsPlanNamePro()
			}
			theme.Text(c, theme.BodyStrong, t.Text, name)
			s.planTag(c, t, bill)
		})
		if line := s.planLine(bill); line != "" {
			theme.Paragraph(c, theme.Meta, t.TextSecondary, line, 2)
		}
		if s.pro.manageErr != "" {
			theme.Paragraph(c, theme.Meta, t.Blocked(), s.pro.manageErr, 2)
		}
	}, func() {
		switch {
		case bill.Subscribed():
			if settingsButton(c, t, "pro-manage", s.say.SettingsProManage(), false).Clicked() {
				s.manageSubscription()
			}
		case !bill.Pro:
			if settingsButton(c, t, "pro-upgrade", s.say.SettingsProUpgrade(), true).Clicked() {
				s.openUpgrade()
			}
		}
	})
}

// planTag is the small word after a Pro plan's name, by status: the done green while the trial
// runs, the blocked red once canceled, the review yellow when the renewal failed, and the accent
// for a paying account. Free has none.
func (s *Settings) planTag(c *ui.Context, t *theme.Tokens, bill data.BillingInfo) {
	if !bill.Pro {
		return
	}
	text, ink, wash := s.say.SettingsPlanPro(), t.AccentInk, t.Accent.Alpha(planTagWash)
	tone := func(word string, status data.Status) {
		base := t.Status(status)
		text, ink, wash = word, base.Mix(t.Text, 0.35), base.Alpha(planTagWash)
	}
	switch {
	case bill.Trialing():
		tone(s.say.SettingsPlanTagTrial(), data.Done)
	case bill.Canceling():
		tone(s.say.SettingsPlanTagCanceled(), data.Blocked)
	case bill.PastDue():
		tone(s.say.SettingsPlanTagPastDue(), data.Review)
	}
	ui.Row(c.Key("plan-tag")).Shrink(0).Padding(1, 6).Radius(6).Background(wash).Children(func() {
		theme.Text(c, theme.Badge, ink, text)
	})
}

// planLine is the row's line: what Pro gives while the account is Free; for a Pro one the
// interval and list price, who bills it when that is PayPal, and the day it charges, renews or
// ends, joined with " · ". A price comes only with a plan that will charge (trial or active);
// without the interval the line has neither.
func (s *Settings) planLine(bill data.BillingInfo) string {
	if !bill.Pro {
		if bill.TrialUsed {
			return s.say.SettingsPlanLine()
		}
		return s.say.SettingsPlanLineTrial()
	}
	var parts []string
	interval, price := "", ""
	switch bill.Interval {
	case "yearly":
		interval, price = s.say.ProYearly(), bill.Yearly
	case "monthly":
		interval, price = s.say.ProMonthly(), bill.Monthly
	}
	if interval != "" {
		if price != "" && !bill.Canceling() && !bill.PastDue() {
			interval += " " + price
		}
		parts = append(parts, interval)
	}
	if bill.ViaPayPal() {
		parts = append(parts, s.say.SettingsPlanViaPayPal())
	}
	if bill.PastDue() {
		parts = append(parts, s.say.SettingsPlanPastDue())
	} else if !bill.PeriodEnd.IsZero() {
		date := s.clock.Date(bill.PeriodEnd, s.app.Now())
		switch {
		case bill.Trialing():
			parts = append(parts, s.say.SettingsPlanTrialEnd(date))
		case bill.Canceling():
			parts = append(parts, s.say.SettingsPlanEnds(date))
		default:
			parts = append(parts, s.say.SettingsPlanRenews(date))
		}
	}
	return strings.Join(parts, " · ")
}
