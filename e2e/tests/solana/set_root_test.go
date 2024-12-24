//go:build e2e
// +build e2e

package e2e_solana

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func (s *SolanaTestSuite) Test_SetRoot() {
	// --- arrange ---
	mcmAddress := s.SolanaChain.SolanaPrograms["mcm"] // FIXME: replace with runtime deployment?
	recipientAddress := s.SolanaChain.PublicKey
	signerEVMAccount := NewEVMTestAccount(s.T())
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}

	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(time.Now().Add(10*time.Second).Unix())).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		AddChainMetadata(s.chainSelector, types.ChainMetadata{MCMAddress: mcmAddress}).
		AddOperation(types.Operation{
			ChainSelector: s.chainSelector,
			Transaction: types.Transaction{
				To:               recipientAddress,
				Data:             []byte("0x"),
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			},
		}).
		Build()
	s.Require().NoError(err)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*mcmsSolana.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{s.chainSelector: mcmsSolana.NewExecutor(s.SolanaClient, auth, encoder)}
	inspectors := map[types.ChainSelector]sdk.Inspector{s.chainSelector: mcmsSolana.NewInspector(s.SolanaClient)}

	// sign proposal
	signable, err := mcms.NewSignable(proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerEVMAccount.PrivateKey))
	s.Require().NoError(err)

	// set config
	configurer := mcmsSolana.NewConfigurer(s.SolanaClient, auth, s.chainSelector)
	_, err = configurer.SetConfig(mcmAddress, &mcmConfig, true)
	s.Require().NoError(err)

	// --- act: call SetRoot ---
	executable, err := mcms.NewExecutable(proposal, executors)
	s.Require().NoError(err)
	signature, err := executable.SetRoot(s.chainSelector)
	s.Require().NoError(err)

	// --- assert ---
	_, err = solana.SignatureFromBase58(signature)
	s.Require().NoError(err)
	// TODO: read data from PDAs and assert root, proofs, signatures, etc
}
