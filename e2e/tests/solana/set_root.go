//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testPDASeedSetRootTest = [32]byte{'t', 'e', 's', 't', '-', 's', 'e', 't', 'r', 'o', 'o', 't'}

// Test_Solana_SetRoot tests the SetRoot functionality by setting a root on the MCM program
// and doing the preload signers setup.
func (s *SolanaTestSuite) Test_Solana_SetRoot() {
	// --- arrange ---
	ctx := context.Background()
	s.SetupMCM(testPDASeedSetRootTest)

	mcmAddress := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedSetRootTest)

	recipientAddress := s.SolanaChain.PublicKey
	signerEVMAccount := NewEVMTestAccount(s.T())
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}

	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	metadata, err := solanasdk.NewSolanaChainMetadata(
		0,
		s.MCMProgramID,
		testPDASeedTimelockConverter,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
		s.Roles[timelock.Bypasser_Role].AccessController.PublicKey())
	validUntil := time.Now().Add(10 * time.Hour).Unix()
	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil)).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		AddChainMetadata(s.ChainSelector, metadata).
		AddOperation(types.Operation{
			ChainSelector: s.ChainSelector,
			Transaction: types.Transaction{
				To:               recipientAddress,
				Data:             []byte("0x"),
				AdditionalFields: json.RawMessage(`{"value": 0}`), // FIXME: not used right now; check with jonghyeon.park
			},
		}).
		Build()
	s.Require().NoError(err)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.ChainSelector].(*solanasdk.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{s.ChainSelector: solanasdk.NewExecutor(encoder, s.SolanaClient, auth)}
	inspectors := map[types.ChainSelector]sdk.Inspector{s.ChainSelector: solanasdk.NewInspector(s.SolanaClient)}

	// sign proposal
	signable, err := mcms.NewSignable(proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerEVMAccount.PrivateKey))
	s.Require().NoError(err)

	// set config
	configurer := solanasdk.NewConfigurer(s.SolanaClient, auth, s.ChainSelector)
	_, err = configurer.SetConfig(ctx, mcmAddress, &mcmConfig, true)
	s.Require().NoError(err)

	// --- act: call SetRoot ---
	executable, err := mcms.NewExecutable(proposal, executors)
	s.Require().NoError(err)
	signature, err := executable.SetRoot(ctx, s.ChainSelector)
	s.Require().NoError(err)

	// --- assert ---
	_, err = solana.SignatureFromBase58(signature.Hash)
	s.Require().NoError(err)

	gotRoot, gotValidUntil, err := inspectors[s.ChainSelector].GetRoot(ctx, mcmAddress)
	s.Require().NoError(err)
	s.Require().Equal(common.HexToHash("0x11329486f2a7bb589320f2a8e9fad50fd5ed9ceeb3c1e2f71491d5ab848c7f60"), gotRoot)
	s.Require().Equal(uint32(validUntil), gotValidUntil)
}
