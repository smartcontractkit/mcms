# Contributing

<!-- TOC -->

- [Contributing](#contributing)
  - [Team Overview](#team-overview)
  - [How to Contribute](#how-to-contribute)
  - [Changesets](#changesets)

<!-- TOC -->

## Team Overview

The Deployment Automation ([@smartcontractkit/deployment-automation](https://github.com/orgs/smartcontractkit/teams/deployment-automation)) team is responsible for the development and maintenance of this repo, and are the primary code owners and reviewers.

## How to Contribute

1. Open a branch from `main` and give it a descriptive name.
2. Make changes on your branch.
3. When you are ready to submit your changes, [create a changeset](#changesets) with `pnpm changeset` and commit the changeset file.
4. Push your branch and open a PR against `main`.
5. Ensure your PR passes all CI checks.
6. Request a review from the Deployment Automation team ([@smartcontractkit/deployment-automation](https://github.com/orgs/smartcontractkit/teams/deployment-automation)).

## Changesets

Changesets are a way to manage changes to the codebase that are not yet released. Here are a few things to keep in mind when you create a changeset:

- Following semantic versioning, select between: major, minor and patch
  - **MAJOR** version when you make incompatible API changes
  - **MINOR** version when you add functionality in a backwards compatible manner
  - **PATCH** version when you make backwards compatible bug fixes.
- When describing the change, try to answer the following:
  - **WHAT** the change is
  - **WHY** the change was made
