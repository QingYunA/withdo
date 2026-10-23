package pbauth_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

// Linking Google to a signed-in account against the real PocketBase: the Google email differs from
// the account's, the identity attaches to the signed-in record, and Continue with Google later
// lands on that account. A Google identity another account already has is refused without
// changing the token or the account.
func TestLinkGoogleAgainstPocketBase(t *testing.T) {
	app := openApp(t)
	google := fakeGoogleEndpoints(t, "someone.else@example.com")
	addGoogle(t, app, google)
	adaSubject, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pbauth.ImportUser(app, "bob@example.com", "Bob", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ada := syncclient.NewPocketBaseSource(base, &memSecrets{})
	if err := ada.SignIn(ctx, base, "ada@example.com", testPassword); err != nil {
		t.Fatal(err)
	}
	if linked, err := ada.GoogleLinked(ctx, base); err != nil || linked {
		t.Fatalf("before linking: linked = %v, %v, want false", linked, err)
	}
	before, _ := ada.Token(ctx)
	if err := ada.LinkGoogle(ctx, base, browse(t)); err != nil {
		t.Fatalf("link: %v", err)
	}
	if after, _ := ada.Token(ctx); after != before {
		t.Error("linking replaced the stored token")
	}
	if linked, err := ada.GoogleLinked(ctx, base); err != nil || !linked {
		t.Fatalf("after linking: linked = %v, %v, want true", linked, err)
	}
	if total, _ := app.CountRecords("users"); total != 2 {
		t.Fatalf("users = %d, want the two imported ones", total)
	}

	// Continue with Google, with an email that matches no account, now lands on Ada.
	fresh := syncclient.NewPocketBaseSource(base, &memSecrets{})
	if err := fresh.SignInWithGoogle(ctx, base, browse(t)); err != nil {
		t.Fatalf("sign in with Google: %v", err)
	}
	token, _ := fresh.Token(ctx)
	if status, got := get(t, base+"/v1/whoami", token); status != http.StatusOK || got != adaSubject {
		t.Fatalf("whoami = %d %q, want Ada %q", status, got, adaSubject)
	}

	// Bob tries the same Google identity.
	bob := syncclient.NewPocketBaseSource(base, &memSecrets{})
	if err := bob.SignIn(ctx, base, "bob@example.com", testPassword); err != nil {
		t.Fatal(err)
	}
	bobToken, _ := bob.Token(ctx)
	if err := bob.LinkGoogle(ctx, base, browse(t)); !errors.Is(err, syncclient.ErrGoogleAlreadyLinked) {
		t.Fatalf("Bob linking Ada's Google = %v, want ErrGoogleAlreadyLinked", err)
	}
	if now, _ := bob.Token(ctx); now != bobToken {
		t.Error("the refused link replaced Bob's token")
	}
	if linked, err := bob.GoogleLinked(ctx, base); err != nil || linked {
		t.Fatalf("Bob linked = %v, %v, want false", linked, err)
	}
}
