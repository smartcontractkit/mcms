# Configuration

The [`ManyChainMultiSigConfig`](https://github.com/smartcontractkit/mcms/blob/70ec727caf84a3fac4fa280ce5fbda3b07df7ee5/pkg/gethwrappers/ManyChainMultiSig.go#L32) data structure in the MCMS contract
is complex and difficult to define based on a desired group structure. To simplify usage, the library provides
a [`Config` wrapper](https://github.com/smartcontractkit/mcms/blob/main/types/config.go) offering a more intuitive way to define MCMS membership structures.

The `Config` is a nested tree structure where a group reaches `quorum` if the sum of `Signers` with
signatures and `GroupSigners` that individually meet their own `quorum` is greater than or equal to the top-level `quorum`.

### Example

Consider the following `Config`:

```
Config{
    Quorum:       3,
    Signers:      ["0x1", "0x2"],
    GroupSigners: [
        {
            Quorum: 1,
            Signers: ["0x3","0x4"],
            GroupSigners: []
        },
        {
            Quorum: 1,
            Signers: ["0x5","0x6"],
            GroupSigners: []
        }
    ],
}
```

> [!NOTE]
> Signers cannot be repeated in this configuration (i.e. they cannot belong to multiple groups)

This configuration represents a membership structure that requires 3 entities to approve, in which any of the following
combinations of signatures would satisfy the top-level quorum of `3`:

1. [`0x1`, `0x2`, `0x3`]
2. [`0x1`, `0x2`, `0x4`]
3. [`0x1`, `0x2`, `0x5`]
4. [`0x1`, `0x2`, `0x6`]
5. [`0x1`, `0x3`, `0x5`]
6. [`0x1`, `0x3`, `0x6`]
7. [`0x1`, `0x4`, `0x5`]
8. [`0x1`, `0x4`, `0x6`]
9. [`0x2`, `0x3`, `0x5`]
10. [`0x2`, `0x3`, `0x6`]
11. [`0x2`, `0x4`, `0x5`]
12. [`0x2`, `0x4`, `0x6`]

Once a satisfactory MCMS Membership configuration is constructed, users can use
the [`ExtractSetConfigInputs`](./config/config.go#L153) function to generate inputs and
call [`SetConfig`](./gethwrappers/ManyChainMultiSig.go#L428)
