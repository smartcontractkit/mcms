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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/samber/lo"
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

type EVMChainMeta struct {
	auth             *bind.TransactOpts
	mcmsContract     *bindings.ManyChainMultiSig
	timelockContract *bindings.RBACTimelock
	chainSelector    mcmtypes.ChainSelector
}

type timelockRoleConfig struct {
	Proposers  []common.Address
	Executors  []common.Address
	Cancellers []common.Address
	Bypassers  []common.Address
}

type ExecutionTestSuite struct {
	suite.Suite
	ChainA          EVMChainMeta
	ChainB          EVMChainMeta
	signerAddresses []common.Address
	deployerKey     common.Address
	e2e.TestSetup
}

func uniqAddresses(addrs []common.Address) []common.Address {
	return lo.Filter(lo.Uniq(addrs), func(addr common.Address, _ int) bool {
		return addr != (common.Address{})
	})
}

func ensureRoleSet(input []common.Address, fallback []common.Address) []common.Address {
	if len(input) == 0 {
		return fallback
	}

	return uniqAddresses(input)
}

// SetupSuite runs before the test suite
func (s *ExecutionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
	// Get deployer's private key
	privateKeyHex := s.Settings.PrivateKeys[0]
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err, "Invalid private key")
	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)

	// Define signer addresses
	s.signerAddresses = []common.Address{
		common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"),
		common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	}

	// Initialize Chain A
	chainIDA, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse Chain A ID")

	s.ChainA.auth, err = bind.NewKeyedTransactorWithChainID(privateKey, chainIDA)
	s.Require().NoError(err, "Failed to create transactor for Chain A")

	chainDetailsA, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.BlockchainA.Out.Family)
	s.Require().NoError(err)
	s.ChainA.chainSelector = mcmtypes.ChainSelector(chainDetailsA.ChainSelector)

	// Initialize Chain B
	chainIDB, ok := new(big.Int).SetString(s.BlockchainB.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse Chain B ID")

	s.ChainB.auth, err = bind.NewKeyedTransactorWithChainID(privateKey, chainIDB)
	s.Require().NoError(err, "Failed to create transactor for Chain B")

	chainDetailsB, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainB.Out.ChainID, s.BlockchainB.Out.Family)
	s.Require().NoError(err)
	s.ChainB.chainSelector = mcmtypes.ChainSelector(chainDetailsB.ChainSelector)

	// Deploy contracts on both chains
	s.ChainA.mcmsContract = s.deployMCMSContract(s.ChainA.auth, s.ClientA)
	roleCfgA := s.defaultTimelockRoleConfig(s.ChainA.mcmsContract.Address(), s.ChainA.auth.From)
	s.ChainA.timelockContract = s.deployTimelockContract(s.ChainA.auth, s.ClientA, s.ChainA.mcmsContract.Address().Hex(), roleCfgA)

	s.ChainB.mcmsContract = s.deployMCMSContract(s.ChainB.auth, s.ClientB)
	roleCfgB := s.defaultTimelockRoleConfig(s.ChainB.mcmsContract.Address(), s.ChainB.auth.From)
	s.ChainB.timelockContract = s.deployTimelockContract(s.ChainB.auth, s.ClientB, s.ChainB.mcmsContract.Address().Hex(), roleCfgB)
}

// TestExecuteProposal executes a proposal after setting the root
func (s *ExecutionTestSuite) TestExecuteProposal() {
	ctx := context.Background()
	opts := &bind.CallOpts{
		Context:     ctx,
		From:        s.ChainA.auth.From, // Set the "from" address (optional)
		BlockNumber: nil,                // Use the latest block (nil by default)
	}
	// Construct example transaction
	role, err := s.ChainA.timelockContract.PROPOSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	s.Require().NoError(err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, s.ChainA.mcmsContract.Address())
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
				s.ChainA.chainSelector: {
					StartingOpCount: 0,
					MCMAddress:      s.ChainA.mcmsContract.Address().Hex(),
				},
			},
		},
		Operations: []mcmtypes.Operation{
			{
				ChainSelector: s.ChainA.chainSelector,
				Transaction: evm.NewTransaction(
					common.HexToAddress(s.ChainA.timelockContract.Address().Hex()),
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
		s.ChainA.chainSelector: evm.NewInspector(s.ClientA),
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
		s.ChainA.chainSelector: evm.NewExecutor(
			encoders[s.ChainA.chainSelector].(*evm.Encoder),
			s.ClientA,
			s.ChainA.auth,
		),
	}

	// Construct executable
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, s.ChainA.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.ClientA, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	root, err := s.ChainA.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(root.Root, [32]byte(tree.Root))
	s.Require().Equal(root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.ClientA, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Check the state of the MCMS contract
	newOpCount, err := s.ChainA.mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)
	s.Require().NotNil(newOpCount)
	s.Require().Equal(uint64(1), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := s.ChainA.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role)
	s.Require().NoError(err)
	// One is added by default
	s.Require().Equal(big.NewInt(2), proposerCount)
	proposer, err := s.ChainA.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	s.Require().NoError(err)
	s.Require().Equal(s.ChainA.mcmsContract.Address().Hex(), proposer.Hex())
}

// TestExecuteProposalMultiple executes 2 proposals to check nonce calculation mechanisms are working
func (s *ExecutionTestSuite) TestExecuteProposalMultiple() {
	ctx := context.Background()
	opts := &bind.CallOpts{
		Context:     ctx,
		From:        s.ChainA.auth.From, // Set the "from" address (optional)
		BlockNumber: nil,                // Use the latest block (nil by default)
	}
	// Construct example transaction
	role, err := s.ChainA.timelockContract.PROPOSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	s.Require().NoError(err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, s.ChainA.auth.From)
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
				s.ChainA.chainSelector: {
					StartingOpCount: 1,
					MCMAddress:      s.ChainA.mcmsContract.Address().Hex(),
				},
			},
		},
		Operations: []mcmtypes.Operation{
			{
				ChainSelector: s.ChainA.chainSelector,
				Transaction: evm.NewTransaction(
					common.HexToAddress(s.ChainA.timelockContract.Address().Hex()),
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
		s.ChainA.chainSelector: evm.NewInspector(s.ClientA),
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
		s.ChainA.chainSelector: evm.NewExecutor(
			encoders[s.ChainA.chainSelector].(*evm.Encoder),
			s.ClientA,
			s.ChainA.auth,
		),
	}

	// Construct executable
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, s.ChainA.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.ClientA, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	root, err := s.ChainA.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(root.Root, [32]byte(tree.Root))
	s.Require().Equal(root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.ClientA, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Check the state of the MCMS contract
	newOpCount, err := s.ChainA.mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)
	s.Require().NotNil(newOpCount)
	s.Require().Equal(uint64(2), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := s.ChainA.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(2), proposerCount)
	proposer, err := s.ChainA.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	s.Require().NoError(err)
	s.Require().Equal(s.ChainA.mcmsContract.Address().Hex(), proposer.Hex())

	role2, err := s.ChainA.timelockContract.BYPASSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	grantRoleData2, err := timelockAbi.Pack("grantRole", role2, s.ChainA.mcmsContract.Address())
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
				s.ChainA.chainSelector: {
					StartingOpCount: 2,
					MCMAddress:      s.ChainA.mcmsContract.Address().Hex(),
				},
			},
		},
		Operations: []mcmtypes.Operation{
			{
				ChainSelector: s.ChainA.chainSelector,
				Transaction: evm.NewTransaction(
					common.HexToAddress(s.ChainA.timelockContract.Address().Hex()),
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
		s.ChainA.chainSelector: evm.NewExecutor(
			encoders2[s.ChainA.chainSelector].(*evm.Encoder),
			s.ClientA,
			s.ChainA.auth,
		),
	}

	// Construct executable
	executable2, err := mcms.NewExecutable(&proposal2, executors2)
	s.Require().NoError(err)

	// SetRoot on the contract
	tx, err = executable2.SetRoot(ctx, s.ChainA.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.ClientA, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	tree2, err := proposal2.MerkleTree()
	s.Require().NoError(err)

	root, err = s.ChainA.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(root.Root, [32]byte(tree2.Root))
	s.Require().Equal(root.ValidUntil, proposal2.ValidUntil)

	// Execute the proposal
	tx, err = executable2.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.ClientA, common.HexToHash(tx.Hash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Check the state of the MCMS contract
	newOpCount, err = s.ChainA.mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)
	s.Require().NotNil(newOpCount)
	s.Require().Equal(uint64(3), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err = s.ChainA.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role2)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(2), proposerCount)
	proposer, err = s.ChainA.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	s.Require().NoError(err)
	s.Require().Equal(s.ChainA.mcmsContract.Address().Hex(), proposer.Hex())
}

// TestExecuteProposalMultipleChains executes a proposal with operations on two different chains
func (s *ExecutionTestSuite) TestExecuteProposalMultipleChains() {
	ctx := context.Background()

	optsA := &bind.CallOpts{
		Context:     ctx,
		From:        s.ChainA.auth.From,
		BlockNumber: nil,
	}
	optsB := &bind.CallOpts{
		Context:     ctx,
		From:        s.ChainB.auth.From,
		BlockNumber: nil,
	}

	// Check the state of the MCMS contract
	opCountA, err := s.ChainA.mcmsContract.GetOpCount(optsA)
	s.Require().NoError(err)
	// Check the state of the MCMS contract
	opCountB, err := s.ChainB.mcmsContract.GetOpCount(optsB)
	s.Require().NoError(err)

	proposalTimelock := mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Kind:                 mcmtypes.KindTimelockProposal,
			Description:          "description",
			ValidUntil:           2004259681,
			OverridePreviousRoot: true,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata: map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
				s.ChainA.chainSelector: {
					StartingOpCount: opCountA.Uint64(),
					MCMAddress:      s.ChainA.mcmsContract.Address().Hex(),
				},
				s.ChainB.chainSelector: {
					StartingOpCount: opCountB.Uint64(),
					MCMAddress:      s.ChainB.mcmsContract.Address().Hex(),
				},
			},
		},
		Action: mcmtypes.TimelockActionSchedule,
		Delay:  mcmtypes.MustParseDuration("0s"),
		TimelockAddresses: map[mcmtypes.ChainSelector]string{
			s.ChainA.chainSelector: s.ChainA.timelockContract.Address().Hex(), s.ChainB.chainSelector: s.ChainB.timelockContract.Address().Hex(),
		},

		Operations: []mcmtypes.BatchOperation{{
			ChainSelector: s.ChainA.chainSelector,
			Transactions: []mcmtypes.Transaction{evm.NewTransaction(
				s.signerAddresses[0],
				[]byte("0x13424"),
				big.NewInt(0),
				"",
				[]string{""},
			)},
		}, {
			ChainSelector: s.ChainB.chainSelector,
			Transactions: []mcmtypes.Transaction{evm.NewTransaction(
				s.signerAddresses[0],
				[]byte("0x13424"),
				big.NewInt(0),
				"",
				[]string{""},
			)},
		},
		},
	}

	proposal, _, err := proposalTimelock.Convert(ctx, map[mcmtypes.ChainSelector]sdk.TimelockConverter{
		s.ChainA.chainSelector: &evm.TimelockConverter{},
		s.ChainB.chainSelector: &evm.TimelockConverter{},
	})
	s.Require().NoError(err)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	// Sign proposal
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.ChainA.chainSelector: evm.NewInspector(s.ClientA),
		s.ChainB.chainSelector: evm.NewInspector(s.ClientB),
	}

	// Construct signable object
	signable, err := mcms.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	err = signable.ValidateConfigs(ctx)
	s.Require().NoError(err)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])))
	s.Require().NoError(err)

	// Validate signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders for both chains
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoderA := encoders[s.ChainA.chainSelector].(*evm.Encoder)
	encoderB := encoders[s.ChainB.chainSelector].(*evm.Encoder)

	// Construct executors for both chains
	executors := map[mcmtypes.ChainSelector]sdk.Executor{
		s.ChainA.chainSelector: evm.NewExecutor(
			encoderA,
			s.ClientA,
			s.ChainA.auth,
		),
		s.ChainB.chainSelector: evm.NewExecutor(
			encoderB,
			s.ClientB,
			s.ChainB.auth,
		),
	}

	// Prepare and execute simulation A
	simulatorA, err := evm.NewSimulator(encoderA, s.ClientA)
	s.Require().NoError(err, "Failed to create simulator for Chain A")
	simulatorB, err := evm.NewSimulator(encoderB, s.ClientB)
	s.Require().NoError(err, "Failed to create simulator for Chain B")
	simulators := map[mcmtypes.ChainSelector]sdk.Simulator{
		s.ChainA.chainSelector: simulatorA,
		s.ChainB.chainSelector: simulatorB,
	}
	signable.SetSimulators(simulators)
	err = signable.Simulate(ctx)
	s.Require().NoError(err)

	// Construct executable object
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on MCMS Contract for both chains
	txA, err := executable.SetRoot(ctx, s.ChainA.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txA.Hash)

	txB, err := executable.SetRoot(ctx, s.ChainB.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txB.Hash)

	// Wait for transactions to be mined
	receiptA, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, common.HexToHash(txA.Hash))
	s.Require().NoError(err, "Failed to mine SetRoot transaction on Chain A")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptA.Status)

	receiptB, err := testutils.WaitMinedWithTxHash(ctx, s.ClientB, common.HexToHash(txB.Hash))
	s.Require().NoError(err, "Failed to mine SetRoot transaction on Chain B")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptB.Status)

	// Validate Contract State and verify root was set A
	rootA, err := s.ChainA.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(rootA.Root, [32]byte(tree.Root.Bytes()))
	s.Require().Equal(rootA.ValidUntil, proposal.ValidUntil)

	// Validate Contract State and verify root was set
	rootB, err := s.ChainA.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(rootB.Root, [32]byte(tree.Root.Bytes()))
	s.Require().Equal(rootB.ValidUntil, proposal.ValidUntil)

	txA, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txA.Hash)

	txB, err = executable.Execute(ctx, 1)
	s.Require().NoError(err)
	s.Require().NotEmpty(txB.Hash)

	// Wait for execution transactions to be mined
	receiptA, err = testutils.WaitMinedWithTxHash(ctx, s.ClientA, common.HexToHash(txA.Hash))
	s.Require().NoError(err, "Failed to mine execution transaction on Chain A")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptA.Status)

	receiptB, err = testutils.WaitMinedWithTxHash(ctx, s.ClientB, common.HexToHash(txB.Hash))
	s.Require().NoError(err, "Failed to mine execution transaction on Chain B")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptB.Status)

	// Verify the operation count is updated on both chains
	newOpCountA, err := s.ChainA.mcmsContract.GetOpCount(optsA)
	s.Require().NoError(err)
	s.Require().Equal(opCountA.Uint64()+1, newOpCountA.Uint64())

	newOpCountB, err := s.ChainB.mcmsContract.GetOpCount(optsB)
	s.Require().NoError(err)
	s.Require().Equal(opCountB.Uint64()+1, newOpCountB.Uint64())

	// Construct executors
	tExecutors := map[mcmtypes.ChainSelector]sdk.TimelockExecutor{
		s.ChainA.chainSelector: evm.NewTimelockExecutor(
			s.ClientA,
			s.ChainA.auth,
		),
		s.ChainB.chainSelector: evm.NewTimelockExecutor(
			s.ClientB,
			s.ChainB.auth,
		),
	}

	// Create new executable
	tExecutable, err := mcms.NewTimelockExecutable(ctx, &proposalTimelock, tExecutors)
	s.Require().NoError(err)

	err = tExecutable.IsReady(ctx)
	s.Require().NoError(err)

	// Execute operation 0
	txA, err = tExecutable.Execute(ctx, 0)
	s.Require().NoError(err)
	// Wait for execution transactions to be mined
	timelockReceiptA, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, common.HexToHash(txA.Hash))
	s.Require().NoError(err, "Failed to mine execution transaction on Chain A")
	s.Require().Equal(types.ReceiptStatusSuccessful, timelockReceiptA.Status)

	// Execute operation 1
	txB, err = tExecutable.Execute(ctx, 1)
	s.Require().NoError(err)
	// Wait for execution transactions to be mined
	timelockReceiptB, err := testutils.WaitMinedWithTxHash(ctx, s.ClientB, common.HexToHash(txB.Hash))
	s.Require().NoError(err, "Failed to mine execution transaction on Chain A")
	s.Require().Equal(types.ReceiptStatusSuccessful, timelockReceiptB.Status)
}

func (s *ExecutionTestSuite) defaultTimelockRoleConfig(mcmsAddr common.Address, operator common.Address) timelockRoleConfig {
	base := uniqAddresses([]common.Address{mcmsAddr, operator})
	return timelockRoleConfig{
		Proposers:  append([]common.Address{}, base...),
		Executors:  append([]common.Address{}, base...),
		Cancellers: append([]common.Address{}, base...),
		Bypassers:  uniqAddresses([]common.Address{operator, mcmsAddr}),
	}
}

// deployContract is a helper to deploy the contract
func (s *ExecutionTestSuite) deployTimelockContract(auth *bind.TransactOpts, client *ethclient.Client, mcmsAddress string, roles timelockRoleConfig) *bindings.RBACTimelock {
	mcmsAddr := common.HexToAddress(mcmsAddress)
	defaultRoles := s.defaultTimelockRoleConfig(mcmsAddr, auth.From)
	roleSet := timelockRoleConfig{
		Proposers:  ensureRoleSet(roles.Proposers, defaultRoles.Proposers),
		Executors:  ensureRoleSet(roles.Executors, defaultRoles.Executors),
		Cancellers: ensureRoleSet(roles.Cancellers, defaultRoles.Cancellers),
		Bypassers:  ensureRoleSet(roles.Bypassers, defaultRoles.Bypassers),
	}
	_, tx, instance, err := bindings.DeployRBACTimelock(
		auth,
		client,
		big.NewInt(0),
		mcmsAddr,
		roleSet.Proposers,
		roleSet.Executors,
		roleSet.Cancellers,
		roleSet.Bypassers,
	)
	s.Require().NoError(err, "Failed to deploy Timelock contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// deployMCMSContract deploys ManyChainMultiSig on the given chain
func (s *ExecutionTestSuite) deployMCMSContract(auth *bind.TransactOpts, client *ethclient.Client) *bindings.ManyChainMultiSig {
	_, tx, instance, err := bindings.DeployManyChainMultiSig(auth, client)
	s.Require().NoError(err, "Failed to deploy MCMS contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine MCMS deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = instance.SetConfig(auth, s.signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set MCMS contract configuration")
	receipt, err = bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine MCMS config transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}
