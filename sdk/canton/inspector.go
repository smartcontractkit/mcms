package canton

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-canton/bindings"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/chainlink-canton/contracts"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	stateClient apiv2.StateServiceClient
	party       string
	role        TimelockRole
}

func NewInspector(stateClient apiv2.StateServiceClient, party string, role TimelockRole) *Inspector {
	return &Inspector{
		stateClient: stateClient,
		party:       party,
		role:        role,
	}
}

// StateServiceClient returns the state service client for resolution (e.g. InstanceAddress to contract ID).
func (i *Inspector) StateServiceClient() apiv2.StateServiceClient {
	return i.stateClient
}

func (i *Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
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
	mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	switch i.role {
	case TimelockRoleProposer:
		return uint64(mcmsContract.Proposer.ExpiringRoot.OpCount), nil
	case TimelockRoleBypasser:
		return uint64(mcmsContract.Bypasser.ExpiringRoot.OpCount), nil
	case TimelockRoleCanceller:
		return uint64(mcmsContract.Canceller.ExpiringRoot.OpCount), nil
	default:
		return 0, fmt.Errorf("unknown timelock role: %s", i.role)
	}
}

func (i *Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	var expiringRoot mcms.ExpiringRoot
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
	validUntil := uint32(timeVal.Unix())

	return root, validUntil, nil
}

func (i *Inspector) GetRootMetadata(ctx context.Context, mcmsAddr string) (types.ChainMetadata, error) {
	mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to get MCMS contract: %w", err)
	}

	var rootMetadata mcms.RootMetadata
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
	mcmAddress := contracts.InstanceID(string(mcmsContract.InstanceId)).RawInstanceAddress(cantontypes.PARTY(mcmsContract.Owner)).InstanceAddress().Hex()
	return types.ChainMetadata{
		StartingOpCount: uint64(rootMetadata.PreOpCount),
		MCMAddress:      mcmAddress,
	}, nil
}

// getMCMSContract queries the active MCMS contract by InstanceAddress (hex).
// mcmsAddr is the InstanceAddress hex string (may be prefixed with "0x").
func (i *Inspector) getMCMSContract(ctx context.Context, mcmsAddr string) (*mcms.MCMS, error) {
	mcmsAddr = strings.TrimPrefix(mcmsAddr, "0x")
	if mcmsAddr == "" {
		return nil, fmt.Errorf("MCMS instance address is required")
	}
	addr := contracts.HexToInstanceAddress(mcmsAddr)
	templateID := mcms.MCMS{}.GetTemplateID()
	activeContract, err := FindActiveContractByInstanceAddress(ctx, i.stateClient, i.party, templateID, addr)
	if err != nil {
		return nil, fmt.Errorf("MCMS contract for InstanceAddress %s: %w", mcmsAddr, err)
	}

	// Wrap for bindings unmarshal
	wrapped := &apiv2.GetActiveContractsResponse_ActiveContract{ActiveContract: activeContract}

	// TODO: MinDelay type from binding doesnt correspond to actual type from contract
	type NoMinDelayMCMS struct {
		Owner              cantontypes.PARTY      `json:"owner"`
		InstanceId         cantontypes.TEXT       `json:"instanceId"`
		ChainId            cantontypes.INT64      `json:"chainId"`
		Proposer           mcms.RoleState         `json:"proposer"`
		Canceller          mcms.RoleState         `json:"canceller"`
		Bypasser           mcms.RoleState         `json:"bypasser"`
		BlockedFunctions   []mcms.BlockedFunction `json:"blockedFunctions"`
		TimelockTimestamps cantontypes.GENMAP     `json:"timelockTimestamps"`
	}
	mcmsContractNoMinDelay, err := bindings.UnmarshalActiveContract[NoMinDelayMCMS](wrapped)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCMS contract: %w", err)
	}

	return &mcms.MCMS{
		Owner:              mcmsContractNoMinDelay.Owner,
		InstanceId:         mcmsContractNoMinDelay.InstanceId,
		ChainId:            mcmsContractNoMinDelay.ChainId,
		Proposer:           mcmsContractNoMinDelay.Proposer,
		Canceller:          mcmsContractNoMinDelay.Canceller,
		Bypasser:           mcmsContractNoMinDelay.Bypasser,
		BlockedFunctions:   mcmsContractNoMinDelay.BlockedFunctions,
		TimelockTimestamps: mcmsContractNoMinDelay.TimelockTimestamps,
		MinDelay:           0, // TODO: Fix bindings type
	}, nil
}

// toConfig converts a Canton MultisigConfig to the chain-agnostic types.Config
func toConfig(bindConfig mcms.MultisigConfig) (*types.Config, error) {
	// Group signers by group index
	signersByGroup := make([][]common.Address, 32) // MCMS supports up to 32 groups

	for _, signer := range bindConfig.Signers {
		groupIdx := int(signer.SignerGroup)
		if groupIdx >= 32 {
			return nil, fmt.Errorf("signer group index %d exceeds maximum of 31", groupIdx)
		}

		// Parse signer address
		addr := common.HexToAddress(string(signer.SignerAddress))
		signersByGroup[groupIdx] = append(signersByGroup[groupIdx], addr)
	}

	// Build the group configs
	groups := make([]types.Config, 32)
	for i := 0; i < 32; i++ {
		signers := signersByGroup[i]
		if signers == nil {
			signers = []common.Address{}
		}

		quorum := uint8(0)
		if i < len(bindConfig.GroupQuorums) {
			quorum = uint8(bindConfig.GroupQuorums[i])
		}

		groups[i] = types.Config{
			Signers:      signers,
			GroupSigners: []types.Config{},
			Quorum:       quorum,
		}
	}

	// Link the group signers; this assumes a group's parent always has a lower index
	// Process in reverse order to build the tree from leaves to root
	for i := 31; i >= 0; i-- {
		parent := uint8(0)
		if i < len(bindConfig.GroupParents) {
			parent = uint8(bindConfig.GroupParents[i])
		}

		// Add non-empty child groups to their parent
		// Skip the root group (i == 0) and empty groups (quorum == 0)
		if i > 0 && groups[i].Quorum > 0 {
			groups[parent].GroupSigners = append([]types.Config{groups[i]}, groups[parent].GroupSigners...)
		}
	}

	// Validate the root group config
	if err := groups[0].Validate(); err != nil {
		return nil, fmt.Errorf("invalid MCMS config: %w", err)
	}

	return &groups[0], nil
}
