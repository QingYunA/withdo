package mcpserver

import (
	"context"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// The user's categories are theirs to make, rename and delete in the window; agents only read the
// list and name a category on a task.

// categoryBrief is a category as list_categories returns it.
type categoryBrief struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Builtin string `json:"builtin,omitempty"`
	Color   string `json:"color"`
}

func (t *tools) registerListCategories(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_categories",
		Title:       "List categories",
		Description: "List the categories the user sorts tasks under, in their order. Work and Life are built in; the user adds the rest.",
		InputSchema: object(map[string]*jsonschema.Schema{}),
	}, t.listCategories)
}

func (t *tools) listCategories(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	cats, err := t.core.ListCategories(ctx)
	if err != nil {
		return t.fail(err)
	}
	briefs := make([]categoryBrief, len(cats))
	names := make([]string, len(cats))
	for i, c := range cats {
		briefs[i] = categoryBrief{ID: c.ID, Name: c.Name, Builtin: c.Builtin, Color: c.Color}
		names[i] = c.Name
	}
	return ok("Categories: "+strings.Join(names, ", "), map[string]any{"categories": briefs})
}

// categoryPatch turns the category name of an update_task call into a patch of its id: null clears
// the category, a name must be one of the user's.
func (t *tools) categoryPatch(ctx context.Context, n domain.Nullable[string]) (domain.Nullable[int64], error) {
	switch {
	case !n.Present:
		return domain.Nullable[int64]{}, nil
	case n.Value == nil:
		return domain.Clear[int64](), nil
	}
	id, err := t.core.ResolveCategory(ctx, *n.Value)
	if err != nil {
		return domain.Nullable[int64]{}, err
	}
	return domain.Set(id), nil
}
