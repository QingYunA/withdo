# M9: categories of the user's own, and 12 x 12 pixel icons

User request (2026-10-19, adhd-todo task #111, second step): M8's Work and Life are fixed. Users make
their own categories, each with a name, a color and an icon. Decided with the user: Work and Life stay
built in (shown as Work/工作 and Life/生活, always there, not renamed, not deleted, but movable and
re-iconable); every icon is a 12 x 12 bitmap kept in the database and drawn in the category's color;
deleting a category makes its tasks unsorted; a task in a repository's project still counts as Work
unless it says otherwise. Two PRs: PR1 is everything but the drawing editor and image import; PR2 is
those two plus multi-color icons.

## Findings that shape it

- Many pixelarticons do not land on a 12 x 12 grid (1-unit coordinates, 2-unit strokes). Each is
  rasterized at 24 x 24, the best of four offsets (0 or 1 unit each way) is taken, and a 2 x 2 block is
  lit when 2 of its 4 cells are. Of 704 non-sharp icons, 347 convert losslessly, 493 with at most 16
  wrong cells of 576 (what the library keeps); the rest (user, gift, globe, book-open) distort. A curated
  icon that does not convert carries a hand-drawn grid in `docs/design/category-icons.txt`.
- `prefixCols` (`internal/domain/overview.go`) prefixes every column without a "." with `t.`, so the
  category name rides in `taskCols` as a subquery, `(SELECT c.name FROM categories c WHERE c.id = category_id)`.

## Schema, migration 012

- `categories(id, builtin, name, icon, color, position, created_at)`. `builtin` is NULL, `work` or `life`
  (unique); the built-ins are stored as Work and Life and the app localizes them by `builtin`. Names are
  unique case-insensitively (`COLLATE NOCASE` index as a backstop to the domain check).
- `icon` is 144 characters, the 12 rows one after the other (row 0 on top, 12 characters each, no
  separators): `.` is empty and `#` a pixel in the category's color. Lower case letters `a` to `l` are the
  colors of a palette (PR2).
- `color` is one of eight keys, `indigo yellow green red blue pink teal orange`, in the order of
  `theme.ProjectColors` (`domain.CategoryColors` holds the list, `theme.CategoryColor` the colors).
  Work is blue, Life green; a new category defaults to the color its name hashes to, with the same FNV-1a
  hash as `theme.ProjectColor(name)`.
- Work (id 1) and Life (id 2) are seeded with the briefcase and the house. `tasks.category_id` references
  `categories` with `ON DELETE SET NULL`, filled from the old `tasks.category`, which is dropped in place
  (`ALTER TABLE ... DROP COLUMN` works on modernc sqlite; rebuilding `tasks` would cascade into reports and
  pictures).
- Schema version 12 is beyond the old TypeScript core, which cannot open such a database.
- `position` is 1-based and contiguous: new categories append, a move renumbers in one transaction.
- Limits: a name is at most 32 characters, may not be an alias of the built-ins (`work`, `life`, `工作`,
  `生活`), there are at most 24 categories, an icon has at least one pixel.

## Domain, MCP, assistant

- `internal/pixmap`: `Bitmap [12]uint16` with hex, grid and SVG forms and `FromRectSVG` for the library
  conversion; `internal/pixmap/cmd/pxicons` and `scripts/category-icons.sh` generate
  `app/theme/category_icons_gen.go` from the pinned pixelarticons 2.4.2 and the curated list.
- `internal/domain/categories.go`: `ListCategories`, `CreateCategory`, `UpdateCategory`, `MoveCategory`,
  `DeleteCategory` (returns the tasks it unsorted), `RestoreCategory` (the Undo: same id and place, and only
  tasks that are still unsorted come back), `ResolveCategory` (a name in any case; Work and Life also
  answer to their key and Chinese name). A `Task` has `CategoryID` and the read-only `Category` name;
  `CreateTaskInput.CategoryID` and `TaskPatch.CategoryID` are ids.
- MCP: `category` on `create_task` / `update_task` is a name, resolved by `ResolveCategory` (an error lists
  the names that exist); `list_categories` is the ninth tool. Agents cannot create, rename or delete
  categories: that is the user's taxonomy, done in the window.
- Assistant: the prompt lists the categories ("use these names, never invent one") and says a category
  page of Mine is where a new task goes. Receipts carry the key for a built-in category and the name for
  the user's own. Undo restores by id and leaves the task unsorted when the category is gone.

## App

- `data.Category{ID, Builtin, Name, Icon, Color, Position}`; a task has `CategoryID` (0 none); `Store` has
  `ListCategories`, `Manager` the create/update/move/delete/restore calls with sentinel errors; the fake
  and the real store behave alike (`eachStore` tests). The seed adds no user category.
- Views: `snapshot.categories`, `App.Category(id)`, `App.workID()`. PR1's first commits keep the UI as it
  is (Work/Life/Unsorted sub-rows, the three-button strip, the inspector row) on top of the ids; the rest
  of PR1 adds dynamic sub-rows, a category page in the task menu, the New/Edit category sheet with the icon
  picker, and sidebar context menus (New, Edit, Move up/down, Delete with a 6 s Undo).
- Icon picker: curated groups (Work, Study, Health, Money, Home, Hobby, Travel, People) before anything is
  typed, a flat grid of matches (English names, curated Chinese keywords) after.

## PR2 (done in branch claude/category-editor)

- `pixmap.Picture [12][12]byte` (0 empty, `#`, `a` to `l`) with Set, Invert (empty and own color swap,
  palette pixels turn empty), Shift, FlipH, Line (Bresenham), Outline, Grid/ParseGrid; `Bitmap` stays for the
  library. Domain and store accept `a` to `l`; `data.Category.Icon` is a Picture.
- `theme.PaletteColors` (a to h are `ProjectColors`); `theme.CategoryIcon(Picture)` is one SVG with a path per
  color. mygo's `ui.Icon` tints every shape, so category icons are drawn with `ui.Image`, which keeps the
  explicit fills and tints only `currentColor` (pixel test in `category_icons_test.go`).
- The sheet has three modes (library, draw, import). The editor is one `Draw`-painted element with a pointer
  stroke state, a stack of pictures for `⌘Z`, and keyboard arrows and Space. Import: `pixmap.Reduce` (ink, crop,
  10 x 10 averages), `Reduced.Otsu` and `Reduced.Picture` (threshold, nearest palette color, 80 % rule), then
  `Picture.Outline`; `data.ReadIconFile` decodes, or rasters an SVG of right angles.
- Deviations: Otsu is not applied when the classes differ by less than 20 % ink (a drawing that fills its
  square would be split by noise); the file panel for the import is `pickIconFiles` (the composer's `pickFiles`
  has no SVG); four pixelarticons (`arrow-left`, `arrow-down`, `minus`, `flip-horizontal-2`) are vendored for
  the editor.

## Risks

- A category pointed to by an open page may be deleted by another process: the app falls back to Mine.
- Undo of a deletion against concurrent MCP writes: restore only fills tasks that are still unsorted.
- Half the library is lossy; search is by English name, and only curated icons carry Chinese keywords.
