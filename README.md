<div align="center">
  <h1>Many Chain Multisig System</h1>
  <a href='https://github.com/smartcontractkit/mcms/actions/workflows/push-main.yml'><img src="https://github.com/smartcontractkit/mcms/actions/workflows/push-main.yml/badge.svg" /></a>
  <br/>
  <br/>
</div>

Many Chain Multisig System (MCMS) provides tools and libraries to deploy, manage and interact with MCMS across multiple chains.

## Development

### Getting Started

Install the developments tools and dependencies to get started.

#### Install `asdf`

[asdf](https://asdf-vm.com/) is a tool version manager. All dependencies used for local development of this repo are managed through `asdf`. To install `asdf`:

1. [Install asdf](https://asdf-vm.com/guide/getting-started.html)
2. Follow the instructions to ensure `asdf` is shimmed into your terminal or development environment

#### Install `task`

[task](https://github.com/go-task/task) is an alternative to `make` and is used to provide commands for everyday development tasks. To install `task`:

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

### Running Linters

Run the linters with:

`task lint`

More `lint` commands can be found by running `task -l`

## Documentation

We use [Docsify](https://docsify.js.org) for our documentations.

```bash
pnpm install
pnpm run docs
````
