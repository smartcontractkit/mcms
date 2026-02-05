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
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	stateClient   apiv2.StateServiceClient
	party         string
	contractCache *mcms.MCMS // Cache MCMS to avoid repeated RPC calls
}

func NewInspector(stateClient apiv2.StateServiceClient, party string) *Inspector {
	return &Inspector{
		stateClient: stateClient,
		party:       party,
	}
}

func (i *Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	if i.contractCache == nil {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
	}

	return toConfig(i.contractCache.Config)
}

func (i *Inspector) GetOpCount(ctx context.Context, mcmsAddr string) (uint64, error) {
	if i.contractCache == nil {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return 0, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
	}

	return uint64(i.contractCache.ExpiringRoot.OpCount), nil
}

func (i *Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	if i.contractCache == nil {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return common.Hash{}, 0, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
	}

	// Parse the root from hex string
	rootStr := string(i.contractCache.ExpiringRoot.Root)
	rootStr = strings.TrimPrefix(rootStr, "0x")
	rootBytes, err := hex.DecodeString(rootStr)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to decode root hash: %w", err)
	}

	root := common.BytesToHash(rootBytes)

	// validUntil is a TIMESTAMP (which wraps time.Time)
	// Convert to Unix timestamp (uint32)
	timeVal := time.Time(i.contractCache.ExpiringRoot.ValidUntil)
	validUntil := uint32(timeVal.Unix())

	return root, validUntil, nil
}

func (i *Inspector) GetRootMetadata(ctx context.Context, mcmsAddr string) (types.ChainMetadata, error) {
	if i.contractCache == nil {
		mcmsContract, err := i.getMCMSContract(ctx, mcmsAddr)
		if err != nil {
			return types.ChainMetadata{}, fmt.Errorf("failed to get MCMS contract: %w", err)
		}
		i.contractCache = mcmsContract
	}

	return types.ChainMetadata{
		StartingOpCount: uint64(i.contractCache.RootMetadata.PreOpCount),
		MCMAddress:      string(i.contractCache.McmsId),
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
		mcmsContract, err := bindings.UnmarshalActiveContract[mcms.MCMS](activeContract)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal MCMS contract: %w", err)
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
