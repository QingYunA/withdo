package store

import (
	"context"
	"database/sql"
)

// MigrateTo and DSN let the tests build a database that stops at an older schema version.
func MigrateTo(ctx context.Context, db *sql.DB, version int) error {
	return migrateTo(ctx, db, version)
}

var DSN = dsn
