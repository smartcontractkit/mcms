package canton

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-canton/bindings"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	stateClient   apiv2.StateServiceClient
	party         string
	contractCache *mcms.MCMS // Cache MCMS to avoid repeated RPC calls
	cachedAddress string     // Track which contract address is cached
	role          TimelockRole
}

func NewInspector(stateClient apiv2.StateServiceClient, party string, role TimelockRole) *Inspector {
	return &Inspector{
		stateClient: stateClient,
		party:       party,
		role:        role,
	}
}

func (i *Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	if i.contractCache == nil || i.cachedAddress != mcmsAddr {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
		i.cachedAddress = mcmsAddr
	}

	switch i.role {
	case TimelockRoleProposer:
		return toConfig(i.contractCache.Proposer.Config)
	case TimelockRoleBypasser:
		return toConfig(i.contractCache.Bypasser.Config)
	case TimelockRoleCanceller:
		return toConfig(i.contractCache.Canceller.Config)
	default:
		return nil, fmt.Errorf("unknown timelock role: %s", i.role)
	}
}

func (i *Inspector) GetOpCount(ctx context.Context, mcmsAddr string) (uint64, error) {
	if i.contractCache == nil || i.cachedAddress != mcmsAddr {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return 0, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
		i.cachedAddress = mcmsAddr
	}

	switch i.role {
	case TimelockRoleProposer:
		return uint64(i.contractCache.Proposer.ExpiringRoot.OpCount), nil
	case TimelockRoleBypasser:
		return uint64(i.contractCache.Bypasser.ExpiringRoot.OpCount), nil
	case TimelockRoleCanceller:
		return uint64(i.contractCache.Canceller.ExpiringRoot.OpCount), nil
	default:
		return 0, fmt.Errorf("unknown timelock role: %s", i.role)
	}
}

func (i *Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	if i.contractCache == nil || i.cachedAddress != mcmsAddr {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return common.Hash{}, 0, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
		i.cachedAddress = mcmsAddr
	}

	var expiringRoot mcms.ExpiringRoot
	switch i.role {
	case TimelockRoleProposer:
		expiringRoot = i.contractCache.Proposer.ExpiringRoot
	case TimelockRoleBypasser:
		expiringRoot = i.contractCache.Bypasser.ExpiringRoot
	case TimelockRoleCanceller:
		expiringRoot = i.contractCache.Canceller.ExpiringRoot
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
	if i.contractCache == nil || i.cachedAddress != mcmsAddr {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return types.ChainMetadata{}, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
		i.cachedAddress = mcmsAddr
	}

	var rootMetadata mcms.RootMetadata
	switch i.role {
	case TimelockRoleProposer:
		rootMetadata = i.contractCache.Proposer.RootMetadata
	case TimelockRoleBypasser:
		rootMetadata = i.contractCache.Bypasser.RootMetadata
	case TimelockRoleCanceller:
		rootMetadata = i.contractCache.Canceller.RootMetadata
	default:
		return types.ChainMetadata{}, fmt.Errorf("unknown timelock role: %s", i.role)
	}

	return types.ChainMetadata{
		StartingOpCount: uint64(rootMetadata.PreOpCount),
		MCMAddress:      string(i.contractCache.InstanceId),
	}, nil
}

// getMCMSContract queries the active MCMS contract by contract ID
func (i *Inspector) getMCMSContract(ctx context.Context, mcmsAddr string) (*mcms.MCMS, error) {
	// Get current ledger offset
	ledgerEndResp, err := i.stateClient.GetLedgerEnd(ctx, &apiv2.GetLedgerEndRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger end: %w", err)
	}

	// Query active contracts at current offset
	activeContractsResp, err := i.stateClient.GetActiveContracts(ctx, &apiv2.GetActiveContractsRequest{
		ActiveAtOffset: ledgerEndResp.GetOffset(),
		EventFormat: &apiv2.EventFormat{
			FiltersByParty: map[string]*apiv2.Filters{
				i.party: {
					Cumulative: []*apiv2.CumulativeFilter{
						{
							IdentifierFilter: &apiv2.CumulativeFilter_TemplateFilter{
								TemplateFilter: &apiv2.TemplateFilter{
									TemplateId: &apiv2.Identifier{
										PackageId:  "#mcms",
										ModuleName: "MCMS.Main",
										EntityName: "MCMS",
									},
									IncludeCreatedEventBlob: false,
								},
							},
						},
					},
				},
			},
			Verbose: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get active contracts: %w", err)
	}
	defer activeContractsResp.CloseSend()

	// Stream through active contracts to find the MCMS contract with matching ID
	for {
		resp, err := activeContractsResp.Recv()
		if errors.Is(err, io.EOF) {
			// Stream ended without finding the contract
			return nil, fmt.Errorf("MCMS contract with ID %s not found", mcmsAddr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to receive active contracts: %w", err)
		}

		activeContract, ok := resp.GetContractEntry().(*apiv2.GetActiveContractsResponse_ActiveContract)
		if !ok {
			continue
		}

		createdEvent := activeContract.ActiveContract.GetCreatedEvent()
		if createdEvent == nil {
			continue
		}

		// Check if contract ID matches
		if createdEvent.ContractId != mcmsAddr {
			continue
		}

		// Use bindings package to unmarshal the contract
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
		mcmsContractNoMinDelay, err := bindings.UnmarshalActiveContract[NoMinDelayMCMS](activeContract)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal MCMS contract: %w", err)
		}

		mcmsContract := &mcms.MCMS{
			Owner:              mcmsContractNoMinDelay.Owner,
			InstanceId:         mcmsContractNoMinDelay.InstanceId,
			ChainId:            mcmsContractNoMinDelay.ChainId,
			Proposer:           mcmsContractNoMinDelay.Proposer,
			Canceller:          mcmsContractNoMinDelay.Canceller,
			Bypasser:           mcmsContractNoMinDelay.Bypasser,
			BlockedFunctions:   mcmsContractNoMinDelay.BlockedFunctions,
			TimelockTimestamps: mcmsContractNoMinDelay.TimelockTimestamps,
			MinDelay:           0, // TODO: Fix bindings type
		}

		return mcmsContract, nil
	}
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
