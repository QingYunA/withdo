package pbauth

import (
	"context"
	"testing"
)

func TestEmailLookupFindsAUsersEmail(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = Close(app) })
	subject, err := ImportUser(app, "buyer@example.com", "Buyer", "a-long-enough-password")
	if err != nil {
		t.Fatal(err)
	}
	lookup := EmailLookup(app)
	for subject, want := range map[string]string{
		subject:       "buyer@example.com",
		"pb:missing1": "",
		"dev-user":    "",
		"pb:":         "",
	} {
		got, err := lookup(context.Background(), subject)
		if err != nil || got != want {
			t.Errorf("lookup(%q) = %q, %v; want %q", subject, got, err, want)
		}
	}
}
