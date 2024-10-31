package evm

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/smartcontractkit/mcms/internal/core/timelock"
	gethwrappers "github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
)

type Account struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
	Opts       *bind.TransactOpts
}

// // TODO: Create this generic function
type BackendConfig struct {
	accounts []Account
	backend  *simulated.Backend
}

func SetupBackendWithAccounts(numSigners uint32) (*BackendConfig, error) {
	keys := make([]*ecdsa.PrivateKey, numSigners)
	auths := make([]*bind.TransactOpts, numSigners)
	for i := uint32(0); i < numSigners; i++ {
		key, _ := crypto.GenerateKey()
		auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
		if err != nil {
			return nil, err
		}
		auth.GasLimit = uint64(8000000)
		keys[i] = key
		auths[i] = auth
	}

	// Setup a simulated backend
	// TODO: remove deprecated call
	//nolint:staticcheck
	genesisAlloc := map[common.Address]ethTypes.Account{}
	accounts := make([]Account, numSigners)
	for i, auth := range auths {
		genesisAlloc[auth.From] = ethTypes.Account{Balance: big.NewInt(1e18)}
		accounts[i] = Account{
			PrivateKey: keys[i],
			Address:    auth.From,
			Opts:       auth,
		}
	}
	backend := simulated.NewBackend(genesisAlloc)
	return &BackendConfig{
		accounts: accounts,
		backend:  backend,
	}, nil
}

// Deploys a timelock contract. The signer will own the contract, can proposa and execute proposals.
func DeployTimelockContract(backend *simulated.Backend, deployer *bind.TransactOpts, signer common.Address) (common.Address, *gethwrappers.RBACTimelock, error) {
	// Deploy the Timelock contract
	address, _, contract, err := gethwrappers.DeployRBACTimelock(
		deployer,
		backend.Client(),
		big.NewInt(0),
		signer,
		[]common.Address{signer},
		[]common.Address{signer},
		[]common.Address{signer},
		[]common.Address{signer},
	)
	if err != nil {
		return common.Address{}, nil, err
	}
	return address, contract, nil
}

func SignAndSend(backend *simulated.Backend, signer *bind.TransactOpts, to common.Address, data []byte, value uint32) error {
	client := backend.Client()
	nonce, err := client.PendingNonceAt(context.Background(), signer.From)
	if err != nil {
		return err
	}
	gasLimit := uint64(100000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	ethTx := ethTypes.NewTransaction(nonce, to, big.NewInt(int64(value)), gasLimit, gasPrice, data)
	signedTx, err := ethTypes.SignTx(ethTx, ethTypes.NewEIP155Signer(big.NewInt(1337)), signer.PrivateKey)

}

func Test_TimelockProposal(t *testing.T) {
	t.Parallel()

	// Setup the environment
	backendConfig, err := SetupBackendWithAccounts(10)
	require.NoError(t, err)

	// Deploy the timelock contract
	account := backendConfig.accounts[0]
	timelockAddress, timelockContract, err := DeployTimelockContract(backendConfig.backend, account.PrivateKey, account.From)
	require.NoError(t, err)

	// Generate the grantRole tx data. Generating the inner tx data is beyond the scope of this library
	timelockAbi, err := gethwrappers.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	role, err := timelockContract.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, backendConfig.accounts[1].From)
	require.NoError(t, err)

	// Creates dummy transactions
	txs := []types.BatchChainOperation{
		{
			ChainIdentifier: 1337,
			Batch: []types.Operation{
				{
					To:           timelockAddress,
					Data:         grantRoleData,
					Value:        big.NewInt(0),
					ContractType: "RBACTimelock",
					Tags:         []string{"grantRole"},
				},
			},
			TimelockAddress: timelockAddress.String(),
		},
	}

	timelockProposal, err := timelock.NewTimelockProposal(timelock.Schedule, txs, "1d")
	require.NoError(t, err)

	// Create a new EVM Timelock Proposal
	evmTimelockProposal := NewEVMTimelockProposal(*timelockProposal)
	wrappedOps, err := evmTimelockProposal.Encode()
	require.NoError(t, err)

	client := backendConfig.backend.Client()
	nonce, err := client.PendingNonceAt(context.Background(), account.From)
	require.NoError(t, err)
	gasLimit := uint64(100000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	// Send wrapped transactions to the backend
	for _, o := range wrappedOps {
		ethTx := ethTypes.NewTransaction(nonce, o.To, o.Value, gasLimit, gasPrice, o.Data)
		signedTx, err := ethTypes.SignTx(ethTx, ethTypes.NewEIP155Signer(big.NewInt(1337)), account.PrivateKey)

		err := client.SendTransaction(context.Background(), signedTx)
		require.NoError(t, err)
	}

}
