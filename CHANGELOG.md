# Changelog

`mygo build` reads the release notes of an update from the section of its version (`## 0.5.0`), and
fails when the file has no such section. The release PR bumps the version and finishes the section.

## 0.5.0

The last version to install by hand: from here on the app updates itself.

- **Automatic updates.** "Check for Updates…" in the app menu, and a daily background check with the
  usual window: release notes, Install Update, Remind Me Later, Skip This Version. Turn the
  background checks off in Settings > General > Software updates.
- **adhd-core comes with the app.** The app links `~/.local/bin/adhd-core`, the path your coding
  agents run, to the copy inside it, so an update brings both. A coding agent keeps running the old
  adhd-core until it is restarted; restart it after an update. `adhd-core --version` prints the
  version.
- **Your own categories**, with 12 x 12 pixel icons in color, an icon editor, and import from a
  picture or SVG. Work and Life stay as the built-in two.
- **Sub-projects**: a project can hang under another in the sidebar (right-click > New sub-project).
- **Recent changes**, a page off by default (Settings > General > Pages) that reads your projects'
  repositories with `gh` and has the assistant explain what merged.
- Fixes: completing a task from a grouped list by its hover glyph, the Summary ledger's two-digit
  counts, and the test suite and SQLite on Windows.

## 0.4.0

- **Summary page**: the day's and the week's shipping, an end-of-day reminder, quiet sound effects.
- **Pictures on tasks**, and **selecting several tasks** to act on them together, each with undo.
- **Pages from agents**: an agent can attach an HTML report or a design draft to its report.
- **System notifications** when an agent finishes or gets stuck, or a pull request fails or merges.
- **Agents set themselves up**: `adhd-core guide` prints what any coding agent needs to add the MCP
  server, and Settings > Agents has more harnesses.
- The pixel look all through the interface.
