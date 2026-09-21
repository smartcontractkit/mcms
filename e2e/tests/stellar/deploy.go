package stellar

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	chainsel "github.com/smartcontractkit/chain-selectors"
	stellarbindings "github.com/smartcontractkit/chainlink-stellar/bindings"
	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"
	stellardeployer "github.com/smartcontractkit/chainlink-stellar/deployment"
	"github.com/smartcontractkit/chainlink-stellar/deployment/cre"
	stellarmcmsutil "github.com/smartcontractkit/chainlink-stellar/deployment/mcmsutil"
	stellarrpc "github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stretchr/testify/require"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// stellarChainEnv holds the shared on-chain bootstrap every Stellar e2e suite
// needs: a funded test account, its signer, and a deployer bound to the
// configured local network.
type stellarChainEnv struct {
	testSetup     e2e.TestSetup
	signer        stellarbindings.Signer
	deployer      *stellardeployer.Deployer
	chainSelector mcmtypes.ChainSelector
	passphrase    string
}

// bootstrapStellarChainEnv funds a fresh Stellar test account and builds the
// deployer used by every Stellar e2e suite.
func bootstrapStellarChainEnv(t *testing.T) stellarChainEnv {
	t.Helper()

	testSetup := *e2e.InitializeSharedTestSetup(t)

	require.NotNil(t, testSetup.StellarClient, "Stellar RPC client is not configured")
	require.NotNil(t, testSetup.StellarChain, "Stellar chain is not configured")
	require.NotNil(t, testSetup.StellarChain.Out, "Stellar chain output is not configured")
	require.NotNil(t, testSetup.StellarChain.Out.NetworkSpecificData, "Stellar network-specific data is not configured")
	require.NotNil(t, testSetup.StellarChain.Out.NetworkSpecificData.StellarNetwork, "Stellar network data is not configured")

	friendbotURL := testSetup.StellarChain.Out.NetworkSpecificData.StellarNetwork.FriendbotURL
	require.NotEmpty(t, friendbotURL, "Stellar Friendbot URL is empty")

	kp, err := keypair.Random()
	require.NoError(t, err, "Failed to generate Stellar test account")

	FundStellarKey(t, friendbotURL, kp)

	t.Logf("Funded Stellar test account %s", kp.Address())

	signer := stellarbindings.NewStellarKeypairSigner(kp)
	passphrase := chainsel.STELLAR_LOCALNET.Passphrase

	deployer := stellardeployer.NewDeployer(
		testSetup.StellarClient,
		passphrase,
		signer.KeypairFull(),
	)

	return stellarChainEnv{
		testSetup:     testSetup,
		signer:        signer,
		deployer:      deployer,
		chainSelector: mcmtypes.ChainSelector(chainsel.STELLAR_LOCALNET.Selector),
		passphrase:    passphrase,
	}
}

// newStellarInspector builds a read-side MCMS inspector against the shared
// RPC client.
func newStellarInspector(
	t *testing.T,
	client *stellarrpc.Client,
	signer stellarbindings.Signer,
	passphrase string,
) *stellarsdk.Inspector {
	t.Helper()

	inspector, err := stellarsdk.NewInspectorWithNetworkPassphrase(client, signer, passphrase)
	require.NoError(t, err, "failed to create Stellar inspector")

	return inspector
}

// deployStellarMCMSContract deploys and initializes an MCMS instance. The
// instance label must be unique across suites: it feeds the deploy salt.
func deployStellarMCMSContract(
	t *testing.T,
	deployer *stellardeployer.Deployer,
	owner string,
	chainSelector mcmtypes.ChainSelector,
	config *mcmtypes.Config,
	instanceLabel string,
) string {
	t.Helper()

	networkIDHex, err := chainsel.StellarChainIdFromSelector(uint64(chainSelector))
	require.NoError(t, err)
	require.True(t, common.IsHexHash(networkIDHex), "invalid Stellar network ID %q", networkIDHex)

	require.LessOrEqual(t, len(instanceLabel), 32, "Soroban instance label exceeds symbol length")

	contractID, err := stellarmcmsutil.DeployMCMS(
		t.Context(),
		deployer,
		owner,
		common.HexToHash(networkIDHex),
		config,
		instanceLabel,
		stellarmcmsutil.TimelockDeploySalt(uint64(chainSelector), instanceLabel),
	)
	require.NoError(t, err, "failed to deploy Stellar MCMS")

	return contractID
}

// deployStellarTimelockContract deploys and initializes an RBAC timelock with
// zero min delay. saltPrefix keeps deployment salts unique across suites.
func deployStellarTimelockContract(
	t *testing.T,
	deployer *stellardeployer.Deployer,
	saltPrefix string,
	deploymentID uint64,
	roles timelockRoleConfig,
) string {
	t.Helper()

	wasm, err := cre.Artifact(cre.TimelockWasm)
	require.NoError(t, err, "failed to load Stellar timelock WASM")

	salt := [32]byte(crypto.Keccak256Hash(
		[]byte(fmt.Sprintf("stellar_%s_timelock_%d", saltPrefix, deploymentID)),
	))

	contractID, err := deployer.DeployContractBytes(t.Context(), wasm, salt)
	require.NoError(t, err, "failed to deploy Stellar timelock")

	client := timelockbindings.NewTimelockClient(deployer, contractID)

	err = client.Initialize(t.Context(), 0, roles.Proposers, roles.Cancellers, roles.Bypassers)
	require.NoError(t, err, "failed to initialize Stellar timelock")

	return contractID
}

// defaultTimelockRoleConfig grants PROPOSER, CANCELLER, and BYPASSER to the
// MCMS instance and the test operator. The timelock defines no executor role:
// execute_batch is permissionless once an operation is ready.
func defaultTimelockRoleConfig(mcmAddress, operator string) timelockRoleConfig {
	base := uniqueStrings([]string{mcmAddress, operator})

	return timelockRoleConfig{
		Proposers:  append([]string(nil), base...),
		Cancellers: append([]string(nil), base...),
		Bypassers:  append([]string(nil), base...),
	}
}

// generateSortedSigners generates count ECDSA signers with strictly
// increasing addresses, as required by the Stellar MCMS contract.
func generateSortedSigners(t *testing.T, count int) ([]*ecdsa.PrivateKey, []common.Address) {
	t.Helper()

	signers := make([]struct {
		key     *ecdsa.PrivateKey
		address common.Address
	}, count)

	for i := range signers {
		key, err := crypto.GenerateKey()
		require.NoError(t, err, "failed to generate Stellar MCMS proposal signer")

		signers[i].key = key
		signers[i].address = crypto.PubkeyToAddress(key.PublicKey)
	}

	sort.Slice(signers, func(i, j int) bool {
		return bytes.Compare(signers[i].address.Bytes(), signers[j].address.Bytes()) < 0
	})

	keys := make([]*ecdsa.PrivateKey, count)
	addresses := make([]common.Address, count)
	for i, signer := range signers {
		keys[i] = signer.key
		addresses[i] = signer.address
	}

	return keys, addresses
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
