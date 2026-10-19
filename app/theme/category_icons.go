package theme

import (
	"strings"
	"sync"

	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// A category's icon is a 12 x 12 one-bit bitmap kept in the database. The picker offers the
// pixelarticons that reduce to that grid with little loss (category_icons_gen.go, written by
// scripts/category-icons.sh) and, first, a curated list in groups (docs/design/category-icons.txt).

// categoryIconRow and categoryIconGroupRow are the shapes of the generated tables.
type (
	categoryIconRow      struct{ name, hex, keywords string }
	categoryIconGroupRow struct {
		name  string
		icons []string
	}
)

// CategoryIconEntry is one icon of the library.
type CategoryIconEntry struct {
	Name     string // the pixelarticons name, such as "coffee"
	Keywords string // Chinese words the search also reads; only the curated icons have them
	Bitmap   pixmap.Bitmap
}

// CategoryIconGroup is a section of the picker: Name is a stable key such as "Work" for the app to
// localize.
type CategoryIconGroup struct {
	Name  string
	Icons []CategoryIconEntry
}

var (
	// CategoryIcons is the whole library, sorted by name.
	CategoryIcons []CategoryIconEntry
	// CategoryIconGroups is the curated list, in the order the picker shows it.
	CategoryIconGroups []CategoryIconGroup

	categoryIconIndex map[string]CategoryIconEntry
)

func init() {
	categoryIconIndex = make(map[string]CategoryIconEntry, len(categoryIconData))
	for _, r := range categoryIconData {
		e := CategoryIconEntry{Name: r.name, Keywords: r.keywords, Bitmap: pixmap.MustParseHex(r.hex)}
		CategoryIcons = append(CategoryIcons, e)
		categoryIconIndex[r.name] = e
	}
	for _, g := range categoryIconGroupData {
		group := CategoryIconGroup{Name: g.name}
		for _, n := range g.icons {
			group.Icons = append(group.Icons, categoryIconIndex[n])
		}
		CategoryIconGroups = append(CategoryIconGroups, group)
	}
}

// CategoryIconNamed finds a library icon by its pixelarticons name.
func CategoryIconNamed(name string) (CategoryIconEntry, bool) {
	e, ok := categoryIconIndex[name]
	return e, ok
}

// SearchCategoryIcons lists the icons whose name or Chinese keywords contain q, ignoring case; "-" in
// a name reads as a space. A blank q matches nothing: the picker shows the curated groups then.
func SearchCategoryIcons(q string) []CategoryIconEntry {
	q = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(q, "-", " ")))
	if q == "" {
		return nil
	}
	var out []CategoryIconEntry
	for _, e := range CategoryIcons {
		if strings.Contains(strings.ReplaceAll(e.Name, "-", " "), q) || strings.Contains(strings.ToLower(e.Keywords), q) {
			out = append(out, e)
		}
	}
	return out
}

var (
	categorySVGMu sync.Mutex
	categorySVGs  = map[pixmap.Bitmap]*ui.SVG{}
)

// CategoryIcon is the bitmap as an SVG in the color of the surrounding text, parsed once per
// bitmap. Draw it at the same whole multiples of 12 pt as the interface icons.
func CategoryIcon(b pixmap.Bitmap) *ui.SVG {
	categorySVGMu.Lock()
	defer categorySVGMu.Unlock()
	if s, ok := categorySVGs[b]; ok {
		return s
	}
	s := ui.MustParseSVG(b.SVG())
	categorySVGs[b] = s
	return s
}
