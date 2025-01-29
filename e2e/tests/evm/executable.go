//go:build e2e
// +build e2e

package evme2e

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// ExecutionTestSuite defines the test suite
type ExecutionTestSuite struct {
	suite.Suite
	mcmsContract     *bindings.ManyChainMultiSig
	chainSelector    mcmtypes.ChainSelector
	deployerKey      common.Address
	signerAddresses  []common.Address
	auth             *bind.TransactOpts
	timelockContract *bindings.RBACTimelock
	e2e.TestSetup
}

// SetupSuite runs before the test suite
func (s *ExecutionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
	// Get deployer's private key
	privateKeyHex := s.Settings.PrivateKeys[0]
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err, "Invalid private key")

	// Define signer addresses
	s.signerAddresses = []common.Address{
		common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"),
		common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	}

	// Parse ChainID from string to int64
	chainID, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.auth = auth

	s.mcmsContract = s.deployMCMSContract()
	s.timelockContract = s.deployTimelockContract(s.mcmsContract.Address().Hex())
	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)

	chainDetails, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.BlockchainA.Out.Family)
	s.Require().NoError(err)
	s.chainSelector = mcmtypes.ChainSelector(chainDetails.ChainSelector)
}

// deployContract is a helper to deploy the contract
func (s *ExecutionTestSuite) deployMCMSContract() *bindings.ManyChainMultiSig {
	_, tx, instance, err := bindings.DeployManyChainMultiSig(s.auth, s.Client)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.Client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = instance.SetConfig(s.auth, s.signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	receipt, err = bind.WaitMined(context.Background(), s.Client, tx)
	s.Require().NoError(err, "Failed to mine configuration transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// deployContract is a helper to deploy the contract
func (s *ExecutionTestSuite) deployTimelockContract(mcmsAddress string) *bindings.RBACTimelock {
	_, tx, instance, err := bindings.DeployRBACTimelock(
		s.auth,
		s.Client,
		big.NewInt(0),
		common.HexToAddress(mcmsAddress),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.Client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// TestExecuteProposal executes a proposal after setting the root
func (s *ExecutionTestSuite) TestExecuteProposal() {
	ctx := context.Background()
	opts := &bind.CallOpts{
		Context:     ctx,
		From:        s.auth.From, // Set the "from" address (optional)
		BlockNumber: nil,         // Use the latest block (nil by default)
	}
	// Construct example transaction
	role, err := s.timelockContract.PROPOSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	s.Require().NoError(err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, s.mcmsContract.Address())
	s.Require().NoError(err)

	// Construct a proposal
	proposal := mcms.Proposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			Kind:                 mcmtypes.KindProposal,
			ValidUntil:           2004259681,
			Signatures:           []mcmtypes.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
				s.chainSelector: {
					StartingOpCount: 0,
					MCMAddress:      s.mcmsContract.Address().Hex(),
				},
			},
		},
		Operations: []mcmtypes.Operation{
			{
				ChainSelector: s.chainSelector,
				Transaction: evm.NewTransaction(
					common.HexToAddress(s.timelockContract.Address().Hex()),
					grantRoleData,
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	// Gen caller map for easy access (we can use geth chainselector for anvil)
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.chainSelector: evm.NewInspector(s.Client),
	}

	// Construct executor
	signable, err := mcms.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	err = signable.ValidateConfigs(ctx)
	s.Require().NoError(err)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	// Construct executors
	executors := map[mcmtypes.ChainSelector]sdk.Executor{
		s.chainSelector: evm.NewExecutor(
			encoders[s.chainSelector].(*evm.Encoder),
			s.Client,
			s.auth,
		),
	}

	// Construct executable
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	root, err := s.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(root.Root, [32]byte(tree.Root))
	s.Require().Equal(root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Check the state of the MCMS contract
	newOpCount, err := s.mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)
	s.Require().NotNil(newOpCount)
	s.Require().Equal(uint64(1), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := s.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(1), proposerCount)
	proposer, err := s.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	s.Require().NoError(err)
	s.Require().Equal(s.mcmsContract.Address().Hex(), proposer.Hex())
}

// TestExecuteProposalMultiple executes 2 proposals to check nonce calculation mechanisms are working
func (s *ExecutionTestSuite) TestExecuteProposalMultiple() {
	ctx := context.Background()
	opts := &bind.CallOpts{
		Context:     ctx,
		From:        s.auth.From, // Set the "from" address (optional)
		BlockNumber: nil,         // Use the latest block (nil by default)
	}
	// Construct example transaction
	role, err := s.timelockContract.PROPOSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	s.Require().NoError(err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, s.auth.From)
	s.Require().NoError(err)

	// Construct a proposal
	proposal := mcms.Proposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			Kind:                 mcmtypes.KindProposal,
			ValidUntil:           2004259681,
			Signatures:           []mcmtypes.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
				s.chainSelector: {
					StartingOpCount: 1,
					MCMAddress:      s.mcmsContract.Address().Hex(),
				},
			},
		},
		Operations: []mcmtypes.Operation{
			{
				ChainSelector: s.chainSelector,
				Transaction: evm.NewTransaction(
					common.HexToAddress(s.timelockContract.Address().Hex()),
					grantRoleData,
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	// Gen caller map for easy access (we can use geth chainselector for anvil)
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.chainSelector: evm.NewInspector(s.Client),
	}

	// Construct executor
	signable, err := mcms.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	// Construct executors
	executors := map[mcmtypes.ChainSelector]sdk.Executor{
		s.chainSelector: evm.NewExecutor(
			encoders[s.chainSelector].(*evm.Encoder),
			s.Client,
			s.auth,
		),
	}

	// Construct executable
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	root, err := s.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(root.Root, [32]byte(tree.Root))
	s.Require().Equal(root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Check the state of the MCMS contract
	newOpCount, err := s.mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)
	s.Require().NotNil(newOpCount)
	s.Require().Equal(uint64(2), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := s.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(2), proposerCount)
	proposer, err := s.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	s.Require().NoError(err)
	s.Require().Equal(s.mcmsContract.Address().Hex(), proposer.Hex())

	role2, err := s.timelockContract.EXECUTORROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	grantRoleData2, err := timelockAbi.Pack("grantRole", role2, s.mcmsContract.Address())
	s.Require().NoError(err)
	// Construct 2nd proposal

	proposal2 := mcms.Proposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			Kind:                 mcmtypes.KindProposal,
			ValidUntil:           2004259681,
			Signatures:           []mcmtypes.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
				s.chainSelector: {
					StartingOpCount: 2,
					MCMAddress:      s.mcmsContract.Address().Hex(),
				},
			},
		},
		Operations: []mcmtypes.Operation{
			{
				ChainSelector: s.chainSelector,
				Transaction: evm.NewTransaction(
					common.HexToAddress(s.timelockContract.Address().Hex()),
					grantRoleData2,
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	// Construct executor
	signable2, err := mcms.NewSignable(&proposal2, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	_, err = signable2.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err = signable2.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders
	encoders2, err := proposal2.GetEncoders()
	s.Require().NoError(err)

	// Construct executors
	executors2 := map[mcmtypes.ChainSelector]sdk.Executor{
		s.chainSelector: evm.NewExecutor(
			encoders2[s.chainSelector].(*evm.Encoder),
			s.Client,
			s.auth,
		),
	}

	// Construct executable
	executable2, err := mcms.NewExecutable(&proposal2, executors2)
	s.Require().NoError(err)

	// SetRoot on the contract
	tx, err = executable2.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	tree2, err := proposal2.MerkleTree()
	s.Require().NoError(err)

	root, err = s.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(root.Root, [32]byte(tree2.Root))
	s.Require().Equal(root.ValidUntil, proposal2.ValidUntil)

	// Execute the proposal
	tx, err = executable2.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Check the state of the MCMS contract
	newOpCount, err = s.mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)
	s.Require().NotNil(newOpCount)
	s.Require().Equal(uint64(3), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err = s.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role2)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(1), proposerCount)
	proposer, err = s.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	s.Require().NoError(err)
	s.Require().Equal(s.mcmsContract.Address().Hex(), proposer.Hex())
}
