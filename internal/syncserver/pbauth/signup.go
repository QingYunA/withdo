package pbauth

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// createUserOnCodeRequest makes sign-up and sign-in one path: PocketBase mails no code to an email
// without an account, so the first request for one creates the user (random password, not verified,
// named after the part before the @) and then lets the request go on. The code sign-in marks it
// verified. The email was validated before this hook runs.
func createUserOnCodeRequest(e *core.RecordCreateOTPRequestEvent) error {
	if e.Record == nil && e.Collection.Name == UsersCollection {
		var form struct {
			Email string `form:"email" json:"email"`
		}
		if err := e.BindBody(&form); err != nil {
			return e.BadRequestError("An error occurred while loading the submitted data.", err)
		}
		typed := strings.TrimSpace(form.Email)
		email := strings.ToLower(typed)
		// PocketBase looked the email up as typed; accounts are stored in lower case.
		if existing, err := e.App.FindAuthRecordByEmail(e.Collection, email); err == nil {
			e.Record = existing
			return e.Next()
		}
		record := core.NewRecord(e.Collection)
		record.SetEmail(email)
		record.SetVerified(false)
		record.SetPassword(security.RandomString(32))
		record.Set("name", strings.SplitN(typed, "@", 2)[0])
		if err := e.App.Save(record); err != nil {
			return err
		}
		e.Record = record
	}
	return e.Next()
}
