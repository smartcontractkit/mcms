# Create, Sign and Execute a Timelock Proposal

The general flow for timelock proposal is similar to the usual [Building Proposals](./building-proposals.md)
usage. The difference is that we have an intermediate proposal conversion step.

The reason we need a conversion step is that the timelock proposal allows users to
batch transactions, meaning that if one tx reverts we want to enforce all of the txs
in the batch to revert too.

To achieve this we need to convert all the txs of a given batch of the timelock
proposal to a single "non-timelock" operation. This operations is a call to the
timelock contract with all the txs as input of the given batch. So then the
mcms contract just does 1 single call to the timelock contract with all the txs
on the batch, and the timelock will be responsible of ensuring the atomicity of the batch.

See the example below:

```golang
package examples

import (
	"crypto/ecdsa"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func main() {
	file, err := os.Open("proposal.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 1. Create the proposal from the JSON data
	proposal, err := mcms.NewTimelockProposal(file)
	if err != nil {
		fmt.Println("Error creating proposal:", err)
		return
	}

	// 1.1 Convert to MCMS proposal
	mcmsProposal, err := proposal.Convert()
	if err != nil {
		return
	}

	// 2. Create the signable type from the proposal
	selector := chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector
	backend := backends.SimulatedBackend{}
	inspectorsMap := make(map[types.ChainSelector]sdk.Inspector)
	inspectorsMap[types.ChainSelector(selector)] = evm.NewEVMInspector(backend)
	signable, err := mcms.NewSignable(&mcmsProposal, inspectorsMap)

	// 3. Sign the proposal bytes
	// This will be generated via ledger, using a private key KMS, etc.
	// For the sake of this example, we will generate a signature using an empty private key
	signer := mcms.NewPrivateKeySigner(&ecdsa.PrivateKey{})
	// Or using ledger, you can call NewLedgerSigner and provide the derivation path as a parameter
	// signerLedger := mcms.NewLedgerSigner([]uint32{44, 60, 0, 0, 0})
	signature, err := signable.Sign(signer)
	if err != nil {
		fmt.Println("Error signing proposal:", err)
		return
	}

	/// 4. Add the signature
	proposal.AppendSignature(signature)
	fmt.Println("Successfully created proposal:", proposal)
}
```