package timelock

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/pkg/config"
	mcm_errors "github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms"
)

var TestAddress = common.HexToAddress("0x1234567890abcdef")
var TestChain1 = mcms.ChainSelector(3379446385462418246)
var TestChain2 = mcms.ChainSelector(16015286601757825753)
var TestChain3 = mcms.ChainSelector(10344971235874465080)

func TestValidate_ValidProposal(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						AdditionalFields: additionalFields,
						Value:            big.NewInt(0),
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		Schedule,
		"1h",
	)

	require.NoError(t, err)
	assert.NotNil(t, proposal)
}

func TestValidate_InvalidOperation(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						Value:            big.NewInt(0),
						AdditionalFields: additionalFields,
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		"invalid",
		"1h",
	)

	require.Error(t, err)
	assert.Nil(t, proposal)
	assert.IsType(t, &mcm_errors.InvalidTimelockOperationError{}, err)
}

func TestValidate_InvalidMinDelaySchedule(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						Value:            big.NewInt(0),
						AdditionalFields: additionalFields,
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		Schedule,
		"invalid",
	)

	require.Error(t, err)
	assert.Nil(t, proposal)
	assert.EqualError(t, err, "time: invalid duration \"invalid\"")
}

func TestValidate_InvalidUntilTimeError(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		1697398311, // Old date (2023-10-15)
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						Value:            big.NewInt(0),
						AdditionalFields: additionalFields,
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		Schedule,
		"invalid",
	)

	require.Error(t, err)
	assert.Nil(t, proposal)
	assert.EqualError(t, err, "invalid valid until: 1697398311")
}

func TestValidate_InvalidNoChainMetadata(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						Value:            big.NewInt(0),
						AdditionalFields: additionalFields,
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		Schedule,
		"1h",
	)

	require.Error(t, err)
	assert.Nil(t, proposal)
	assert.EqualError(t, err, "missing chain metadata for chain 3379446385462418246")
}

func TestValidate_InvalidNoTransactions(t *testing.T) {
	t.Parallel()

	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"Sample description",
		[]BatchChainOperation{},
		Schedule,
		"1h",
	)

	require.Error(t, err)
	assert.Nil(t, proposal)
	assert.EqualError(t, err, "no transactions")
}

func TestValidate_InvalidNoDescription(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						Value:            big.NewInt(0),
						AdditionalFields: additionalFields,
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		Schedule,
		"1h",
	)

	require.Error(t, err)
	assert.Nil(t, proposal)
	assert.EqualError(t, err, "invalid description: ")
}

func TestValidate_InvalidVersion(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"test",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						Value:            big.NewInt(0),
						AdditionalFields: additionalFields,
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		Schedule,
		"1h",
	)

	require.Error(t, err)
	assert.Nil(t, proposal)
	assert.EqualError(t, err, "invalid version: ")
}

func TestValidate_InvalidMinDelayBypassShouldBeValid(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: TestAddress,
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               TestAddress,
						Value:            big.NewInt(0),
						AdditionalFields: additionalFields,
						Data:             common.Hex2Bytes("0x"),
						ContractType:     "Sample contract",
						Tags:             []string{"tag1", "tag2"},
					},
				},
			},
		},
		Bypass,
		"invalid",
	)

	require.NoError(t, err)
	assert.NotNil(t, proposal)
}

// Constructs a simulated backend with a ManyChainMultiSig contract and a RBACTimelock contract
// The Admin of the RBACTimelock is itself and the RBACTimelock owns the ManyChainMultiSig
func setupSimulatedBackendWithMCMSAndTimelock(numSigners uint64) ([]*ecdsa.PrivateKey, []*bind.TransactOpts, *backends.SimulatedBackend, *gethwrappers.ManyChainMultiSig, *gethwrappers.RBACTimelock, error) {
	// Generate a private key
	keys := make([]*ecdsa.PrivateKey, numSigners)
	auths := make([]*bind.TransactOpts, numSigners)
	for i := range numSigners {
		key, _ := crypto.GenerateKey()
		auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		auth.GasLimit = uint64(8000000)
		keys[i] = key
		auths[i] = auth
	}

	// Setup a simulated backend
	// TODO: remove deprecated call
	//nolint:staticcheck
	genesisAlloc := map[common.Address]core.GenesisAccount{}
	for _, auth := range auths {
		// TODO: remove deprecated call
		//nolint:staticcheck
		genesisAlloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(1e18)}
	}
	blockGasLimit := uint64(8000000)
	// TODO: remove deprecated call
	//nolint:staticcheck
	sim := backends.NewSimulatedBackend(genesisAlloc, blockGasLimit)

	// Deploy a ManyChainMultiSig contract with any of the signers
	mcmAddr, tx, mcmsContract, err := gethwrappers.DeployManyChainMultiSig(auths[0], sim)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the contract to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Check the receipt status
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, nil, nil, nil, nil, errors.New("contract deployment failed")
	}

	// Set a valid config
	signers := make([]common.Address, numSigners)
	for i, auth := range auths {
		signers[i] = auth.From
	}

	// Set the config
	castedNumSigner, err := mcms.SafeCastUint64ToUint8(numSigners)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	cfg := &config.Config{
		Quorum:       castedNumSigner,
		Signers:      signers,
		GroupSigners: []config.Config{},
	}
	quorums, parents, signersAddresses, signerGroups, err := cfg.ExtractSetConfigInputs()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	tx, err = mcmsContract.SetConfig(auths[0], signersAddresses, signerGroups, quorums, parents, false)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Deploy a timelock contract for testing
	_, tx, timelock, err := gethwrappers.DeployRBACTimelock(
		auths[0],
		sim,
		big.NewInt(0),
		auths[0].From, // Temporarily set the admin to the first signer
		[]common.Address{mcmAddr},
		[]common.Address{mcmAddr, auths[0].From},
		[]common.Address{mcmAddr},
		[]common.Address{mcmAddr},
	)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the contract to be mined
	_, err = bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Transfer the ownership of the ManyChainMultiSig to the timelock
	tx, err = mcmsContract.TransferOwnership(auths[0], timelock.Address())
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Construct payload for Accepting the ownership of the ManyChainMultiSig
	mcmsAbi, err := gethwrappers.ManyChainMultiSigMetaData.GetAbi()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	acceptOwnershipData, err := mcmsAbi.Pack("acceptOwnership")
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Accept the ownership of the ManyChainMultiSig
	tx, err = timelock.BypasserExecuteBatch(auths[0], []gethwrappers.RBACTimelockCall{
		{
			Target: mcmsContract.Address(),
			Data:   acceptOwnershipData,
			Value:  big.NewInt(0),
		},
	})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Give the timelock admin rights
	role, err := timelock.ADMINROLE(&bind.CallOpts{})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	tx, err = timelock.GrantRole(auths[0], role, timelock.Address())
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Revoking the admin rights of the first signer
	tx, err = timelock.RevokeRole(auths[0], role, auths[0].From)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return keys, auths, sim, mcmsContract, timelock, nil
}

func TestE2E_ValidScheduleAndExecuteProposalOneTx(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	keys, auths, sim, mcmsObj, timelock, err := setupSimulatedBackendWithMCMSAndTimelock(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	assert.NotNil(t, timelock)

	// Construct example transaction to grant EOA the PROPOSER role
	role, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := gethwrappers.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, auths[0].From)
	require.NoError(t, err)

	// Validate Contract State and verify role does not exist
	hasRole, err := timelock.HasRole(&bind.CallOpts{}, role, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)

	// Construct example transaction
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address(),
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: timelock.Address(),
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               timelock.Address(),
						AdditionalFields: additionalFields,
						Value:            big.NewInt(0),
						Data:             grantRoleData,
					},
				},
			},
		},
		Schedule,
		"5s",
	)
	require.NoError(t, err)
	assert.NotNil(t, proposal)

	// Gen caller map for easy access
	callers := map[mcms.ChainSelector]mcms.ContractDeployBackend{TestChain1: sim}

	// Construct executor
	executor, err := proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err := executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err := crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err := mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err := executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check all the logs
	var operationId common.Hash
	for _, log := range receipt.Logs {
		event, perr := timelock.ParseCallScheduled(*log)
		if perr == nil {
			operationId = event.Id
		}
	}

	// Validate Contract State and verify operation was scheduled
	grantRoleCall := []gethwrappers.RBACTimelockCall{
		{
			Target: timelock.Address(),
			Value:  big.NewInt(0),
			Data:   grantRoleData,
		},
	}

	isOperation, err := timelock.IsOperation(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperation)
	isOperationPending, err := timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationPending)
	isOperationReady, err := timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationReady)

	// sleep for 5 seconds and then mine a block
	require.NoError(t, sim.AdjustTime(5*time.Second))
	sim.Commit() // Note < 1.14 geth needs a commit after adjusting time.

	// Check that the operation is now ready
	isOperationReady, err = timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationReady)

	// Execute the operation
	tx, err = timelock.ExecuteBatch(auths[0], grantRoleCall, ZERO_HASH, ZERO_HASH)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err = bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check that the operation is done
	isOperationDone, err := timelock.IsOperationDone(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationDone)

	// Check that the operation is no longer pending
	isOperationPending, err = timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationPending)

	// Validate Contract State and verify role was granted
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, role, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
}

func TestE2E_ValidScheduleAndCancelProposalOneTx(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	keys, auths, sim, mcmsObj, timelock, err := setupSimulatedBackendWithMCMSAndTimelock(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	assert.NotNil(t, timelock)

	// Construct example transaction to grant EOA the PROPOSER role
	role, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := gethwrappers.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, auths[0].From)
	require.NoError(t, err)

	// Validate Contract State and verify role does not exist
	hasRole, err := timelock.HasRole(&bind.CallOpts{}, role, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)

	// Construct example transaction
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address(),
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: timelock.Address(),
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               timelock.Address(),
						AdditionalFields: additionalFields,
						Value:            big.NewInt(0),
						Data:             grantRoleData,
					},
				},
			},
		},
		Schedule,
		"5s",
	)
	require.NoError(t, err)
	assert.NotNil(t, proposal)

	// Gen caller map for easy access
	callers := map[mcms.ChainSelector]mcms.ContractDeployBackend{TestChain1: sim}

	// Construct executor
	executor, err := proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err := executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err := crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err := mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err := executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check all the logs
	var operationId common.Hash
	for _, log := range receipt.Logs {
		event, perr := timelock.ParseCallScheduled(*log)
		if perr == nil {
			operationId = event.Id
		}
	}

	// Check operation state and see that it was scheduled
	isOperation, err := timelock.IsOperation(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperation)
	isOperationPending, err := timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationPending)
	isOperationReady, err := timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationReady)

	// Get and validate the current operation count
	currOpCount, err := mcmsObj.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, currOpCount.Int64(), int64(len(proposal.Transactions)))

	// Generate a new proposal to cancel the operation
	// Update the proposal Operation to Cancel
	// Update the proposal ChainMetadata StartingOpCount to the current operation count
	proposal.Operation = Cancel
	proposal.ChainMetadata[TestChain1] = mcms.ChainMetadata{
		StartingOpCount: currOpCount.Uint64(),
		MCMAddress:      mcmsObj.Address(),
	}

	// Construct executor
	executor, err = proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err = executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err = crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err = mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err = executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err = executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err = mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err = bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Verify operation state and confirm it was cancelled
	isOperation, err = timelock.IsOperation(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperation)
	isOperationPending, err = timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationPending)
	isOperationReady, err = timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationReady)
}

func TestE2E_ValidBypassProposalOneTx(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	keys, auths, sim, mcmsObj, timelock, err := setupSimulatedBackendWithMCMSAndTimelock(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	assert.NotNil(t, timelock)

	// Construct example transaction to grant EOA the PROPOSER role
	role, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := gethwrappers.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, auths[0].From)
	require.NoError(t, err)

	// Validate Contract State and verify role does not exist
	hasRole, err := timelock.HasRole(&bind.CallOpts{}, role, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)

	// Construct example transaction
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address(),
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: timelock.Address(),
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch: []mcms.Operation{
					{
						To:               timelock.Address(),
						AdditionalFields: additionalFields,
						Value:            big.NewInt(0),
						Data:             grantRoleData,
					},
				},
			},
		},
		Bypass,
		"",
	)
	require.NoError(t, err)
	assert.NotNil(t, proposal)

	// Gen caller map for easy access
	callers := map[mcms.ChainSelector]mcms.ContractDeployBackend{TestChain1: sim}

	// Construct executor
	executor, err := proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err := executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err := crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err := mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err := executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Validate Contract State and verify role was granted
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, role, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
}

func TestE2E_ValidScheduleAndExecuteProposalOneBatchTx(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	keys, auths, sim, mcmsObj, timelock, err := setupSimulatedBackendWithMCMSAndTimelock(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	assert.NotNil(t, timelock)

	// Construct example transactions
	proposerRole, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelock.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelock.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := gethwrappers.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]mcms.Operation, 3)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole} {
		data, perr := timelockAbi.Pack("grantRole", role, auths[0].From)
		require.NoError(t, perr)
		operations[i] = mcms.Operation{
			To:               timelock.Address(),
			AdditionalFields: additionalFields,
			Value:            big.NewInt(0),
			Data:             data,
		}
	}

	// Validate Contract State and verify role does not exist
	hasRole, err := timelock.HasRole(&bind.CallOpts{}, proposerRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, bypasserRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, cancellerRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)

	// Construct example transaction
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address(),
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: timelock.Address(),
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch:         operations,
			},
		},
		Schedule,
		"5s",
	)
	require.NoError(t, err)
	assert.NotNil(t, proposal)

	// Gen caller map for easy access
	callers := map[mcms.ChainSelector]mcms.ContractDeployBackend{TestChain1: sim}

	// Construct executor
	executor, err := proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err := executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err := crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err := mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err := executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check all the logs
	var operationId common.Hash
	for _, log := range receipt.Logs {
		event, perr := timelock.ParseCallScheduled(*log)
		if perr == nil {
			operationId = event.Id
		}
	}

	// Validate Contract State and verify operation was scheduled
	grantRoleCalls := []gethwrappers.RBACTimelockCall{
		{
			Target: timelock.Address(),
			Value:  big.NewInt(0),
			Data:   operations[0].Data,
		},
		{
			Target: timelock.Address(),
			Value:  big.NewInt(0),
			Data:   operations[1].Data,
		},
		{
			Target: timelock.Address(),
			Value:  big.NewInt(0),
			Data:   operations[2].Data,
		},
	}

	isOperation, err := timelock.IsOperation(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperation)
	isOperationPending, err := timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationPending)
	isOperationReady, err := timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationReady)

	// sleep for 5 seconds and then mine a block
	require.NoError(t, sim.AdjustTime(5*time.Second))
	sim.Commit()

	// Check that the operation is now ready
	isOperationReady, err = timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationReady)

	// Execute the operation
	tx, err = timelock.ExecuteBatch(auths[0], grantRoleCalls, ZERO_HASH, ZERO_HASH)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err = bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check that the operation is done
	isOperationDone, err := timelock.IsOperationDone(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationDone)

	// Check that the operation is no longer pending
	isOperationPending, err = timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationPending)

	// Validate Contract State and verify role was granted
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, proposerRole, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, bypasserRole, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, cancellerRole, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
}

func TestE2E_ValidScheduleAndCancelProposalOneBatchTx(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	keys, auths, sim, mcmsObj, timelock, err := setupSimulatedBackendWithMCMSAndTimelock(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	assert.NotNil(t, timelock)

	// Construct example transactions
	proposerRole, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelock.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelock.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := gethwrappers.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]mcms.Operation, 3)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole} {
		data, perr := timelockAbi.Pack("grantRole", role, auths[0].From)
		require.NoError(t, perr)
		operations[i] = mcms.Operation{
			To:               timelock.Address(),
			AdditionalFields: additionalFields,
			Value:            big.NewInt(0),
			Data:             data,
		}
	}

	// Validate Contract State and verify role does not exist
	hasRole, err := timelock.HasRole(&bind.CallOpts{}, proposerRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, bypasserRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, cancellerRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)

	// Construct example transaction
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address(),
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: timelock.Address(),
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch:         operations,
			},
		},
		Schedule,
		"5s",
	)
	require.NoError(t, err)
	assert.NotNil(t, proposal)

	// Gen caller map for easy access
	callers := map[mcms.ChainSelector]mcms.ContractDeployBackend{TestChain1: sim}

	// Construct executor
	executor, err := proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err := executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err := crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err := mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err := executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check all the logs
	var operationId common.Hash
	for _, log := range receipt.Logs {
		event, perr := timelock.ParseCallScheduled(*log)
		if perr == nil {
			operationId = event.Id
		}
	}

	// Check operation state and see that it was scheduled
	isOperation, err := timelock.IsOperation(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperation)
	isOperationPending, err := timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.True(t, isOperationPending)
	isOperationReady, err := timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationReady)

	// Get and validate the current operation count
	currOpCount, err := mcmsObj.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, currOpCount.Int64(), int64(len(proposal.Transactions)))

	// Generate a new proposal to cancel the operation
	// Update the proposal Operation to Cancel
	// Update the proposal ChainMetadata StartingOpCount to the current operation count
	proposal.Operation = Cancel
	proposal.ChainMetadata[TestChain1] = mcms.ChainMetadata{
		StartingOpCount: currOpCount.Uint64(),
		MCMAddress:      mcmsObj.Address(),
	}

	// Construct executor
	executor, err = proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err = executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err = crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err = mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err = executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err = executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err = mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err = bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Verify operation state and confirm it was cancelled
	isOperation, err = timelock.IsOperation(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperation)
	isOperationPending, err = timelock.IsOperationPending(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationPending)
	isOperationReady, err = timelock.IsOperationReady(&bind.CallOpts{}, operationId)
	require.NoError(t, err)
	assert.False(t, isOperationReady)
}

func TestE2E_ValidBypassProposalOneBatchTx(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	keys, auths, sim, mcmsObj, timelock, err := setupSimulatedBackendWithMCMSAndTimelock(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	assert.NotNil(t, timelock)

	// Construct example transactions
	proposerRole, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelock.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelock.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := gethwrappers.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]mcms.Operation, 3)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole} {
		data, perr := timelockAbi.Pack("grantRole", role, auths[0].From)
		require.NoError(t, perr)
		operations[i] = mcms.Operation{
			To:               timelock.Address(),
			AdditionalFields: additionalFields,
			Value:            big.NewInt(0),
			Data:             data,
		}
	}

	// Validate Contract State and verify role does not exist
	hasRole, err := timelock.HasRole(&bind.CallOpts{}, proposerRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, bypasserRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, cancellerRole, auths[0].From)
	require.NoError(t, err)
	assert.False(t, hasRole)

	// Construct example transaction
	proposal, err := NewMCMSWithTimelockProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address(),
			},
		},
		map[mcms.ChainSelector]common.Address{
			TestChain1: timelock.Address(),
		},
		"Sample description",
		[]BatchChainOperation{
			{
				ChainSelector: TestChain1,
				Batch:         operations,
			},
		},
		Bypass,
		"",
	)
	require.NoError(t, err)
	assert.NotNil(t, proposal)

	// Gen caller map for easy access
	callers := map[mcms.ChainSelector]mcms.ContractDeployBackend{TestChain1: sim}

	// Construct executor
	executor, err := proposal.ToExecutor(true)
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// Get the hash to sign
	hash, err := executor.SigningHash()
	require.NoError(t, err)

	// Sign the hash
	sig, err := crypto.Sign(hash.Bytes(), keys[0])
	require.NoError(t, err)

	// Construct a signature
	sigObj, err := mcms.NewSignatureFromBytes(sig)
	require.NoError(t, err)
	executor.Proposal.Signatures = append(proposal.Signatures, sigObj)

	// Validate the signatures
	quorumMet, err := executor.ValidateSignatures(callers)
	assert.True(t, quorumMet)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executor.SetRootOnChain(sim, auths[0], TestChain1)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(executor.Tree.Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executor.ExecuteOnChain(sim, auths[0], 0)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Validate Contract State and verify role was granted
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, proposerRole, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, bypasserRole, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
	hasRole, err = timelock.HasRole(&bind.CallOpts{}, cancellerRole, auths[0].From)
	require.NoError(t, err)
	assert.True(t, hasRole)
}

func TestTimelockProposalFromFile(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	mcmsProposal := MCMSWithTimelockProposal{
		MCMSProposal: mcms.MCMSProposal{
			Version:              "MCMSWithTimelock",
			ValidUntil:           4128029039,
			Signatures:           []mcms.Signature{},
			OverridePreviousRoot: false,
			Description:          "Test Proposal",
			ChainMetadata: map[mcms.ChainSelector]mcms.ChainMetadata{
				mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector): {
					StartingOpCount: 0,
					MCMAddress:      common.Address{},
				},
			},
		},
		TimelockAddresses: make(map[mcms.ChainSelector]common.Address),
		Transactions: []BatchChainOperation{
			{
				ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Batch: []mcms.Operation{
					{
						AdditionalFields: additionalFields,
					},
				},
			},
		},
		Operation: Schedule,
		MinDelay:  "1h",
	}

	tempFile, err := os.CreateTemp("", "timelock.json")
	require.NoError(t, err)

	proposalBytes, err := json.Marshal(mcmsProposal)
	require.NoError(t, err)
	err = os.WriteFile(tempFile.Name(), proposalBytes, 0600)
	require.NoError(t, err)

	fileProposal, err := NewMCMSWithTimelockProposalFromFile(tempFile.Name())
	require.NoError(t, err)
	assert.EqualValues(t, mcmsProposal, *fileProposal)
}

const validJsonProposal = `{
  "chainMetadata": {
    "16015286601757825753": {
      "mcmAddress": "0x0000000000000000000000000000000000000000",
      "startingOpCount": 0
    }
  },
  "description": "Test proposal",
  "minDelay": "1d",
  "operation": "schedule",
  "overridePreviousRoot": true,
  "signatures": null,
  "timelockAddresses": {},
  "transactions": [
    {
      "batch": [
        {
          "AdditionalFields": {
            "value": 0
          },
          "contractType": "",
          "data": "ZGF0YQ==",
          "tags": null,
          "to": "0x0000000000000000000000000000000000000000",
          "value": 0
        }
      ],
      "chainSelector": 16015286601757825753
    }
  ],
  "validUntil": 4128029039,
  "version": "MCMSWithTimelock"
}`

func TestMCMSWithTimelockProposal_MarshalJSON(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	tests := []struct {
		name       string
		proposal   MCMSWithTimelockProposal
		wantErr    bool
		expectJSON string
	}{
		{
			name: "successful marshalling",
			proposal: MCMSWithTimelockProposal{
				MCMSProposal: mcms.MCMSProposal{
					Version:     "MCMSWithTimelock",
					ValidUntil:  4128029039,
					Description: "Test proposal",
					ChainMetadata: map[mcms.ChainSelector]mcms.ChainMetadata{
						mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector): {
							StartingOpCount: 0,
							MCMAddress:      common.Address{},
						},
					},
					OverridePreviousRoot: true,
				},
				Operation:         Schedule,
				MinDelay:          "1d",
				TimelockAddresses: map[mcms.ChainSelector]common.Address{},
				Transactions: []BatchChainOperation{
					{
						ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
						Batch: []mcms.Operation{
							{
								To:               common.HexToAddress("0x0"),
								AdditionalFields: additionalFields,
								Data:             []byte("data"),
								Value:            big.NewInt(0),
							},
						},
					},
				},
			},
			wantErr:    false,
			expectJSON: validJsonProposal,
		},
		{
			name: "error during marshalling transactions",
			proposal: MCMSWithTimelockProposal{
				Transactions: []BatchChainOperation{
					{
						ChainSelector: mcms.ChainSelector(1),
						Batch:         nil, // This will cause an error because Batch should not be nil
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(&tt.proposal)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.expectJSON, string(got))
			}
		})
	}
}

func TestMCMSWithTimelockProposal_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	// Prepare the compact version of the validJsonProposal
	var compactBuffer bytes.Buffer
	err := json.Compact(&compactBuffer, []byte(validJsonProposal))
	require.NoError(t, err)

	// Use the compact JSON as the one-liner version
	compactJsonProposal := compactBuffer.String()

	// Preparing the additional fields
	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
		expected MCMSWithTimelockProposal
	}{
		{
			name:     "successful unmarshalling",
			jsonData: compactJsonProposal,
			wantErr:  false,
			expected: MCMSWithTimelockProposal{
				MCMSProposal: mcms.MCMSProposal{
					Version:     "MCMSWithTimelock",
					ValidUntil:  4128029039,
					Description: "Test proposal",
					ChainMetadata: map[mcms.ChainSelector]mcms.ChainMetadata{
						mcms.ChainSelector(16015286601757825753): {
							StartingOpCount: 0,
							MCMAddress:      common.Address{},
						},
					},
					OverridePreviousRoot: true,
				},
				Operation:         Schedule,
				MinDelay:          "1d",
				TimelockAddresses: map[mcms.ChainSelector]common.Address{},
				Transactions: []BatchChainOperation{
					{
						ChainSelector: mcms.ChainSelector(16015286601757825753),
						Batch: []mcms.Operation{
							{
								To:               common.HexToAddress("0x0000000000000000000000000000000000000000"),
								AdditionalFields: additionalFields,
								Data:             []byte("data"),
								Value:            big.NewInt(0),
							},
						},
					},
				},
			},
		},
		{
			name: "error during unmarshalling invalid JSON",
			jsonData: `{
				"version":"1.0",
				"validUntil":123456789,
				"description":"Test proposal",
				"operation":"invalid_operation"
			}`, // invalid operation field
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got MCMSWithTimelockProposal
			err := json.Unmarshal([]byte(tt.jsonData), &got)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
