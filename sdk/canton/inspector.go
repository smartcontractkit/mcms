package canton

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	mcmsapi "github.com/smartcontractkit/chainlink-canton/bindings/generated/mcms/api"
	"github.com/smartcontractkit/chainlink-canton/contracts"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	stateClient apiv2.StateServiceClient
	// The parties that own the MCMS deployment.
	mcmsParties []string
	role        TimelockRole
}

func NewInspector(stateClient apiv2.StateServiceClient, mcmsParties []string, role TimelockRole) *Inspector {
	return &Inspector{
		stateClient: stateClient,
		mcmsParties: mcmsParties,
		role:        role,
	}
}

// StateServiceClient returns the state service client for resolution (e.g. InstanceAddress to contract ID).
func (i *Inspector) StateServiceClient() apiv2.StateServiceClient {
	return i.stateClient
}

func (i *Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	mcmsContract, err := GetMCMSContract(ctx, i.stateClient, i.mcmsParties, mcmsAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	switch i.role {
	case TimelockRoleProposer:
		return toConfig(mcmsContract.Proposer.Config)
	case TimelockRoleBypasser:
		return toConfig(mcmsContract.Bypasser.Config)
	case TimelockRoleCanceller:
		return toConfig(mcmsContract.Canceller.Config)
	default:
		return nil, fmt.Errorf("unknown timelock role: %s", i.role)
	}
}

func (i *Inspector) GetOpCount(ctx context.Context, mcmsAddr string) (uint64, error) {
	mcmsContract, err := GetMCMSContract(ctx, i.stateClient, i.mcmsParties, mcmsAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	switch i.role {
	case TimelockRoleProposer:
		return safecast.Int64ToUint64(int64(mcmsContract.Proposer.ExpiringRoot.OpCount))
	case TimelockRoleBypasser:
		return safecast.Int64ToUint64(int64(mcmsContract.Bypasser.ExpiringRoot.OpCount))
	case TimelockRoleCanceller:
		return safecast.Int64ToUint64(int64(mcmsContract.Canceller.ExpiringRoot.OpCount))
	default:
		return 0, fmt.Errorf("unknown timelock role: %s", i.role)
	}
}

func (i *Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	mcmsContract, err := GetMCMSContract(ctx, i.stateClient, i.mcmsParties, mcmsAddr)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	var expiringRoot mcmsapi.ExpiringRoot
	switch i.role {
	case TimelockRoleProposer:
		expiringRoot = mcmsContract.Proposer.ExpiringRoot
	case TimelockRoleBypasser:
		expiringRoot = mcmsContract.Bypasser.ExpiringRoot
	case TimelockRoleCanceller:
		expiringRoot = mcmsContract.Canceller.ExpiringRoot
	default:
		return common.Hash{}, 0, fmt.Errorf("unknown timelock role: %s", i.role)
	}

	// Parse the root from hex string
	rootStr := string(expiringRoot.Root)
	rootStr = strings.TrimPrefix(rootStr, "0x")
	rootBytes, err := hex.DecodeString(rootStr)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to decode root hash: %w", err)
	}

	root := common.BytesToHash(rootBytes)

	// validUntil is a TIMESTAMP (which wraps time.Time)
	// Convert to Unix timestamp (uint32)
	timeVal := time.Time(expiringRoot.ValidUntil)
	validUntil, err := safecast.Int64ToUint32(timeVal.Unix())
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("valid until out of range: %w", err)
	}

	return root, validUntil, nil
}

func (i *Inspector) GetRootMetadata(ctx context.Context, mcmsAddr string) (types.ChainMetadata, error) {
	mcmsContract, err := GetMCMSContract(ctx, i.stateClient, i.mcmsParties, mcmsAddr)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	var rootMetadata mcmsapi.RootMetadata
	switch i.role {
	case TimelockRoleProposer:
		rootMetadata = mcmsContract.Proposer.RootMetadata
	case TimelockRoleBypasser:
		rootMetadata = mcmsContract.Bypasser.RootMetadata
	case TimelockRoleCanceller:
		rootMetadata = mcmsContract.Canceller.RootMetadata
	default:
		return types.ChainMetadata{}, fmt.Errorf("unknown timelock role: %s", i.role)
	}

	// For Canton, MCMAddress is the InstanceAddress hex (stable across SetRoot/ExecuteOp)
	mcmAddress := contracts.InstanceID(string(mcmsContract.InstanceId)).RawInstanceAddress(mcmsContract.Owner).InstanceAddress().Hex()
	startingOpCount, castErr := safecast.Int64ToUint64(int64(rootMetadata.PreOpCount))
	if castErr != nil {
		return types.ChainMetadata{}, fmt.Errorf("pre op count out of range: %w", castErr)
	}

	return types.ChainMetadata{
		StartingOpCount: startingOpCount,
		MCMAddress:      mcmAddress,
	}, nil
}

// toConfig converts a Canton MultisigConfig to the chain-agnostic types.Config
func toConfig(bindConfig mcmsapi.MultisigConfig) (*types.Config, error) {
	// Group signers by group index
	signersByGroup := make([][]common.Address, maxMCMSGroups)

	for _, signer := range bindConfig.Signers {
		groupIdx := int(signer.SignerGroup)
		if groupIdx < 0 || groupIdx >= maxMCMSGroups {
			return nil, fmt.Errorf("signer group index %d out of range [0, %d)", groupIdx, maxMCMSGroups)
		}

		// Parse signer address
		addr := common.HexToAddress(string(signer.SignerAddress))
		signersByGroup[groupIdx] = append(signersByGroup[groupIdx], addr)
	}

	// Build the group configs
	groups := make([]types.Config, maxMCMSGroups)
	for i := range maxMCMSGroups {
		signers := signersByGroup[i]
		if signers == nil {
			signers = []common.Address{}
		}

		quorum := uint8(0)
		if i < len(bindConfig.GroupQuorums) {
			q, convErr := safecast.IntToUint8(int(bindConfig.GroupQuorums[i]))
			if convErr != nil {
				return nil, fmt.Errorf("group quorum for group %d: %w", i, convErr)
			}
			quorum = q
		}

		groups[i] = types.Config{
			Signers:      signers,
			GroupSigners: []types.Config{},
			Quorum:       quorum,
		}
	}

	// Link the group signers; this assumes a group's parent always has a lower index
	// Process in reverse order to build the tree from leaves to root
	for i := maxMCMSGroups - 1; i >= 0; i-- {
		parent := uint8(0)
		if i < len(bindConfig.GroupParents) {
			p, convErr := safecast.IntToUint8(int(bindConfig.GroupParents[i]))
			if convErr != nil {
				return nil, fmt.Errorf("group parent for group %d: %w", i, convErr)
			}
			parent = p
		}

		// Add non-empty child groups to their parent
		// Skip the root group (i == 0) and empty groups (quorum == 0)
		if i > 0 && groups[i].Quorum > 0 {
			if int(parent) >= len(groups) {
				return nil, fmt.Errorf("group parent index %d out of range", parent)
			}
			groups[parent].GroupSigners = append([]types.Config{groups[i]}, groups[parent].GroupSigners...)
		}
	}

	// Validate the root group config
	if err := groups[0].Validate(); err != nil {
		return nil, fmt.Errorf("invalid MCMS config: %w", err)
	}

	return &groups[0], nil
}
