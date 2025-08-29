package sui

import (
	"crypto/ecdsa"
	"encoding/json"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

type ProposerConfig struct {
	Count  int
	Quorum uint8
	Keys   []*ecdsa.PrivateKey
	Config *types.Config
}
type BypasserConfig struct {
	Count  int
	Quorum uint8
	Keys   []*ecdsa.PrivateKey
	Config *types.Config
}

func CreateProposerConfig(count int, quorum uint8) *ProposerConfig {
	proposers := make([]common.Address, count)
	proposerKeys := make([]*ecdsa.PrivateKey, count)

	for i := range proposers {
		proposerKeys[i], _ = crypto.GenerateKey()
		proposers[i] = crypto.PubkeyToAddress(proposerKeys[i].PublicKey)
	}
	slices.SortFunc(proposers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})

	return &ProposerConfig{
		Count:  count,
		Quorum: quorum,
		Keys:   proposerKeys,
		Config: &types.Config{
			Quorum:  quorum,
			Signers: proposers[:],
		},
	}
}

func CreateBypasserConfig(count int, quorum uint8) *BypasserConfig {
	bypassers := make([]common.Address, count)
	bypasserKeys := make([]*ecdsa.PrivateKey, count)

	for i := range bypassers {
		bypasserKeys[i], _ = crypto.GenerateKey()
		bypassers[i] = crypto.PubkeyToAddress(bypasserKeys[i].PublicKey)
	}
	slices.SortFunc(bypassers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})

	return &BypasserConfig{
		Count:  count,
		Quorum: quorum,
		Keys:   bypasserKeys,
		Config: &types.Config{
			Quorum:  quorum,
			Signers: bypassers[:],
		},
	}
}

type ProposalBuilderConfig struct {
	Version        string
	Description    string
	ChainSelector  types.ChainSelector
	MCMSPackageId  string
	Role           suisdk.TimelockRole
	CurrentOpCount uint64
	Action         types.TimelockAction
	Delay          *types.Duration
}

func CreateTimelockProposalBuilder(config ProposalBuilderConfig, operations []types.BatchOperation) *mcms.TimelockProposalBuilder {
	validUntilMs := uint32(time.Now().Add(time.Hour * 24).Unix())

	builder := mcms.NewTimelockProposalBuilder().
		SetVersion(config.Version).
		SetValidUntil(validUntilMs).
		SetDescription(config.Description).
		AddTimelockAddress(config.ChainSelector, config.MCMSPackageId).
		AddChainMetadata(config.ChainSelector, types.ChainMetadata{
			StartingOpCount:  config.CurrentOpCount,
			MCMAddress:       config.MCMSPackageId,
			AdditionalFields: mustMarshal(suisdk.AdditionalFieldsMetadata{Role: config.Role}),
		})

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

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return data
}
