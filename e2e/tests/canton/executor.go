//go:build e2e

package canton

import (
	"crypto/ecdsa"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	mcmscore "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// mcmsExecutorSetup holds shared setup (MCMS + config + counter + signers) for suites that need it.
// It has no Test* methods, so embedding it only adds SetupSuite and fields; test methods come from the embedding suite.
type mcmsExecutorSetup struct {
	TestSuite

	// Test signers
	signers       []*ecdsa.PrivateKey
	signerAddrs   []common.Address
	sortedSigners []*ecdsa.PrivateKey
	sortedWallets []*mcmscore.PrivateKeySigner

	// Counter contract for testing ExecuteOp
	counterInstanceID string
	counterCID        string
}

// SetupSuite runs before the test suite.
func (s *mcmsExecutorSetup) SetupSuite() {
	s.TestSuite.SetupSuite()

	// Create 3 signers for 2-of-3 multisig
	s.signers = make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < 3; i++ {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.signers[i] = key
	}

	// Sort signers by address
	signersCopy := make([]*ecdsa.PrivateKey, len(s.signers))
	copy(signersCopy, s.signers)
	slices.SortFunc(signersCopy, func(a, b *ecdsa.PrivateKey) int {
		addrA := crypto.PubkeyToAddress(a.PublicKey)
		addrB := crypto.PubkeyToAddress(b.PublicKey)
		return addrA.Cmp(addrB)
	})
	s.sortedSigners = signersCopy
	s.sortedWallets = make([]*mcmscore.PrivateKeySigner, len(s.sortedSigners))

	// Derive sorted addresses from sorted signers to ensure they correspond
	s.signerAddrs = make([]common.Address, len(s.sortedSigners))
	for i, signer := range s.sortedSigners {
		s.sortedWallets[i] = mcmscore.NewPrivateKeySigner(signer)
		s.signerAddrs[i] = crypto.PubkeyToAddress(signer.PublicKey)
	}

	// Deploy MCMS with config
	config := s.create2of3Config()
	s.DeployMCMSWithConfig(config)

	// Deploy Counter contract for ExecuteOp tests
	s.DeployCounterContract()
}

func (s *mcmsExecutorSetup) create2of3Config() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.signerAddrs,
	}
}
