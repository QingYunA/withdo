# Changelog

Each release has a section here. When updates are on (`docs/plans/m10-auto-update.md`), `mygo build`
reads the update window's notes from the section of the version (`## 0.5.0`) and fails without one.

## 0.5.0

Upgrading from 0.4.0: quit your coding agents first and back up `~/Library/Application Support/adhd-todo/`.
The database moves from schema 11 to 13, which 0.4.0's adhd-core cannot read; on its first launch
the app points `~/.local/bin/adhd-core` at its own copy (the old file is kept as `adhd-core.old`).

- **adhd-core comes with the app.** The app links `~/.local/bin/adhd-core`, the path your coding
  agents run, to the copy inside it, so installing the app brings both. A coding agent keeps running
  the old adhd-core until it is restarted; restart it after installing. `adhd-core --version` prints
  the version.
- **Your own categories**, with 12 x 12 pixel icons in color, an icon editor, and import from a
  picture or SVG. Work and Life stay as the built-in two.
- **Sub-projects**: a project can hang under another in the sidebar (right-click > New sub-project).
- **Recent changes**, a page off by default (Settings > General > Pages) that reads your projects'
  repositories with `gh` and has the assistant explain what merged.
- **Summary**: quiet sound effects while the day is shipped (Settings > General can mute them), and
  six people in the scene, with figures of their own for Gemini, Cursor, Copilot and OpenCode.
- Fixes: completing a task from a grouped list by its hover glyph, the Summary ledger's two-digit
  counts, and the test suite and SQLite on Windows.

## 0.4.0

- **Summary page**: the day's and the week's shipping, and an end-of-day reminder.
- **Pictures on tasks**, and **selecting several tasks** to act on them together, each with undo.
- **Pages from agents**: an agent can attach an HTML report or a design draft to its report.
- **System notifications** when an agent finishes or gets stuck, or a pull request fails or merges.
- **Agents set themselves up**: `adhd-core guide` prints what any coding agent needs to add the MCP
  server, and Settings > Agents has more harnesses.
- The pixel look all through the interface.
