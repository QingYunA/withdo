package pbauth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// EmailLookup returns a syncserver.EmailLookup over app: the email of the PocketBase user a
// `pb:<record id>` subject names. A subject of another kind (a development token) or a user that is
// gone has no email, which is "" without an error.
func EmailLookup(app core.App) func(ctx context.Context, subject string) (string, error) {
	return func(_ context.Context, subject string) (string, error) {
		id, ok := strings.CutPrefix(subject, SubjectPrefix)
		if !ok || id == "" {
			return "", nil
		}
		record, err := app.FindRecordById(UsersCollection, id)
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		return record.Email(), nil
	}
}
