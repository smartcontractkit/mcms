# Integrating a New Chain Family

This guide provides a comprehensive overview for engineers adding MCMS (Many Chain Multi-Sig) support to a new blockchain family.

## Introduction

### What is MCMS?

MCMS is a cross-chain governance system that allows for secure execution of operations across multiple blockchain families. The chain family SDK provides the necessary abstractions to integrate MCMS with different blockchain ecosystems.

### Integration Overview

Adding support for a new chain family involves:

1. Implementing core SDK interfaces for contract interaction
2. Creating chain-specific encoders and decoders
3. Writing comprehensive unit and end-to-end tests
4. Optionally implementing simulation and timelock functionality

### Prerequisites

Before starting, you should have:

- Strong understanding of Go programming
- Deep knowledge of the target blockchain's architecture, transaction format, and smart contract capabilities
- Familiarity with MCMS core concepts (see [Key Concepts](../key-concepts/configuration.md))
- Understanding of the target chain's RPC interfaces and client libraries

## SDK Interfaces Overview

All chain family integrations must implement interfaces defined in the `/sdk` folder. Here's a complete overview:

| Interface           | Status       | Purpose                                                   | Definition                                                                                            |
|---------------------|--------------|-----------------------------------------------------------|-------------------------------------------------------------------------------------------------------|
| `Executor`          | **Required** | Execute MCMS operations on-chain                          | [executor.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/executor.go)                     |
| `Inspector`         | **Required** | Query MCMS contract state                                 | [inspector.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/inspector.go)                   |
| `Encoder`           | **Required** | Hash operations and metadata                              | [encoder.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/encoder.go)                       |
| `ConfigTransformer` | **Required** | Convert between chain-agnostic and chain-specific configs | [config_transformer.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/config_transformer.go) |
| `Configurer`        | **Required** | Update MCMS contract configuration                        | [configurer.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/configurer.go)                 |
| `Decoder`           | Optional     | Decode transaction data for human readability             | [decoder.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/decoder.go)                       |
| `Simulator`         | Optional     | Simulate transactions before execution                    | [simulator.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/simulator.go)                   |
| `TimelockExecutor`  | **Required** | Execute timelock operations                               | [timelock_executor.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/timelock_executor.go)   |
| `TimelockInspector` | **Required** | Query timelock contract state                             | [timelock_inspector.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/timelock_inspector.go) |
| `TimelockConverter` | **Required** | Convert batch operations to timelock operations           | [timelock_converter.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/timelock_converter.go) |

## Required Interfaces

### Executor Interface

The `Executor` is the primary interface for executing MCMS operations on your chain. It embeds both `Inspector` and `Encoder` interfaces.

**Interface Definition:** [sdk/executor.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/executor.go)

**Key Methods:**

- `ExecuteOperation(ctx, metadata, nonce, proof, op)` - Executes a single MCMS operation
- `SetRoot(ctx, metadata, proof, root, validUntil, signatures)` - Sets a new Merkle root with signatures

**Implementation Examples:**

- [EVM Executor](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/executor.go) - Uses go-ethereum bindings
- [Solana Executor](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/executor.go) - Uses Solana program instructions
- [Aptos Executor](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/executor.go) - Uses Move entry functions
- [Sui Executor](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/executor.go) - Uses Sui Move transactions

**Key Considerations:**

- Return `types.TransactionResult` with transaction hash and chain family
- Handle chain-specific transaction signing and submission
- Properly format proofs and signatures for your chain

### Inspector Interface

The `Inspector` queries on-chain state of MCMS contracts.

**Interface Definition:** [sdk/inspector.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/inspector.go)

**Key Methods:**

- `GetConfig(ctx, mcmAddr)` - Retrieves current MCMS configuration
- `GetOpCount(ctx, mcmAddr)` - Gets the current operation count
- `GetRoot(ctx, mcmAddr)` - Returns the current Merkle root and valid until timestamp
- `GetRootMetadata(ctx, mcmAddr)` - Gets metadata for the current root

**Implementation Examples:**

- [EVM Inspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/inspector.go)
- [Solana Inspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/inspector.go)
- [Aptos Inspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/inspector.go)
- [Sui Inspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/inspector.go)

**Key Considerations:**

- Use chain-specific RPC clients to query contract state
- Parse chain-specific data structures into common `types.Config` format
- Handle chain-specific address formats

### Encoder Interface

The `Encoder` creates chain-specific hashes for operations and metadata.

**Interface Definition:** [sdk/encoder.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/encoder.go)

**Key Methods:**

- `HashOperation(opCount, metadata, op)` - Creates a hash of an operation
- `HashMetadata(metadata)` - Creates a hash of chain metadata

**Implementation Examples:**

- [EVM Encoder](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/encoder.go) - Uses Solidity keccak256 encoding
- [Solana Encoder](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/encoder.go) - Uses Borsh serialization + SHA256
- [Aptos Encoder](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/encoder.go) - Uses BCS serialization + SHA3-256
- [Sui Encoder](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/encoder.go) - Uses BCS serialization + Blake2b

**Key Considerations:**

- Match the exact hashing algorithm used by your on-chain contract
- Ensure byte-level encoding matches contract expectations
- Test hash compatibility with contract thoroughly

### ConfigTransformer Interface

The `ConfigTransformer` converts between chain-agnostic `types.Config` and chain-specific configuration structures.

**Interface Definition:** [sdk/config_transformer.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/config_transformer.go)

**Key Methods:**

- `ToChainConfig(cfg, chainSpecificConfig)` - Converts to chain-specific format
- `ToConfig(onchainConfig)` - Converts from chain-specific format to common format

**Implementation Examples:**

- [EVM ConfigTransformer](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/config_transformer.go)
- [Solana ConfigTransformer](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/config_transformer.go)
- [Aptos ConfigTransformer](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/config_transformer.go)
- [Sui ConfigTransformer](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/config_transformer.go)

**Key Considerations:**

- Handle address format conversions between chain-specific and common formats
- Preserve all configuration fields during round-trip conversion
- Support chain-specific configuration parameters via the generic `C` type parameter

### Configurer Interface

The `Configurer` updates MCMS contract configuration on-chain.

**Interface Definition:** [sdk/configurer.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/configurer.go)

**Key Methods:**

- `SetConfig(ctx, mcmAddr, cfg, clearRoot)` - Updates the MCMS configuration

**Implementation Examples:**

- [EVM Configurer](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/configurer.go)
- [Solana Configurer](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/configurer.go)
- [Aptos Configurer](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/configurer.go)
- [Sui Configurer](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/configurer.go)

**Key Considerations:**

- Requires administrative/owner privileges on the MCMS contract
- Handle the `clearRoot` parameter to optionally clear the current root
- Return transaction result with hash and status

## Optional Interfaces

### Decoder Interface

Implement `Decoder` if your chain supports decoding transaction calldata into human-readable format.

**Interface Definition:** [sdk/decoder.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/decoder.go)

**Key Methods:**

- `Decode(op, contractInterfaces)` - Decodes transaction data using contract interfaces

**Return Type:** [sdk/decoded_operation.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/decoded_operation.go)

**Implementations:**

- [EVM Decoder](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/decoder.go) - Uses ABI parsing
- [Aptos Decoder](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/decoder.go) - Uses Move ABI
- [Sui Decoder](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/decoder.go) - Uses Move ABI
- **Note:** Solana does not implement decoding

**When to Implement:**

- Your chain has a standardized interface definition language (like ABI)
- Transaction calldata can be parsed into method names and arguments
- Useful for debugging and operation visualization

### Simulator Interface

Implement `Simulator` if your chain supports transaction simulation/dry-run before execution.

**Interface Definition:** [sdk/simulator.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/simulator.go)

**Key Methods:**

- `SimulateSetRoot(ctx, originCaller, metadata, proof, root, validUntil, signatures)` - Simulates setting a root
- `SimulateOperation(ctx, metadata, operation)` - Simulates executing an operation

**Implementations:**

- [EVM Simulator](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/simulator.go) - Uses `eth_call`
- [Solana Simulator](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/simulator.go) - Uses `simulateTransaction` RPC
- [Sui Simulator](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/simulator.go) - Uses `devInspectTransactionBlock`
- **Note:** Aptos does not currently implement simulation

**When to Implement:**

- Your chain's RPC supports simulation/dry-run capabilities
- Allows validation before actual on-chain execution
- Helps detect errors early without spending gas

## Timelock Interfaces

If your chain supports timelock functionality (scheduled/delayed execution), implement these interfaces:

### TimelockExecutor Interface

**Interface Definition:** [sdk/timelock_executor.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/timelock_executor.go)

Embeds `TimelockInspector` and adds the `Execute` method for executing scheduled operations.

**Implementations:**

- [EVM TimelockExecutor](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/timelock_executor.go)
- [Solana TimelockExecutor](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/timelock_executor.go)
- [Aptos TimelockExecutor](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/timelock_executor.go)
- [Sui TimelockExecutor](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/timelock_executor.go)

### TimelockInspector Interface

**Interface Definition:** [sdk/timelock_inspector.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/timelock_inspector.go)

Queries timelock contract state including roles, operation status, and minimum delay.

**Key Methods:**

- `GetProposers`, `GetExecutors`, `GetBypassers`, `GetCancellers` - Query role members
- `IsOperation`, `IsOperationPending`, `IsOperationReady`, `IsOperationDone` - Check operation status
- `GetMinDelay` - Get minimum timelock delay

**Implementations:**

- [EVM TimelockInspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/timelock_inspector.go)
- [Solana TimelockInspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/timelock_inspector.go)
- [Aptos TimelockInspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/timelock_inspector.go)
- [Sui TimelockInspector](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/timelock_inspector.go)

### TimelockConverter Interface

**Interface Definition:** [sdk/timelock_converter.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/timelock_converter.go)

Converts batch operations into chain-specific timelock operations.

**Key Methods:**

- `ConvertBatchToChainOperations` - Converts batch to chain operations with timelock scheduling

**Implementations:**

- [EVM TimelockConverter](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/timelock_converter.go)
- [Solana TimelockConverter](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/timelock_converter.go)
- [Aptos TimelockConverter](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/timelock_converter.go)
- [Sui TimelockConverter](https://github.com/smartcontractkit/mcms/blob/main/sdk/sui/timelock_converter.go)

## Implementation Guidelines

### Package Structure

Create a new package under `/sdk/<chain-family>/` with the following typical structure:

```
sdk/
└── <chain-family>/
    ├── encoder.go                    # Encoder implementation
    ├── encoder_test.go               # Encoder tests
    ├── executor.go                   # Executor implementation
    ├── executor_test.go              # Executor tests
    ├── inspector.go                  # Inspector implementation
    ├── inspector_test.go             # Inspector tests
    ├── configurer.go                 # Configurer implementation
    ├── configurer_test.go            # Configurer tests
    ├── config_transformer.go         # ConfigTransformer implementation
    ├── config_transformer_test.go    # ConfigTransformer tests
    ├── decoder.go                    # Decoder implementation (optional)
    ├── decoder_test.go               # Decoder tests
    ├── simulator.go                  # Simulator implementation (optional)
    ├── simulator_test.go             # Simulator tests
    ├── timelock_executor.go          # TimelockExecutor (if supported)
    ├── timelock_executor_test.go
    ├── timelock_inspector.go         # TimelockInspector (if supported)
    ├── timelock_inspector_test.go
    ├── timelock_converter.go         # TimelockConverter (if supported)
    ├── timelock_converter_test.go
    ├── transaction.go                # Transaction utilities
    ├── transaction_test.go
    ├── utils.go                      # Chain-specific helpers
    ├── utils_test.go
    └── mocks/                        # Generated mocks for testing
        └── ...
```

**Reference:** See [sdk/evm/](https://github.com/smartcontractkit/mcms/tree/main/sdk/evm) for a complete example.

### Chain-Specific Considerations

#### Transaction Formatting

- Each chain has unique transaction structures (e.g., EVM calldata, Solana instructions, Move entry functions)
- Ensure your implementation correctly formats transactions for contract calls
- Handle nonce/sequence numbers according to chain requirements

#### Address Encoding

- Implement conversion between chain-specific address formats (hex, base58, bech32, etc.) and common representations
- Store addresses in a consistent format within `types.Config`
- See [types/chain_metadata.go](https://github.com/smartcontractkit/mcms/blob/main/types/chain_metadata.go) for metadata requirements

#### Signature Handling

- Different chains use different signature schemes (ECDSA, EdDSA, etc.)
- Ensure signature verification matches on-chain expectations
- Handle signature serialization correctly (r,s,v format vs compact format, etc.)

#### Chain Metadata Requirements

- Define required metadata fields in `types.ChainMetadata`
- Include MCMS contract address and chain selector
- Add chain-specific parameters as needed

#### Additional Fields in Operations

- Use `types.Transaction.AdditionalFields` (JSON) for chain-specific data
- Examples: Solana account lists, Aptos type arguments, Sui object references
- Document expected structure for your chain

### Error Handling

Use the `/sdk/errors/` package for standardized error handling:

**Reference:** [sdk/errors/errors.go](https://github.com/smartcontractkit/mcms/blob/main/sdk/errors/errors.go)

**Common Error Patterns:**

- Wrap errors with context using `fmt.Errorf("operation failed: %w", err)`
- Return specific errors for common failure cases (insufficient signatures, invalid proof, etc.)
- Use typed errors for cases that callers may need to handle specifically

## Testing Requirements

### Unit Tests

Each interface implementation needs a corresponding `_test.go` file with comprehensive coverage (>80%). Test all public methods with both success and failure cases using table-driven tests. Mock external dependencies (RPC clients, contracts) in `sdk/<chain>/mocks/`.

**Test Examples:**
- [EVM Tests](https://github.com/smartcontractkit/mcms/blob/main/sdk/evm/executor_test.go) | [Mock Examples](https://github.com/smartcontractkit/mcms/tree/main/sdk/evm/mocks)
- [Solana Tests](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/encoder_test.go) | [Mocks](https://github.com/smartcontractkit/mcms/tree/main/sdk/solana/mocks)
- [Aptos Tests](https://github.com/smartcontractkit/mcms/blob/main/sdk/aptos/inspector_test.go) | [Mocks](https://github.com/smartcontractkit/mcms/tree/main/sdk/aptos/mocks)

### E2E Tests

Create test suite under `/e2e/tests/<chain-family>/` covering:

| Test Category | Example | Key Coverage |
|---------------|---------|--------------|
| **Config Management** | [solana/set_config.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/set_config.go) | Set/update config, retrieve and verify, clearRoot flag |
| **Root Operations** | [solana/set_root.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/set_root.go) | Set root with signatures, quorum requirements, expiration |
| **Operation Execution** | [solana/execute.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/execute.go) | Execute with valid proof, verify effects, test invalid proofs |
| **Contract Inspection** | [solana/inspection.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/inspection.go) | Query config, op count, root, metadata |
| **Simulation** (optional) | [solana/simulator.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/simulator.go) | Simulate valid/invalid ops, verify no state changes |
| **Timelock Conversion** (optional) | [solana/timelock_converter.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/timelock_converter.go) | Convert batch to timelock ops, verify IDs and actions |
| **Timelock Execution** (optional) | [solana/timelock_execution.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/timelock_execution.go) | Schedule with delay, execute after delay, predecessors |
| **Timelock Inspection** (optional) | [solana/timelock_inspection.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/timelock_inspection.go) | Query roles, operation status, minimum delay |
| **Timelock Cancellation** (optional) | [aptos/timelock_cancel.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/aptos/timelock_cancel.go) | Cancel pending ops, verify cancellation |

**Test Suite Setup:**
1. Create `e2e/config.<chain>.toml` ([example](https://github.com/smartcontractkit/mcms/blob/main/e2e/config.evm.toml))
2. Update [e2e/tests/setup.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/setup.go) with blockchain node and RPC clients
3. Add suite to [e2e/tests/runner_test.go](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/runner_test.go)
4. Create helpers in `common.go` ([example](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/common.go))

## Reference Implementations

When implementing your integration, refer to these existing implementations:

1. **EVM**: [sdk/evm/](https://github.com/smartcontractkit/mcms/tree/main/sdk/evm) - Most mature, includes all features
2. **Solana**: [sdk/solana/](https://github.com/smartcontractkit/mcms/tree/main/sdk/solana) - Excellent example of chain-specific complexity
3. **Aptos**: [sdk/aptos/](https://github.com/smartcontractkit/mcms/tree/main/sdk/aptos) - Move-based chain without simulation
4. **Sui**: [sdk/sui/](https://github.com/smartcontractkit/mcms/tree/main/sdk/sui) - Recent addition with good patterns

