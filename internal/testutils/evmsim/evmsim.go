// package evmsim implements a simulated EVM chain for testing purposes.
package evmsim

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

const (
	// DefaultGasLimit is the default gas limit for each transaction in the simulated chain
	DefaultGasLimit = uint64(8000000)

	// DefaultBalance is the default balance for each account in the simulated chain
	DefaultBalance = 1e18

	// SimulatedChainID is the chain ID used for the simulated chain. EVM Simulated chains always use 1337
	//
	// https://pkg.go.dev/github.com/ethereum/go-ethereum/ethclient/simulated#NewBackend
	SimulatedChainID = 1337
)

// SimulatedChain represents a simulated chain with a backend and a list of signers.
type SimulatedChain struct {
	Backend *simulated.Backend
	Signers []*Signer
}

// Signer represents a signer with a private key.
type Signer struct {
	PrivateKey *ecdsa.PrivateKey
}

// NewTransactOpts creates a new transact options with the signer's private key and sets default
// values.
func (s *Signer) NewTransactOpts(t *testing.T) *bind.TransactOpts {
	t.Helper()

	auth, err := bind.NewKeyedTransactorWithChainID(s.PrivateKey, big.NewInt(SimulatedChainID))
	require.NoError(t, err)

	// Set default values
	auth.GasLimit = DefaultGasLimit

	return auth
}

// Address extracts the address from the signer's private key.
func (s *Signer) Address(t *testing.T) common.Address {
	t.Helper()

	publicKeyECDSA, ok := s.PrivateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("error casting public key from crypto to ecdsa")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

// NewSimulatedChain creates a new simulated chain with the given number of signers.
func NewSimulatedChain(t *testing.T, numSigners uint64) SimulatedChain {
	t.Helper()

	// Generate a private key
	signers := make([]*Signer, 0, numSigners)
	for range numSigners {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)

		signers = append(signers, &Signer{PrivateKey: key})
	}

	// Setup the simulated backend
	genesisAlloc := gethTypes.GenesisAlloc{}
	for _, s := range signers {
		genesisAlloc[s.Address(t)] = gethTypes.Account{
			Balance: big.NewInt(DefaultBalance),
		}
	}

	sim := simulated.NewBackend(genesisAlloc,
		simulated.WithBlockGasLimit(DefaultGasLimit),
	)

	return SimulatedChain{
		Backend: sim,
		Signers: signers,
	}
}

// DeployMCMContract Deploy a ManyChainMultiSig contract with the signer
func (s *SimulatedChain) DeployMCMContract(
	t *testing.T, signer *Signer,
) (*bindings.ManyChainMultiSig, *gethTypes.Transaction) {
	t.Helper()

	// Deploy a ManyChainMultiSig contract with the signer
	_, tx, contract, err := bindings.DeployManyChainMultiSig(signer.NewTransactOpts(t), s.Backend.Client())
	require.NoError(t, err)

	// Mine a block
	s.Backend.Commit()

	return contract, tx
}

// Set the MCMS contract configuration. The signers of the contract are set to all available signers in the MCMS config
func (s *SimulatedChain) SetMCMSConfig(
	t *testing.T, signer *Signer, contract *bindings.ManyChainMultiSig,
) *gethTypes.Transaction {
	t.Helper()

	signers := make([]common.Address, 0, len(s.Signers))
	for _, s := range s.Signers {
		signers = append(signers, s.Address(t))
	}

	quorom, err := safecast.IntToUint8(len(s.Signers)) // Quorum is set to the number of signers
	require.NoError(t, err)

	cfg := &types.Config{
		Quorum:       quorom,
		Signers:      signers,
		GroupSigners: []types.Config{},
	}

	transformer := evm.ConfigTransformer{}
	bindConfig, err := transformer.ToChainConfig(*cfg, nil)
	require.NoError(t, err)

	signerAddrs := make([]common.Address, len(bindConfig.Signers))
	signerGroups := make([]uint8, len(bindConfig.Signers))
	for i, signer := range bindConfig.Signers {
		signerAddrs[i] = signer.Addr
		signerGroups[i] = signer.Group
	}

	tx, err := contract.SetConfig(
		signer.NewTransactOpts(t), signerAddrs, signerGroups, bindConfig.GroupQuorums, bindConfig.GroupParents, false,
	)
	require.NoError(t, err)

	// Mine a block
	s.Backend.Commit()

	return tx
}

// Deploy a RBACTimelock contract with the signer. The admin addr is usually set to the MCMS contract address.
func (s *SimulatedChain) DeployRBACTimelock(
	t *testing.T,
	signer *Signer,
	adminAddr common.Address,
	proposers []common.Address,
	executors []common.Address,
	bypassers []common.Address,
	cancellers []common.Address,
) (*bindings.RBACTimelock, *gethTypes.Transaction) {
	t.Helper()

	_, tx, contract, err := bindings.DeployRBACTimelock(
		signer.NewTransactOpts(t),
		s.Backend.Client(),
		big.NewInt(0),
		adminAddr,
		proposers,
		executors,
		cancellers,
		bypassers,
	)
	require.NoError(t, err)

	// Mine a block
	s.Backend.Commit()

	return contract, tx
}
