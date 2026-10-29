package views

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The terms and the privacy policy a sign-up agrees to, on the website.
const termsURL = "https://qingyuna.github.io/withdo/terms.html"

// syncPasswordMin is the shortest password the server takes; the strength bar has four steps.
const (
	syncPasswordMin   = 8
	syncStrengthSteps = 4
)

// syncTabs is the card's first screen: the Sign in | Sign up switch, Google, and the form of the
// tab picked. Switching keeps the email.
func (s *Settings) syncTabs(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	busy := snap.Phase == data.SyncSigningIn
	signUp := snap.SignInStep == data.SignInStepSignUp
	selected := 0
	if signUp {
		selected = 1
	}
	labels := []string{s.say.SettingsSyncSignInShort(), s.say.SettingsSyncSignUpShort()}
	if pick := segmentedFill(c, t, "sync-tabs", s.say.SettingsSyncTabs(), labels, selected); pick >= 0 && !busy {
		if pick == 1 {
			svc.SyncSetSignInStep(data.SignInStepSignUp)
		} else {
			svc.SyncSetSignInStep(data.SignInStepSignIn)
		}
	}
	if snap.GoogleAvailable {
		s.syncGoogleButton(c, t, svc, busy)
	}
	if signUp {
		s.syncSignUpForm(c, t, svc, snap, busy)
	} else {
		s.syncSignInForm(c, t, svc, snap, busy)
	}
}

// syncLink is a link of the card in the size of the text around it.
func syncLink(c *ui.Context, t *theme.Tokens, style theme.Style, key, text, label string) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Padding(2, 4).Radius(theme.RadiusControl).Shrink(0).Label(label).Cursor(ui.CursorPointer)
	ink := t.Accent
	if btn.Hovered() {
		ink = t.Accent.Mix(t.Text, 0.2)
	}
	btn.Children(func() { theme.Text(c, style, ink, text) })
	return btn
}

// syncLabel is the label above a field.
func (s *Settings) syncLabel(c *ui.Context, t *theme.Tokens, text string) {
	theme.Text(c, syncLabelText, t.TextSecondary, text).Margin(10, 2, 4)
}

// syncEmailField is the Email label and field; bad outlines it in status.blocked. Enter calls
// onEnter.
func (s *Settings) syncEmailField(c *ui.Context, t *theme.Tokens, busy, bad bool, onEnter func()) {
	s.syncLabel(c, t, s.say.SettingsSyncEmail())
	well := fieldWell(c, t, "sync-email", 0, func() {
		entered, _ := draftField(c, t, s.syncFields, draftSpec{
			name: "sync-email", label: s.say.SettingsSyncEmail(),
			placeholder: s.say.SettingsSyncEmailPlaceholder(), focus: s.syncFocusAccount,
			focusNow: s.syncFocusAccount, disabled: busy,
		}, &s.syncLogin)
		if entered {
			onEnter()
		}
	})
	s.syncFocusAccount = false
	switch {
	case busy:
		well.Opacity(0.5)
	case bad:
		well.Border(theme.Hairline, t.Blocked())
	case well.FocusWithin():
		well.Border(theme.Hairline, t.Accent)
	}
}

// syncPasswordWell is the password field with its quiet Show at the right end. It keeps its text
// in s.syncPassword; bad outlines it in status.blocked.
func (s *Settings) syncPasswordWell(c *ui.Context, t *theme.Tokens, key, label string, busy, bad bool, onEnter func()) {
	well := fieldWellTrailing(c, t, key, 0, func() {
		entered, _ := draftField(c, t, s.syncFields, draftSpec{
			name: key, label: label, secret: !s.syncShowPassword,
			focusNow: s.syncFocusPassword, disabled: busy,
		}, &s.syncPassword)
		if entered {
			onEnter()
		}
	}, func() {
		text := s.say.SettingsSyncShowPassword()
		if s.syncShowPassword {
			text = s.say.SettingsSyncHidePassword()
		}
		// Show is for the pointer: Tab goes from the email to the password to the button.
		toggle := ui.Box(c.Key("sync-show")).Padding(2, 4).Shrink(0).Cursor(ui.CursorPointer).Label(text)
		toggle.Children(func() { theme.Text(c, syncSmall, t.TextTertiary, text) })
		if !busy && toggle.Clicked() {
			s.syncShowPassword = !s.syncShowPassword
			s.syncFocusPassword = true
		}
	})
	s.syncFocusPassword = false
	switch {
	case busy:
		well.Opacity(0.5)
	case bad:
		well.Border(theme.Hairline, t.Blocked())
	case well.FocusWithin():
		well.Border(theme.Hairline, t.Accent)
	}
}

// syncSignInForm is the Sign in tab: email, password with Forgot password?, the button, and the
// way to Sign up.
func (s *Settings) syncSignInForm(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot, busy bool) {
	ready := func() bool { return strings.TrimSpace(s.syncLogin) != "" && s.syncPassword != "" }
	submit := func() {
		if !busy && ready() {
			svc.SyncSignIn(strings.TrimSpace(s.syncLogin), s.syncPassword)
		}
	}
	wrong := snap.SignInFailure == data.SignInFailureCredentials
	s.syncEmailField(c, t, busy, false, func() {
		if s.syncPassword == "" {
			s.syncFocusPassword = true // Enter on the email goes on to the password
		}
		submit()
	})
	// The password block ends with the button and Forgot password?, so that Tab reaches the password
	// and then the button; the link is drawn at the right of the label.
	ui.Column(c.Key("sync-pass-block")).Shrink(0).Children(func() {
		s.syncLabel(c, t, s.say.SettingsSyncPassword())
		s.syncPasswordWell(c, t, "sync-password", s.say.SettingsSyncPassword(), busy, wrong, submit)
		s.syncStatusLine(c, t, snap, false)
		if busy {
			s.syncBusyLine(c, t, snap)
		}
		button := settingsButton(c, t, "sync-signin", s.say.SettingsSyncSignInShort(), true).FillWidth().Height(32).Margin(14, 0, 0)
		if busy || !ready() {
			button.Disabled(true).Opacity(0.45)
		}
		if button.Clicked() {
			submit()
		}
		link := syncLink(c, t, theme.Meta, "sync-forgot", s.say.SettingsSyncForgotPassword(), s.say.SettingsSyncForgotPassword()).
			Absolute().Right(0).Top(4)
		if busy {
			link.Disabled(true).Opacity(0.45)
		} else if link.Clicked() {
			svc.SyncSetSignInStep(data.SignInStepResetEmail)
		}
	})
	s.syncSwap(c, t, svc, busy, s.say.SettingsSyncNoAccount(), s.say.SettingsSyncSignUpShort(), data.SignInStepSignUp)
}

// syncSwap is "<question> <link>" under a tab's button; the link picks the other tab, the email kept.
func (s *Settings) syncSwap(c *ui.Context, t *theme.Tokens, svc SyncService, busy bool, question, link string, step data.SignInStep) {
	ui.Row(c.Key("sync-swap")).Gap(2).AlignItems(ui.Center).Justify(ui.Center).Margin(12, 0, 0).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, question)
		l := syncLink(c, t, theme.Meta, "sync-swap-link", link, question+" "+link)
		if busy {
			l.Disabled(true).Opacity(0.45)
		} else if l.Clicked() {
			svc.SyncSetSignInStep(step)
		}
	})
}

// syncSignUpForm is the Sign up tab: email, a password with its strength bar, Create account and the
// terms. An email that already has an account says so under its field, with a link to Sign in.
func (s *Settings) syncSignUpForm(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot, busy bool) {
	ready := func() bool { return looksLikeEmail(s.syncLogin) && passwordLongEnough(s.syncPassword) }
	submit := func() {
		if !busy && ready() {
			svc.SyncSignUp(strings.TrimSpace(s.syncLogin), s.syncPassword, syncLang())
		}
	}
	taken := snap.SignInFailure == data.SignInFailureEmailTaken
	invalid := snap.SignInFailure == data.SignInFailureInvalidEmail
	s.syncEmailField(c, t, busy, taken || invalid, func() {
		if s.syncPassword == "" {
			s.syncFocusPassword = true
		}
		submit()
	})
	switch {
	case taken:
		ui.Row(c.Key("sync-taken")).Gap(4).AlignItems(ui.Center).Margin(5, 2, 0).Children(func() {
			theme.Text(c, syncSmall, t.Blocked(), s.say.SettingsSyncEmailTaken())
			if syncLink(c, t, syncSmall, "sync-taken-signin", s.say.SettingsSyncGoSignIn(), s.say.SettingsSyncEmailTaken()+" "+s.say.SettingsSyncGoSignIn()).Clicked() {
				svc.SyncSetSignInStep(data.SignInStepSignIn)
			}
		})
	case invalid:
		theme.Paragraph(c, syncSmall, t.Blocked(), s.say.SettingsSyncInvalidEmail(), 0).Margin(5, 2, 0)
	}
	ui.Column(c.Key("sync-pass-block")).Shrink(0).Children(func() {
		s.syncLabel(c, t, s.say.SettingsSyncPassword())
		s.syncPasswordWell(c, t, "sync-password", s.say.SettingsSyncPassword(), busy,
			snap.SignInFailure == data.SignInFailureWeakPassword, submit)
		s.syncStrengthBar(c, t)
		s.syncPasswordHint(c, t, snap)
	})
	button := settingsButton(c, t, "sync-create", s.say.SettingsSyncCreateAccount(), true).FillWidth().Height(32).Margin(14, 0, 0)
	if busy || !ready() {
		button.Disabled(true).Opacity(0.45)
	}
	if button.Clicked() {
		submit()
	}
	s.syncStatusLine(c, t, snap, busy, data.SignInFailureEmailTaken, data.SignInFailureInvalidEmail, data.SignInFailureWeakPassword)
	s.syncTermsLine(c, t)
}

// syncTermsLine is the 11.5 pt line under Create account, with the two links to the website.
func (s *Settings) syncTermsLine(c *ui.Context, t *theme.Tokens) {
	ui.Column(c.Key("sync-terms")).FillWidth().Shrink(0).AlignItems(ui.Center).Margin(10, 0, 0).Children(func() {
		theme.Paragraph(c, syncTerms, t.TextTertiary, s.say.SettingsSyncTermsLead(), 0).TextAlign(ui.Center)
		ui.Row(c.Key("sync-terms-links")).Gap(3).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			if syncLink(c, t, syncTerms, "sync-terms-tos", s.say.SettingsSyncTermsOfService(), s.say.SettingsSyncTermsOfService()).Clicked() {
				s.app.Act.OpenURL(termsURL)
			}
			theme.Text(c, syncTerms, t.TextTertiary, s.say.SettingsSyncTermsAnd())
			if syncLink(c, t, syncTerms, "sync-terms-privacy", s.say.SettingsSyncPrivacyPolicy(), s.say.SettingsSyncPrivacyPolicy()).Clicked() {
				s.app.Act.OpenURL(privacyURL)
			}
		})
	})
}

// syncPasswordHint is "At least 8 characters." under the strength bar, in status.blocked when the
// server refused a short password.
func (s *Settings) syncPasswordHint(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot) {
	ink := t.TextSecondary
	if snap.SignInFailure == data.SignInFailureWeakPassword {
		ink = t.Blocked()
	}
	theme.Paragraph(c, syncSmall, ink, s.say.SettingsSyncPasswordMin(), 0).Margin(5, 2, 0)
}

// syncStrengthBar is four steps in the done green under a new password: the length and the kinds
// of characters.
func (s *Settings) syncStrengthBar(c *ui.Context, t *theme.Tokens) {
	score := passwordStrength(s.syncPassword)
	ui.Row(c.Key("sync-strength")).FillWidth().Shrink(0).Gap(3).Margin(6, 0, 0).Children(func() {
		for i := 0; i < syncStrengthSteps; i++ {
			ink := t.Border
			if i < score {
				ink = t.Status(data.Done)
			}
			ui.Box(c.Key("sync-strength-" + itoa(i))).Grow(1).Height(3).Background(ink)
		}
	})
}

// passwordLongEnough is the Create account rule: at least 8 characters.
func passwordLongEnough(password string) bool {
	return utf8.RuneCountInString(password) >= syncPasswordMin
}

// passwordStrength is 0 for a password under 8 characters, else one step for the length, one more
// from 12, one for two kinds of characters (lower case, upper case, digits, the rest) and one for
// three kinds from 10 characters.
func passwordStrength(password string) int {
	n := utf8.RuneCountInString(password)
	if n < syncPasswordMin {
		return 0
	}
	var lower, upper, digit, other bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		default:
			other = true
		}
	}
	kinds := 0
	for _, k := range []bool{lower, upper, digit, other} {
		if k {
			kinds++
		}
	}
	score := 1
	if n >= 12 {
		score++
	}
	if kinds >= 2 {
		score++
	}
	if kinds >= 3 && n >= 10 {
		score++
	}
	return score
}
