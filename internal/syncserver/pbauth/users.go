package pbauth

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// ImportUser creates a user with an email, a display name and a password, marked as verified, and
// returns its sync subject. It fails when the email is already taken. The password goes straight
// to PocketBase, which stores a hash; it is never logged or put into an error.
func ImportUser(app core.App, email, name, password string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return "", errors.New("an email and a password are required")
	}
	users, err := app.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		return "", fmt.Errorf("find the users collection: %w", err)
	}
	if _, err := app.FindAuthRecordByEmail(users, email); err == nil {
		return "", fmt.Errorf("a user with the email %s already exists", email)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("look up %s: %w", email, err)
	}
	record := core.NewRecord(users)
	record.SetEmail(email)
	record.SetVerified(true)
	record.SetPassword(password)
	record.Set("name", strings.TrimSpace(name))
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("create the user: %w", err)
	}
	return SubjectPrefix + record.Id, nil
}

// UpsertSuperuser creates the superuser with this email, or sets a new password for the existing
// one. Superusers sign in to PocketBase's admin UI, which is reached over an ssh tunnel.
func UpsertSuperuser(app core.App, email, password string) error {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return errors.New("an email and a password are required")
	}
	superusers, err := app.FindCachedCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return fmt.Errorf("find the superusers collection: %w", err)
	}
	record, err := app.FindAuthRecordByEmail(superusers, email)
	if errors.Is(err, sql.ErrNoRows) {
		record = core.NewRecord(superusers)
		record.SetEmail(email)
	} else if err != nil {
		return fmt.Errorf("look up %s: %w", email, err)
	}
	record.SetPassword(password)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("save the superuser: %w", err)
	}
	return nil
}
