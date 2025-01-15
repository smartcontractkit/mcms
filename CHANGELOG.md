# @smartcontractkit/mcms

## 0.6.0

### Minor Changes

- [#231](https://github.com/smartcontractkit/mcms/pull/231) [`a8447e1`](https://github.com/smartcontractkit/mcms/commit/a8447e1727147fe21ff7f2b8186ceeebeef47a23) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(solana): timelock inspection - operation statuses check

- [#209](https://github.com/smartcontractkit/mcms/pull/209) [`a71dd79`](https://github.com/smartcontractkit/mcms/commit/a71dd79442551bdb757b94263d48b9ef7aa2b3b8) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - Add the `Configurer` component and `SetConfig` call to the Solana SDK.

- [#223](https://github.com/smartcontractkit/mcms/pull/223) [`4adb968`](https://github.com/smartcontractkit/mcms/commit/4adb96870c0a3daac98095656d0fea0753367b0d) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - Add a "context" parameter to all APIs that interact with a blockchain.

- [#211](https://github.com/smartcontractkit/mcms/pull/211) [`be76399`](https://github.com/smartcontractkit/mcms/commit/be76399a414053345b0b6e8e5b1eff951a3efd7e) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(solana): support get opdata, root and root metadata

### Patch Changes

- [#215](https://github.com/smartcontractkit/mcms/pull/215) [`9f39403`](https://github.com/smartcontractkit/mcms/commit/9f394035272baa4f2fcfb33d626486ba113841d7) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - Add the `Executor` component and `SetRoot` call to the Solana SDK.

- [#225](https://github.com/smartcontractkit/mcms/pull/225) [`7c9cd3d`](https://github.com/smartcontractkit/mcms/commit/7c9cd3d08c4a04cb1dd596b643976a9b96807149) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - Add PDA finders and ContractAddress parser to the Solana SDK

- [#228](https://github.com/smartcontractkit/mcms/pull/228) [`b953973`](https://github.com/smartcontractkit/mcms/commit/b953973f62b2c2876f55cb050541a3f990cc1ea7) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(solana): setProgramID on inspection methods

## 0.5.0

### Minor Changes

- [#214](https://github.com/smartcontractkit/mcms/pull/214) [`04cb547`](https://github.com/smartcontractkit/mcms/commit/04cb5474e8ce890566a4e48739a1d917f245c72f) Thanks [@akhilchainani](https://github.com/akhilchainani)! - Add helper function for timelock proposal execution

- [#210](https://github.com/smartcontractkit/mcms/pull/210) [`8431019`](https://github.com/smartcontractkit/mcms/commit/8431019b9a672edf0c257982d677dcf04897c770) Thanks [@akhilchainani](https://github.com/akhilchainani)! - Implement timelock executor interface + EVM implementation

## 0.4.0

### Minor Changes

- [#183](https://github.com/smartcontractkit/mcms/pull/183) [`6dd7030`](https://github.com/smartcontractkit/mcms/commit/6dd7030d76efaa44e75332421c44b1adc1f31728) Thanks [@akhilchainani](https://github.com/akhilchainani)! - Implement EVM Simulations

### Patch Changes

- [#188](https://github.com/smartcontractkit/mcms/pull/188) [`a8db3c4`](https://github.com/smartcontractkit/mcms/commit/a8db3c4ba39d8067bd96fda915544dd17808d599) Thanks [@ecPablo](https://github.com/ecPablo)! - Fix bug in nonce calculation when multiple proposals were executed

## 0.3.0

### Minor Changes

- [#165](https://github.com/smartcontractkit/mcms/pull/165) [`03682a2`](https://github.com/smartcontractkit/mcms/commit/03682a2772b4771f5af05d1ebd49b0a54e30beaf) Thanks [@ecPablo](https://github.com/ecPablo)! - Add SetConfig support for EVM MCMS contract.

## 0.2.0

### Major Changes

- Refactored public API and internal packages to support multiple chain families.
