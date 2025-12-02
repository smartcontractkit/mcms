//go:build e2e

package solanae2e

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testPDASeedSetRootSimulateTest = [32]byte{'t', 'e', 's', 't', '-', 's', 'e', 't', 'r', 'o', 'o', 't', '-', 's', 'i', 'm', 'u', 'l', 'a', 't', 'e'}

func (s *TestSuite) TestSimulator_SimulateSetRoot() {
	s.SetupMCM(testPDASeedSetRootSimulateTest)
	ctx := context.Background()

	mcmAddress := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedSetRootSimulateTest)

	signerEVMAccount := NewEVMTestAccount(s.T())
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}

	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	validUntil := time.Now().Add(10 * time.Hour).Unix()
	solanaTx, err := solanasdk.NewTransaction(
		auth.PublicKey().String(),
		[]byte("test data"),
		nil,
		[]*solana.AccountMeta{{PublicKey: auth.PublicKey(), IsSigner: true, IsWritable: false}},
		"unit-tests",
		[]string{"test"},
	)
	s.Require().NoError(err)
	metadata, err := solanasdk.NewChainMetadata(
		0,
		s.MCMProgramID,
		testPDASeedSetRootSimulateTest,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
		s.Roles[timelock.Bypasser_Role].AccessController.PublicKey())
	s.Require().NoError(err)
	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil)).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		AddChainMetadata(s.ChainSelector, metadata).
		AddOperation(types.Operation{
			ChainSelector: s.ChainSelector,
			Transaction:   solanaTx,
		}).
		Build()
	s.Require().NoError(err)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.ChainSelector].(*solanasdk.Encoder)
	executor := solanasdk.NewExecutor(encoder, s.SolanaClient, auth)
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

	// simulate set root
	metadata = proposal.ChainMetadata[s.ChainSelector]
	metadataHash, err := encoder.HashMetadata(metadata)
	s.Require().NoError(err)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	proof, err := tree.GetProof(metadataHash)
	s.Require().NoError(err)

	hash, err := proposal.SigningHash()
	s.Require().NoError(err)

	// Sort signatures by recovered address
	sortedSignatures := slices.Clone(proposal.Signatures) // Clone so we don't modify the original
	slices.SortFunc(sortedSignatures, func(a, b types.Signature) int {
		recoveredSignerA, _ := a.Recover(hash)
		recoveredSignerB, _ := b.Recover(hash)

		return recoveredSignerA.Cmp(recoveredSignerB)
	})

	simulator := solanasdk.NewSimulator(executor)

	err = simulator.SimulateSetRoot(ctx, "", metadata, proof, [32]byte(tree.Root.Bytes()), proposal.ValidUntil, sortedSignatures)
	var sErr solanasdk.SimulateError
	if errors.As(err, &sErr) {
		s.T().Logf("LOGS:")
		for _, log := range sErr.Logs() {
			s.T().Log("  " + log)
		}
	}
	s.Require().NoError(err)
}

func (s *TestSuite) TestSimulator_SimulateOperation() {
	ctx := context.Background()

	recipientAddress, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)

	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	encoder := solanasdk.NewEncoder(s.ChainSelector, 1, false)
	executor := solanasdk.NewExecutor(encoder, s.SolanaClient, auth)
	simulator := solanasdk.NewSimulator(executor)

	ix, err := system.NewTransferInstruction(
		1*solana.LAMPORTS_PER_SOL,
		auth.PublicKey(),
		recipientAddress.PublicKey()).ValidateAndBuild()
	s.Require().NoError(err)

	ixData, err := ix.Data()
	s.Require().NoError(err)

	tx, err := solanasdk.NewTransaction(solana.SystemProgramID.String(), ixData, nil, ix.Accounts(), "System", []string{})
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
