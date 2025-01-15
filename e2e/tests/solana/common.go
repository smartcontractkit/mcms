//go:build e2e
// +build e2e

package solanae2e

import (
	"crypto/ecdsa"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/types"
)

// -----------------------------------------------------------------------------
// Constants and globals

// this key matches the public key in the config.toml so it gets funded by the genesis block
const privateKey = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

var testPDASeed = [32]byte{'t', 'e', 's', 't', '-', 'm', 'c', 'm'}

// -----------------------------------------------------------------------------
// EVMTestAccount is data type wrapping attributes typically needed when managing
// ethereum accounts.
type EVMTestAccount struct {
	Address       common.Address
	HexAddress    string
	PrivateKey    *ecdsa.PrivateKey
	HexPrivateKey string
}

// NewTestAccount generates a new ecdsa key and returns a TestAccount structure
// with the associated attributes.
func NewEVMTestAccount(t *testing.T) EVMTestAccount {
	t.Helper()

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	publicKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	require.True(t, ok)

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return EVMTestAccount{
		Address:       address,
		HexAddress:    address.Hex(),
		PrivateKey:    privateKey,
		HexPrivateKey: hexutil.Encode(crypto.FromECDSA(privateKey))[2:],
	}
}

func generateTestEVMAccounts(t *testing.T, numAccounts int) []EVMTestAccount {
	t.Helper()

	testAccounts := make([]EVMTestAccount, numAccounts)
	for i := range testAccounts {
		testAccounts[i] = NewEVMTestAccount(t)
	}

	slices.SortFunc(testAccounts, func(a, b EVMTestAccount) int {
		return strings.Compare(strings.ToLower(a.HexAddress), strings.ToLower(b.HexAddress))
	})

	return testAccounts
}

// -----------------------------------------------------------------------------
// SolanaTestSuite
type SolanaTestSuite struct {
	suite.Suite
	e2e.TestSetup

	ChainSelector types.ChainSelector
	MCMProgramID  solana.PublicKey
}

// SetupSuite runs before the test suite
func (s *SolanaTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
	s.MCMProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])

	details, err := cselectors.GetChainDetailsByChainIDAndFamily(s.SolanaChain.ChainID, cselectors.FamilySolana)
	s.Require().NoError(err)
	s.ChainSelector = types.ChainSelector(details.ChainSelector)

	s.SetupMCM()
}
