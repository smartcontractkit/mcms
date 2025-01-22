//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"

	solana2 "github.com/smartcontractkit/mcms/e2e/utils/solana"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

// Test_Simulator_SimulateOperation tests the operation simulation functionality.
func (s *SolanaTestSuite) Test_Simulator_SimulateOperation() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	recipientAddress, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)

	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	solana2.FundAccounts(s.T(), ctx, []solana.PublicKey{auth.PublicKey()}, 20*solana.LAMPORTS_PER_SOL, s.SolanaClient)

	// Create a new simulator
	simulator, err := solanasdk.NewSimulator(
		s.SolanaClient,
		auth,
		solanasdk.NewEncoder(s.ChainSelector, 1, false),
	)
	s.Require().NoError(err)

	ix, err := system.NewTransferInstruction(
		1*solana.LAMPORTS_PER_SOL,
		auth.PublicKey(),
		recipientAddress.PublicKey()).ValidateAndBuild()
	s.Require().NoError(err)
	ixData, err := ix.Data()
	s.Require().NoError(err)

	tx, err := solanasdk.NewTransaction(solana.SystemProgramID.String(), ixData, ix.Accounts(), "System", []string{})
	s.Require().NoError(err)
	op := types.Operation{
		Transaction:   tx,
		ChainSelector: s.ChainSelector,
	}
	metadata := types.ChainMetadata{
		MCMAddress: s.MCMProgramID.String(),
	}
	err = simulator.SimulateOperation(ctx, metadata, op)
	s.Require().NoError(err)
}
