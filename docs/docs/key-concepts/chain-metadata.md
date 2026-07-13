# Chain Metadata

Chain Metadata is shared across all proposal types and contains the information that allow transactions to be hashed and executed for it's respective chain. It is a map of chain-specific configuration for each blockchain involved in the proposal. The key of the object is the chain selector ID, and the value is the metadata object. An entry is required for every chain referenced in the proposal's operations.

### Chain Metadata Structure

```json
{
  "16015286601757825753": {
    "startingOpCount": 1,
    "mcmAddress": "0x0"
  }
}
```

### Chain Selector ID

**Key** uint64<br/>
The chain selector ID matching the chain based on the [Chain Selectors](https://github.com/smartcontractkit/chain-selectors) library.

### Metadata Object

**startingOpCount** uint64<br/>
The starting operation count, typically used for parallel signing processes.

---

**mcmAddress** string<br/>
The MCM contract address that will process this proposal on the respective chain.

---

**additionalFields** object _optional_<br/>
Chain-family-specific fields encoded as JSON. Structure depends on the chain family (see below).

### Solana Additional Fields

Solana chain metadata uses `additionalFields` for the Timelock role access-controller accounts and, for bypass proposals, the execute fee payer.

| Field | Required | When used |
| --- | --- | --- |
| `proposerRoleAccessController` | yes | schedule conversion |
| `cancellerRoleAccessController` | yes | cancel conversion |
| `bypasserRoleAccessController` | yes | bypass conversion |
| `executePayer` | no | bypass only — account that pays (and therefore signs) the outer MCM execute transaction |

Example Solana `chainMetadata` entry:

```json
"5013781088424303360": {
  "startingOpCount": 0,
  "mcmAddress": "<programId>.<seed>",
  "additionalFields": {
    "proposerRoleAccessController": "...",
    "cancellerRoleAccessController": "...",
    "bypasserRoleAccessController": "...",
    "executePayer": "<base58 execute-payer pubkey>"
  }
}
```

#### `executePayer`

When the execute payer also appears in a bypass operation's `remaining_accounts` (for example as a BPF upgrade spill / close recipient), the Solana runtime always presents the fee payer as `IsSigner=true` at execution time. Off-chain conversion otherwise defaults remaining accounts to non-signer. Without recording `executePayer` in chain metadata, the Merkle leaf hashed off-chain does not match on-chain proof verification and execution fails with `ProofCannotBeVerified`.

**When to set it:** Solana **bypass** proposals where the fee-payer pubkey is listed as a writable remaining account. Omit for schedule/cancel; the converter ignores `executePayer` for non-bypass actions.

**Go helper:** `AdditionalFieldsMetadata.WithExecutePayer(pk)` in [`sdk/solana/chain_metadata.go`](https://github.com/smartcontractkit/mcms/blob/main/sdk/solana/chain_metadata.go).

**Reference scenario:** [`e2e/tests/solana/timelock_bypass_payer_collision.go`](https://github.com/smartcontractkit/mcms/blob/main/e2e/tests/solana/timelock_bypass_payer_collision.go).
