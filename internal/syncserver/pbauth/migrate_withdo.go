package pbauth

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// legacyAppName is what the earlier migrations called the product before it was renamed.
const legacyAppName = "ADHD Todo"

func init() {
	core.AppMigrations.Register(renameToWithdo, nil, "1790000400_adhd_rename_withdo.go")
}

// renameToWithdo gives a database that already ran the earlier migrations the product's new name:
// PocketBase's app name, the mail sender's name when the owner set it to the old name in the admin
// UI, and the sign-in code mail's subject and body, which the OTP migration saved with the old name. Only the old name is replaced, so a template the owner edited in the
// admin UI keeps its own wording. A database made after the rename has nothing left to replace, so
// running it again changes nothing.
func renameToWithdo(txApp core.App) error {
	settings := txApp.Settings()
	if settings.Meta.AppName != appName || settings.Meta.SenderName == legacyAppName {
		settings.Meta.AppName = appName
		if settings.Meta.SenderName == legacyAppName {
			settings.Meta.SenderName = appName
		}
		if err := txApp.Save(settings); err != nil {
			return err
		}
	}
	users, err := txApp.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		return err
	}
	tpl := &users.OTP.EmailTemplate
	subject := strings.ReplaceAll(tpl.Subject, legacyAppName, appName)
	body := strings.ReplaceAll(tpl.Body, legacyAppName, appName)
	if subject == tpl.Subject && body == tpl.Body {
		return nil
	}
	tpl.Subject, tpl.Body = subject, body
	return txApp.Save(users)
}
