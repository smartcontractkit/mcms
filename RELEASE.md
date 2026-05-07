# Release Process

<!-- TOC -->

- [Release Process](#release-process)
  - [Release Please](#release-please)
  - [Semantic Versioning](#semantic-versioning)
  - [Preparing a Release](#preparing-a-release)
  - [How to Release](#how-to-release)

<!-- TOC -->

## Release Please

This repo uses [Release Please](https://github.com/googleapis/release-please) to manage releases. Do not run `pnpm changeset` or add changeset files for releases.

Release Please determines the next release from Conventional Commit messages on `main`. Because PR titles are linted, use a Conventional Commit title that describes the user-visible impact of the change:

- `feat: add support for new deployment workflow` for backwards-compatible features
- `fix: handle missing workflow artifacts` for backwards-compatible bug fixes
- `perf: reduce datastore lookup latency` for performance improvements
- `feat!: remove deprecated workflow input` or a commit body with `BREAKING CHANGE:` for incompatible API changes

Other types such as `docs`, `test`, `ci`, `chore`, `build`, `style`, and `refactor` are allowed, but they are hidden from the changelog by the current Release Please configuration and usually do not create a release by themselves.

When writing the PR title and description, make it clear:

- **WHAT** the change is
- **WHY** the change was made
- Whether the change is breaking or requires downstream action

## Semantic Versioning

Release Please follows semantic versioning when choosing the next version:

- **MAJOR** version when you make incompatible API changes
- **MINOR** version when you add functionality in a backwards-compatible manner
- **PATCH** version when you make backwards-compatible bug fixes

## Preparing a Release

The `release-please` workflow runs after changes are merged to `main`. When Release Please finds releasable changes, it opens or updates a release PR that includes the version bump, changelog updates, and `.release-please-manifest.json` changes.

## How to Release

1. Review the Release Please PR and confirm the changelog and version are correct.
2. Approve or request approval to merge the Release Please PR.
3. Merge the Release Please PR into `main`.
4. The `release-please` workflow will create the GitHub release and push the version tag automatically.
