package sui

import (
	"crypto/ecdsa"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/stretchr/testify/require"
)

type RoleConfig struct {
	Count  int
	Quorum uint8
	Keys   []*ecdsa.PrivateKey
	Config *types.Config
}

func CreateConfig(count int, quorum uint8) *RoleConfig {
	signers := make([]common.Address, count)
	signerKeys := make([]*ecdsa.PrivateKey, count)

	for i := range signers {
		signerKeys[i], _ = crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(signerKeys[i].PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})

	return &RoleConfig{
		Count:  count,
		Quorum: quorum,
		Keys:   signerKeys,
		Config: &types.Config{
			Quorum:  quorum,
			Signers: signers[:],
		},
	}
}

type ProposalBuilderConfig struct {
	Version            string
	Description        string
	ChainSelector      types.ChainSelector
	McmsObjID          string
	TimelockObjID      string
	AccountObjID       string
	RegistryObjID      string
	DeployerStateObjID string
	McmsPackageID      string
	Role               suisdk.TimelockRole
	CurrentOpCount     uint64
	Action             types.TimelockAction
	Delay              *types.Duration
}

func CreateTimelockProposalBuilder(t *testing.T, config ProposalBuilderConfig, operations []types.BatchOperation) *mcms.TimelockProposalBuilder {
	t.Helper()
	validUntilMs := uint32(time.Now().Add(time.Hour * 24).Unix())

	metadata, err := suisdk.NewChainMetadata(config.CurrentOpCount, config.Role, config.McmsPackageID, config.McmsObjID, config.AccountObjID, config.RegistryObjID, config.TimelockObjID, config.DeployerStateObjID)
	require.NoError(t, err)

	builder := mcms.NewTimelockProposalBuilder().
		SetVersion(config.Version).
		SetValidUntil(validUntilMs).
		SetDescription(config.Description).
		AddTimelockAddress(config.ChainSelector, config.TimelockObjID).
		AddChainMetadata(config.ChainSelector, metadata)

	for _, op := range operations {
		builder.AddOperation(op)
	}

	builder.SetAction(config.Action)
	if config.Delay != nil {
		builder.SetDelay(*config.Delay)
	}

	return builder
}

func SignProposal(proposal *mcms.Proposal, inspectorsMap map[types.ChainSelector]sdk.Inspector, keys []*ecdsa.PrivateKey, quorum int) (*mcms.Signable, error) {
	signable, err := mcms.NewSignable(proposal, inspectorsMap)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(keys) && i < quorum; i++ {
		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(keys[i]))
		if err != nil {
			return nil, err
		}
	}

	return signable, nil
}
