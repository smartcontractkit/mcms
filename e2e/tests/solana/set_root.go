//go:build e2e
// +build e2e

package solanae2e

import (
	"encoding/json"
	"fmt"
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
	ctx := s.contextWithLogger()

	s.SetupMCM(testPDASeedSetRootTest)
	s.SetupTimelock(testPDASeedSetRootTest, 1*time.Second)

	mcmAddress := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedSetRootTest)

	recipientAddress := s.SolanaChain.PublicKey
	signerEVMAccount := NewEVMTestAccount(s.T())
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}

	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	metadata, err := solanasdk.NewChainMetadata(
		0,
		s.MCMProgramID,
		testPDASeedSetRootTest,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
		s.Roles[timelock.Bypasser_Role].AccessController.PublicKey())
	s.Require().NoError(err)
	buildProposal := func() *mcms.Proposal {
		validUntil := time.Now().Add(10 * time.Hour)
		proposal, perr := mcms.NewProposalBuilder().
			SetVersion("v1").
			SetValidUntil(uint32(validUntil.Unix())).
			SetDescription(fmt.Sprintf("proposal to test SetRoot - %v", validUntil.UnixMilli())).
			SetOverridePreviousRoot(true).
			AddChainMetadata(s.ChainSelector, metadata).
			AddOperation(types.Operation{
				ChainSelector: s.ChainSelector,
				Transaction: types.Transaction{
					To:               recipientAddress,
					Data:             []byte("0x"),
					AdditionalFields: json.RawMessage(`{"value": 0, "accounts": []}`),
				},
			}).
			Build()
		s.Require().NoError(perr)

		return proposal
	}

	proposal := buildProposal()
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

	s.Run("single signer", func() {
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
		s.Require().Equal(common.HexToHash("0x2b970fa3b929cafc45e8740e5123ebf150c519813bcf4d9c7284518fd5720108"), gotRoot)
		s.Require().Equal(proposal.ValidUntil, gotValidUntil)
	})

	s.Run("multiple signers - exceeds default CU Limit", func() {
		s.T().Setenv("MCMS_SOLANA_MAX_RETRIES", "20")

		signers := []common.Address{}
		mcmsSigners := []*mcms.PrivateKeySigner{}
		for range 20 {
			signer := NewEVMTestAccount(s.T())
			signers = append(signers, signer.Address)
			mcmsSigners = append(mcmsSigners, mcms.NewPrivateKeySigner(signer.PrivateKey))
		}
		multiSignersMcmConfig := types.Config{Quorum: uint8(len(signers)), Signers: signers}

		proposal = buildProposal()

		signable, err := mcms.NewSignable(proposal, inspectors)
		s.Require().NoError(err)
		s.Require().NotNil(signable)
		for _, mcmsSigner := range mcmsSigners {
			_, err = signable.SignAndAppend(mcmsSigner)
		}
		s.Require().NoError(err)

		// set config
		configurer := solanasdk.NewConfigurer(s.SolanaClient, auth, s.ChainSelector)
		_, err = configurer.SetConfig(ctx, mcmAddress, &multiSignersMcmConfig, true)
		s.Require().NoError(err)

		executable, err := mcms.NewExecutable(proposal, executors)
		s.Require().NoError(err)

		// --- act: call SetRoot with "0" CU limit - will use Solana's default ---
		s.T().Setenv("MCMS_SOLANA_COMPUTE_UNIT_LIMIT", "0")
		_, err = executable.SetRoot(ctx, s.ChainSelector)
		// --- assert ---
		s.Require().ErrorContains(err, "Computational budget exceeded")

		// --- act: call SetRoot with no CU limit envvar; will use max limit: 1.4M ---
		s.T().Setenv("MCMS_SOLANA_COMPUTE_UNIT_LIMIT", "")
		signature, err := executable.SetRoot(ctx, s.ChainSelector)
		s.Require().NoError(err)
		// --- assert ---
		_, err = solana.SignatureFromBase58(signature.Hash)
		s.Require().NoError(err)
	})
}
