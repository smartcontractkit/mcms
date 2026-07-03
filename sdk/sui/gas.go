package sui

// DefaultExecuteGasBudget is the gas budget for MCMS Sui executor PTBs, including
// timelock execute batches that upgrade large packages such as CCIP.
//
// chainlink-sui integration tests use 10 SUI for publishing and big MCMS proposals.
// The previous 500M mid-PTB budget was insufficient for full package upgrades.
const DefaultExecuteGasBudget uint64 = 10_000_000_000
