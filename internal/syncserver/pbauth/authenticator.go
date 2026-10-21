package pbauth

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

// SubjectPrefix starts every subject this package returns, so a PocketBase user can never share a
// subject with a Zitadel one.
const SubjectPrefix = "pb:"

// Authenticator accepts the auth token PocketBase issued to a record of the users collection and
// uses `pb:<record id>` as the subject. A superuser's token, another collection's and any other
// token type are not accepted.
type Authenticator struct {
	app core.App
}

// NewAuthenticator returns an Authenticator over an opened app.
func NewAuthenticator(app core.App) *Authenticator { return &Authenticator{app: app} }

// Subject implements syncserver.Authenticator.
func (a *Authenticator) Subject(r *http.Request) (string, error) {
	token, err := syncserver.BearerToken(r)
	if err != nil {
		return "", err
	}
	record, err := a.app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil || record.Collection().Name != UsersCollection {
		return "", syncserver.ErrUnauthenticated
	}
	return SubjectPrefix + record.Id, nil
}
