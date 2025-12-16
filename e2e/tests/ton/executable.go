//go:build e2e

package tone2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/stretchr/testify/suite"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	toncommon "github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/hash"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/ethereum/go-ethereum/common"

	mcmslib "github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/internal/testutils"

	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
)

type ExecutionTestSuite struct {
	suite.Suite

	signers []testutils.ECDSASigner

	// Sign proposals across multiple chains, execute and verify on Chain A
	ChainA types.ChainSelector
	ChainB types.ChainSelector
	ChainC types.ChainSelector

	// Chain A metadata
	mcmsAddr     string
	timelockAddr string

	wallet *wallet.Wallet

	e2e.TestSetup
}

// SetupSuite runs before the test suite
func (s *ExecutionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	// Init wallet
	var err error
	s.wallet, err = tvm.MyLocalTONWalletDefault(s.TonClient)
	s.Require().NoError(err)

	// Generate few test signers
	s.signers = testutils.MakeNewECDSASigners(2)

	// Initialize chains
	details, err := cselectors.GetChainDetailsByChainIDAndFamily(s.TonBlockchain.ChainID, s.TonBlockchain.Family)
	s.Require().NoError(err)
	s.ChainA = types.ChainSelector(details.ChainSelector)

	s.ChainB = types.ChainSelector(cselectors.GETH_TESTNET.Selector)
	s.ChainC = types.ChainSelector(cselectors.GETH_DEVNET_2.Selector)

	// Deploy contracts on chain A (the one we execute on)
	s.deployMCMSContract(hash.CRC32("test.executable.mcms"))
	s.deployTimelockContract(hash.CRC32("test.executable.timelock"))
}

// TestExecuteProposal executes a proposal after setting the root
func (s *ExecutionTestSuite) TestExecuteProposal() {
	ctx := context.Background()

	// Construct a proposal

	// Construct a TON transaction to grant a role

	// Grant role data
	grantRoleData, err := tlb.ToCell(rbac.GrantRole{
		QueryID: 0x1,
		Role:    tlbe.NewUint256(timelock.RoleProposer),
		Account: address.MustParseAddr(s.mcmsAddr),
	})
	s.Require().NoError(err)

	opTX, err := mcmston.NewTransaction(
		address.MustParseAddr(s.timelockAddr),
		grantRoleData.ToBuilder().ToSlice(),
		tlb.MustFromTON("0.1").Nano(),
		"RBACTimelock",
		[]string{"RBACTimelock", "GrantRole"},
	)
	s.Require().NoError(err)

	proposal := mcmslib.Proposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:              "v1",
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			Kind:                 types.KindProposal,
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				s.ChainA: {
					StartingOpCount:  0,
					MCMAddress:       s.mcmsAddr,
					AdditionalFields: testOpAdditionalFields,
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: s.ChainA,
				Transaction:   opTX,
			},
		},
	}

	tree, err := proposal.MerkleTree()
	s.Require().NotNil(tree)
	s.Require().NoError(err)

	// Gen caller map for easy access (we can use geth chainselector for anvil)
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.ChainA: mcmston.NewInspector(s.TonClient),
	}

	// Construct executor
	signable, err := mcmslib.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	err = signable.ValidateConfigs(ctx)
	s.Require().NoError(err)

	_, err = signable.SignAndAppend(mcmslib.NewPrivateKeySigner(s.signers[1].Key))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	// Construct executors
	encoder := encoders[s.ChainA].(*mcmston.Encoder)
	executor, err := mcmston.NewExecutor(encoder, s.TonClient, s.wallet, tlb.MustFromTON("0.1"))
	s.Require().NoError(err)
	executors := map[types.ChainSelector]sdk.Executor{
		s.ChainA: executor,
	}

	// Construct executable
	executable, err := mcmslib.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on the contract
	res, err := executable.SetRoot(ctx, s.ChainA)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Validate Contract State and verify root was set A
	rootARoot, rootAValidUntil, err := inspectors[s.ChainA].GetRoot(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(rootARoot, common.Hash([32]byte(tree.Root.Bytes())))
	s.Require().Equal(rootAValidUntil, proposal.ValidUntil)

	// Execute the proposal
	res, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok = res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Verify the operation count is updated on chain A
	newOpCountA, err := inspectors[s.ChainA].GetOpCount(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), newOpCountA)

	// // Check the state of the timelock contract
	// proposerCount, err := s.ChainA.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role)
	// s.Require().NoError(err)
	// // One is added by default
	// s.Require().Equal(big.NewInt(2), proposerCount)
	// proposer, err := s.ChainA.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	// s.Require().NoError(err)
	// s.Require().Equal(s.ChainA.mcmsContract.Address().Hex(), proposer.Hex())
}

// TestExecuteProposalMultiple executes 2 proposals to check nonce calculation mechanisms are working
func (s *ExecutionTestSuite) TestExecuteProposalMultiple() {
	ctx := context.Background()

	// Construct a TON transaction to grant a role

	// Grant role data
	grantRoleData, err := tlb.ToCell(rbac.GrantRole{
		QueryID: 0x1,
		Role:    tlbe.NewUint256(timelock.RoleProposer),
		Account: s.wallet.Address(),
	})
	s.Require().NoError(err)

	opTX, err := mcmston.NewTransaction(
		address.MustParseAddr(s.timelockAddr),
		grantRoleData.ToBuilder().ToSlice(),
		tlb.MustFromTON("0.1").Nano(),
		"RBACTimelock",
		[]string{"RBACTimelock", "GrantRole"},
	)
	s.Require().NoError(err)

	// Construct a proposal
	proposal := mcmslib.Proposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:              "v1",
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			Kind:                 types.KindProposal,
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				s.ChainA: {
					StartingOpCount:  1,
					MCMAddress:       s.mcmsAddr,
					AdditionalFields: testOpAdditionalFields,
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: s.ChainA,
				Transaction:   opTX,
			},
		},
	}

	tree, err := proposal.MerkleTree()
	s.Require().NotNil(tree)
	s.Require().NoError(err)

	// Gen caller map for easy access (we can use geth chainselector for anvil)
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.ChainA: mcmston.NewInspector(s.TonClient),
	}

	// Construct executor
	signable, err := mcmslib.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	_, err = signable.SignAndAppend(mcmslib.NewPrivateKeySigner(s.signers[1].Key))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	// Construct executors
	encoder := encoders[s.ChainA].(*mcmston.Encoder)
	executor, err := mcmston.NewExecutor(encoder, s.TonClient, s.wallet, tlb.MustFromTON("0.1"))
	s.Require().NoError(err)
	executors := map[types.ChainSelector]sdk.Executor{
		s.ChainA: executor,
	}

	// Construct executable
	executable, err := mcmslib.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on the contract
	res, err := executable.SetRoot(ctx, s.ChainA)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Validate Contract State and verify root was set A
	rootARoot, rootAValidUntil, err := inspectors[s.ChainA].GetRoot(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(rootARoot, common.Hash([32]byte(tree.Root.Bytes())))
	s.Require().Equal(rootAValidUntil, proposal.ValidUntil)

	// Execute the proposal
	res, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok = res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Verify the operation count is updated on chain A
	newOpCountA, err := inspectors[s.ChainA].GetOpCount(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(uint64(2), newOpCountA)

	// // Check the state of the timelock contract
	// proposerCount, err := s.ChainA.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role)
	// s.Require().NoError(err)
	// s.Require().Equal(big.NewInt(2), proposerCount)
	// proposer, err := s.ChainA.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	// s.Require().NoError(err)
	// s.Require().Equal(s.ChainA.mcmsContract.Address().Hex(), proposer.Hex())

	// Construct 2nd proposal

	// Construct a TON transaction to grant a role
	// Grant role data
	grantRoleData2, err := tlb.ToCell(rbac.GrantRole{
		QueryID: 0x1,
		Role:    tlbe.NewUint256(timelock.RoleBypasser),
		Account: address.MustParseAddr(s.mcmsAddr),
	})
	s.Require().NoError(err)

	opTX2, err := mcmston.NewTransaction(
		address.MustParseAddr(s.timelockAddr),
		grantRoleData2.ToBuilder().ToSlice(),
		tlb.MustFromTON("0.1").Nano(),
		"RBACTimelock",
		[]string{"RBACTimelock", "GrantRole"},
	)
	s.Require().NoError(err)

	proposal2 := mcmslib.Proposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:              "v1",
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			Kind:                 types.KindProposal,
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				s.ChainA: {
					StartingOpCount:  2,
					MCMAddress:       s.mcmsAddr,
					AdditionalFields: testOpAdditionalFields,
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: s.ChainA,
				Transaction:   opTX2,
			},
		},
	}

	// Construct executor
	signable2, err := mcmslib.NewSignable(&proposal2, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable2)

	_, err = signable2.SignAndAppend(mcmslib.NewPrivateKeySigner(s.signers[1].Key))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err = signable2.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// TODO: this is not needed?
	// // Construct encoders
	// encoders2, err := proposal2.GetEncoders()
	// s.Require().NoError(err)

	// Construct executors
	executors2 := map[types.ChainSelector]sdk.Executor{
		s.ChainA: executor,
	}

	// Construct executable
	executable2, err := mcmslib.NewExecutable(&proposal2, executors2)
	s.Require().NoError(err)

	// SetRoot on the contract
	res, err = executable2.SetRoot(ctx, s.ChainA)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok = res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	tree2, err := proposal2.MerkleTree()
	s.Require().NotNil(tree2)
	s.Require().NoError(err)

	// Validate Contract State and verify root was set A
	rootARoot, rootAValidUntil, err = inspectors[s.ChainA].GetRoot(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(rootARoot, common.Hash([32]byte(tree2.Root.Bytes())))
	s.Require().Equal(rootAValidUntil, proposal2.ValidUntil)

	// Execute the proposal
	res, err = executable2.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok = res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Check the state of the MCMS contract
	// Verify the operation count is updated on chain A
	newOpCountA, err = inspectors[s.ChainA].GetOpCount(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(uint64(3), newOpCountA)

	// TODO (ton): verify actual state changes
	// // Check the state of the timelock contract
	// proposerCount, err = s.ChainA.timelockContract.GetRoleMemberCount(&bind.CallOpts{}, role2)
	// s.Require().NoError(err)
	// s.Require().Equal(big.NewInt(2), proposerCount)
	// proposer, err = s.ChainA.timelockContract.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	// s.Require().NoError(err)
	// s.Require().Equal(s.ChainA.mcmsContract.Address().Hex(), proposer.Hex())
}

// TestExecuteProposalMultipleChains executes a proposal with operations on two different chains
func (s *ExecutionTestSuite) TestExecuteProposalMultipleChains() {
	ctx := context.Background()

	// Op counts before execution
	inspectorA := mcmston.NewInspector(s.TonClient)
	opCountA, err := inspectorA.GetOpCount(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	opCountB := uint64(0)
	opCountC := uint64(0)

	// Construct a TON transaction to grant a role

	// Sends some funds to MCMS contract
	opTX, err := mcmston.NewTransaction(
		address.MustParseAddr(s.mcmsAddr),
		cell.BeginCell().ToSlice(), // empty message (top up)
		tlb.MustFromTON("0.1").Nano(),
		"RBACTimelock",
		[]string{"RBACTimelock", "TopUp"},
	)
	s.Require().NoError(err)

	// Dummy transaction for EVM chains B/C
	dummyTX := evm.NewTransaction(
		s.signers[0].Address(),
		[]byte("0x13424"),
		big.NewInt(0),
		"",
		[]string{""},
	)

	proposalTimelock := mcmslib.TimelockProposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:              "v1",
			Kind:                 types.KindTimelockProposal,
			Description:          "description",
			ValidUntil:           2004259681,
			OverridePreviousRoot: true,
			Signatures:           []types.Signature{},
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				s.ChainA: {
					StartingOpCount:  opCountA,
					MCMAddress:       s.mcmsAddr,
					AdditionalFields: testOpAdditionalFields,
				},
				s.ChainB: {
					StartingOpCount: opCountB,
					MCMAddress:      "0xdead0001",
				},
				s.ChainC: {
					StartingOpCount: opCountC,
					MCMAddress:      "0xdead1001",
				},
			},
		},
		Action: types.TimelockActionSchedule,
		Delay:  types.MustParseDuration("0s"),
		TimelockAddresses: map[types.ChainSelector]string{
			s.ChainA: s.timelockAddr,
			s.ChainB: "0xdead0002",
			s.ChainC: "0xdead1002",
		},
		Operations: []types.BatchOperation{
			{
				ChainSelector: s.ChainA,
				Transactions:  []types.Transaction{opTX},
			},
			{
				ChainSelector: s.ChainB,
				Transactions:  []types.Transaction{dummyTX},
			},
			{
				ChainSelector: s.ChainC,
				Transactions:  []types.Transaction{dummyTX},
			},
		},
	}

	proposal, _, err := proposalTimelock.Convert(ctx, map[types.ChainSelector]sdk.TimelockConverter{
		s.ChainA: mcmston.NewTimelockConverter(),
		s.ChainB: &evm.TimelockConverter{},
		s.ChainC: &evm.TimelockConverter{},
	})
	s.Require().NoError(err)

	tree, err := proposal.MerkleTree()
	s.Require().NotNil(tree)
	s.Require().NoError(err)

	// Sign proposal
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.ChainA: inspectorA,
		s.ChainB: s.newMockEVMInspector(proposal.ChainMetadata[s.ChainB]),
		s.ChainC: s.newMockEVMInspector(proposal.ChainMetadata[s.ChainC]),
	}

	// Construct signable object
	signable, err := mcmslib.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	err = signable.ValidateConfigs(ctx)
	s.Require().NoError(err)

	_, err = signable.SignAndAppend(mcmslib.NewPrivateKeySigner(s.signers[1].Key))
	s.Require().NoError(err)

	// Validate signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders for both chains
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoderA := encoders[s.ChainA].(*mcmston.Encoder)
	encoderB := encoders[s.ChainB].(*evm.Encoder)
	encoderC := encoders[s.ChainC].(*evm.Encoder)

	// Construct executors for both chains
	executors := map[types.ChainSelector]sdk.Executor{
		s.ChainA: must(mcmston.NewExecutor(encoderA, s.TonClient, s.wallet, tlb.MustFromTON("0.1"))),
		s.ChainB: evm.NewExecutor(encoderB, nil, nil), // No need to execute on Chain B or C
		s.ChainC: evm.NewExecutor(encoderC, nil, nil),
	}

	// // Prepare and execute simulation A
	// simulatorB, err := evm.NewSimulator(encoderB, s.ClientA)
	// s.Require().NoError(err, "Failed to create simulator for Chain A")
	// simulatorC, err := evm.NewSimulator(encoderC, s.ClientB)
	// s.Require().NoError(err, "Failed to create simulator for Chain B")
	// simulators := map[types.ChainSelector]sdk.Simulator{
	// 	s.ChainB: simulatorB,
	// 	s.ChainC: simulatorC,
	// }
	// signable.SetSimulators(simulators)
	// err = signable.Simulate(ctx)
	// s.Require().NoError(err)

	// Construct executable object
	executable, err := mcmslib.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on MCMS Contract for Chain A (only)
	res, err := executable.SetRoot(ctx, s.ChainA)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Validate Contract State and verify root was set A
	rootARoot, rootAValidUntil, err := inspectors[s.ChainA].GetRoot(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(rootARoot, common.Hash([32]byte(tree.Root.Bytes())))
	s.Require().Equal(rootAValidUntil, proposal.ValidUntil)

	res, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	// Wait for transaction to be mined
	tx, ok = res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Verify the operation count is updated on chain A
	newOpCountA, err := inspectorA.GetOpCount(ctx, s.mcmsAddr)
	s.Require().NoError(err)
	s.Require().Equal(opCountA+1, newOpCountA)

	// Construct executors
	tExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.ChainA: must(mcmston.NewTimelockExecutor(s.TonClient, s.wallet, tlb.MustFromTON("0.2"))),
		s.ChainB: evm.NewTimelockExecutor(nil, nil), // No need to execute on Chain B or C
		s.ChainC: evm.NewTimelockExecutor(nil, nil),
	}

	// Create new executable
	tExecutable, err := mcmslib.NewTimelockExecutable(ctx, &proposalTimelock, tExecutors)
	s.Require().NoError(err)

	// Notice: skipped as fails on sdk/evm.TimelockInspector.IsOperationReady
	// Could be enabled with an evm TimelockExecutor/Inspector mock similar
	// err = tExecutable.IsReady(ctx)
	// s.Require().NoError(err)

	// Execute operation 0
	res, err = tExecutable.Execute(ctx, 0)
	s.Require().NoError(err)

	// Wait for transaction to be mined
	tx, ok = res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Execute operation 1
	// TODO: expect error as unitialized?
}

var testOpAdditionalFields = json.RawMessage(fmt.Sprintf(`{"value": %d}`, tlb.MustFromTON("0.1").Nano().Uint64()))

// TODO (ton): duplicated with timelock_inspection.go
func (s *ExecutionTestSuite) deployTimelockContract(id uint32) {
	ctx := s.T().Context()
	amount := tlb.MustFromTON("1.5") // TODO: high gas

	data := timelock.EmptyDataFrom(id)
	mcmsAddr := address.MustParseAddr(s.mcmsAddr)
	// When deploying the contract, send the Init message to initialize the Timelock contract
	accounts := []toncommon.WrappedAddress{
		toncommon.WrappedAddress{
			WrappedAddress: mcmsAddr,
		},
		toncommon.WrappedAddress{
			WrappedAddress: s.wallet.Address(),
		},
	}
	body := timelock.Init{
		QueryID:                  0,
		MinDelay:                 0,
		Admin:                    mcmsAddr,
		Proposers:                accounts,
		Executors:                accounts,
		Cancellers:               accounts,
		Bypassers:                accounts,
		ExecutorRoleCheckEnabled: true,
		OpFinalizationTimeout:    0,
	}

	timelockAddr, err := DeployTimelockContract(ctx, s.TonClient, s.wallet, amount, data, body)
	s.Require().NoError(err)
	s.timelockAddr = timelockAddr.String()
}

// TODO (ton): duplicated with set_root.go
func (s *ExecutionTestSuite) deployMCMSContract(id uint32) {
	ctx := s.T().Context()

	// TODO: when MCMS is out of gas, executions fail silently
	// - trace doesn't return error, but opCount doesn't increase
	amount := tlb.MustFromTON("10")
	chainID, err := strconv.ParseInt(s.TonBlockchain.ChainID, 10, 64)
	s.Require().NoError(err)
	data := mcms.EmptyDataFrom(id, s.wallet.Address(), chainID)
	mcmsAddr, err := DeployMCMSContract(ctx, s.TonClient, s.wallet, amount, data)
	s.Require().NoError(err)
	s.mcmsAddr = mcmsAddr.String()

	// Set configuration
	configurerTON, err := mcmston.NewConfigurer(s.wallet, amount)
	s.Require().NoError(err)

	config := &types.Config{
		Quorum:  1,
		Signers: []common.Address{s.signers[0].Address()},
		GroupSigners: []types.Config{
			{
				Quorum:       1,
				Signers:      []common.Address{s.signers[1].Address()},
				GroupSigners: []types.Config{},
			},
		},
	}

	clearRoot := true
	res, err := configurerTON.SetConfig(ctx, s.mcmsAddr, config, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	s.Require().NotNil(res)

	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx.Description)

	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)
}

func (s *ExecutionTestSuite) newMockEVMInspector(rootMetadata types.ChainMetadata) sdk.Inspector {
	return mockEVMInspector{
		config: &types.Config{
			Quorum:  1,
			Signers: []common.Address{s.signers[0].Address()},
			GroupSigners: []types.Config{
				{
					Quorum:       1,
					Signers:      []common.Address{s.signers[1].Address()},
					GroupSigners: []types.Config{},
				},
			},
		},
		opCount:      0,
		root:         common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		rootMetadata: rootMetadata,
	}
}

// Implements sdk.Inspector
type mockEVMInspector struct {
	config       *types.Config
	opCount      uint64
	root         common.Hash
	rootMetadata types.ChainMetadata
}

func (i mockEVMInspector) GetConfig(ctx context.Context, mcmAddr string) (*types.Config, error) {
	return i.config, nil
}

func (i mockEVMInspector) GetOpCount(ctx context.Context, mcmAddr string) (uint64, error) {
	return i.opCount, nil
}

func (i mockEVMInspector) GetRoot(ctx context.Context, mcmAddr string) (common.Hash, uint32, error) {
	return i.root, 0, nil
}

func (i mockEVMInspector) GetRootMetadata(ctx context.Context, mcmAddr string) (types.ChainMetadata, error) {
	return i.rootMetadata, nil
}
