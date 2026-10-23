package syncserver

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

// statusError is an error with the HTTP status it should become. Handlers return one for every
// expected failure so the client gets a useful status instead of a generic 500.
type statusError struct {
	status  int
	message string
	// code is a machine-readable reason clients may switch on; empty for most errors.
	code string
}

func (e *statusError) Error() string { return e.message }

func statusErrorf(status int, format string, args ...any) error {
	return &statusError{status: status, message: fmt.Sprintf(format, args...)}
}

func badRequest(format string, args ...any) error {
	return statusErrorf(http.StatusBadRequest, format, args...)
}

func notFound(format string, args ...any) error {
	return statusErrorf(http.StatusNotFound, format, args...)
}

func forbidden(format string, args ...any) error {
	return statusErrorf(http.StatusForbidden, format, args...)
}

// CodeDeviceNotInAccount is the "code" of the 403 a device id the caller's account does not own
// gets: the client is signed in as a different account than the one it was linked with.
const CodeDeviceNotInAccount = "device_not_in_account"

// deviceNotInAccount is the 403 for a device that is not in the caller's account. The message
// stays what older clients matched on; the code is for newer ones.
func deviceNotInAccount() error {
	return &statusError{status: http.StatusForbidden, message: "device does not belong to this user", code: CodeDeviceNotInAccount}
}

func conflict(format string, args ...any) error {
	return statusErrorf(http.StatusConflict, format, args...)
}

// writeError maps err to a response. Errors without a status are logged and become a 500 without
// their detail, so internal state does not leak to the client.
func writeError(w http.ResponseWriter, err error) {
	var se *statusError
	if errors.As(err, &se) {
		body := map[string]string{"error": se.message}
		if se.code != "" {
			body["code"] = se.code
		}
		writeJSON(w, se.status, body)
		return
	}
	log.Printf("sync server: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
