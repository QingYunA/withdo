package domain

import (
	"errors"
	"fmt"
)

// FieldError is a caller-fixable problem. Field names the offending input so
// tool errors can point at it.
type FieldError struct {
	Field    string
	Msg      string
	NotFound bool
	// Conflict says the value is fine but already taken.
	Conflict bool
}

func (e *FieldError) Error() string {
	if e.Field == "" {
		return e.Msg
	}
	return e.Field + ": " + e.Msg
}

func invalid(field, format string, args ...any) error {
	return &FieldError{Field: field, Msg: fmt.Sprintf(format, args...)}
}

func notFound(field, what string, id int64) error {
	return &FieldError{Field: field, Msg: fmt.Sprintf("%s %d not found", what, id), NotFound: true}
}

// IsNotFound reports whether err says a referenced row does not exist.
func IsNotFound(err error) bool {
	var fe *FieldError
	return errors.As(err, &fe) && fe.NotFound
}

func conflict(field, format string, args ...any) error {
	return &FieldError{Field: field, Msg: fmt.Sprintf(format, args...), Conflict: true}
}

// IsConflict reports whether err says a value is already used by another row.
func IsConflict(err error) bool {
	var fe *FieldError
	return errors.As(err, &fe) && fe.Conflict
}
