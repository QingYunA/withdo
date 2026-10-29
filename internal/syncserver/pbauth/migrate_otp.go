package pbauth

import (
	"github.com/pocketbase/pocketbase/core"
)

const (
	codeLength       = 6
	codeSeconds      = 600
	codeSubject      = core.EmailPlaceholderOTP + " is your " + appName + " code"
	mailsPerMinute   = 3
	guessesPerMinute = 10
)

// codeBody is the mail with the sign-in code. English only: the product is for people outside China.
const codeBody = `<div style="font-family:-apple-system,Segoe UI,Helvetica,Arial,sans-serif;color:#1d1d1f;max-width:420px">
<p>Your sign-in code:</p>
<p style="font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:34px;font-weight:600;letter-spacing:8px;margin:16px 0">` + core.EmailPlaceholderOTP + `</p>
<p>It works for 10 minutes. If you didn't ask for it, you can ignore this email; nobody can sign in without the code.</p>
<p style="color:#6e6e73">&mdash; ` + appName + ` &middot; todo.sri-ai.xyz</p>
</div>`

func init() {
	core.AppMigrations.Register(enableCodeSignIn, nil, "1790000200_adhd_users_otp.go")
}

// enableCodeSignIn turns on the one-time code for the users collection (P2; since v42 the code
// verifies an email at sign-up and resets a password, see migrate_signup.go), keeps password sign-in, keeps MFA off and
// limits the three mail-sending or guessable endpoints. PocketBase matches a rule label to
// "<collection>:<action>" (apis/middlewares_rate_limit.go), the same shape as the password rule.
func enableCodeSignIn(txApp core.App) error {
	users, err := txApp.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		return err
	}
	users.PasswordAuth.Enabled = true
	users.OTP.Enabled = true
	users.OTP.Length = codeLength
	users.OTP.Duration = codeSeconds
	users.OTP.EmailTemplate = core.EmailTemplate{Subject: codeSubject, Body: codeBody}
	users.MFA.Enabled = false
	if err := txApp.Save(users); err != nil {
		return err
	}

	settings := txApp.Settings()
	settings.RateLimits.Enabled = true
	for label, max := range map[string]int{
		UsersCollection + ":requestOTP":           mailsPerMinute,
		UsersCollection + ":authWithOTP":          guessesPerMinute,
		UsersCollection + ":requestPasswordReset": mailsPerMinute,
	} {
		if !hasRateLimit(settings, label) {
			settings.RateLimits.Rules = append(settings.RateLimits.Rules,
				core.RateLimitRule{Label: label, Duration: 60, MaxRequests: max})
		}
	}
	return txApp.Save(settings)
}

func hasRateLimit(settings *core.Settings, label string) bool {
	for _, rule := range settings.RateLimits.Rules {
		if rule.Label == label {
			return true
		}
	}
	return false
}
