package stellar

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/suite"

	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"
	stellardeployer "github.com/smartcontractkit/chainlink-stellar/deployment"
	"github.com/smartcontractkit/chainlink-stellar/deployment/cre"
	stellarmcmsutil "github.com/smartcontractkit/chainlink-stellar/deployment/mcmsutil"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

type ChainMeta struct {
	MCMAddress      string
	TimelockAddress string
}

type timelockRoleConfig struct {
	Proposers  []string
	Cancellers []string
	Bypassers  []string
}

type proposalSigner struct {
	key     *ecdsa.PrivateKey
	address common.Address
}

type ExecutionTestSuite struct {
	suite.Suite
	e2e.TestSetup

	deployer      *stellardeployer.Deployer
	chainSelector mcmtypes.ChainSelector
	passphrase    string

	proposalSignerKeys []*ecdsa.PrivateKey
	signerAddresses    []common.Address
	mcmsConfig         *mcmtypes.Config

	deploymentCounter uint64
}

func (s *ExecutionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	s.Require().NotNil(
		s.StellarClient,
		"Stellar RPC client is not configured",
	)
	s.Require().NotNil(
		s.StellarSigner,
		"Stellar transaction signer is not configured",
	)

	s.chainSelector = mcmtypes.ChainSelector(
		chainsel.STELLAR_LOCALNET.Selector,
	)
	s.passphrase = chainsel.STELLAR_LOCALNET.Passphrase

	s.deployer = stellardeployer.NewDeployer(
		s.StellarClient,
		s.passphrase,
		s.StellarSigner,
	)

	s.initializeProposalSigners()

	// Group 0 contains signer 0.
	// Group 1 contains signer 1 and is a child of group 0.
	s.mcmsConfig = &mcmtypes.Config{
		Quorum:  1,
		Signers: []common.Address{s.signerAddresses[0]},
		GroupSigners: []mcmtypes.Config{
			{
				Quorum: 1,
				Signers: []common.Address{
					s.signerAddresses[1],
				},
			},
		},
	}
}

func (s *ExecutionTestSuite) initializeProposalSigners() {
	s.T().Helper()

	const signerCount = 2

	signers := make([]proposalSigner, signerCount)

	for i := 0; i < signerCount; i++ {
		key, err := crypto.GenerateKey()
		s.Require().NoError(
			err,
			"failed to generate Stellar MCMS proposal signer",
		)

		signers[i] = proposalSigner{
			key:     key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
	}

	// The Stellar MCMS contract requires signer addresses to be strictly
	// increasing. Keep each private key paired with its sorted address.
	sort.Slice(signers, func(i, j int) bool {
		return bytes.Compare(
			signers[i].address.Bytes(),
			signers[j].address.Bytes(),
		) < 0
	})

	s.proposalSignerKeys = make(
		[]*ecdsa.PrivateKey,
		signerCount,
	)
	s.signerAddresses = make(
		[]common.Address,
		signerCount,
	)

	for i, signer := range signers {
		s.proposalSignerKeys[i] = signer.key
		s.signerAddresses[i] = signer.address
	}
}

func (s *ExecutionTestSuite) newInspector() *stellarsdk.Inspector {
	s.T().Helper()

	inspector, err :=
		stellarsdk.NewInspectorWithNetworkPassphrase(
			s.StellarClient,
			s.StellarSigner,
			s.passphrase,
		)
	s.Require().NoError(
		err,
		"failed to create Stellar inspector",
	)

	return inspector
}

func (s *ExecutionTestSuite) deployChain() ChainMeta {
	s.T().Helper()

	deploymentID := s.nextDeploymentID()

	mcmAddress := s.deployMCMSContract(deploymentID)

	roles := s.defaultTimelockRoleConfig(
		mcmAddress,
		s.StellarSigner.Address(),
	)

	timelockAddress := s.deployTimelockContract(
		deploymentID,
		roles,
	)

	return ChainMeta{
		MCMAddress:      mcmAddress,
		TimelockAddress: timelockAddress,
	}
}

func (s *ExecutionTestSuite) deployMCMSContract(
	deploymentID uint64,
) string {
	s.T().Helper()

	networkIDHex, err := chainsel.StellarChainIdFromSelector(
		uint64(s.chainSelector),
	)
	s.Require().NoError(err)
	s.Require().True(
		common.IsHexHash(networkIDHex),
		"invalid Stellar network ID %q",
		networkIDHex,
	)

	instanceLabel := fmt.Sprintf("e2e_%d", deploymentID)
	s.Require().LessOrEqual(
		len(instanceLabel),
		32,
		"Soroban instance label exceeds symbol length",
	)

	contractID, err := stellarmcmsutil.DeployMCMS(
		s.T().Context(),
		s.deployer,
		s.StellarSigner.Address(),
		common.HexToHash(networkIDHex),
		s.mcmsConfig,
		instanceLabel,
		stellarmcmsutil.MCMSDeploySalt(
			uint64(s.chainSelector),
			instanceLabel,
		),
	)
	s.Require().NoError(
		err,
		"failed to deploy Stellar MCMS",
	)

	return contractID
}

func (s *ExecutionTestSuite) deployTimelockContract(
	deploymentID uint64,
	roles timelockRoleConfig,
) string {
	s.T().Helper()

	wasm, err := cre.Artifact(cre.TimelockWasm)
	s.Require().NoError(
		err,
		"failed to load Stellar timelock WASM",
	)

	salt := [32]byte(
		crypto.Keccak256Hash(
			[]byte(
				fmt.Sprintf(
					"stellar_execution_timelock_%d",
					deploymentID,
				),
			),
		),
	)

	contractID, err := s.deployer.DeployContractBytes(
		s.T().Context(),
		wasm,
		salt,
	)
	s.Require().NoError(
		err,
		"failed to deploy Stellar timelock",
	)

	client := timelockbindings.NewTimelockClient(
		s.deployer,
		contractID,
	)

	err = client.Initialize(
		s.T().Context(),
		0,
		roles.Proposers,
		roles.Cancellers,
		roles.Bypassers,
	)
	s.Require().NoError(
		err,
		"failed to initialize Stellar timelock",
	)

	return contractID
}

func (s *ExecutionTestSuite) defaultTimelockRoleConfig(
	mcmAddress string,
	operator string,
) timelockRoleConfig {
	s.T().Helper()

	base := uniqueStrings([]string{
		mcmAddress,
		operator,
	})

	return timelockRoleConfig{
		Proposers:  append([]string(nil), base...),
		Cancellers: append([]string(nil), base...),
		Bypassers:  append([]string(nil), base...),
	}
}

func (s *ExecutionTestSuite) transferMCMSOwnershipToTimelock(
	chain ChainMeta,
) {
	s.T().Helper()

	configurer := stellarsdk.NewConfigurer(s.deployer)

	_, err := configurer.TransferOwnership(
		s.T().Context(),
		chain.MCMAddress,
		chain.TimelockAddress,
	)
	s.Require().NoError(
		err,
		"failed to transfer MCMS ownership to timelock",
	)
}

func (s *ExecutionTestSuite) acceptMCMSOwnershipThroughTimelock(
	chain ChainMeta,
) {
	s.T().Helper()

	encodedPayload := s.mustEncodeInvokePayload(
		"accept_ownership",
		nil,
	)

	payload, err := stellarsdk.DecodeSorobanInvokePayload(
		encodedPayload,
	)
	s.Require().NoError(err)

	calls := timelockbindings.Calls{
		Inner: []timelockbindings.Call{
			{
				Target:   chain.MCMAddress,
				Function: "accept_ownership",
				ArgsXdr:  payload.ArgsXDR,
			},
		},
	}

	timelockClient := timelockbindings.NewTimelockClient(
		s.deployer,
		chain.TimelockAddress,
	)

	err = timelockClient.BypasserExecuteBatch(
		s.T().Context(),
		s.StellarSigner.Address(),
		calls,
	)
	s.Require().NoError(
		err,
		"timelock failed to accept MCMS ownership",
	)

	mcmsClient := mcmsbindings.NewMcmsClient(
		s.deployer,
		chain.MCMAddress,
	)

	owner, err := mcmsClient.Owner(s.T().Context())
	s.Require().NoError(err)
	s.Require().NotNil(owner)
	s.Require().Equal(
		chain.TimelockAddress,
		*owner,
		"timelock should own MCMS",
	)
}

func (s *ExecutionTestSuite) mustEncodeInvokePayload(
	function string,
	args []xdr.ScVal,
) []byte {
	s.T().Helper()

	payload, err := stellarsdk.EncodeSorobanInvokePayload(
		function,
		args,
	)
	s.Require().NoError(err)

	return payload
}

func (s *ExecutionTestSuite) nextDeploymentID() uint64 {
	s.deploymentCounter++

	return s.deploymentCounter
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
