package pbauth

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Languages the code mails come in. The app sends one of them as `lang` when it signs a user up.
const (
	langEnglish = "en"
	langChinese = "zh-Hans"
)

// prepareNewUser runs on every create request of the users collection (a sign-up from the app, or
// the record Google's OAuth2 sign-in makes): the email is stored in lower case, the way the old
// code path stored it, and a user with no name is named after the part before the @. The client
// never asks for a name; it is changed later in the Account pane.
func prepareNewUser(e *core.RecordRequestEvent) error {
	if email := strings.ToLower(strings.TrimSpace(e.Record.Email())); email != "" {
		e.Record.SetEmail(email)
		if strings.TrimSpace(e.Record.GetString("name")) == "" {
			e.Record.Set("name", strings.SplitN(email, "@", 2)[0])
		}
	}
	return e.Next()
}

// findUserByTypedEmail looks the email up in lower case, which is how accounts are stored:
// PocketBase's own lookup is case sensitive, so "Ada@Example.com" would find nothing, and a request
// for a code would be answered with the dummy otpId and no mail.
func findUserByTypedEmail(e *core.RecordCreateOTPRequestEvent) error {
	if e.Record == nil {
		var form struct {
			Email string `form:"email" json:"email"`
		}
		if err := e.BindBody(&form); err == nil {
			email := strings.ToLower(strings.TrimSpace(form.Email))
			if existing, err := e.App.FindAuthRecordByEmail(e.Collection, email); err == nil {
				e.Record = existing
			}
		}
	}
	return e.Next()
}
