-- Categories the user keeps: Work and Life are built in (builtin 'work' / 'life', stored in English
-- and localized by the app) and cannot be renamed or deleted; the rest are the user's own.
-- icon is a 12 x 12 picture as 144 characters, row 0 first and 12 characters a row: '.' is empty
-- and '#' a pixel in the category's color (the lower case letters are kept for the colors of a
-- later palette). color names one of the eight project colors, in the order of theme.ProjectColors.
-- position is 1-based and contiguous.
CREATE TABLE categories (
  id         INTEGER PRIMARY KEY,
  builtin    TEXT UNIQUE CHECK (builtin IN ('work','life')),
  name       TEXT NOT NULL,
  icon       TEXT NOT NULL CHECK (length(icon) = 144),
  color      TEXT NOT NULL CHECK (color IN ('indigo','yellow','green','red','blue','pink','teal','orange')),
  position   INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX categories_name ON categories(name COLLATE NOCASE);

-- The briefcase and the house of the interface icons, as pictures.
INSERT INTO categories (id, builtin, name, icon, color, position, created_at) VALUES
  (1, 'work', 'Work',
     '............'
     || '.....##.....'
     || '....#..#....'
     || '..########..'
     || '.#........#.'
     || '.#........#.'
     || '.#........#.'
     || '.#........#.'
     || '.#........#.'
     || '.#........#.'
     || '..########..'
     || '............',
   'blue', 1, strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  (2, 'life', 'Life',
     '............'
     || '.....##.....'
     || '....#..#....'
     || '...#....#...'
     || '..#......#..'
     || '.#........#.'
     || '.#...##...#.'
     || '.#..#..#..#.'
     || '.#..#..#..#.'
     || '.#..#..#..#.'
     || '..########..'
     || '............',
   'green', 2, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));

-- tasks.category (007) held 'work', 'life' or NULL; the id replaces it. A deleted category leaves
-- its tasks unsorted. The column is dropped in place: rebuilding tasks would cascade into its
-- reports and pictures.
ALTER TABLE tasks ADD COLUMN category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL;
UPDATE tasks SET category_id = (SELECT id FROM categories WHERE builtin = tasks.category) WHERE category IS NOT NULL;
ALTER TABLE tasks DROP COLUMN category;
CREATE INDEX tasks_category ON tasks(category_id);
