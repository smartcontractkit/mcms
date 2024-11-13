# Operations

Operations contain the information required to execute transactions. Metadata fields (`Contract Types` and `Tags`) are optional and can be used to add helpful references to contracts' ABI, simplify operations like decoding payloads, or to categorize operations.

There are two types of operations which are used in proposals depending on the proposal type:

### MCM Proposal Operations

Operations in an MCM proposal are used to execute a single transaction per operation. The transaction data is encoded in the `transaction` field.

<!-- panels:start -->

<!-- div:left-panel -->

```json
{
    "chainSelector": "16015286601757825753",
    "transaction": {
        "to": "0xb",
        "data": "ZGF0YQ==",
        "additionalFields": {
            "value": 0
        },
        "contractType": "<CONTRACT_TYPE>",
        "tags": [
            "tag1"
        ]
    }
}
```

<!-- div:right-panel -->

**chainSelector** uint64<br/>
The chain selector id for the chain that the operation will be executed on.

---

**transaction** object<br/>
The [transaction](#transactions) to be executed.

<!-- panels:end -->

### Timelock Proposal Operations

Operations in an Timelock Proposal can be used to a batch of transaction per operation. The transaction data to be encoded is in the `transactions` field.

<!-- panels:start -->

<!-- div:left-panel -->

```json
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
        }
    ]
}
```

<!-- div:right-panel -->

**chainSelector** uint64<br/>
The chain selector id for the chain that the operation will be executed on.

---

**transactions** object<br/>
The [transactions](#transactions) to be executed.



<!-- panels:end -->

### Transactions

Transactions represent the atomic operations that can be executed on a chain. They are composed of the following fields:

<!-- panels:start -->

<!-- div:left-panel -->

```json
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
}
```

<!-- div:right-panel -->

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