<div align="center">
  <h1>Many Chain Multisig System</h1>
  <a href='https://github.com/smartcontractkit/mcms/actions/workflows/push-main.yml'><img src="https://github.com/smartcontractkit/mcms/actions/workflows/push-main.yml/badge.svg" /></a>
  <a href="https://miniature-adventure-5kwz5w3.pages.github.io/" rel="nofollow">
    <img src="https://img.shields.io/static/v1?label=docs&message=latest&color=blue" alt="Official documentation">
  </a>
  <br/>
  <br/>
</div>

Many Chain Multisig System (MCMS) provides tools and libraries to deploy, manage and interact with MCMS across multiple
chains.

## Development

### Getting Started

Install the developments tools and dependencies to get started.

#### Install `asdf`

[asdf](https://asdf-vm.com/) is a tool version manager. All dependencies used for local development of this repo are
managed through `asdf`. To install `asdf`:

1. [Install asdf](https://asdf-vm.com/guide/getting-started.html)
2. Follow the instructions to ensure `asdf` is shimmed into your terminal or development environment

#### Install `task`

[task](https://github.com/go-task/task) is an alternative to `make` and is used to provide commands for everyday
development tasks. To install `task`:

1. Add the asdf task plugin: `asdf plugin add task`
2. Install `task` with `asdf install task`
3. Run `task -l` to see available commands

#### Installing Dependencies

Now that you have `asdf` and `task` installed, you can install the dependencies for this repo:

`task install:tools`

### Running Tests

Run the entire test suite with:

`task test`

More `test` commands can be found by running `task -l`

### Running E2E tests

We are using [Chainlink Testing Framework](https://github.com/smartcontractkit/chainlink-testing-framework) for E2E
tests, so you'll need to setup a `config.toml` you can use the `config.toml.example` in the `e2e`
repo

```shell
CTF_CONFIGS=../config.toml go test -tags=e2e -v ./e2e/...
```

### Running Linters

Run the linters with:

`task lint`

More `lint` commands can be found by running `task -l`

## Documentation

We use [Docsify](https://docsify.js.org) to generate our documentation. You can modify the docs by editing the markdown
files in the [`docs`](https://github.com/smartcontractkit/mcms/tree/main/docs) directory.

Run the local documentation server with:

```
task docs
```

## Contributing

For instructions on how to contribute to `mcms`,
see [CONTRIBUTING.md](https://github.com/smartcontractkit/mcms/blob/main/CONTRIBUTING.md)

## Releasing

For instructions on how to release `mcms`,
see [RELEASE.md](https://github.com/smartcontractkit/mcms/blob/main/RELEASE.md)