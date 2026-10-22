package views

import (
	"strings"
	"time"
	"unicode"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The code step's cells (ui-spec, Settings > Account > Code).
const (
	syncCodeLen        = 6
	syncCodeCellW      = 38
	syncCodeCellH      = 46
	syncCodeGap        = 8
	syncFormWidth      = 280
	syncCodeDashWidth  = 8
	syncCodeDashHeight = 2
)

// syncCodeDigit is a digit of the code: 22 pt mono 600.
var syncCodeDigit = theme.Style{Size: 22, Line: 28, Weight: 600, Mono: true}

// syncCodeSignInCard is the signed-out card of a PocketBase server: an email, then a 6-digit code
// mailed to it, with a password as the optional other way.
func (s *Settings) syncCodeSignInCard(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	group(c, t, "sync-hero", func() {
		ui.Column(c.Key("sync-hero-"+string(snap.SignInStep))).FillWidth().AlignItems(ui.Center).Padding(28, 40, 18).Children(func() {
			switch snap.SignInStep {
			case data.SignInStepCode:
				s.syncCodeStep(c, t, svc, snap)
			case data.SignInStepPassword:
				s.syncPasswordStep(c, t, svc, snap)
			case data.SignInStepResetSent:
				s.syncResetStep(c, t, svc, snap)
			default:
				s.syncEmailStep(c, t, svc, snap)
			}
		})
	})
}

// syncHeroHead is the icon and the title every step starts with.
func (s *Settings) syncHeroHead(c *ui.Context, t *theme.Tokens, icon *ui.SVG, title string) {
	ui.Icon(c, icon).Size(theme.IconLarge, theme.IconLarge).TextColor(t.Accent)
	hero := theme.Input
	hero.Weight = 600
	theme.Paragraph(c, hero, t.Text, title, 0).TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(12, 0, 0)
}

// syncEmailLine is "<lead> <email> · Use a different email": the address in text, the link back to
// the email step with the address kept.
func (s *Settings) syncEmailLine(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot, lead string, busy bool) {
	ui.Row(c.Key("sync-email-line")).Gap(4).AlignItems(ui.Center).Justify(ui.Center).Margin(6, 0, 0).Children(func() {
		if lead != "" {
			theme.Text(c, theme.Meta, t.TextSecondary, lead)
		}
		theme.Text(c, theme.Meta, t.Text, snap.SignInEmail)
		theme.Text(c, theme.Meta, t.TextSecondary, "·")
		link := settingsLink(c, t, "sync-change-email", s.say.SettingsSyncChangeEmail(), s.say.SettingsSyncChangeEmail())
		if busy {
			link.Disabled(true).Opacity(0.45)
		} else if link.Clicked() {
			s.syncLogin = snap.SignInEmail
			s.syncFocusAccount = true
			svc.SyncSetSignInStep(data.SignInStepEmail)
		}
	})
}

// syncEmailStep is "Sign in or sign up": one email field and Continue with email.
func (s *Settings) syncEmailStep(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	busy := snap.Phase == data.SyncSigningIn
	s.syncHeroHead(c, t, theme.IconUser, s.say.SettingsSyncEmailTitle())
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncEmailBody(), 0).
		TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(6, 0, 0)
	ui.Column(c.Key("sync-form")).Width(syncFormWidth).Shrink(0).Margin(16, 0, 0).Children(func() {
		well := fieldWell(c, t, "sync-email", syncFormWidth, func() {
			entered, _ := draftField(c, t, s.syncFields, draftSpec{
				name: "sync-email", label: s.say.SettingsSyncEmail(),
				placeholder: s.say.SettingsSyncEmailPlaceholder(), focus: s.syncFocusAccount,
				disabled: busy,
			}, &s.syncLogin)
			if entered {
				s.submitSyncEmail(svc, busy)
			}
		})
		if busy {
			well.Opacity(0.5)
		} else if well.FocusWithin() {
			well.Border(1, t.Accent)
		}
	})
	s.syncFocusAccount = false
	button := settingsButton(c, t, "sync-continue-email", s.say.SettingsSyncContinueEmail(), true).Width(syncFormWidth).Margin(8, 0, 0)
	if busy || !looksLikeEmail(s.syncLogin) {
		button.Disabled(true).Opacity(0.45)
	}
	if button.Clicked() {
		s.submitSyncEmail(svc, busy)
	}
	s.syncStatusLine(c, t, snap, busy)
	theme.Paragraph(c, theme.Meta, t.TextTertiary, s.say.SettingsSyncLocalNote(), 0).
		TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(8, 0, 0)
}

// submitSyncEmail asks for a code for the typed address.
func (s *Settings) submitSyncEmail(svc SyncService, busy bool) {
	if busy || !looksLikeEmail(s.syncLogin) {
		return
	}
	svc.SyncRequestCode(strings.TrimSpace(s.syncLogin))
}

// looksLikeEmail is the button's rule: something, an @, something with no space in it. The server
// is the judge of the rest.
func looksLikeEmail(raw string) bool {
	email := strings.TrimSpace(raw)
	at := strings.Index(email, "@")
	return at > 0 && at < len(email)-1 && !strings.ContainsAny(email, " \t")
}

// syncStatusLine is what sits under the button or the cells: the busy indicator while the
// server works, else the failure in status.blocked.
func (s *Settings) syncStatusLine(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot, busy bool) {
	switch {
	case busy:
		s.syncBusyLine(c, t, snap)
	case snap.SignInFailure != data.SignInFailureNone:
		theme.Paragraph(c, theme.Meta, t.Blocked(), s.syncCodeFailureText(snap.SignInFailure), 0).
			TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(10, 0, 0)
	}
}

// syncBusyLine is the pixel busy indicator with what the server is doing.
func (s *Settings) syncBusyLine(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot) {
	text := s.say.SettingsSyncCheckingAccount()
	switch snap.SignInStage {
	case data.SignInStageSending:
		text = s.say.SettingsSyncSendingCode()
	case data.SignInStageDownloading:
		text = s.say.SettingsSyncDownloadingTasks()
	case data.SignInStageChecking, data.SignInStageNone:
		if snap.SignInStep == data.SignInStepCode {
			text = s.say.SettingsSyncCheckingCode()
		}
	}
	reduce := c.Preferences().ReduceMotion
	if !reduce {
		c.After(theme.SpinnerStep)
	}
	ui.Row(c.Key("sync-busy")).Gap(6).AlignItems(ui.Center).Justify(ui.Center).Margin(12, 0, 0).Children(func() {
		theme.PixelSpinner(c, theme.IconSmall, t.Accent, theme.SpinnerFrame(s.app.Now(), reduce))
		theme.Text(c, theme.Meta, t.TextSecondary, text)
	})
}

// syncCodeFailureText words a failure of the code flow or of the password step.
func (s *Settings) syncCodeFailureText(failure data.SignInFailure) string {
	switch failure {
	case data.SignInFailureWrongCode:
		return s.say.SettingsSyncWrongCode()
	case data.SignInFailureCodeExpired:
		return s.say.SettingsSyncCodeExpired()
	case data.SignInFailureTooMany:
		return s.say.SettingsSyncTooManyTriesMinute()
	case data.SignInFailureCredentials:
		return s.say.SettingsSyncWrongEmailPassword()
	}
	return s.syncSignInFailureText(failure)
}

// syncCodeStep is "Check your email": six cells over one real text input.
func (s *Settings) syncCodeStep(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	busy := snap.Phase == data.SyncSigningIn
	s.syncHeroHead(c, t, theme.IconMail, s.say.SettingsSyncCodeTitle())
	s.syncEmailLine(c, t, svc, snap, s.say.SettingsSyncCodeSentTo(), busy)

	s.syncCode = digitsOnly(s.syncCode, syncCodeLen)
	if len(s.syncCode) == syncCodeLen && s.syncCode != s.syncCodeTried && !busy {
		s.syncCodeTried = s.syncCode
		svc.SyncSignInWithCode(s.syncCode)
	}
	s.syncCodeCells(c, t, busy)
	s.syncFocusCode = false

	if busy {
		s.syncBusyLine(c, t, snap)
		return
	}
	if snap.SignInFailure != data.SignInFailureNone {
		theme.Paragraph(c, theme.Meta, t.Blocked(), s.syncCodeFailureText(snap.SignInFailure), 0).
			TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(10, 0, 0)
	}
	left := snap.ResendAt.Sub(s.app.Now())
	if left > 0 {
		c.After(time.Second)
		if snap.SignInFailure == data.SignInFailureNone {
			theme.Text(c, theme.Meta, t.TextTertiary, s.say.SettingsSyncResendIn(countdown(left))).Margin(10, 0, 0)
		}
	} else if settingsLink(c, t, "sync-resend", s.say.SettingsSyncResendCode(), s.say.SettingsSyncResendCode()).Margin(8, 0, 0).Clicked() {
		svc.SyncResendCode()
	}
	if settingsLink(c, t, "sync-use-password", s.say.SettingsSyncUsePassword(), s.say.SettingsSyncUsePassword()).Margin(4, 0, 0).Clicked() {
		svc.SyncSetSignInStep(data.SignInStepPassword)
	}
}

// syncCodeCells draws the six cells, a dash after the third, and over them the one input that
// takes the typing, a paste and an input method's text. The first empty cell has the focus ring.
func (s *Settings) syncCodeCells(c *ui.Context, t *theme.Tokens, busy bool) {
	digits := []rune(s.syncCode)
	ui.Row(c.Key("sync-code-cells")).Gap(syncCodeGap).Margin(16, 0, 0).Shrink(0).AlignItems(ui.Center).Children(func() {
		for i := 0; i < syncCodeLen; i++ {
			if i == 3 {
				ui.Box(c.Key("sync-code-dash")).Size(syncCodeDashWidth, syncCodeDashHeight).Shrink(0).Background(t.TextTertiary)
			}
			cell := ui.Box(c.Key("sync-code-cell-"+itoa(i))).Size(syncCodeCellW, syncCodeCellH).Shrink(0).Center().
				Radius(theme.RadiusControl).Background(t.Bg).Border(1, t.ComposerBorder())
			if busy {
				cell.Opacity(0.5)
			} else if i == len(digits) {
				cell.Border(1, t.Accent)
			}
			if i < len(digits) {
				cell.Children(func() { theme.Text(c, syncCodeDigit, t.Text, string(digits[i])) })
			}
		}
		field := ui.TextInputBase(c.Key("sync-code"), &s.syncCode).Absolute().Left(0).Top(0).Right(0).Bottom(0).Opacity(0).
			Label(s.say.SettingsSyncCodeLabel()).OnPaste(func(text string) bool {
			if code := digitsOnly(text, syncCodeLen); code != "" {
				s.syncCode = code
				return true
			}
			return true
		})
		if busy {
			field.Disabled(true).ReadOnly(true)
		}
		if s.syncFocusCode || !busy {
			field.AutoFocus()
		}
		if s.syncFocusCode && !busy {
			field.Focus()
		}
	})
}

// digitsOnly keeps the digits of text, at most n of them.
func digitsOnly(text string, n int) string {
	var b strings.Builder
	for _, r := range text {
		if unicode.IsDigit(r) && r < 0x80 {
			b.WriteRune(r)
			if b.Len() == n {
				break
			}
		}
	}
	return b.String()
}

// countdown is the time left to resend as m:ss, counted up to the whole second.
func countdown(left time.Duration) string {
	secs := int((left + time.Second - 1) / time.Second)
	m, sec := secs/60, secs%60
	return itoa(m) + ":" + string(rune('0'+sec/10)) + string(rune('0'+sec%10))
}

// syncPasswordStep is "Sign in with a password".
func (s *Settings) syncPasswordStep(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	busy := snap.Phase == data.SyncSigningIn
	s.syncHeroHead(c, t, theme.IconUser, s.say.SettingsSyncPasswordTitle())
	s.syncEmailLine(c, t, svc, snap, "", busy)
	submit := func() {
		if busy || s.syncPassword == "" {
			return
		}
		svc.SyncSignIn(snap.SignInEmail, s.syncPassword)
	}
	ui.Column(c.Key("sync-form")).Width(syncFormWidth).Shrink(0).Margin(16, 0, 0).Children(func() {
		well := fieldWell(c, t, "sync-password", syncFormWidth, func() {
			entered, _ := draftField(c, t, s.syncFields, draftSpec{
				name: "sync-password", label: s.say.SettingsSyncPassword(),
				placeholder: s.say.SettingsSyncPassword(), secret: true,
				focus: true, focusNow: s.syncFocusPassword, disabled: busy,
			}, &s.syncPassword)
			if entered {
				submit()
			}
		})
		if busy {
			well.Opacity(0.5)
		} else if well.FocusWithin() {
			well.Border(1, t.Accent)
		}
	})
	s.syncFocusPassword = false
	button := settingsButton(c, t, "sync-signin", s.say.SettingsSyncSignInShort(), true).Width(syncFormWidth).Margin(8, 0, 0)
	if busy || s.syncPassword == "" {
		button.Disabled(true).Opacity(0.45)
	}
	if button.Clicked() {
		submit()
	}
	s.syncStatusLine(c, t, snap, busy)
	if busy {
		return
	}
	ui.Row(c.Key("sync-password-links")).Gap(4).AlignItems(ui.Center).Margin(8, 0, 0).Children(func() {
		if settingsLink(c, t, "sync-forgot", s.say.SettingsSyncForgotPassword(), s.say.SettingsSyncForgotPassword()).Clicked() {
			svc.SyncRequestPasswordReset(snap.SignInEmail)
		}
		theme.Text(c, theme.Meta, t.TextTertiary, "·")
		if settingsLink(c, t, "sync-use-code", s.say.SettingsSyncUseCode(), s.say.SettingsSyncUseCode()).Clicked() {
			svc.SyncRequestCode(snap.SignInEmail)
		}
	})
}

// syncResetStep is the envelope after a reset link was mailed.
func (s *Settings) syncResetStep(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	s.syncHeroHead(c, t, theme.IconMail, s.say.SettingsSyncResetTitle())
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncResetBody(snap.SignInEmail), 0).
		TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(6, 0, 0)
	if settingsButton(c, t, "sync-back", s.say.SettingsSyncBackToSignIn(), false).Margin(14, 0, 0).Clicked() {
		svc.SyncSetSignInStep(data.SignInStepPassword)
	}
	ui.Row(c.Key("sync-reset-or")).AlignItems(ui.Center).Margin(10, 0, 0).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncResetOrPrefix())
		if settingsLink(c, t, "sync-use-code", s.say.SettingsSyncUseCode(), s.say.SettingsSyncUseCode()).Clicked() {
			svc.SyncRequestCode(snap.SignInEmail)
		}
		theme.Text(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncResetOrSuffix())
	})
}

// syncNoteStepChange resets what a step keeps when the card moves to another one: the code is
// emptied and focused on arriving at the code step, the password likewise.
func (s *Settings) syncNoteStepChange(snap data.SyncSnapshot) {
	if snap.SignInStep == s.syncPrevStep {
		return
	}
	s.syncPrevStep = snap.SignInStep
	s.syncCode, s.syncCodeTried, s.syncFocusCode = "", "", true
	s.syncPassword = ""
	s.syncFocusPassword = snap.SignInStep == data.SignInStepPassword
	if snap.SignInStep == data.SignInStepEmail {
		s.syncFocusAccount = true
	}
}
