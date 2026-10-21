package pbauth

import (
	"github.com/pocketbase/pocketbase/core"
)

// UsersCollection is the PocketBase auth collection the clients sign in to.
const UsersCollection = "users"

// appName is what PocketBase's emails and admin UI call the product.
const appName = "ADHD Todo"

func init() {
	core.AppMigrations.Register(setupUsers, nil, "1790000000_adhd_users_setup.go")
}

// setupUsers makes the built-in users collection what P1 needs: password sign-in by email and
// nothing else. Sign-up, one-time codes, OAuth2, MFA and the new-location mail stay off until the
// phases that need them, and the collection's API rules are closed so the REST API cannot list,
// create or edit users; accounts are made with `adhd-sync pb-import-user` for now.
func setupUsers(txApp core.App) error {
	settings := txApp.Settings()
	settings.Meta.AppName = appName
	if err := txApp.Save(settings); err != nil {
		return err
	}
	users, err := txApp.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		return err
	}
	users.ListRule, users.ViewRule, users.CreateRule, users.UpdateRule, users.DeleteRule = nil, nil, nil, nil, nil
	users.PasswordAuth.Enabled = true
	users.PasswordAuth.IdentityFields = []string{core.FieldNameEmail}
	users.OTP.Enabled = false
	users.OAuth2.Enabled = false
	users.MFA.Enabled = false
	users.AuthAlert.Enabled = false
	return txApp.Save(users)
}
