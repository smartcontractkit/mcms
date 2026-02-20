package canton

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/smartcontractkit/go-daml/pkg/types"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/chainlink-canton/contracts"
)

// FindActiveContractByInstanceAddress finds an active contract by its instance address.
// It returns an error if there are multiple or zero active contracts matching the instance address.
// TODO: copied from chainlink-canton deployment/utils/operations/contract/exercise.go to avoid importing
// unwanted dependencies. We should move the helper function to the bindings package and use it here.
func FindActiveContractByInstanceAddress(ctx context.Context, stateService apiv2.StateServiceClient, party, templateID string, instanceAddress contracts.InstanceAddress) (*apiv2.ActiveContract, error) {
	ledgerEndResp, err := stateService.GetLedgerEnd(ctx, &apiv2.GetLedgerEndRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger end: %w", err)
	}

	packageID, moduleName, entityName, err := parseTemplateIDFromString(templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template ID: %w", err)
	}

	activeContractsResp, err := stateService.GetActiveContracts(ctx, &apiv2.GetActiveContractsRequest{
		ActiveAtOffset: ledgerEndResp.GetOffset(),
		EventFormat: &apiv2.EventFormat{
			FiltersByParty: map[string]*apiv2.Filters{
				party: {
					Cumulative: []*apiv2.CumulativeFilter{
						{
							IdentifierFilter: &apiv2.CumulativeFilter_TemplateFilter{
								TemplateFilter: &apiv2.TemplateFilter{
									TemplateId: &apiv2.Identifier{
										PackageId:  packageID,
										ModuleName: moduleName,
										EntityName: entityName,
									},
									IncludeCreatedEventBlob: true,
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

	var activeContract *apiv2.ActiveContract
	for {
		activeContractResp, err := activeContractsResp.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to receive active contracts: %w", err)
		}

		if c, ok := activeContractResp.GetContractEntry().(*apiv2.GetActiveContractsResponse_ActiveContract); ok {
			createArguments := c.ActiveContract.GetCreatedEvent().GetCreateArguments()
			if createArguments == nil {
				continue
			}

			var contractInstanceID string
			for _, field := range createArguments.GetFields() {
				if field.GetLabel() == "instanceId" {
					contractInstanceID = field.GetValue().GetText()
					break
				}
			}
			if contractInstanceID == "" {
				continue
			}

			instanceID := contracts.InstanceID(contractInstanceID)
			signatories := c.ActiveContract.GetCreatedEvent().GetSignatories()
			if len(signatories) != 1 {
				continue
			}
			gotAddress := instanceID.RawInstanceAddress(types.PARTY(signatories[0])).InstanceAddress()

			if instanceAddress != gotAddress {
				continue
			}

			if activeContract != nil {
				return nil, fmt.Errorf("multiple active contracts found for InstanceAddress %s", instanceAddress.String())
			}
			activeContract = c.ActiveContract
		}
	}

	if activeContract == nil {
		return nil, fmt.Errorf("no active contract found for InstanceAddress %s", instanceAddress.String())
	}

	return activeContract, nil
}

// FindActiveContractIDByInstanceAddress returns the active contract ID for the given instance address.
func FindActiveContractIDByInstanceAddress(ctx context.Context, stateService apiv2.StateServiceClient, party, templateID string, instanceAddress contracts.InstanceAddress) (string, error) {
	activeContract, err := FindActiveContractByInstanceAddress(ctx, stateService, party, templateID, instanceAddress)
	if err != nil {
		return "", err
	}
	return activeContract.GetCreatedEvent().GetContractId(), nil
}

// ResolveMCMSContractID resolves an MCMS InstanceAddress (hex string) to the current active contract ID.
// instanceAddressHex is the hex-encoded InstanceAddress (keccak256 of "instanceId@party"); it may be prefixed with "0x".
func ResolveMCMSContractID(ctx context.Context, stateService apiv2.StateServiceClient, party, instanceAddressHex string) (string, error) {
	instanceAddressHex = strings.TrimPrefix(instanceAddressHex, "0x")
	if instanceAddressHex == "" {
		return "", fmt.Errorf("instance address hex is required")
	}
	addr := contracts.HexToInstanceAddress(instanceAddressHex)
	templateID := mcms.MCMS{}.GetTemplateID()
	return FindActiveContractIDByInstanceAddress(ctx, stateService, party, templateID, addr)
}

// IsInstanceAddressHex returns true if s looks like an InstanceAddress hex string (64 hex chars, optional 0x prefix).
// Canton contract IDs use a different format; when we have 0x-prefixed 64-char hex we treat it as InstanceAddress.
func IsInstanceAddressHex(s string) bool {
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}

// ResolveContractIDIfInstanceAddress returns the current MCMS contract ID if cid is InstanceAddress hex;
// otherwise returns cid unchanged. Use when building TargetCids so Canton receives real contract IDs.
func ResolveContractIDIfInstanceAddress(ctx context.Context, stateService apiv2.StateServiceClient, party, cid string) (string, error) {
	if !IsInstanceAddressHex(cid) {
		return cid, nil
	}
	return ResolveMCMSContractID(ctx, stateService, party, cid)
}
