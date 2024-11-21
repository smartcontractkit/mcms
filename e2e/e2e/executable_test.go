//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/e2e/testdata/anvil"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// ExecutionTestSuite defines the test suite
type ExecutionTestSuite struct {
	suite.Suite
	client           *ethclient.Client
	mcmsContract     *bindings.ManyChainMultiSig
	deployerKey      common.Address
	signerAddresses  []common.Address
	auth             *bind.TransactOpts
	timelockContract *bindings.RBACTimelock
}

// SetupSuite runs before the test suite
func (s *ExecutionTestSuite) SetupSuite() {
	// Load the configuration
	in, err := framework.Load[Config](s.T())
	s.Require().NoError(err, "Failed to load configuration")

	// Initialize the blockchain
	bc, err := blockchain.NewBlockchainNetwork(in.BlockchainA)
	s.Require().NoError(err, "Failed to initialize blockchain network")

	// Initialize Ethereum client
	wsURL := bc.Nodes[0].HostWSUrl
	client, err := ethclient.DialContext(context.Background(), wsURL)
	s.Require().NoError(err, "Failed to initialize Ethereum client")
	s.client = client

	// Get deployer's private key
	privateKeyHex := in.Settings.PrivateKey
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err, "Invalid private key")

	// Define signer addresses
	s.signerAddresses = []common.Address{
		common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"),
		common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	}

	// Parse ChainID from string to int64
	chainID, ok := new(big.Int).SetString(in.BlockchainA.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.auth = auth

	s.mcmsContract = s.deployMCMSContract()
	s.timelockContract = s.deployTimelockContract(s.mcmsContract.Address().Hex())
	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)
}

// deployContract is a helper to deploy the contract
func (s *ExecutionTestSuite) deployMCMSContract() *bindings.ManyChainMultiSig {
	_, tx, instance, err := bindings.DeployManyChainMultiSig(s.auth, s.client)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = instance.SetConfig(s.auth, s.signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	receipt, err = bind.WaitMined(context.Background(), s.client, tx)
	s.Require().NoError(err, "Failed to mine configuration transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// deployContract is a helper to deploy the contract
func (s *ExecutionTestSuite) deployTimelockContract(mcmsAddress string) *bindings.RBACTimelock {
	_, tx, instance, err := bindings.DeployRBACTimelock(
		s.auth,
		s.client,
		big.NewInt(0),
		common.HexToAddress(mcmsAddress),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// TestGetConfig checks contract configuration
func (s *ExecutionTestSuite) TestExecute() {
	opts := &bind.CallOpts{
		Context:     context.Background(), // Use a proper context
		From:        s.auth.From,          // Set the "from" address (optional)
		BlockNumber: nil,                  // Use the latest block (nil by default)
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
				chaintest.Chain1Selector: {
					StartingOpCount: 0,
					MCMAddress:      s.mcmsContract.Address().Hex(),
				},
			},
		},
		Operations: []mcmtypes.Operation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: evm.NewOperation(
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
		chaintest.Chain1Selector: evm.NewInspector(s.client),
	}

	// Construct executor
	signable, err := mcms.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(anvil.Accounts[1].PrivateKey)))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	// Construct executors
	executors := map[mcmtypes.ChainSelector]sdk.Executor{
		chaintest.Chain1Selector: evm.NewExecutor(
			encoders[chaintest.Chain1Selector].(*evm.Encoder),
			s.client,
			s.auth,
		),
	}

	// Construct executable
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(chaintest.Chain1Selector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txHash)

	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.client, common.HexToHash(txHash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	root, err := s.mcmsContract.GetRoot(&bind.CallOpts{})
	s.Require().NoError(err)
	s.Require().Equal(root.Root, [32]byte(tree.Root))
	s.Require().Equal(root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	txHash, err = executable.Execute(0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txHash)

	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.client, common.HexToHash(txHash))
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
