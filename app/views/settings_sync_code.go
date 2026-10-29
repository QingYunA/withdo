package views

import (
	"strings"
	"time"
	"unicode"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// The signed-out card (ui-spec, Settings > Account): 340 pt wide, 20 pt of padding, so 300 pt of
// fields. The code step's cells are 34 x 42 pt, 6 pt apart.
const (
	syncCardWidth      float32 = 340
	syncCardPadX       float32 = 20
	syncInnerWidth     float32 = syncCardWidth - 2*syncCardPadX
	syncCodeLen                = 6
	syncCodeCellW      float32 = 34
	syncCodeCellH      float32 = 42
	syncCodeGap        float32 = 6
	syncCodeDashWidth  float32 = 8
	syncCodeDashHeight float32 = 2
)

// syncCodeDigit is a digit of the code: 18 pt mono 600.
var syncCodeDigit = theme.Style{Size: 18, Line: 24, Weight: 600, Mono: true}

// syncLabelText is the 12 pt 500 label above a field, and syncSmall the quiet lines under a card.
var (
	syncLabelText = theme.Style{Size: 13, Line: 16, Weight: 500}
	syncSmall     = theme.Style{Size: 12, Line: 16, Weight: 400}
	syncTerms     = theme.Style{Size: 11.5, Line: 16, Weight: 400}
)

// syncLang is the language the code mails are written in: the app's, English or Simplified Chinese.
func syncLang() string {
	if strings.HasPrefix(strings.ToLower(i18n.Locale()), "zh") {
		return "zh-Hans"
	}
	return "en"
}

// syncCodeSignInCard is the signed-out card of a PocketBase server: the Sign in | Sign up tabs, or
// one of the steps that follow an action (Verify, Forgot password), or the wait for Google.
func (s *Settings) syncCodeSignInCard(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	googleWait := snap.Phase == data.SyncSigningIn && snap.SignInStage == data.SignInStageGoogle
	// The tabs share one key, so that switching them keeps the keyboard on the switch.
	key := "tabs"
	switch {
	case googleWait:
		key = "google"
	case !isTabStep(snap.SignInStep):
		key = string(snap.SignInStep)
	}
	ui.Column(c.Key("sync-auth-"+key)).FillWidth().Shrink(0).AlignItems(ui.Center).Margin(2, 0, 0).Children(func() {
		s.syncCard(c, t, "card", func() {
			switch {
			case googleWait:
				s.syncGoogleStep(c, t, svc)
			case snap.SignInStep == data.SignInStepVerify:
				s.syncVerifyStep(c, t, svc, snap)
			case snap.SignInStep == data.SignInStepResetEmail:
				s.syncResetEmailStep(c, t, svc, snap)
			case snap.SignInStep == data.SignInStepResetCode:
				s.syncResetCodeStep(c, t, svc, snap)
			default:
				s.syncTabs(c, t, svc, snap)
			}
		})
		switch {
		case googleWait:
		case snap.SignInStep == data.SignInStepVerify:
			s.syncNoteCard(c, t, s.say.SettingsSyncVerifyNote())
		case snap.SignInStep == data.SignInStepSignIn || snap.SignInStep == data.SignInStepSignUp:
			theme.Paragraph(c, syncSmall, t.TextTertiary, syncLocalNote(s.say), 0).
				TextAlign(ui.Center).Width(syncCardWidth).Margin(12, 0, 0)
		}
	})
}

// syncCard is the white 340 pt card the signed-out steps sit in.
func (s *Settings) syncCard(c *ui.Context, t *theme.Tokens, key string, body func()) {
	ui.Column(c.Key("sync-"+key)).Width(syncCardWidth).Shrink(0).Padding(16, syncCardPadX, 18).Radius(theme.SettingsRadius).
		Background(t.SettingsGroup()).Border(1, t.Border.Alpha(0.08)).Children(body)
}

// syncNoteCard is the quiet card under the verify step.
func (s *Settings) syncNoteCard(c *ui.Context, t *theme.Tokens, text string) {
	ui.Column(c.Key("sync-note-card")).Width(syncCardWidth).Shrink(0).Margin(16, 0, 0).Padding(12, 16).Radius(theme.SettingsRadius).
		Background(t.SettingsGroup()).Border(1, t.Border.Alpha(0.08)).Children(func() {
		theme.Paragraph(c, syncSmall, t.TextSecondary, text, 0)
	})
}

// syncHeroHead is the title every step after the tabs starts with.
func (s *Settings) syncHeroHead(c *ui.Context, t *theme.Tokens, title string) {
	hero := theme.Input
	hero.Weight = 600
	theme.Paragraph(c, hero, t.Text, title, 0).TextAlign(ui.Center)
}

// syncChangeEmailLine is the lead line and "Use a different email" under a step's title (the spec puts
// them on one line with a dot; at 300 pt they do not fit): the link goes to step with the address kept.
func (s *Settings) syncChangeEmailLine(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot, lead string, step data.SignInStep, busy bool) {
	if lead != "" {
		theme.Paragraph(c, theme.Meta, t.TextSecondary, lead, 0).TextAlign(ui.Center).Margin(2, 0, 0)
	}
	ui.Row(c.Key("sync-email-line")).Gap(4).AlignItems(ui.Center).Justify(ui.Center).Margin(0, 0, 0).Children(func() {
		link := syncLink(c, t, theme.Meta, "sync-change-email", s.say.SettingsSyncChangeEmail(), s.say.SettingsSyncChangeEmail())
		if busy {
			link.Disabled(true).Opacity(0.45)
		} else if link.Clicked() {
			s.syncLogin = snap.SignInEmail
			s.syncFocusAccount = true
			svc.SyncSetSignInStep(step)
		}
	})
}

// looksLikeEmail is the button's rule: something, an @, something with no space in it. The server
// is the judge of the rest.
func looksLikeEmail(raw string) bool {
	email := strings.TrimSpace(raw)
	at := strings.Index(email, "@")
	return at > 0 && at < len(email)-1 && !strings.ContainsAny(email, " \t")
}

// syncStatusLine is what sits under the button or the cells: the busy indicator while the server
// works, else the failure in status.blocked, unless it is one of skip (a step words those at the
// field they belong to).
func (s *Settings) syncStatusLine(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot, busy bool, skip ...data.SignInFailure) {
	if busy {
		s.syncBusyLine(c, t, snap)
		return
	}
	failure := snap.SignInFailure
	if failure == data.SignInFailureNone {
		return
	}
	for _, other := range skip {
		if failure == other {
			return
		}
	}
	theme.Paragraph(c, syncSmall, t.Blocked(), s.syncCodeFailureText(failure), 0).Margin(6, 2, 0)
}

// syncBusyLine is the pixel busy indicator with what the server is doing.
func (s *Settings) syncBusyLine(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot) {
	text := s.say.SettingsSyncCheckingAccount()
	switch snap.SignInStage {
	case data.SignInStageSending:
		text = s.say.SettingsSyncSendingCode()
	case data.SignInStageCreating:
		text = s.say.SettingsSyncCreatingAccount()
	case data.SignInStageDownloading:
		text = s.say.SettingsSyncDownloadingTasks()
	case data.SignInStageChecking, data.SignInStageNone:
		if snap.SignInStep == data.SignInStepVerify || snap.SignInStep == data.SignInStepResetCode {
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

// syncCodeFailureText words a failure of the sign-in card.
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
	case data.SignInFailureEmailTaken:
		return s.say.SettingsSyncEmailTaken()
	case data.SignInFailureInvalidEmail:
		return s.say.SettingsSyncInvalidEmail()
	case data.SignInFailureWeakPassword:
		return s.say.SettingsSyncPasswordMin()
	case data.SignInFailureGoogleCancelled:
		return s.say.SettingsSyncGoogleCancelled()
	case data.SignInFailureGoogleFailed:
		return s.say.SettingsSyncGoogleFailed()
	}
	return s.syncSignInFailureText(failure)
}

// syncVerifyStep is "Verify your email": six cells over one real text input, the address, and the
// way back to Sign up.
func (s *Settings) syncVerifyStep(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	busy := snap.Phase == data.SyncSigningIn
	s.syncHeroHead(c, t, s.say.SettingsSyncVerifyTitle())
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncVerifySent(snap.SignInEmail), 0).
		TextAlign(ui.Center).Margin(4, 0, 0)
	s.syncChangeEmailLine(c, t, svc, snap, s.say.SettingsSyncVerifyAuto(), data.SignInStepSignUp, busy)

	s.syncCode = digitsOnly(s.syncCode, syncCodeLen)
	if len(s.syncCode) == syncCodeLen && s.syncCode != s.syncCodeTried && !busy {
		s.syncCodeTried = s.syncCode
		svc.SyncSignInWithCode(s.syncCode)
	}
	s.syncCodeCells(c, t, busy)
	s.syncFocusCode = false
	s.syncCodeFooter(c, t, svc, snap, busy)
}

// syncCodeFooter is what sits under the cells: the busy line or the failure, and the resend
// countdown and link.
func (s *Settings) syncCodeFooter(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot, busy bool) {
	if busy {
		s.syncBusyLine(c, t, snap)
		return
	}
	failed := snap.SignInFailure != data.SignInFailureNone
	if failed {
		theme.Paragraph(c, theme.Meta, t.Blocked(), s.syncCodeFailureText(snap.SignInFailure), 0).
			TextAlign(ui.Center).Margin(10, 0, 0)
	}
	left := snap.ResendAt.Sub(s.app.Now())
	switch {
	case left > 0:
		c.After(time.Second)
		if !failed {
			theme.Paragraph(c, syncSmall, t.TextTertiary, s.say.SettingsSyncResendIn(countdown(left)), 0).
				TextAlign(ui.Center).Margin(10, 0, 0)
		}
	default:
		ui.Row(c.Key("sync-resend-row")).Justify(ui.Center).Margin(6, 0, 0).Children(func() {
			if syncLink(c, t, theme.Meta, "sync-resend", s.say.SettingsSyncResendCode(), s.say.SettingsSyncResendCode()).Clicked() {
				svc.SyncResendCode()
			}
		})
	}
}

// syncCodeCells draws the six cells, a dash after the third, and over them the one input that
// takes the typing, a paste and an input method's text. The first empty cell has the focus ring.
func (s *Settings) syncCodeCells(c *ui.Context, t *theme.Tokens, busy bool) {
	digits := []rune(s.syncCode)
	ui.Row(c.Key("sync-code-cells")).Gap(syncCodeGap).Margin(14, 0, 4).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		for i := 0; i < syncCodeLen; i++ {
			if i == 3 {
				ui.Box(c.Key("sync-code-dash")).Size(syncCodeDashWidth, syncCodeDashHeight).Shrink(0).Background(t.TextTertiary)
			}
			cell := ui.Box(c.Key("sync-code-cell-"+itoa(i))).Size(syncCodeCellW, syncCodeCellH).Shrink(0).Center().
				Radius(theme.RadiusControl).Background(t.BgSubtle).Border(theme.Hairline, t.Border) // cells read on the white card
			if busy {
				cell.Opacity(0.5)
			} else if i == len(digits) {
				cell.Border(theme.Hairline, t.Accent)
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

// syncResetEmailStep is Forgot password's first step: the email (kept from Sign in) and Send code.
func (s *Settings) syncResetEmailStep(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	busy := snap.Phase == data.SyncSigningIn
	s.syncHeroHead(c, t, s.say.SettingsSyncResetTitle())
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncResetEmailBody(), 0).TextAlign(ui.Center).Margin(4, 0, 4)
	send := func() {
		if !busy && looksLikeEmail(s.syncLogin) {
			svc.SyncRequestResetCode(strings.TrimSpace(s.syncLogin))
		}
	}
	s.syncEmailField(c, t, busy, false, send)
	button := settingsButton(c, t, "sync-send-code", s.say.SettingsSyncSendCode(), true).FillWidth().Height(32).Margin(14, 0, 0)
	if busy || !looksLikeEmail(s.syncLogin) {
		button.Disabled(true).Opacity(0.45)
	}
	if button.Clicked() {
		send()
	}
	s.syncStatusLine(c, t, snap, busy)
	s.syncBackToSignIn(c, t, svc, busy)
}

// syncResetCodeStep is Forgot password's second step: the six cells, the new password with its
// bar, and Set and sign in.
func (s *Settings) syncResetCodeStep(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	busy := snap.Phase == data.SyncSigningIn
	s.syncHeroHead(c, t, s.say.SettingsSyncResetTitle())
	theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncResetSent(snap.SignInEmail), 0).
		TextAlign(ui.Center).Margin(4, 0, 0)
	s.syncChangeEmailLine(c, t, svc, snap, "", data.SignInStepResetEmail, busy)
	s.syncCode = digitsOnly(s.syncCode, syncCodeLen)
	if len(s.syncCode) == syncCodeLen && s.syncCode != s.syncCodeTried {
		s.syncCodeTried = s.syncCode
		s.syncFocusPassword = true // the sixth digit moves on to the new password
	}
	s.syncCodeCells(c, t, busy)
	s.syncFocusCode = false

	ready := func() bool { return len(s.syncCode) == syncCodeLen && passwordLongEnough(s.syncPassword) }
	submit := func() {
		if !busy && ready() {
			svc.SyncResetPassword(s.syncCode, s.syncPassword)
		}
	}
	ui.Column(c.Key("sync-pass-block")).Shrink(0).Children(func() {
		s.syncLabel(c, t, s.say.SettingsSyncNewPassword())
		s.syncPasswordWell(c, t, "sync-password", s.say.SettingsSyncNewPassword(), busy,
			snap.SignInFailure == data.SignInFailureWeakPassword, submit)
		s.syncStrengthBar(c, t)
		s.syncPasswordHint(c, t, snap)
	})
	button := settingsButton(c, t, "sync-reset-done", s.say.SettingsSyncResetDone(), true).FillWidth().Height(32).Margin(14, 0, 0)
	if busy || !ready() {
		button.Disabled(true).Opacity(0.45)
	}
	if button.Clicked() {
		submit()
	}
	if busy {
		s.syncBusyLine(c, t, snap)
	} else {
		s.syncStatusLine(c, t, snap, false, data.SignInFailureWeakPassword)
		left := snap.ResendAt.Sub(s.app.Now())
		if left > 0 {
			c.After(time.Second)
		} else if snap.SignInFailure != data.SignInFailureNone || len(s.syncCode) < syncCodeLen {
			ui.Row(c.Key("sync-resend-row")).Justify(ui.Center).Margin(6, 0, 0).Children(func() {
				if syncLink(c, t, theme.Meta, "sync-resend", s.say.SettingsSyncResendCode(), s.say.SettingsSyncResendCode()).Clicked() {
					svc.SyncResendCode()
				}
			})
		}
	}
	s.syncBackToSignIn(c, t, svc, busy)
}

// syncBackToSignIn is the quiet link under both reset steps.
func (s *Settings) syncBackToSignIn(c *ui.Context, t *theme.Tokens, svc SyncService, busy bool) {
	ui.Row(c.Key("sync-back-row")).Justify(ui.Center).Margin(10, 0, 0).Children(func() {
		link := syncLink(c, t, theme.Meta, "sync-back", s.say.SettingsSyncBackToSignIn(), s.say.SettingsSyncBackToSignIn())
		if busy {
			link.Disabled(true).Opacity(0.45)
		} else if link.Clicked() {
			svc.SyncSetSignInStep(data.SignInStepSignIn)
		}
	})
}

// isTabStep says whether step is one of the card's two tabs.
func isTabStep(step data.SignInStep) bool {
	return step == data.SignInStepSignIn || step == data.SignInStepSignUp
}

// syncNoteStepChange resets what a step keeps when the card moves to another one: the code and the
// password are emptied, and the keyboard goes where the step's work starts.
func (s *Settings) syncNoteStepChange(snap data.SyncSnapshot) {
	if snap.SignInStep == s.syncPrevStep {
		return
	}
	between := isTabStep(s.syncPrevStep) && isTabStep(snap.SignInStep)
	s.syncPrevStep = snap.SignInStep
	s.syncCode, s.syncCodeTried, s.syncFocusCode = "", "", true
	s.syncPassword, s.syncShowPassword = "", false
	if between {
		return // the switch keeps the keyboard, so Left and Right go on working
	}
	switch snap.SignInStep {
	case data.SignInStepSignIn, data.SignInStepSignUp:
		// The email is kept across the tabs: with one in the field the keyboard goes on to the password.
		s.syncFocusAccount = strings.TrimSpace(s.syncLogin) == ""
		s.syncFocusPassword = !s.syncFocusAccount
	case data.SignInStepResetEmail:
		s.syncFocusAccount, s.syncFocusPassword = true, false
	default:
		s.syncFocusPassword = false
	}
}
