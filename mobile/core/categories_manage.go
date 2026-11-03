package core

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/QingYunA/adhd-todo/app/categoryicons"
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

// categoryRow is one row of Settings > Categories.
type categoryRow struct {
	Category data.Category
	// Open counts the open tasks under the category as the desktop's Settings counts them: a
	// repository's tasks under Work, and no subtasks.
	Open int
}

// CategoriesWithCounts returns the categories in the user's order with their open task counts, as
// JSON:
//
//	[{"Category":{...},"Open":12}, …]
//
// Category is data.Category (see Categories). The tasks that belong to no category are not a row;
// Work and Life are always there.
func (c *Core) CategoriesWithCounts() (string, error) {
	snap, err := c.real.Snapshot()
	if err != nil {
		return "", shortError(err)
	}
	hasRepo := map[int]bool{}
	for _, p := range snap.Projects {
		hasRepo[p.ID] = p.HasRepo()
	}
	work := 0
	for _, cat := range snap.Categories {
		if cat.Builtin == data.BuiltinWork {
			work = cat.ID
		}
	}
	open := map[int]int{}
	for _, t := range snap.Tasks {
		if t.IsSubtask() || t.Status.Terminal() {
			continue
		}
		open[data.CategoryOf(t, func(id *int) bool { return id != nil && hasRepo[*id] }, work)]++
	}
	rows := make([]categoryRow, 0, len(snap.Categories))
	for _, cat := range snap.Categories {
		rows = append(rows, categoryRow{Category: cat, Open: open[cat.ID]})
	}
	return marshal(rows)
}

// categoryInput is the JSON of CreateCategory and UpdateCategory. Every key is optional but the
// name of a new category.
//
//	{"name":"Health","color":"green","icon":"heart"}
//
// icon is the name of a library icon (CategoryIcons); an empty or absent one keeps the icon on an
// update and takes the library's first curated icon on a create. color is one of the eight keys
// the desktop uses: indigo, yellow, green, red, blue, pink, teal, orange; an empty color on a
// create takes the one the name hashes to.
type categoryInput struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
	Icon  *string `json:"icon"`
}

// libraryIcon is the picture of a library icon, and false for a name the library does not have.
func libraryIcon(name string) (pixmap.Picture, bool) {
	e, ok := categoryicons.Named(name)
	return e.Bitmap.Picture(), ok
}

// CreateCategory adds a category at the end of the list and returns it as a data.Category JSON.
// See categoryInput for the keys. A name that is empty, longer than 32 characters or taken, or a
// 25th category, fails with the desktop's sentence.
func (c *Core) CreateCategory(inputJSON string) (string, error) {
	var in categoryInput
	if err := json.Unmarshal([]byte(inputJSON), &in); err != nil {
		return "", errors.New("could not read the new category")
	}
	icon := categoryicons.Groups[0].Icons[0].Bitmap.Picture()
	if name := deref(in.Icon); name != "" {
		picture, ok := libraryIcon(name)
		if !ok {
			return "", errors.New("the icon is not in the library")
		}
		icon = picture
	}
	made, err := c.real.CreateCategory(data.NewCategory{Name: deref(in.Name), Icon: icon, Color: deref(in.Color)})
	if err != nil {
		return "", manageError(err)
	}
	return marshal(made)
}

// UpdateCategory renames, recolors or redraws a category (see categoryInput for the keys; an absent
// key is left alone) and returns it as a data.Category JSON. Work and Life keep their names.
func (c *Core) UpdateCategory(id int64, patchJSON string) (string, error) {
	var in categoryInput
	if err := json.Unmarshal([]byte(patchJSON), &in); err != nil {
		return "", errors.New("could not read the change")
	}
	patch := data.CategoryPatch{Name: in.Name, Color: in.Color}
	if name := deref(in.Icon); name != "" {
		picture, ok := libraryIcon(name)
		if !ok {
			return "", errors.New("the icon is not in the library")
		}
		patch.Icon = &picture
	}
	updated, err := c.real.UpdateCategory(int(id), patch)
	if err != nil {
		return "", manageError(err)
	}
	return marshal(updated)
}

// DeleteCategory removes a category of the user's (Work and Life cannot go); the tasks under it
// become unsorted. It returns the snapshot RestoreCategory needs as JSON, which is opaque: pass it
// back unchanged to undo the delete.
func (c *Core) DeleteCategory(id int64) (string, error) {
	deleted, err := c.dom.DeleteCategory(context.Background(), id)
	if err != nil {
		return "", shortError(err)
	}
	return marshal(deleted)
}

// RestoreCategory puts back a category DeleteCategory removed, with its id at its old place, and
// gives it back the tasks that are still unsorted. It is the Undo of DeleteCategory.
func (c *Core) RestoreCategory(snapshotJSON string) error {
	var deleted domain.DeletedCategory
	if err := json.Unmarshal([]byte(snapshotJSON), &deleted); err != nil {
		return errors.New("could not read the deleted category")
	}
	if _, err := c.dom.RestoreCategory(context.Background(), deleted); err != nil {
		return shortError(err)
	}
	return nil
}

// MoveCategory puts a category at index (0-based) in the list, Work and Life included.
func (c *Core) MoveCategory(id int64, index int64) error {
	if err := c.real.MoveCategory(int(id), int(index)); err != nil {
		return shortError(err)
	}
	return nil
}

// libraryIconJSON is one icon of CategoryIcons.
type libraryIconJSON struct {
	Name     string       `json:"name"`
	Keywords string       `json:"keywords"` // Chinese words the search also reads, "" for most
	Icon     [12][12]byte `json:"icon"`
}

// libraryGroupJSON is a section of CategoryIcons.
type libraryGroupJSON struct {
	Name  string            `json:"name"`
	Icons []libraryIconJSON `json:"icons"`
}

// CategoryIcons returns the icon library the category sheet picks from, as JSON:
//
//	{"groups":[{"name":"Work","icons":[{"name":"briefcase","keywords":"…","icon":[[0,…],…]}]}]}
//
// For a blank query it is the curated groups in the picker's order (the group names are stable keys
// for the app to localize); otherwise it is one group with an empty name holding the icons whose
// English name or Chinese keywords contain the query. icon is the 12 x 12 picture in the form
// Category.Icon has (0 empty, 35 a pixel in the category's color), and name is what CreateCategory
// and UpdateCategory take as icon.
func (c *Core) CategoryIcons(query string) (string, error) {
	groups := []libraryGroupJSON{}
	if found := categoryicons.Search(query); found != nil || query != "" {
		groups = append(groups, libraryGroupJSON{Icons: libraryIcons(found)})
	} else {
		for _, g := range categoryicons.Groups {
			groups = append(groups, libraryGroupJSON{Name: g.Name, Icons: libraryIcons(g.Icons)})
		}
	}
	return marshal(map[string]any{"groups": groups})
}

func libraryIcons(entries []categoryicons.Entry) []libraryIconJSON {
	out := make([]libraryIconJSON, 0, len(entries))
	for _, e := range entries {
		out = append(out, libraryIconJSON{Name: e.Name, Keywords: e.Keywords, Icon: e.Bitmap.Picture()})
	}
	return out
}
