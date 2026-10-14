package domain

import "encoding/json"

// Nullable is a patch value with three states: omitted (the zero value, leave
// the field alone), set to null (clear it) and set to a value.
type Nullable[T any] struct {
	Present bool
	Value   *T // nil with Present means "clear"
}

// Set returns a Nullable that assigns v.
func Set[T any](v T) Nullable[T] { return Nullable[T]{Present: true, Value: &v} }

// Clear returns a Nullable that sets the field to null.
func Clear[T any]() Nullable[T] { return Nullable[T]{Present: true} }

// UnmarshalJSON marks the value present; JSON null clears it.
func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
	n.Present = true
	if string(b) == "null" {
		n.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

// apply returns the field's new value given its current one.
func (n Nullable[T]) apply(current *T) *T {
	if !n.Present {
		return current
	}
	return n.Value
}
