# MCMS Proposal

The **MCMS Proposal** is a structured document that defines a set of operations to be executed across multiple blockchains. Proposals are typically created to carry out specific transactions, like contract interactions or asset transfers, in a coordinated manner across different chains.

> [!NOTE]
> An MCMS Proposal is used specifically when controlling only MCM contract.
> If you require timelock and batching functionality, consider using the [RBAC Timelock proposal](./timelock-proposal.md).

## Proposal Structure

The MCMS Proposal can be serialized into a JSON document with the following structure:

<!-- panels:start -->
<!-- div:left-panel -->
```json
{
  "version": "v1",
  "kind": "Proposal",
  "description": "Set a value on the contract",
  "validUntil": "1920671473",
  "overridePreviousRoot": false,
  "signatures": [
    {
        "r": "0x1",
        "s": "0x2",
        "v": 0,
    },
    {
        "r": "0x3",
        "s": "0x4",
        "v": 0,
    },
  ],
  "chainMetadata": {
    "16015286601757825753": {
      "startingOpCount": 1,
      "mcmAddress": "0x0"
    }
  },
  "transactions": [
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
    },
    {
      "chainSelector": "16015286601757825753",
      "to": "0xb",
      "payload": "ZGF0YQ==",
      "additionalFields": {
        "value": 0
      },
      "contractType": "<CONTRACT_TYPE>",
      "tags": [
        "tag1"
      ]
    }
  ]
}
```

<!-- div:right-panel -->

### Proposal Field Descriptions

**version** string<br/>
The version of the proposal format to ensure backward compatibility for different parsers. Only `v1` is supported at the moment.

---

**kind** string<br/>
Specifies the type of proposal. In this case, it should be set to `Proposal`.

---

**description** string _optional_<br/>
A human-readable (and typically generated) description intended to give signers context for the proposed change.

---

**validUntil** uint32<br/>
A Unix timestamp that specifies the proposal's expiration. If the proposal is not executed before this time, it becomes invalid.

---

**signatures** array of objects<br/>
A list of cryptographic proposal signatures of the signers, where each element represents one Signature object with their R,S and V values. These ensure that the proposal has been agreed upon by the necessary parties.

---

**chainMetadata** object<br/>
Maps the chain-specific configuration for each blockchain involved in the proposal. The key of the object is the chain selector ID, and the value is the metadata object. An entry is required for every chain referenced in the proposal's operations.

For more details about the chain metadata, see [Chain Metadata](/key-concepts/operations-and-chain-metadata.md#chain-metadata).

---

**transactions** array of objects<br/>
A list of operations to be executed across chains.

For more details about the operations, see [Operations](/key-concepts/operations-and-chain-metadata.md#operations).
<!-- panels:end -->