package pbauth

import (
	"errors"
	"sync"
	"time"

	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// ResetPasswordPath is the route that sets a new password with a code, so the app never needs a
// reset page in a browser. Body: {otpId, code, password, passwordConfirm}; the answer is the same
// as auth-with-otp's (token and record).
const ResetPasswordPath = "/api/withdo/reset-password"

// invalidCodeMessage is the one answer for a wrong code, an expired code, a code that never
// existed (request-otp answers an email with no account with an otpId and sends nothing) and a
// code PocketBase did not mail. It is auth-with-otp's own message, so the client has one mapping.
const invalidCodeMessage = "Invalid or expired OTP"

// A code is six digits, so guesses are limited per user like PocketBase limits auth-with-otp:
// five tries in three minutes.
const (
	guessesPerWindow = 5
	guessWindow      = 3 * time.Minute
)

var guesses = guessLimiter{tries: map[string][]time.Time{}}

type guessLimiter struct {
	mu    sync.Mutex
	tries map[string][]time.Time
}

// allow records one try for the user and reports whether it is within the limit.
func (g *guessLimiter) allow(userID string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	recent := g.tries[userID][:0]
	for _, t := range g.tries[userID] {
		if now.Sub(t) < guessWindow {
			recent = append(recent, t)
		}
	}
	if len(recent) >= guessesPerWindow {
		g.tries[userID] = recent
		return false
	}
	g.tries[userID] = append(recent, now)
	return true
}

type resetPasswordForm struct {
	OTPId           string `form:"otpId" json:"otpId"`
	Code            string `form:"code" json:"code"`
	Password        string `form:"password" json:"password"`
	PasswordConfirm string `form:"passwordConfirm" json:"passwordConfirm"`
}

// resetPassword checks the code the way auth-with-otp does (it must exist, belong to the users
// collection, not be expired, match, and be tried at most five times in three minutes), sets the
// new password, verifies the email (the code proves it) and answers with a token and the record.
// An email with no account has no OTP record, so it fails like a wrong code.
func resetPassword(e *core.RequestEvent) error {
	form := &resetPasswordForm{}
	if err := e.BindBody(form); err != nil {
		return e.BadRequestError("An error occurred while loading the submitted data.", err)
	}
	// The password is checked first: it says nothing about any account.
	if n := len(form.Password); n < minPasswordLength || n > 71 {
		return e.BadRequestError("An error occurred while validating the submitted data.", validation.Errors{
			"password": validation.NewError("validation_min_text_constraint", "Must be at least 8 characters."),
		})
	}
	if form.Password != form.PasswordConfirm {
		return e.BadRequestError("An error occurred while validating the submitted data.", validation.Errors{
			"passwordConfirm": validation.NewError("validation_values_mismatch", "Values don't match."),
		})
	}
	if form.OTPId == "" || form.Code == "" {
		return e.BadRequestError(invalidCodeMessage, errors.New("missing otpId or code"))
	}

	collection, err := e.App.FindCachedCollectionByNameOrId(UsersCollection)
	if err != nil {
		return e.InternalServerError("", err)
	}
	otp, err := e.App.FindOTPById(form.OTPId)
	if err != nil {
		return e.BadRequestError(invalidCodeMessage, err)
	}
	if otp.CollectionRef() != collection.Id || otp.HasExpired(collection.OTP.DurationTime()) {
		return e.BadRequestError(invalidCodeMessage, errors.New("the OTP is for another collection or expired"))
	}
	record, err := e.App.FindRecordById(collection, otp.RecordRef())
	if err != nil {
		return e.BadRequestError(invalidCodeMessage, err)
	}
	if !guesses.allow(record.Id, time.Now()) {
		return e.TooManyRequestsError("Too many attempts, please try again later with a new OTP.", nil)
	}
	// Only a code PocketBase mailed to the account's own email proves the email.
	if !otp.ValidatePassword(form.Code) || otp.SentTo() == "" || otp.SentTo() != record.Email() {
		return e.BadRequestError(invalidCodeMessage, errors.New("incorrect code"))
	}

	// One use: this code and any other still open for the user are gone.
	if others, err := e.App.FindAllOTPsByRecord(record); err == nil {
		for _, other := range others {
			_ = e.App.Delete(other)
		}
	}
	record.SetPassword(form.Password)
	record.SetVerified(true)
	if err := e.App.Save(record); err != nil {
		return e.BadRequestError("Failed to set the password.", err)
	}
	return apis.RecordAuthResponse(e, record, core.MFAMethodOTP, nil)
}
