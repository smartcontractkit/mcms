# Many Chain Multisig System

## Motivation

The **Many Chain Multisig System (MCMS)** is a contract-agnostic, cross-chain multisig platform designed to provide standardized, secure, and scalable multisig contract operations. The need for this platform arises from the challenges teams face in deploying and securing their contracts using multisig across various blockchains. Existing solutions, such as Gnosis Safe, are not designed for multi-chain environments, making it difficult to manage multisig operations across different products and blockchains.

MCMS addresses this challenge by offering a standardized interface that facilitates scalable multisig operations across multiple chains. The library provides tools to deploy and manage multisig setups while ensuring interoperability, scalability, and security. By leveraging MCMS, teams can focus on their core protocols and products without needing to reinvent their multisig management processes.

## Key Features

- **Contract Management Across Chains:** Simplifies contract management by allowing users to send a set of transactions to multiple chains using a single set of signatures. This cross-chain capability streamlines operations, reducing complexity and overhead in managing contract security across multiple blockchains.
- **Security:** Ensures the security of products and users by defending against various attack vectors.
- **Product Agnosticism:** Integrates seamlessly with various products and protocols without requiring product-specific modifications.
- **Cross-Chain Interoperability:** Standardizes multisig operations across multiple chains, enabling product teams to expand to new chains easily. MCMS allows the execution of the same set of operations on different chains with minimal additional configuration.
- **Proposal Generation:** Allows users to generate valid MCMS proposals with transactions to manage and configure their product-specific smart contracts.
- **Proposal Simulation:** Enables local simulation of proposals before sending them on-chain, allowing issues with transactions to be caught early.
- **Proposal Execution:** Sends proposals on-chain for execution across multiple chains.
- **Proposal Inspection:** Presents proposals in human-friendly formats to facilitate reviews and collaboration.

## Getting Started

To get started with MCMS, explore the documentation sections:

- **[Key Concepts](./key-concepts/configuration.md)**: Learn about the core concepts and architecture of MCMS
- **[Usage](./usage/configuration.md)**: Step-by-step guides for using MCMS in your projects
- **[Contributing](./contributing/integrating-new-chain-guide.md)**: Information for developers who want to contribute to MCMS

## Quick Links

- [GitHub Repository](https://github.com/smartcontractkit/mcms)
- [Changelog](https://github.com/smartcontractkit/mcms/blob/main/CHANGELOG.md)