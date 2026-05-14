# Contributing

<!-- TOC -->

- [Contributing](#contributing)
  - [Team Overview](#team-overview)
  - [How to Contribute](#how-to-contribute)
  - [Conventional Commits](#conventional-commits)

<!-- TOC -->

## Team Overview

The Operations Platform ([@smartcontractkit/operations-platform](https://github.com/orgs/smartcontractkit/teams/operations-platform)) team is responsible for the development and maintenance of this repo, and are the primary code owners and reviewers.

## How to Contribute

1. Open a branch from `main` and give it a descriptive name.
2. Make changes on your branch.
3. Push your branch and open a PR against `main`.
4. Give the PR a Conventional Commit title. This title is used by Release Please to decide the next version and changelog entry when the PR is merged.
5. Ensure your PR passes all CI checks.
6. Request a review from the Operations Platform team ([@smartcontractkit/operations-platform](https://github.com/orgs/smartcontractkit/teams/operations-platform)).

## Conventional Commits

PR titles must follow the Conventional Commits format:

```text
<type>: <description>
```

Release Please uses the merged PR titles on `main` to decide whether to create a release and which semantic version bump to apply:

- `feat: add support for new deployment workflow` creates a minor release.
- `fix: handle missing workflow artifacts` creates a patch release.
- `perf: reduce datastore lookup latency` creates a patch release.
- `feat!: remove deprecated workflow input` creates a major release.
- `docs: update release instructions`, `test: add workflow deploy coverage`, `ci: update pull request checks`, `chore: update dependencies`, `build: update go version`, `style: format generated code`, and `refactor: simplify deploy input handling` are allowed but usually do not create a release by themselves.

Use `!` after the type, or include `BREAKING CHANGE:` in the commit body, when the change is incompatible with existing users.

For details on how Release Please uses Conventional Commits and semantic versioning, see [RELEASE.md](./RELEASE.md).

