# Merkle Tree Signing

In the MCMS, Merkle tree signing is used to efficiently compose and verify multiple operations (Ops) in a proposal.
This approach leverages cryptographic proofs provided by a Merkle tree, ensuring that a set of transactions
can be signed collectively, maintaining both simplicity and security.

We have adopted the [OpenZeppelin Merkle tree](https://docs.openzeppelin.com/contracts-cairo/0.16.0/api/merkle-tree)
implementation for its robustness and widespread use in
smart contract systems. Here's an overview of how Merkle tree signing is applied in MCMS:

## Why OpenZeppelin's Merkle Tree?

We chose the OpenZeppelin Merkle tree on the Smart Contracts due to its tried-and-tested design and simplicity.
The decision to avoid more complex solutions like multiproofs or Merkle tries (which support key-value pairs)
aligns with the "Keep It Simple, Stupid" (KISS) principle. This approach avoids unnecessary complexity while still
providing the needed cryptographic security.

### Separate Merkle Tree Implementation for the Go Library

While the EVM smart contracts rely on Open Zeppelin's implementation, we have a separate Merkle
tree implementation in the OffChain Golang Library. You can check the
implementation [here](https://github.com/smartcontractkit/mcms/blob/main/pkg/merkle/merkle_tree.go)

#### Why 2 separate Merkle tree implementations?

We need an On-chain implementation (OpenZeppelin for Evm) in order to check if a given hash
belongs to the merkle root saved on the contract. However the on-chain implementation is not responsible for
the merkle tree construction itself. This root that gets saved on the contract (via the `setRoot()` function call) is
generated OffChain via the Go lib, which has the 2nd merkle tree implementation.

The Go-lib implementation can be reused for non-evm cases as well, however we'll most likely need
separate merkle tree implementations whenever any new non-evm chain needs to get supported.
 
## Key Decisions

* **No Multiproofs:** In the context of Merkle tree signing, multiproofs refer to a more advanced method that allows
  for verifying multiple leaves (operations) in the tree with a single proof. While this sounds efficient in certain
  scenarios, it adds complexity to the implementation and verification process. However, for simplicity and ease of
  verification, we stick to the single proof approach.

* **No Sparse Merkle Trees:** We do not use sparse Merkle trees, which store key-value pairs and can efficiently
  represent large, sparsely populated key spaces. While these trees could assign a unique key to each operation (Op),
  preventing duplicate or conflicting operations (e.g., with the same nonce and multisig contract), the practical
  benefit is minimal. Malicious signers can already take more harmful actions, such as approving conflicting
  transactions directly. Therefore, avoiding sparse Merkle trees simplifies the implementation without compromising
  security, keeping the design straightforward while maintaining strong cryptographic integrity.

* **No Unique Key for Ops:** By not using sparse Merkle trees
  (which support key-value pairs), two Ops with the same nonce and multisig
  contract could be included in the tree. However, this is a relatively minor
  concern because the potential risks are far outweighed by the simplicity and efficiency of the current approach.

## Merkle Tree Structure:

A Merkle tree is a binary tree where each leaf node contains a cryptographic hash of a data element
(in our case, a transaction). Each non-leaf node is a hash of its two child nodes. This structure allows for a
**Merkle root**, a single hash that represents the entire set of transactions. A Merkle proof is then used to verify the
inclusion of a particular transaction in the tree without needing to check the entire dataset.

## Merkle Tree Construction

Each Op (operation) is represented as a hash in the Merkle tree.
The Merkle tree is constructed by hashing pairs of operations until
only one root hash remains—the Merkle root.

### Merkle Proofs:

A Merkle proof consists of the hashes required to reconstruct the Merkle root from a specific transaction hash. This
allows anyone to verify that a particular operation is part of the batch.
In practice, once the Merkle root is signed by the signers, any operation within the tree can be verified by presenting
its Merkle proof, ensuring its inclusion in the signed batch of transactions.

### Simplified Verification:

Since only the root hash needs to be signed, signers don't need to sign each operation individually. This greatly
reduces the signing overhead in multi-transaction proposals.
Benefits of Merkle Tree Signing:

## Example:

Let's say we have a proposal with three operations:

```
Transfer 1 ETH to Address A
Transfer 2 ETH to Address B
Transfer 3 ETH to Address C
```

Each operation is hashed and included as a leaf in the Merkle tree:

```
    Root
   /    \
 Hash1  Hash2
 /  \   /  \
 Op1  Op2  Op3

```

The Merkle root is then signed by the multisig signers.
When it’s time to verify any individual operation (e.g., the transfer to Address B),
a **Merkle proof** can be used to verify that this operation is part of the
signed batch without needing to check all operations.
