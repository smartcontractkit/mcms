# TimeLock Proposal

The **MCMS + Timelock Implementation** is an extended proposal document designed for teams that incorporate the `RBACTimelock` contract as part of their contract ownership structure. It builds upon the base MCMS proposal structure by adding timelock configurations and actions like scheduling, cancelling, or bypassing transactions.

## Features

- **Timelock Configurations**: Timelocks allow delaying the execution of transactions, giving signers time to review and, if necessary, cancel operations.
- **Batching**: Transactions can be grouped into batches to be executed together, ensuring atomicity.
- **Parallel Signing**: The nonce offset helps manage nonces when signing transactions across different chains simultaneously.

## Timelock Proposal Structure

<!-- panels:start -->
<!-- div:left-panel -->

```json
{
  "version": "v1",
  "kind": "TimelockProposal",
  "description": "Set a value on the contract",
  "validUntil": "1920671473",
  "action": "schedule",
  "delay": "24h",
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
  "timelockAddresses": {
    "16015286601757825753": "0x0g"
  },
  "operations": [
    {
      "chainSelector": "16015286601757825753",
      "transactions": [
        {
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
      ]
    },
    {
      "chainSelector": "16015286601757825753",
      "transactions": [
        {
          "to": "0xb",
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
          "to": "0xc",
          "data": "ZGF0YQ==",
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
  ]
}
```

<!-- div:right-panel -->

### Proposal Field Descriptions

**version** string<br/>
The version of the proposal format to ensure backward compatibility for different parsers. Only `v1` is supported at the moment.

---

**kind** string<br/>
Specifies the type of proposal. In this case, it should be set to `TimelockProposal`.

---

**description** string _optional_<br/>
A human-readable (and typically generated) description intended to give signers context for the proposed change.

---

**validUntil** uint32<br/>
A Unix timestamp that specifies the proposal's expiration. If the proposal is not executed before this time, it becomes invalid.

---

**action** string<br/>
Specifies the high-level action for the proposal. Can be one of:
- `schedule`: Sets up transactions to execute after a delay.
- `cancel`: Cancels previously scheduled transactions.
- `bypass`: Directly executes transactions, skipping the timelock.

---

**delay** string<br/>
The delay duration that applies to all transactions, ensuring that they are held in the timelock for the specified time. The format is a string with a number followed by a time unit (e.g., `24h`).

---

**signatures** array of objects<br/>
A list of cryptographic proposal signatures of the signers, where each element represents one Signature object with their R,S and V values. These ensure that the proposal has been agreed upon by the necessary parties.

---

**chainMetadata** object<br/>
Maps the chain-specific configuration for each blockchain involved in the proposal. The key of the object is the chain selector ID, and the value is the metadata object. An entry is required for every chain referenced in the proposal's operations.

For more details about the chain metadata, see [Chain Metadata](./chain-metadata.md).

---

**operations** array of objects<br/>
A list of operations to be executed across chains. Each operation contains a batch of transaction to be executed atomically. Each transaction in the batch has the same fields as a regular transaction (e.g., `to`, `data`, `value`).

For more details about the operations, see [Operations](./operations.md).
<!-- panels:end -->
