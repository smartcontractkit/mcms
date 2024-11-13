# Chain Metadata

Chain Metadata is shared across all proposal types and contains the information that allow transactions to be hashed and executed for it's respective chain. It is a map of chain-specific configuration for each blockchain involved in the proposal. The key of the object is the chain selector ID, and the value is the metadata object. An entry is required for every chain referenced in the proposal's operations.

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
