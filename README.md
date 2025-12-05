<div align="center">
  <h1>Many Chain Multisig System</h1>
  <a href='https://github.com/smartcontractkit/mcms/actions/workflows/push-main.yml'><img src="https://github.com/smartcontractkit/mcms/actions/workflows/push-main.yml/badge.svg" /></a>
  <a href="https://smartcontractkit.github.io/mcms/intro/" rel="nofollow">
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

### Running Unit Tests

`task test:unit`

### Running E2E Tests

We use [Chainlink Testing Framework](https://github.com/smartcontractkit/chainlink-testing-framework) for E2E tests. Run them with:

`task test:e2e`

For verbose output just pass -v as command argument:

`task test:e2e -- -v`

By default, we use `anvil` evm. If you want to run e2e tests with specific configuration different chain etc. you need to specify path to the config
after default one to override or add to the previous values. It's pattern of CTF more [here](https://smartcontractkit.github.io/chainlink-testing-framework/framework/test_configuration_overrides.html):

`task test:e2e CTF_CONFIGS=../config.toml,../custom_configs/avax_fuji.toml`

#### Generating MCM Solana compiled program

To run e2e tests for solana blockchain, we need to have the MCM compiled program.
MCM Solana program is located in [chainlink-ccip](https://github.com/smartcontractkit/chainlink-ccip/tree/main/chains/solana/contracts/programs) repo.
We can run `go generate -tags=e2e ./e2e/...` to pull in the latest version of the program from that repo and compile it.
The output will be saved in `e2e/artifacts/solana/` folder.

### Running Ledger Signing Test

For real ledger signing verification you can run:
`task test:ledger`

Remember to connect usb device, unlock it and open ethereum app.

### Running Linters

Run the linters with:

`task lint`

More `lint` commands can be found by running `task -l`

## Documentation

We use [Docusaurus](https://docusaurus.io/) to generate our documentation. You can modify the docs by editing the markdown
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
