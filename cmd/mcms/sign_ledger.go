package mcms

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
)

func newSignLedgerCmd(proposalPath string) *cobra.Command {
	var derivationPath string

	cmd := &cobra.Command{
		Use:   "sign-ledger",
		Short: "Sign a proposal with a ledger",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load Proposal
			proposal, err := loadProposal(proposalPath)
			if err != nil {
				fmt.Printf("Error loading proposal: %s\n", err)
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

			// Parse the derivation path
			path, err := accounts.ParseDerivationPath(derivationPath)
			if err != nil {
				return fmt.Errorf("failed to parse derivation path: %w", err)
			}

			signer := mcms.NewLedgerSigner(path)
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

	cmd.Flags().StringVar(&derivationPath, "derivationPath", "m/44'/60'/0'/0/0", "The derivation path for the ledger")

	return cmd
}
