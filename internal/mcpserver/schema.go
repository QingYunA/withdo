package mcpserver

import (
	"github.com/google/jsonschema-go/jsonschema"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// Small builders for the hand-written tool input schemas. The schemas are
// written out rather than reflected from Go types so the wire contract that
// agents depend on (names, optionality, bounds, enums) is visible in one place.

func ptr[T any](v T) *T { return &v }

func object(props map[string]*jsonschema.Schema, required ...string) *jsonschema.Schema {
	return &jsonschema.Schema{Type: "object", Properties: props, Required: required}
}

func text(desc string) *jsonschema.Schema {
	return &jsonschema.Schema{Type: "string", MinLength: ptr(1), Description: desc}
}

// anyText is a string that may be empty.
func anyText(desc string) *jsonschema.Schema {
	return &jsonschema.Schema{Type: "string", Description: desc}
}

func id(desc string) *jsonschema.Schema {
	return &jsonschema.Schema{Type: "integer", Minimum: ptr(1.0), Description: desc}
}

func enum[T ~string](desc string, values []T) *jsonschema.Schema {
	vs := make([]any, len(values))
	for i, v := range values {
		vs[i] = string(v)
	}
	return &jsonschema.Schema{Type: "string", Enum: vs, Description: desc}
}

func statusEnum(desc string) *jsonschema.Schema   { return enum(desc, domain.Statuses) }
func priorityEnum(desc string) *jsonschema.Schema { return enum(desc, domain.Priorities) }
func categoryEnum(desc string) *jsonschema.Schema { return enum(desc, domain.Categories) }

// timestamp is an ISO 8601 date or datetime; the handler normalises it to UTC.
func timestamp(desc string) *jsonschema.Schema { return text(desc) }

// nullable lets a property be JSON null as well as its normal type. Bounds such
// as minLength apply only to the non-null branch.
func nullable(s *jsonschema.Schema, desc string) *jsonschema.Schema {
	out := *s
	out.Type, out.Types, out.Description = "", []string{s.Type, "null"}, desc
	if len(out.Enum) > 0 {
		out.Enum = append(append([]any(nil), out.Enum...), nil)
	}
	return &out
}

func withDesc(s *jsonschema.Schema, desc string) *jsonschema.Schema {
	out := *s
	out.Description = desc
	return &out
}

var (
	prURL = &jsonschema.Schema{Type: "string", Format: "uri",
		Description: "URL of the pull request, e.g. https://github.com/owner/repo/pull/12"}
	cwdProp = text("Absolute path of the working directory; used to find the git repo, project and branch")
)
