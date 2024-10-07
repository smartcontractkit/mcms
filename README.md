# mcms

Tools/Libraries to Deploy/Manage/Interact with MCMS

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
