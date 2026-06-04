package canton

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	mcmsapi "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/mcms/api"
	mcmscore "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/mcms/core"
	"github.com/smartcontractkit/chainlink-canton/contracts"
	damltypes "github.com/smartcontractkit/go-daml/pkg/types"

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

	return expiringRootOpCount(mcmsContract, i.role)
}

func expiringRootOpCount(mcmsContract *mcmscore.MCMS, role TimelockRole) (uint64, error) {
	switch role {
	case TimelockRoleProposer:
		return safecast.Int64ToUint64(int64(mcmsContract.Proposer.ExpiringRoot.OpCount))
	case TimelockRoleBypasser:
		return safecast.Int64ToUint64(int64(mcmsContract.Bypasser.ExpiringRoot.OpCount))
	case TimelockRoleCanceller:
		return safecast.Int64ToUint64(int64(mcmsContract.Canceller.ExpiringRoot.OpCount))
	default:
		return 0, fmt.Errorf("unknown timelock role: %s", role)
	}
}

func (i *Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	mcmsContract, err := GetMCMSContract(ctx, i.stateClient, i.mcmsParties, mcmsAddr)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	expiringRoot, err := expiringRootForRole(mcmsContract, i.role)
	if err != nil {
		return common.Hash{}, 0, err
	}

	return rootFromExpiringRoot(expiringRoot)
}

func expiringRootForRole(mcmsContract *mcmscore.MCMS, role TimelockRole) (mcmsapi.ExpiringRoot, error) {
	switch role {
	case TimelockRoleProposer:
		return mcmsContract.Proposer.ExpiringRoot, nil
	case TimelockRoleBypasser:
		return mcmsContract.Bypasser.ExpiringRoot, nil
	case TimelockRoleCanceller:
		return mcmsContract.Canceller.ExpiringRoot, nil
	default:
		return mcmsapi.ExpiringRoot{}, fmt.Errorf("unknown timelock role: %s", role)
	}
}

func rootFromExpiringRoot(expiringRoot mcmsapi.ExpiringRoot) (common.Hash, uint32, error) {
	root := common.HexToHash(string(expiringRoot.Root))

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

	return chainMetadataFromMCMSContract(mcmsContract, i.role)
}

func rootMetadataForRole(mcmsContract *mcmscore.MCMS, role TimelockRole) (mcmsapi.RootMetadata, error) {
	switch role {
	case TimelockRoleProposer:
		return mcmsContract.Proposer.RootMetadata, nil
	case TimelockRoleBypasser:
		return mcmsContract.Bypasser.RootMetadata, nil
	case TimelockRoleCanceller:
		return mcmsContract.Canceller.RootMetadata, nil
	default:
		return mcmsapi.RootMetadata{}, fmt.Errorf("unknown timelock role: %s", role)
	}
}

// chainMetadataFromMCMSContract builds chain metadata from on-chain MCMS state.
// MCMAddress is the canonical InstanceAddress hex (from InstanceId + Owner), not the lookup key.
func chainMetadataFromMCMSContract(mcmsContract *mcmscore.MCMS, role TimelockRole) (types.ChainMetadata, error) {
	rootMetadata, err := rootMetadataForRole(mcmsContract, role)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	mcmAddress := contracts.InstanceID(string(mcmsContract.InstanceId)).RawInstanceAddress(mcmsContract.Owner).InstanceAddress().Hex()
	startingOpCount, err := expiringRootOpCount(mcmsContract, role)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	// Before the first SetRoot, role RootMetadata may still be zeroed while the MCMS
	// template field ChainId (and multisig id) are authoritative.
	chainID := int64(rootMetadata.ChainId)
	if chainID <= 0 {
		chainID = int64(mcmsContract.ChainId)
	}
	multisigID := string(rootMetadata.MultisigId)
	if multisigID == "" {
		multisigID = fmt.Sprintf(
			"%s@%s-%s",
			string(mcmsContract.InstanceId),
			string(mcmsContract.Owner),
			strings.ToLower(role.String()),
		)
	}

	additionalFields := AdditionalFieldsMetadata{
		ChainId:    chainID,
		MultisigId: multisigID,
		InstanceId: string(mcmsContract.InstanceId),
	}
	if validateErr := additionalFields.Validate(); validateErr != nil {
		return types.ChainMetadata{}, fmt.Errorf("invalid root metadata from ledger: %w", validateErr)
	}

	additionalFieldsBytes, err := json.Marshal(additionalFields)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("marshal canton additional fields: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount:  startingOpCount,
		MCMAddress:       mcmAddress,
		AdditionalFields: additionalFieldsBytes,
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

	if len(bindConfig.GroupQuorums) > maxMCMSGroups {
		return nil, fmt.Errorf("group quorums length %d exceeds maximum %d", len(bindConfig.GroupQuorums), maxMCMSGroups)
	}
	if len(bindConfig.GroupParents) > maxMCMSGroups {
		return nil, fmt.Errorf("group parents length %d exceeds maximum %d", len(bindConfig.GroupParents), maxMCMSGroups)
	}
	groupQuorums := make([]damltypes.INT64, maxMCMSGroups)
	copy(groupQuorums, bindConfig.GroupQuorums)
	groupParents := make([]damltypes.INT64, maxMCMSGroups)
	copy(groupParents, bindConfig.GroupParents)

	// Build the group configs
	groups := make([]types.Config, maxMCMSGroups)
	for i := range maxMCMSGroups {
		signers := signersByGroup[i]
		if signers == nil {
			signers = []common.Address{}
		}

		q, convErr := safecast.IntToUint8(int(groupQuorums[i]))
		if convErr != nil {
			return nil, fmt.Errorf("group quorum for group %d: %w", i, convErr)
		}

		groups[i] = types.Config{
			Signers:      signers,
			GroupSigners: []types.Config{},
			Quorum:       q,
		}
	}

	// Link the group signers; this assumes a group's parent always has a lower index
	// Process in reverse order to build the tree from leaves to root
	for i := maxMCMSGroups - 1; i >= 0; i-- {
		parent, convErr := safecast.IntToUint8(int(groupParents[i]))
		if convErr != nil {
			return nil, fmt.Errorf("group parent for group %d: %w", i, convErr)
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
