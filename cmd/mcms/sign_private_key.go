package mcms

import (
	"fmt"
	"os"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
)

func newSignPrivateKeyCmd(proposalPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-raw-private-key",
		Short: "Sign a proposal with a raw private key",
		Long:  `Configure a private key in a .env file (using the PRIVATE_KEY var) and sign a proposal with it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load Proposal
			proposal, err := mcms.LoadProposal(proposalPath)
			if err != nil {
				fmt.Printf("Error loading proposal: %s\n", err)
				return err
			}

			// Load Private Key
			pk, err := loadPrivateKey()
			if err != nil {
				fmt.Printf("Error loading private key: %s\n", err)
				return err
			}

			// Get Encoders
			encoders, err := proposal.GetEncoders()
			if err != nil {
				fmt.Printf("Error getting encoders: %s\n", err)
				return err
			}

			// Load RPCs
			clients, err := loadRPCs(getMapKeys(encoders))
			if err != nil {
				return err
			}

			// Create Inspectors
			inspectors := make(map[types.ChainSelector]sdk.Inspector)
			for chain, client := range clients {
				chainInspector := evm.NewInspector(client)
				inspectors[chain] = chainInspector
			}

			// Convert proposal to signable
			s, err := proposal.Signable(inspectors)

			signer := mcms.NewPrivateKeySigner(pk)
			sig, err := s.Sign(signer)
			if err != nil {
				fmt.Printf("Error signing proposal: %s\n", err)
				return err
			}

			// Append signature
			proposal.AppendSignature(sig)

			// open the file
			file, err := os.Open(proposalPath)
			if err != nil {
				fmt.Println("Error opening file:", err)
				return err
			}

			// Write proposal to file
			return proposal.Write(file)
		},
	}

	return cmd
}
