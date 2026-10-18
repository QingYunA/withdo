package mcpserver

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The SDK checks a call's arguments against the tool's input schema before our handler runs and
// answers a mismatch with the validator's own words, e.g.
//
//	validating "arguments": validating root: validating /properties/task_id: type: <invalid reflect.Value> has type "null", want "integer"
//
// rewriteSchemaErrors turns those into the style of the domain's errors, "task_id: must be an integer".

const schemaErrPrefix = `validating "arguments": `

var (
	schemaPropertyRE = regexp.MustCompile(`validating /properties/([^/:]+)`)
	schemaWantRE     = regexp.MustCompile(`want (?:one of )?"([^"]*)"$`)
	schemaBoundRE    = regexp.MustCompile(`^(minimum|maximum|exclusiveMinimum|exclusiveMaximum): .* (?:less|greater) than(?: or equal to)? ([-+0-9.eE]+)$`)
	schemaMinLenRE   = regexp.MustCompile(`fewer than (\d+)$`)
	schemaEnumRE     = regexp.MustCompile(`any of: \[(.*)\]$`)
	schemaMissingRE  = regexp.MustCompile(`"([^"]+)"`)
)

// rewriteSchemaErrors is receiving middleware: it rewords the isError result of a tools/call whose
// arguments failed schema validation, and passes everything else through.
func rewriteSchemaErrors(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		call, ok := res.(*mcp.CallToolResult)
		if err != nil || method != "tools/call" || !ok || !call.IsError || len(call.Content) == 0 {
			return res, err
		}
		tc, ok := call.Content[0].(*mcp.TextContent)
		if !ok || !strings.HasPrefix(tc.Text, schemaErrPrefix) {
			return res, err
		}
		msg := schemaErrorMessage(strings.TrimPrefix(tc.Text, schemaErrPrefix))
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: msg}}, IsError: true}, nil
	}
}

// schemaErrorMessage rewords one validator message, "<field>: <what is wrong>".
func schemaErrorMessage(raw string) string {
	// The innermost failure is the last "validating <path>: " step; the rest is its keyword error.
	field, detail := "", raw
	if i := strings.LastIndex(raw, "validating "); i >= 0 {
		if _, after, found := strings.Cut(raw[i:], ": "); found {
			detail = after
		}
	}
	if m := schemaPropertyRE.FindStringSubmatch(raw); m != nil {
		field = m[1]
	}
	if strings.HasPrefix(detail, "required: ") {
		names := schemaMissingRE.FindAllStringSubmatch(detail, -1)
		missing := make([]string, len(names))
		for i, n := range names {
			missing[i] = n[1]
		}
		return strings.Join(missing, ", ") + ": is required"
	}
	what := schemaProblem(detail)
	if field == "" {
		return "arguments: " + what
	}
	return field + ": " + what
}

// schemaProblem says what one keyword error means, e.g. "must be an integer".
func schemaProblem(detail string) string {
	keyword, _, _ := strings.Cut(detail, ": ")
	switch keyword {
	case "type":
		if m := schemaWantRE.FindStringSubmatch(detail); m != nil {
			types := strings.Split(m[1], ", ")
			for i, t := range types {
				if t != "null" {
					types[i] = withArticle(t)
				}
			}
			return "must be " + strings.Join(types, " or ")
		}
	case "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum":
		if m := schemaBoundRE.FindStringSubmatch(detail); m != nil {
			if limit, err := strconv.ParseFloat(m[2], 64); err == nil {
				words := map[string]string{"minimum": "at least", "maximum": "at most",
					"exclusiveMinimum": "greater than", "exclusiveMaximum": "less than"}
				return fmt.Sprintf("must be %s %s", words[m[1]], strconv.FormatFloat(limit, 'g', -1, 64))
			}
		}
	case "minLength":
		if m := schemaMinLenRE.FindStringSubmatch(detail); m != nil {
			if m[1] == "1" {
				return "must not be empty"
			}
			return "must be at least " + m[1] + " characters"
		}
	case "enum":
		if m := schemaEnumRE.FindStringSubmatch(detail); m != nil {
			values := strings.Fields(m[1])
			nullable := false
			kept := values[:0:0]
			for _, v := range values {
				if v == "<nil>" {
					nullable = true
					continue
				}
				kept = append(kept, v)
			}
			out := "must be one of " + strings.Join(kept, ", ")
			if nullable {
				out += " or null"
			}
			return out
		}
	}
	return "is invalid"
}

func withArticle(noun string) string {
	if strings.ContainsRune("aeiou", rune(noun[0])) {
		return "an " + noun
	}
	return "a " + noun
}
