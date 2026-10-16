package mobilesync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validID(s string) bool   { return uuidPattern.MatchString(s) }
func validType(s string) bool { return s == "project" || s == "task" || s == "report" }

func decodeStrict(raw []byte, v any) error {
	// encoding/json matches struct fields case-insensitively and permits
	// duplicate object keys. Both would let stored raw data disagree with the
	// struct we validated, so check the exact public keys before decoding.
	keys := json.NewDecoder(bytes.NewReader(raw))
	if err := checkJSONKeys(keys, reflect.TypeOf(v)); err != nil {
		return err
	}
	if _, err := keys.Token(); err != io.EOF {
		return errors.New("invalid JSON document")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func checkJSONKeys(d *json.Decoder, schema reflect.Type) error {
	for schema != nil && schema.Kind() == reflect.Pointer {
		schema = schema.Elem()
	}
	if schema == reflect.TypeFor[json.RawMessage]() {
		schema = nil // Its exact entity shape is checked separately by decodeEntity.
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	delimiter, object := token.(json.Delim)
	if !object {
		return nil // The normal decoder checks scalar types.
	}
	if delimiter == '{' {
		var fields map[string]reflect.Type
		if schema != nil && schema.Kind() == reflect.Struct {
			fields = map[string]reflect.Type{}
			for i := 0; i < schema.NumField(); i++ {
				field := schema.Field(i)
				name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
				if field.PkgPath != "" || name == "-" {
					continue
				}
				if name == "" {
					name = field.Name
				}
				fields[name] = field.Type
			}
		}
		seen := map[string]bool{}
		for d.More() {
			keyToken, err := d.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok || seen[key] {
				return errors.New("duplicate or invalid JSON key")
			}
			seen[key] = true
			child, known := fields[key]
			if fields != nil && !known {
				return errors.New("unknown JSON key")
			}
			if err := checkJSONKeys(d, child); err != nil {
				return err
			}
		}
	} else if delimiter == '[' {
		var child reflect.Type
		if schema != nil && (schema.Kind() == reflect.Slice || schema.Kind() == reflect.Array) {
			child = schema.Elem()
		}
		for d.More() {
			if err := checkJSONKeys(d, child); err != nil {
				return err
			}
		}
	} else {
		return errors.New("invalid JSON delimiter")
	}
	_, err = d.Token() // Consume the matching closing delimiter.
	return err
}

// Require the complete public record shape, including explicit nullable fields.
// encoding/json otherwise accepts null for a string and turns it into "",
// leaving the raw record unsafe for a TypeScript client expecting strings.
func decodeEntity(raw []byte, v any) error {
	if err := decodeStrict(raw, v); err != nil {
		return err
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var expected, actual map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &expected); err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &actual); err != nil {
		return err
	}
	for field, value := range expected {
		got, ok := actual[field]
		if !ok || (bytes.Equal(got, []byte("null")) && !bytes.Equal(value, []byte("null"))) {
			return errors.New("incomplete entity fields")
		}
	}
	return nil
}

func timestamp(s string) bool {
	_, err := time.Parse(time.RFC3339Nano, s)
	return err == nil
}

func optionalTimestamp(s *string) bool { return s == nil || timestamp(*s) }
func optionalID(s *string) bool        { return s == nil || validID(*s) }
func sameID(a, b *string) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}
func nonempty(s string, max int) bool { return strings.TrimSpace(s) != "" && len(s) <= max }

func validateMutation(ctx context.Context, q store.Querier, m Mutation, cur *Record, deleted bool) (string, error) {
	if deleted {
		var dependents int
		query := "SELECT count(*) FROM sync_entities WHERE type='task' AND deleted=0 AND json_extract(data,'$.projectId')=?"
		if m.Type == "task" {
			query = "SELECT count(*) FROM sync_entities WHERE type='task' AND deleted=0 AND json_extract(data,'$.parentId')=?"
		}
		if err := q.QueryRowContext(ctx, query, m.EntityID).Scan(&dependents); err != nil {
			return "", err
		}
		if dependents > 0 {
			return "dependent_tasks_exist", nil
		}
		return "", nil
	}
	switch m.Type {
	case "project":
		var p project
		if err := decodeEntity(m.Data, &p); err != nil || p.ID != m.EntityID || !nonempty(p.Name, 500) || len(p.RepoSlug) > 1000 || !timestamp(p.CreatedAt) || !timestamp(p.UpdatedAt) || !optionalTimestamp(p.ArchivedAt) {
			return "invalid_project", nil
		}
		if cur != nil && !cur.Deleted {
			var old project
			if err := json.Unmarshal(cur.Data, &old); err != nil {
				return "", err
			}
			if old.CreatedAt != p.CreatedAt {
				return "created_at_is_immutable", nil
			}
		}
	case "task":
		var t task
		if err := decodeEntity(m.Data, &t); err != nil || t.ID != m.EntityID || !nonempty(t.Title, 2000) || len(t.Notes) > 50000 || len(t.Branch) > 1000 || len(t.PRURL) > 2000 || !domain.Status(t.Status).Valid() || !domain.Priority(t.Priority).Valid() || (t.Source != "user" && t.Source != "agent") || !optionalID(t.ProjectID) || !optionalID(t.ParentID) || !timestamp(t.CreatedAt) || !timestamp(t.UpdatedAt) || !optionalTimestamp(t.DueAt) || !optionalTimestamp(t.RemindAt) || !optionalTimestamp(t.CompletedAt) {
			return "invalid_task", nil
		}
		if cur != nil && !cur.Deleted {
			var old task
			if err := json.Unmarshal(cur.Data, &old); err != nil {
				return "", err
			}
			if old.CreatedAt != t.CreatedAt {
				return "created_at_is_immutable", nil
			}
		}
		if t.ProjectID != nil {
			p, err := getRecord(ctx, q, "project", *t.ProjectID)
			if err != nil {
				return "", err
			}
			if p == nil || p.Deleted {
				return "project_not_found", nil
			}
		}
		// Bound ancestry and reject cycles and project mismatches, including a
		// reparent under an existing descendant.
		at := t.ParentID
		for depth := 0; at != nil; depth++ {
			if *at == t.ID || depth >= 100 {
				return "invalid_parent", nil
			}
			p, err := getRecord(ctx, q, "task", *at)
			if err != nil {
				return "", err
			}
			if p == nil || p.Deleted {
				return "parent_not_found", nil
			}
			var parent task
			if err := json.Unmarshal(p.Data, &parent); err != nil {
				return "", err
			}
			if !sameID(t.ProjectID, parent.ProjectID) {
				return "parent_project_mismatch", nil
			}
			at = parent.ParentID
		}
		// Changing a parent's project cannot silently orphan existing children.
		var mismatch int
		if err := q.QueryRowContext(ctx, `SELECT count(*) FROM sync_entities WHERE type='task' AND deleted=0
			AND json_extract(data,'$.parentId')=? AND json_extract(data,'$.projectId') IS NOT ?`, t.ID, t.ProjectID).Scan(&mismatch); err != nil {
			return "", err
		}
		if mismatch != 0 {
			return "child_project_mismatch", nil
		}
	case "report":
		var r report
		if err := decodeEntity(m.Data, &r); err != nil || r.ID != m.EntityID || !validID(r.TaskID) || !nonempty(r.Agent, 200) || !nonempty(r.Summary, 20000) || len(r.NextStep) > 20000 || !timestamp(r.CreatedAt) || (r.Status != nil && !domain.Status(*r.Status).Valid()) {
			return "invalid_report", nil
		}
		t, err := getRecord(ctx, q, "task", r.TaskID)
		if err != nil {
			return "", err
		}
		if t == nil || t.Deleted {
			return "task_not_found", nil
		}
	}
	return "", nil
}
