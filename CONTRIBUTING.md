# Contributing

Thanks for helping. Bug reports, translations, design notes and code are all welcome.

## Before you start

- **Bugs and ideas**: open an issue first for anything bigger than a small fix, so we can agree on the
  approach before you spend time on it.
- **Security problems**: do not open a public issue. Report them privately through
  [GitHub's security advisories](https://github.com/QingYunA/withdo/security/advisories/new).
- **The CLA**: pull requests from outside the project need a signed
  [Contributor License Agreement](CLA.md). The CLA bot asks you to sign it, once, on your first pull
  request. You keep the copyright; the agreement lets the project ship your work in the open-source
  app and in the hosted service, and promises that it stays open source.

## Set up

You need Go (the version in `go.mod`). Android work also needs JDK 17.

```sh
git config core.hooksPath .githooks              # once: the static checks run on every commit
cd app && go tool mygo dev                       # run the app against your real database and agent configs
go run ./app -fake                               # or with demo data
```

## While you work

- Run `scripts/check.sh` often (gofmt, vet, i18n, short tests), and `scripts/check.sh full` before you
  open the pull request. Android changes: `scripts/check.sh android`.
- Changes to the desktop interface: run `scripts/screens.sh` and commit the changed PNGs in
  `docs/design/screens`. Android interface: `scripts/android-screens.sh`.
- The interface follows `docs/design/ui-spec.md`; every picture follows its **Pixel language** section
  (pixel art on a fixed grid, no system icons or emoji). Find a section with
  `grep -n '^#' docs/design/ui-spec.md`.
- New user-facing text goes through `app/locales` (English is the source) and, on Android,
  `strings.xml`.

## Pull requests

- One change per pull request, with tests for what it fixes or adds.
- Commit messages: `<type>: <description>`, where type is one of `feat`, `fix`, `refactor`, `docs`,
  `test`, `chore`, `perf`, `ci`.
- Say in the description what changed, why, and how you checked it; attach screenshots for interface
  changes.

## License

By contributing you agree that your work is released under the project's
[AGPL-3.0 license](LICENSE) and the terms of the [CLA](CLA.md).
