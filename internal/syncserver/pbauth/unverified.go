package pbauth

import (
	"strings"

	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// ErrorCodeEmailNotVerified is the code in `data.email.code` of the 403 that auth-with-password
// answers when the password is right but the email has not been verified with a code yet. The
// client then requests a code and shows its Verify step. A wrong password is still the usual 400,
// so the answer never tells someone without the password whether an account exists.
const ErrorCodeEmailNotVerified = "email_not_verified"

// refuseUnverifiedPasswordSignIn stops an unverified user from getting a token with its password.
// Accounts are stored in lower case and PocketBase's lookup is case sensitive, so an email typed
// with capitals is looked up in lower case first.
func refuseUnverifiedPasswordSignIn(e *core.RecordAuthWithPasswordRequestEvent) error {
	if e.Record == nil && strings.Contains(e.Identity, "@") {
		if found, err := e.App.FindAuthRecordByEmail(e.Collection, strings.ToLower(strings.TrimSpace(e.Identity))); err == nil {
			e.Record = found
		}
	}
	if e.Record != nil && !e.Record.Verified() && e.Record.ValidatePassword(e.Password) {
		return e.ForbiddenError("The email is not verified.", validation.Errors{
			"email": validation.NewError(ErrorCodeEmailNotVerified, "The email is not verified."),
		})
	}
	return e.Next()
}

// verifyOnCode marks the user verified when the code came from the user's own email, before
// PocketBase's handler runs. PocketBase does the same but also replaces the password with a random
// one ("pre-hijacking" protection for the code-only sign-up); here the user chose a password at
// sign-up, so that step would throw it away. Nobody can sign in with the password before this
// moment, and the reset route replaces it with one the email's owner picks.
func verifyOnCode(e *core.RecordAuthWithOTPRequestEvent) error {
	if !e.Record.Verified() && e.OTP.SentTo() != "" && e.Record.Email() == e.OTP.SentTo() {
		e.Record.SetVerified(true)
		if err := e.App.Save(e.Record); err != nil {
			return err
		}
	}
	return e.Next()
}
