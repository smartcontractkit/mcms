# Signing Proposals

For signing proposals, we use the methods that come with the `Proposal` type.

```golang
package examples

import (
	"fmt"
	"os"

	"github.com/smartcontractkit/mcms"
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
	proposal, err := mcms.NewProposal(file)
	if err != nil {
		fmt.Println("Error creating proposal:", err)
		return
	}

	// 2. Get proposal bytes for signature
	bytes, err := proposal.SigningMessage()

	// 3. Sign the actual bytes
	// This should be generated via ledger, using a private key KMS, etc.
	// For the sake of this example, we will generate a signature using a random private key
	// and then convert it to bytes
	signedBytes, err := types.NewSignatureFromBytes(bytes[:])
	if err != nil {
		fmt.Println("Error creating signature:", err)
		return
	}
	/// 4. Add the signature
	proposal.AppendSignature(signedBytes)
	fmt.Println("Successfully signed proposal:", proposal)
}

```