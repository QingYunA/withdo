package pbauth

import (
	"github.com/pocketbase/pocketbase/core"
)

// oauth2CreateRule lets only an OAuth2 sign-in create a user through the REST API: PocketBase's
// auth-with-oauth2 makes the new record by an internal create request whose context is "oauth2",
// while a plain POST to /api/collections/users/records has the context "default" and is refused.
// Migration 1790000500 (migrate_signup.go) widens the rule for the password sign-up and keeps this branch.
const oauth2CreateRule = `@request.context = "oauth2"`

func init() {
	core.AppMigrations.Register(enableOAuth2, nil, "1790000300_adhd_users_oauth2.go")
}

// enableOAuth2 turns OAuth2 on for the users collection (P3, "Continue with Google") and maps the
// provider's display name to the name field. It adds no provider: the owner sets the Google client
// id and secret in PocketBase's settings by hand, so no secret is in the code. An email that
// already has an account signs in to that account.
func enableOAuth2(txApp core.App) error {
	users, err := txApp.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		return err
	}
	rule := oauth2CreateRule
	users.CreateRule = &rule
	users.OAuth2.Enabled = true
	users.OAuth2.MappedFields.Name = "name"
	return txApp.Save(users)
}
