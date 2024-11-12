# Operations & Chain Metadata

Operations and Chain Metadata are shared across all proposal types and contain the information that allow transactions to be executed across chains.

## Chain Metadata

ChainMetadata is a map of chain-specific configuration for each blockchain involved in the proposal. The key of the object is the chain selector ID, and the value is the metadata object. An entry is required for every chain referenced in the proposal's operations.

<!-- panels:start -->
<!-- div:left-panel -->
### Chain Metadata Structure

```json
{
  "16015286601757825753": {
    "startingOpCount": 1,
    "mcmAddress": "0x0"
  }
}
```

<!-- div:right-panel -->

### Chain Selector ID

**Key** uint64<br/>
The chain selector ID matching the chain based on the [Chain Selectors](https://github.com/smartcontractkit/chain-selectors) library.

### Metadata Object

**startingOpCount** uint64<br/>
The starting operation count, typically used for parallel signing processes.

---

**mcmAddress** string<br/>
The MCM contract address that will process this proposal on the respective chain.

<!-- panels:end -->

## Operations

Operations contain the information required to execute transactions. Metadata fields (`Contract Types` and `Tags`) are optional and can be used to add helpful references to contracts' ABI, simplify operations like decoding payloads, or to categorize operations.

<!-- panels:start -->

<!-- div:left-panel -->

```json
{
  "chainSelector": "16015286601757825753",
  "to": "0xa",
  "data": "ZGF0YQ==",
  "additionalFields": {
    "value": 0
  },
  "contractType": "<CONTRACT_TYPE>",
  "tags": [
    "tag1"
  ]
}
```

<!-- div:right-panel -->

**chainSelector** uint64<br/>
The chain selector id for the chain that the operation will be executed on.

---

**to** string<br/>
The target contract address.

---

**data** string<br/>
The encoded data (hexadecimal) to be sent with the transaction.

---

**additionalFields** object<br/>
A chain family specific object with data relevant for the execution of operations on each chain.

---

**contractType** string _optional_<br/>
A pointer to the contract's ABI or other relevant metadata. This field is not included in the Merkle tree's hashed transaction data.

---

**tags** array _optional_<br/>
Tags for categorizing or describing transactions. This field is not included in the Merkle tree's hashed transaction data.

<!-- panels:end -->