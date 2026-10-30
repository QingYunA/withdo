package pbauth

import (
	"github.com/pocketbase/pocketbase/core"
)

const (
	// minPasswordLength is the shortest password a sign-up or a reset accepts.
	minPasswordLength = 8
	// createsPerMinute is how many accounts one address may make in a minute.
	createsPerMinute = 5
	// resetsPerMinute is how many reset requests one address may make in a minute.
	resetsPerMinute = 10
	// resetLabel is the rate limit label of the custom reset route (PocketBase matches a label
	// against "<METHOD> <path>" and "<path>").
	resetLabel = "/api/withdo/reset-password"
)

// signUpCreateRule lets anyone create a user with an email and a password through the REST API
// (v42), but never choose the id or the token key, and never set `verified`: the field is also
// refused by PocketBase's own record form for anyone who is not a superuser (the rule is the
// second lock), so a user is verified only by the code mailed to its email. The OAuth2 sign-in
// keeps its own branch: its internal create request has the context "oauth2".
const signUpCreateRule = `@request.context = "oauth2" || ` +
	`(@request.body.id:isset = false && @request.body.verified:isset = false && @request.body.tokenKey:isset = false)`

func init() {
	core.AppMigrations.Register(enablePasswordSignUp, nil, "1790000500_adhd_users_signup.go")
}

// enablePasswordSignUp opens sign-up with a password (at least 8 characters), adds the `lang` field
// the code mails read, and rate limits users:create and the reset route.
func enablePasswordSignUp(txApp core.App) error {
	users, err := txApp.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		return err
	}
	rule := signUpCreateRule
	users.CreateRule = &rule
	if password, ok := users.Fields.GetByName(core.FieldNamePassword).(*core.PasswordField); ok {
		password.Min = minPasswordLength
	}
	if users.Fields.GetByName("lang") == nil {
		users.Fields.Add(&core.TextField{Name: "lang", Pattern: `^(` + langEnglish + `|` + langChinese + `)$`, Max: 16})
	}
	if err := txApp.Save(users); err != nil {
		return err
	}

	settings := txApp.Settings()
	settings.RateLimits.Enabled = true
	for label, max := range map[string]int{
		UsersCollection + ":create": createsPerMinute,
		resetLabel:                  resetsPerMinute,
	} {
		if !hasRateLimit(settings, label) {
			settings.RateLimits.Rules = append(settings.RateLimits.Rules,
				core.RateLimitRule{Label: label, Duration: 60, MaxRequests: max})
		}
	}
	return txApp.Save(settings)
}
