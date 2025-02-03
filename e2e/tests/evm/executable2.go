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

// Execution2TestSuite defines the test suite
type ExecutionTestSuite2 struct {
	suite.Suite
	mcmsContractA     *bindings.ManyChainMultiSig
	mcmsContractB     *bindings.ManyChainMultiSig
	timelockContractA *bindings.RBACTimelock
	timelockContractB *bindings.RBACTimelock
	chainSelectorA    mcmtypes.ChainSelector
	chainSelectorB    mcmtypes.ChainSelector
	deployerKey       common.Address
	signerAddresses   []common.Address
	authA             *bind.TransactOpts
	authB             *bind.TransactOpts
	executorsKey      common.Address
	e2e.TestSetup
}

// SetupSuite initializes both chains
func (s *ExecutionTestSuite2) SetupSuite() {
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

	// Initialize Chain A
	chainIDA, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse Chain A ID")

	s.authA, err = bind.NewKeyedTransactorWithChainID(privateKey, chainIDA)
	s.Require().NoError(err, "Failed to create transactor for Chain A")

	chainDetailsA, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.BlockchainA.Out.Family)
	s.Require().NoError(err)
	s.chainSelectorA = mcmtypes.ChainSelector(chainDetailsA.ChainSelector)

	// Initialize Chain B
	chainIDB, ok := new(big.Int).SetString(s.BlockchainB.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse Chain B ID")

	s.authB, err = bind.NewKeyedTransactorWithChainID(privateKey, chainIDB)
	s.Require().NoError(err, "Failed to create transactor for Chain B")

	chainDetailsB, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainB.Out.ChainID, s.BlockchainB.Out.Family)
	s.Require().NoError(err)
	s.chainSelectorB = mcmtypes.ChainSelector(chainDetailsB.ChainSelector)

	// Deploy contracts on both chains
	s.mcmsContractA = s.deployMCMSContract(s.authA, s.Client)
	s.timelockContractA = s.deployTimelockContract(s.authA, s.Client, s.mcmsContractA.Address().Hex())

	s.mcmsContractB = s.deployMCMSContract(s.authB, s.ClientB)
	s.timelockContractB = s.deployTimelockContract(s.authB, s.ClientB, s.mcmsContractB.Address().Hex())

	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)
}

// deployMCMSContract deploys ManyChainMultiSig on the given chain
func (s *ExecutionTestSuite2) deployMCMSContract(auth *bind.TransactOpts, client *ethclient.Client) *bindings.ManyChainMultiSig {
	_, tx, instance, err := bindings.DeployManyChainMultiSig(auth, client)
	s.Require().NoError(err, "Failed to deploy MCMS contract")

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine MCMS deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0, 1}
	groupQuorums := [32]uint8{1, 1}
	groupParents := [32]uint8{0, 0}
	clearRoot := true

	tx, err = instance.SetConfig(auth, s.signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set MCMS contract configuration")
	receipt, err = bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine MCMS config transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// deployTimelockContract deploys RBACTimelock on the given chain
func (s *ExecutionTestSuite2) deployTimelockContract(auth *bind.TransactOpts, client *ethclient.Client, mcmsAddress string) *bindings.RBACTimelock {
	_, tx, instance, err := bindings.DeployRBACTimelock(
		auth,
		client,
		big.NewInt(0),
		common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"),
		[]common.Address{common.HexToAddress(mcmsAddress)},
		[]common.Address{common.HexToAddress(mcmsAddress)},
		[]common.Address{common.HexToAddress(mcmsAddress)},
		[]common.Address{},
	)
	s.Require().NoError(err, "Failed to deploy Timelock contract")

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine Timelock deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// TestExecuteProposal executes a proposal with operations on two different chains
func (s *ExecutionTestSuite2) TestExecuteProposal2() {
	ctx := context.Background()

	optsA := &bind.CallOpts{
		Context:     ctx,
		From:        s.authA.From,
		BlockNumber: nil,
	}
	optsB := &bind.CallOpts{
		Context:     ctx,
		From:        s.authB.From,
		BlockNumber: nil,
	}

	proposalTimelock := mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Kind:                 mcmtypes.KindTimelockProposal,
			Description:          "description",
			ValidUntil:           2004259681,
			OverridePreviousRoot: false,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata: map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
				s.chainSelectorA: {
					StartingOpCount: 0,
					MCMAddress:      s.mcmsContractA.Address().Hex(),
				},
				s.chainSelectorB: {
					StartingOpCount: 0,
					MCMAddress:      s.mcmsContractB.Address().Hex(),
				},
			},
		},
		Action: mcmtypes.TimelockActionSchedule,
		Delay:  mcmtypes.MustParseDuration("0s"),
		TimelockAddresses: map[mcmtypes.ChainSelector]string{
			s.chainSelectorA: s.timelockContractA.Address().Hex(), s.chainSelectorB: s.timelockContractB.Address().Hex(),
		},

		Operations: []mcmtypes.BatchOperation{{
			ChainSelector: s.chainSelectorA,
			Transactions: []mcmtypes.Transaction{evm.NewTransaction(
				s.signerAddresses[0],
				[]byte("0x13424"),
				big.NewInt(0),
				"",
				[]string{""},
			)},
		}, {
			ChainSelector: s.chainSelectorB,
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
		s.chainSelectorA: &evm.TimelockConverter{},
		s.chainSelectorB: &evm.TimelockConverter{},
	})
	s.Require().NoError(err)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	// Sign proposal
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.chainSelectorA: evm.NewInspector(s.Client),
		s.chainSelectorB: evm.NewInspector(s.ClientB),
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
	encoderA := encoders[s.chainSelectorA].(*evm.Encoder)
	encoderB := encoders[s.chainSelectorB].(*evm.Encoder)

	// Construct executors for both chains
	executors := map[mcmtypes.ChainSelector]sdk.Executor{
		s.chainSelectorA: evm.NewExecutor(
			encoderA,
			s.Client,
			s.authA,
		),
		s.chainSelectorB: evm.NewExecutor(
			encoderB,
			s.ClientB,
			s.authB,
		),
	}

	// Prepare and execute simulation A
	simulatorA, err := evm.NewSimulator(encoderA, s.Client)
	simulatorB, err := evm.NewSimulator(encoderB, s.ClientB)
	s.Require().NoError(err, "Failed to create simulator")
	simulators := map[mcmtypes.ChainSelector]sdk.Simulator{
		s.chainSelectorA: simulatorA,
		s.chainSelectorB: simulatorB,
	}
	signable.SetSimulators(simulators)
	err = signable.Simulate(ctx)
	s.Require().NoError(err)

	// Construct executable object
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on MCMS Contract for both chains
	txA, err := executable.SetRoot(ctx, s.chainSelectorA)
	s.Require().NoError(err)
	s.Require().NotEmpty(txA.Hash)

	txB, err := executable.SetRoot(ctx, s.chainSelectorB)
	s.Require().NoError(err)
	s.Require().NotEmpty(txB.Hash)

	// Wait for transactions to be mined
	receiptA, err := testutils.WaitMinedWithTxHash(ctx, s.Client, common.HexToHash(txA.Hash))
	s.Require().NoError(err, "Failed to mine SetRoot transaction on Chain A")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptA.Status)

	receiptB, err := testutils.WaitMinedWithTxHash(ctx, s.ClientB, common.HexToHash(txB.Hash))
	s.Require().NoError(err, "Failed to mine SetRoot transaction on Chain B")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptB.Status)

	// Validate Contract State and verify root was set A
	rootA, err := s.mcmsContractA.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(rootA.Root, [32]byte(tree.Root.Bytes()))
	s.Require().Equal(rootA.ValidUntil, proposal.ValidUntil)

	// Validate Contract State and verify root was set
	rootB, err := s.mcmsContractB.GetRoot(&bind.CallOpts{})
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
	receiptA, err = testutils.WaitMinedWithTxHash(ctx, s.Client, common.HexToHash(txA.Hash))
	s.Require().NoError(err, "Failed to mine execution transaction on Chain A")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptA.Status)

	receiptB, err = testutils.WaitMinedWithTxHash(ctx, s.ClientB, common.HexToHash(txB.Hash))
	s.Require().NoError(err, "Failed to mine execution transaction on Chain B")
	s.Require().Equal(types.ReceiptStatusSuccessful, receiptB.Status)

	// Verify the operation count is updated on both chains
	newOpCountA, err := s.mcmsContractA.GetOpCount(optsA)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), newOpCountA.Uint64())

	newOpCountB, err := s.mcmsContractB.GetOpCount(optsB)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), newOpCountB.Uint64())

	// Construct executors
	tExecutors := map[mcmtypes.ChainSelector]sdk.TimelockExecutor{
		s.chainSelectorA: evm.NewTimelockExecutor(
			s.Client,
			s.authA,
		),
		s.chainSelectorB: evm.NewTimelockExecutor(
			s.ClientB,
			s.authB,
		),
	}

	// Create new executable
	tExecutable, err := mcms.NewTimelockExecutable(&proposalTimelock, tExecutors)
	s.Require().NoError(err)

	err = tExecutable.IsReady(ctx)
	s.Require().NoError(err)

	// Execute operation 0
	txA, err = tExecutable.Execute(ctx, 0)
	s.Require().NoError(err)
	// Wait for execution transactions to be mined
	timelockReceiptA, err := testutils.WaitMinedWithTxHash(ctx, s.Client, common.HexToHash(txA.Hash))
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
