# Operations & Chain Metadata

TODO

The objects are shared across all proposal types and contain the information that allow transactions to be executed across chains.

## Chain Metadata

**Key**
    - **CHAIN_SELECTOR**: The blockchain selector ID, a `uint64` value matching the chain based on the [Chain Selectors](https://github.com/smartcontractkit/chain-selectors)

**Metadata Object**
    - **startingOpCount**: Starting operation count, typically used for parallel signing processes.
    - **mcmAddress**: The MCM contract address that will process this proposal on the respective

## Operations

- **Human-Readable Breadcrumbs**: Non-signed metadata such as `contractType` and `tags` can be used to give additional
  context to the operations, especially during debugging or reviews. Included in both timelock and non-timelock proposal
  types.
- **ContractTypes / ABI Pointers**: These fields allow developers to add helpful references to contracts' ABI, which can
  simplify operations like decoding payloads.
- **Tags**: Tags can help in filtering or categorizing operations, adding flexibility to proposal management.

- **chain_metadata**: Contains chain-specific configuration for each blockchain involved in the proposal:
    - **CHAIN_SELECTOR**: The blockchain selector ID, a `uin64` value matching the chain based on
      the [Chain Selectors Repo Structure](https://github.com/smartcontractkit/chain-selectors)
    - **startingOpCount**: Starting operation count, typically used for parallel signing processes.
    - **mcmAddress**: The MCM contract address that will process this proposal on the respective chain.

- **transactions**: A list of transactions to be executed across chains:
    - **chain**: The blockchain identifier for the specific transaction.
    - **to**: The target contract address.
    - **payload**: The encoded payload (hexadecimal) to be sent with the transaction.
    - **additionalFields**: A chain-specific object with data relevant for the execution of operations on each chain.
    - **contractType** (optional): A pointer to the contract's ABI or other relevant metadata.
    - **tags** (optional): Tags are being considered for categorizing or describing transactions. For
      example, `"EXAMPLE_TAG"`. -->