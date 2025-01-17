//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"crypto/ecdsa"
	"slices"
	"strings"
	"testing"
	"time"

	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

// -----------------------------------------------------------------------------
// Constants and globals

// this key matches the public key in the config.toml so it gets funded by the genesis block
const privateKey = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

// EVMTestAccount is data type wrapping attributes typically needed when managing
// ethereum accounts.
type EVMTestAccount struct {
	Address       evmCommon.Address
	HexAddress    string
	PrivateKey    *ecdsa.PrivateKey
	HexPrivateKey string
}

// NewEVMTestAccount generates a new ecdsa key and returns a TestAccount structure
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

// generateTestEVMAccounts generates a slice of EVMTestAccount structures
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

// SolanaTestSuite is the base test suite for all solana e2e tests
type SolanaTestSuite struct {
	suite.Suite
	e2e.TestSetup

	ChainSelector types.ChainSelector
	MCMProgramID  solana.PublicKey
}

// SetupMCM initializes the MCM account with the given PDA seed
func (s *SolanaTestSuite) SetupMCM(pdaSeed [32]byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	bindings.SetProgramID(s.MCMProgramID)

	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	configPDA, err := solanasdk.FindConfigPDA(s.MCMProgramID, pdaSeed)
	s.Require().NoError(err)
	rootMetadataPDA, err := solanasdk.FindRootMetadataPDA(s.MCMProgramID, pdaSeed)
	s.Require().NoError(err)
	expiringRootAndOpCountPDA, err := solanasdk.FindExpiringRootAndOpCountPDA(s.MCMProgramID, pdaSeed)
	s.Require().NoError(err)

	// get program data account
	data, err := s.SolanaClient.GetAccountInfoWithOpts(ctx, s.MCMProgramID, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentConfirmed,
	})
	s.Require().NoError(err)

	// decode program data
	var programData struct {
		DataType uint32
		Address  solana.PublicKey
	}
	err = bin.UnmarshalBorsh(&programData, data.Bytes())
	s.Require().NoError(err)

	ix, err := bindings.NewInitializeInstruction(
		uint64(s.ChainSelector),
		pdaSeed,
		configPDA,
		wallet.PublicKey(),
		solana.SystemProgramID,
		s.MCMProgramID,
		programData.Address,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
	).ValidateAndBuild()
	s.Require().NoError(err)

	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, wallet, rpc.CommitmentConfirmed)

	// get config and validate
	var configAccount bindings.MultisigConfig
	err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, configPDA, rpc.CommitmentConfirmed, &configAccount)
	s.Require().NoError(err, "failed to get account data")

	s.Require().Equal(uint64(s.ChainSelector), configAccount.ChainId)
	s.Require().Equal(wallet.PublicKey(), configAccount.Owner)
}

// SetupSuite runs before the test suite
func (s *SolanaTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
	s.MCMProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])
	details, err := cselectors.GetChainDetailsByChainIDAndFamily(s.SolanaChain.ChainID, cselectors.FamilySolana)
	s.Require().NoError(err)
	s.ChainSelector = types.ChainSelector(details.ChainSelector)
}
