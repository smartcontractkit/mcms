# Usage

(OUT OF DATE)

The packages in this directory provide a set of tools that users can use to interact with the MCMS and Timelock
contracts

## Deployment & Configuration

### MCMS

Deploy the MCMS contract using
the [`DeployManyChainMultiSig`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/gethwrappers/ManyChainMultiSig.go#L76)
function

### Timelock

Deploy the RBACTimelock using
the [DeployRBACTimelock](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/gethwrappers/RBACTimelock.go#L47x)
function

Users can configure other addresses with certain roles using the [`GrantRole`](./gethwrappers/RBACTimelock.go#L667)
and [`RevokeRole`](./gethwrappers/RBACTimelock.go#L727) functions

Note: These configurations can only be done by the admin, so it's probably easier to set the deployer as the admin until
configuration is as desired, then use the [`RenounceRole`](./gethwrappers/RBACTimelock.go#L715) to give up `admin`
privileges

### CallProxy

Deploy the call proxy using
the [`DeployCallProxy`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/gethwrappers/CallProxy.go#L41)
function

Note: the `target` in the CallProxy is only configurable during deployment and cannot be set after the fact

## Proposals

Once relevant MCMS/RBACTimelock/CallProxy contracts are deployed, the way users can interact with these contracts is
through a [`Proposal`](./proposal/mcms/proposal.go#L18). At it's core, a `Proposal` is just a list of (currently only
EVM) operations that are to be executed through the MCMS, along with additional metadata about individual transactions
and the proposal as a whole. Proposals come in two flavors:

1. [`MCMSProposal`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/proposal/mcms/proposal.go#L19):
   Represents a simple list of operations (`to`,`value`,`data`) that are to be executed through the mcms with no
   transformation
2. [`MCMSWithTimelockProposal`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/proposal/timelock/mcm_with_timelock.go#L25):
   Represents a list of operations that are to be wrapped in a given timelock operation (`Schedule`,`Cancel`,`Bypass`)
   before being executed through the MCMS. More details about this flavor can be
   found [below](#nuances-of-mcmswithtimelockproposals)

### Construction

Proposal types can be constructed with their respective `NewProposal...` functions. For
example, [`NewMCMSWithTimelockProposal`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/proposal/timelock/mcm_with_timelock.go#L70)
and [`NewMCMSProposal`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/proposal/mcms/proposal.go#L37)

### Proposal Validation

Proposal types all contain a relevant `Validate()` function that can be used to validate the proposal format. This
function is executed by default when using the constructors but for proposals that are incrementally constructed, this
function can be used to revalidate.

### Proposal Signing

`cd cmd && go run main.go --help`

### Proposal Execution

The library provides two functions to help with the execution of an MCMS Proposal:

1. [`SetRootOnChain`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/proposal/mcms/executor.go#L308):
   Given auth and a ChainIdentifer, calls `setRoot` on the target MCMS for that given chainSelector.
2. [`ExecuteOnChain`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/proposal/mcms/executor.go#L348):
   Given auth and an index, calls `execute` on the target MCMS for that given operation.

### Nuances of MCMSWithTimelockProposals

The [`MCMSWithTimelockProposal`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/proposal/timelock/mcm_with_timelock.go#L25)
is an extension of the `MCMSProposal` and has the following additional fields:

1. `Operation`: One of <`Schedule` | `Cancel` | `Bypass`> which determines how to wrap each call in `Transactions`,
   wrapping calls in `scheduleBatch`,`cancel`, and `bypasserExecuteBatch` calls, respectively
2. `MinDelay` is a string representation of a Go `time.Duration` ("1s", "1h", "1d", etc.). This field is only required
   when `Operation == Schedule` and sets the delay for each transaction to be the provided value in seconds
3. `TimelockAddresses` is a map of `ChainSelector` to the target `RBACTimelock` address for each chain.
4. Each element in `Transactions` is now an array of operations that are all to be wrapped in a single `scheduleBatch`
   or `bypasserExecuteBatch` call and executed atomically. There is no concept of batching natively available in the
   MCMS contract which is why this is only available in the RBACTimelock flavor.
