# adhd-todo

- Checks: `scripts/check.sh` while working, `scripts/check.sh full` before opening a PR. Run a command that takes over 2 minutes (`check.sh full`, `screens.sh`) with `run_in_background` and wait for its notification. Wait for PR CI with the ccd_pr monitor, not `gh pr checks --watch`. When CI fails but local checks pass, reproduce with `scripts/check.sh linux` before guessing a cause.
- UI changes: run `scripts/screens.sh` and commit the changed PNGs in `docs/design/screens`.
- Android UI changes: run `scripts/android-screens.sh` and commit the changed PNGs in `docs/design/screens/android`. Android checks (Kotlin unit tests + lint, needs a JDK 17): `scripts/check.sh android`.
- Design work: mockups are `docs/design/mockups/vN.html` (+ png via `docs/design/tools/shot.sh`); decisions go into `docs/design/ui-spec.md`, and graphics follow its Pixel language section.
- The spec is ~2500 lines: `grep -n '^#' docs/design/ui-spec.md` to find a section.
- Releases ship macOS and Windows x64 every time: `scripts/build-release.sh` builds both into `app/build/release/`; steps in `docs/plans/m10-auto-update.md` (Release steps).
