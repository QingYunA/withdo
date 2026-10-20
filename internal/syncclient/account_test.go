package syncclient_test

import (
	"context"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// TestS3UIa2Account is S3-UI-a acceptance criterion 2 on the client: after a fake-issuer sign-in the
// source knows the ID token's name and email, and a refresh (in a new source that only has the
// stored refresh token) learns them again. Logout forgets them.
func TestS3UIa2Account(t *testing.T) {
	const clientID = "adhd-desktop"
	ctx := context.Background()
	issuer := newFakeOIDC(t, clientID)
	secrets := newFakeSecrets()
	source := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
	if account, ok := source.Account(); ok {
		t.Fatalf("Account before sign-in = %+v, want none", account)
	}
	if err := source.SignIn(ctx, issuer.URL, "ada@example.com", "correct horse"); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	assertAccount(t, source, "Ada Lovelace", "ada@example.com")

	// A fresh source has only the refresh token. Token refreshes it and reads the new ID token.
	refreshed := syncclient.NewOIDCSource(issuer.URL, clientID, nil, secrets)
	if account, ok := refreshed.Account(); ok {
		t.Fatalf("Account before the first token = %+v, want none", account)
	}
	if _, err := refreshed.Token(ctx); err != nil {
		t.Fatalf("Token: %v", err)
	}
	assertAccount(t, refreshed, "Ada Lovelace", "ada@example.com")

	if err := refreshed.Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if account, ok := refreshed.Account(); ok {
		t.Fatalf("Account after logout = %+v, want none", account)
	}

	// A static development token has no account.
	static := syncclient.StaticToken("token")
	if _, ok := any(static).(interface {
		Account() (syncclient.Account, bool)
	}); ok {
		t.Fatal("StaticToken grew an Account method; a static token must have no account")
	}
}

func assertAccount(t *testing.T, source *syncclient.OIDCSource, name, email string) {
	t.Helper()
	account, ok := source.Account()
	if !ok {
		t.Fatal("Account() reports no account")
	}
	if account.Name != name || account.Email != email {
		t.Fatalf("Account = %+v, want %s <%s>", account, name, email)
	}
}
