// Package categoryicons is the library a category's icon is picked from: the pixelarticons that
// reduce to a 12 x 12 grid with little loss (data_gen.go, written by scripts/category-icons.sh)
// and, first, a curated list in groups (docs/design/category-icons.txt). It holds no drawing code,
// so the desktop's theme and the phone's core both read it.
package categoryicons

import (
	"strings"

	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

// categoryIconRow and categoryIconGroupRow are the shapes of the generated tables.
type (
	categoryIconRow      struct{ name, hex, keywords string }
	categoryIconGroupRow struct {
		name  string
		icons []string
	}
)

// Entry is one icon of the library.
type Entry struct {
	Name     string // the pixelarticons name, such as "coffee"
	Keywords string // Chinese words the search also reads; only the curated icons have them
	Bitmap   pixmap.Bitmap
}

// Group is a section of the picker: Name is a stable key such as "Work" for the app to localize.
type Group struct {
	Name  string
	Icons []Entry
}

var (
	// All is the whole library, sorted by name.
	All []Entry
	// Groups is the curated list, in the order the picker shows it.
	Groups []Group

	index map[string]Entry
)

func init() {
	index = make(map[string]Entry, len(categoryIconData))
	for _, r := range categoryIconData {
		e := Entry{Name: r.name, Keywords: r.keywords, Bitmap: pixmap.MustParseHex(r.hex)}
		All = append(All, e)
		index[r.name] = e
	}
	for _, g := range categoryIconGroupData {
		group := Group{Name: g.name}
		for _, n := range g.icons {
			group.Icons = append(group.Icons, index[n])
		}
		Groups = append(Groups, group)
	}
}

// Named finds a library icon by its pixelarticons name.
func Named(name string) (Entry, bool) {
	e, ok := index[name]
	return e, ok
}

// Search lists the icons whose name or Chinese keywords contain q, ignoring case; "-" in a name
// reads as a space. A blank q matches nothing: the picker shows the curated groups then.
func Search(q string) []Entry {
	q = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(q, "-", " ")))
	if q == "" {
		return nil
	}
	var out []Entry
	for _, e := range All {
		if strings.Contains(strings.ReplaceAll(e.Name, "-", " "), q) || strings.Contains(strings.ToLower(e.Keywords), q) {
			out = append(out, e)
		}
	}
	return out
}
