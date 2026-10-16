# M7: handing over, going to the agent, the calendar's day card

User feedback (2026-10-16) on M6 in use: list rows read as unrelated to their group; a row handed
over by an agent could only be answered, never put aside or accepted; "Review" and "Answer" opened
the inspector although the agent is somewhere else; and on the calendar, "2 more" could not be
clicked and a task could not be added right on a day. Four small parts, each its own commit.

## P1: list rows sit in under their group's name

- In a project, the Inbox and the Done group, rows start 26 pt in from the group header, the 18 pt
  avatar under the name rather than under the glyph, so the header reads as their parent. The title
  is as far from the avatar as the header's name is from its glyph. `Show all N →` lines up with the
  rows' titles. Rows that carry a status glyph (Mine, Completed, search) are unchanged.

## P2: "Not now" and "Accept" on rows from agents

- Rows in "From your agents" get a 12 pt close icon at the far right, shown on hover or keyboard
  selection, in a slot that is always reserved so nothing shifts. A click hides the row without
  opening it or changing the task. What is hidden is kept in `dismissed.json` next to the database,
  as the row's `Since`: the row comes back when the agent reports again or GitHub files a new event.
  Entries of tasks that are gone are dropped. A hidden row leaves the sidebar count, the badge and
  the hop.
- Rows of tasks in review get **Accept** (`收下`) next to Review or Open PR: it marks the task done
  and the row fades like any completion.

## P3: going to the agent

- Migration `005_report_cwd.sql`: reports store the `cwd` the agent passed to `report_progress`
  (`''` when it passed none; the server's own folder is not stored). The folder of a row is the
  newest report that has one, else the project's folder, else none.
- Review and Answer on a handed-over row no longer open the inspector (the row still does). They go
  where Settings > Agents > "Open agent work in" says, saved in `prefs.json` next to the database:
  the agent's app (default; `open -a`, the editors also open the folder), Terminal, an editor
  (Cursor, Visual Studio Code or Zed, the first installed) or Finder. An agent with no app, or an
  app that is not installed, falls back to a terminal at the folder. Failures show one quiet line
  under the row's reason.
- The task menu of an agent's task has an `Open in…` page with the four places, for one open that
  does not change the Settings choice. A place with nowhere to go is drawn quiet and does nothing.
- **Limit**: MCP gives no session id, so an agent's app is only brought forward; the exact session
  cannot be opened.

## P4: the calendar's day card

- A day with more items than its cell lists had "N more" as plain text, and a task could only be
  added by selecting a day and typing in the composer at the top. Now "N more" is a link, a hovered
  cell shows a `+`, and a double click on a cell does the same: all open the **day card**.
- The card (`app/views/calendar_day.go`) is an overlay panel whose top left sits on the cell's, grown
  outward, left or up near the window's edges. It lists every item of the day (the cell's rows,
  taller; a click opens the task and closes the card), and has an input that adds a plain task due
  that day at 18:00, keeping the card open. Esc, a click outside, the close button, a change of month
  and leaving the page close it. While it is open the window's shortcuts wait, so the arrows do not
  move the day.
- It opens in four pixel steps, no tween: `theme.DayCardAt` gives 0.55, 0.7, 0.85 every 50 ms and 1
  from 150 ms, done at 200 ms, scaling width and height from the corner it grows from with the
  contents clipped. A frame is asked for every 16 ms only while it plays; Reduce Motion shows it at
  full size at once. Time is `App.Now`, so the tests move a clock.
- mygo notes: an element's `Clicked` is taken by the deepest element that asks for clicks, so "N more"
  and the `+` take their clicks alone; a double click is two `Clicked` and one `DoubleClicked` on
  the cell (tested in `calendar_day_test.go`). The card's backdrop is a `Modal` box in the overlay,
  which takes the presses outside the card and holds back the window's shortcuts.
- Strings: `calendarDay*`, `a11yCalendarShowAll`, `a11yCalendarAddTo`, `a11yCalendarDayCard`,
  `a11yCalendarDayClose` in both catalogs.
- Screens: `calendar-day-card-{light,dark,zh-light}`.

## Done when

- Tests, vet and gofmt clean; the screens regenerated and looked at in light, dark and Chinese; ui-spec
  updated for each part.
