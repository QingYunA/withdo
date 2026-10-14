package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestNullableJSON(t *testing.T) {
	type body struct {
		Notes domain.Nullable[string] `json:"notes"`
		N     domain.Nullable[int64]  `json:"n"`
	}
	var omitted, cleared, set body
	for in, out := range map[string]*body{`{}`: &omitted, `{"notes":null}`: &cleared, `{"notes":"x","n":4}`: &set} {
		if err := json.Unmarshal([]byte(in), out); err != nil {
			t.Fatal(err)
		}
	}
	if omitted.Notes.Present {
		t.Error("omitted field must not be present")
	}
	if !cleared.Notes.Present || cleared.Notes.Value != nil {
		t.Errorf("null must be present and nil: %+v", cleared.Notes)
	}
	if !set.Notes.Present || *set.Notes.Value != "x" || *set.N.Value != 4 {
		t.Errorf("set values wrong: %+v", set)
	}
	if err := json.Unmarshal([]byte(`{"n":"nope"}`), &body{}); err == nil {
		t.Error("wrong type should fail")
	}
}
