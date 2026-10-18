# adhd-todo

- Checks: `scripts/check.sh` while working, `scripts/check.sh full` before opening a PR. Run tests in the foreground with a Bash timeout; no polling with sleep.
- UI changes: run `scripts/screens.sh` and commit the changed PNGs in `docs/design/screens`.
- Design work: mockups are `docs/design/mockups/vN.html` (+ png via `docs/design/tools/shot.sh`); decisions go into `docs/design/ui-spec.md`, and graphics follow its Pixel language section.
- The spec is ~1200 lines: `grep -n '^#' docs/design/ui-spec.md` to find a section.
