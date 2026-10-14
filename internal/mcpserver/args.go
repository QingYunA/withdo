package mcpserver

import (
	"net/url"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// Argument checks the JSON schema cannot express. Each error names its field.

func fieldErr(field, msg string) error { return &domain.FieldError{Field: field, Msg: msg} }

// timestampArg normalises an optional ISO 8601 date or datetime; "" means absent.
func timestampArg(field, s string) (*string, error) {
	if s == "" {
		return nil, nil
	}
	v, err := domain.ParseTimestamp(s)
	if err != nil {
		return nil, fieldErr(field, err.Error())
	}
	return &v, nil
}

// nullableTimestampArg is timestampArg for patch fields that may be cleared.
func nullableTimestampArg(field string, n domain.Nullable[string]) (domain.Nullable[string], error) {
	if !n.Present || n.Value == nil {
		return n, nil
	}
	v, err := domain.ParseTimestamp(*n.Value)
	if err != nil {
		return n, fieldErr(field, err.Error())
	}
	return domain.Set(v), nil
}

// urlArg checks that s is an absolute URL; "" means absent.
func urlArg(field, s string) error {
	if s == "" {
		return nil
	}
	if u, err := url.Parse(s); err != nil || u.Scheme == "" || u.Host == "" {
		return fieldErr(field, "must be an absolute URL, e.g. https://github.com/owner/repo/pull/12")
	}
	return nil
}

func nullableURLArg(field string, n domain.Nullable[string]) error {
	if n.Present && n.Value != nil {
		return urlArg(field, *n.Value)
	}
	return nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
