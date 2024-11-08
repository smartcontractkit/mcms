# Release Process

<!-- TOC -->

- [Release Process](#release-process)
    - [Preparing a Release](#preparing-a-release)
    - [How to Release](#how-to-release)

<!-- TOC -->

### Preparing a Release

After every PR with a changeset is merged, a changesets CI job will create or update a "Version Packages" PR, which contains the release version and information about the changes.

### How to Release

1. Approve or request approval to merge the "Version Packages" PR.
2. Merge the "Version Packages" PR.
3. This will trigger the release workflow, automatically releasing a new version and pushing a tag for the version. Check the [release view](https://github.com/smartcontractkit/mcms/releases) to confirm the latest release.
