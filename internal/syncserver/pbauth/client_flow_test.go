package pbauth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

// The real client against the real embedded PocketBase: what the app and the phone core call.
func TestClientSignUpVerifyResetAgainstPocketBase(t *testing.T) {
	ctx := context.Background()
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	source := syncclient.NewPocketBaseSource(base, &memSecrets{})

	// Sign up mails the first code; the account cannot sign in yet.
	otpID, err := source.SignUp(ctx, base, "Grace@Example.com", testPassword, "en")
	if err != nil || otpID == "" {
		t.Fatalf("SignUp = %q, %v", otpID, err)
	}
	m := waitMail(t, mails)
	code := anyCode.FindString(m.subject)
	if code == "" {
		t.Fatalf("mail = %+v", m)
	}
	if err := source.SignIn(ctx, base, "grace@example.com", testPassword); !errors.Is(err, syncclient.ErrEmailNotVerified) {
		t.Fatalf("unverified sign-in = %v, want ErrEmailNotVerified", err)
	}
	if err := source.SignIn(ctx, base, "grace@example.com", "wrong password"); !errors.Is(err, syncclient.ErrInvalidCredentials) {
		t.Fatalf("wrong password = %v, want ErrInvalidCredentials", err)
	}

	// Errors of a sign-up.
	if _, err := source.SignUp(ctx, base, "grace@example.com", testPassword, "en"); !errors.Is(err, syncclient.ErrEmailTaken) {
		t.Errorf("taken email = %v, want ErrEmailTaken", err)
	}
	if _, err := source.SignUp(ctx, base, "not an email", testPassword, "en"); !errors.Is(err, syncclient.ErrInvalidEmail) {
		t.Errorf("malformed email = %v, want ErrInvalidEmail", err)
	}
	if _, err := source.SignUp(ctx, base, "ada@example.com", "short", "en"); !errors.Is(err, syncclient.ErrWeakPassword) {
		t.Errorf("short password = %v, want ErrWeakPassword", err)
	}

	// The code verifies and signs in; the password then works on its own.
	if err := source.SignInWithCode(ctx, base, otpID, "000000"); !errors.Is(err, syncclient.ErrInvalidCode) {
		t.Fatalf("wrong code = %v, want ErrInvalidCode", err)
	}
	if err := source.SignInWithCode(ctx, base, otpID, code); err != nil {
		t.Fatalf("SignInWithCode: %v", err)
	}
	if account, ok := source.Account(); !ok || account.Email != "grace@example.com" || account.Name != "grace" {
		t.Fatalf("account = %+v, %v", account, ok)
	}
	if err := source.SignIn(ctx, base, "grace@example.com", testPassword); err != nil {
		t.Fatalf("password sign-in after verifying: %v", err)
	}

	// Forgot password: a code, then a new password, signed in.
	resetID, err := source.RequestCode(ctx, base, "grace@example.com")
	if err != nil {
		t.Fatal(err)
	}
	resetCode := anyCode.FindString(waitMail(t, mails).subject)
	if err := source.ResetPassword(ctx, base, resetID, resetCode, "short"); !errors.Is(err, syncclient.ErrWeakPassword) {
		t.Errorf("short new password = %v, want ErrWeakPassword", err)
	}
	if err := source.ResetPassword(ctx, base, resetID, "000000", "the new password"); !errors.Is(err, syncclient.ErrInvalidCode) {
		t.Errorf("wrong reset code = %v, want ErrInvalidCode", err)
	}
	if err := source.ResetPassword(ctx, base, resetID, resetCode, "the new password"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if err := source.SignIn(ctx, base, "grace@example.com", "the new password"); err != nil {
		t.Fatalf("sign-in with the new password: %v", err)
	}
	// An email with no account answers like a wrong code.
	ghostID, err := source.RequestCode(ctx, base, "nobody@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := source.ResetPassword(ctx, base, ghostID, "123456", "the new password"); !errors.Is(err, syncclient.ErrInvalidCode) {
		t.Errorf("reset for no account = %v, want ErrInvalidCode", err)
	}
}
