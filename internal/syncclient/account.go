package syncclient

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// Account is who the user signed in as, as the ID token's claims name them. It is only what the
// Settings pane shows; the server keeps using the access token's subject.
type Account struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// accountFromIDToken reads name and email from an ID token without verifying it. The token came
// straight from the issuer's token endpoint over TLS, and the claims only drive the account display;
// the server verifies every request's access token itself. ok is false when the payload is not a
// JWT or names no person.
func accountFromIDToken(idToken string) (Account, bool) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return Account{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Account{}, false
	}
	var claims struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return Account{}, false
	}
	account := Account{Name: strings.TrimSpace(claims.Name), Email: strings.TrimSpace(claims.Email)}
	if account.Name == "" && account.Email == "" {
		return Account{}, false
	}
	return account, true
}
