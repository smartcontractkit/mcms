package mcms

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func LoadProposal(filePath string) (ProposalInterface, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close() // Ensure the file is closed when done

	// Temporary struct to read the proposal kind
	type TemporaryProposal struct {
		ProposalKind types.ProposalKind `json:"kind"`
	}

	// Read the proposal kind
	var tempProposal TemporaryProposal
	err = json.NewDecoder(file).Decode(&tempProposal)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, err
	}

	switch tempProposal.ProposalKind {
	case types.KindProposal:
		return NewProposal(file)
	case types.KindTimelockProposal:
		return NewTimelockProposal(file)
	default:
		return nil, errors.New("unknown proposal type")
	}
}

// BatchToChainOperation converts a batch of chain operations to a single types.ChainOperation for
// different chains
func BatchToChainOperation(
	bops types.BatchOperation,
	timelockAddr string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
) (types.Operation, common.Hash, error) {
	chainFamily, err := types.GetChainSelectorFamily(bops.ChainSelector)
	if err != nil {
		return types.Operation{}, common.Hash{}, err
	}

	var converter sdk.TimelockConverter
	switch chainFamily {
	case cselectors.FamilyEVM:
		converter = &evm.TimelockConverter{}
	}

	return converter.ConvertBatchToChainOperation(
		bops, timelockAddr, delay, action, predecessor,
	)
}
