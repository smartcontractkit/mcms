package timelock

import (
	"time"

	"encoding/json"

	"encoding/json"

	mcmsTypes "github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"
	"github.com/smartcontractkit/mcms/pkg/proposal/timelock/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms"
)

var ZERO_HASH = common.Hash{}

type MCMSWithTimelockProposal struct {
	mcms.MCMSProposal

	Operation types.TimelockOperationType `json:"operation"` // Always 'schedule', 'cancel', or 'bypass'

	// i.e. 1d, 1w, 1m, 1y
	MinDelay string `json:"minDelay"`

	TimelockAddresses map[mcmsTypes.ChainIdentifier]common.Address `json:"timelockAddresses"`

	// Overridden: Operations to be executed after wrapping in a timelock
	Transactions []types.BatchChainOperation `json:"transactions"`
}

// timeLockProposalValidateBasic basic validation for an MCMS proposal
func timeLockProposalValidateBasic(timelockProposal MCMSWithTimelockProposal) error {
	// Get the current Unix timestamp as an int64
	currentTime := time.Now().Unix()

	currentTimeCasted, err := mcms.SafeCastIntToUint32(int(currentTime))
	if err != nil {
		return err
	}
	if timelockProposal.ValidUntil <= currentTimeCasted {
		// ValidUntil is a Unix timestamp, so it should be greater than the current time
		return &errors.InvalidValidUntilError{
			ReceivedValidUntil: timelockProposal.ValidUntil,
		}
	}
	if len(timelockProposal.ChainMetadata) == 0 {
		return &errors.NoChainMetadataError{}
	}

	if len(timelockProposal.Transactions) == 0 {
		return &errors.NoTransactionsError{}
	}

	if timelockProposal.Description == "" {
		return &errors.InvalidDescriptionError{
			ReceivedDescription: timelockProposal.Description,
		}
	}

	return nil
}
func NewMCMSWithTimelockProposal(
	version string,
	validUntil uint32,
	signatures []mcms.Signature,
	overridePreviousRoot bool,
	chainMetadata map[mcmsTypes.ChainIdentifier]mcms.ChainMetadata,
	timelockAddresses map[mcmsTypes.ChainIdentifier]common.Address,
	description string,
	transactions []types.BatchChainOperation,
	operation types.TimelockOperationType,
	minDelay string,
) (*MCMSWithTimelockProposal, error) {
	proposal := MCMSWithTimelockProposal{
		MCMSProposal: mcms.MCMSProposal{
			Version:              version,
			ValidUntil:           validUntil,
			Signatures:           signatures,
			OverridePreviousRoot: overridePreviousRoot,
			Description:          description,
			ChainMetadata:        chainMetadata,
		},
		Operation:         operation,
		MinDelay:          minDelay,
		TimelockAddresses: timelockAddresses,
		Transactions:      transactions,
	}

	errValidate := proposal.Validate()
	if errValidate != nil {
		return nil, errValidate
	}

	return &proposal, nil
}

func NewMCMSWithTimelockProposalFromFile(filePath string) (*MCMSWithTimelockProposal, error) {
	var out MCMSWithTimelockProposal
	errFromFile := mcms.FromFile(filePath, &out)
	if errFromFile != nil {
		return nil, errFromFile
	}

	return &out, nil
}

// MarshalJSON due to the struct embedding we need to separate the marshalling in 3 phases.
func (m *MCMSWithTimelockProposal) MarshalJSON() ([]byte, error) {
	// First, marshal the Transactions field from MCMSWithTimelockProposal
	transactionsBytes, err := json.Marshal(struct {
		Transactions []BatchChainOperation `json:"transactions"`
	}{
		Transactions: m.Transactions,
	})
	if err != nil {
		return nil, err
	}

	// Then, marshal the embedded MCMSProposal directly
	// Exclude transactions from the embedded MCMSProposal, they are on the parent struct.
	m.MCMSProposal.Transactions = nil
	mcmsProposalBytes, err := json.Marshal(m.MCMSProposal)
	if err != nil {
		return nil, err
	}

	// Finally, marshal the remaining fields specific to MCMSWithTimelockProposal
	mcmsWithTimelockFieldsBytes, err := json.Marshal(struct {
		Operation         TimelockOperation                       `json:"operation"`
		MinDelay          string                                  `json:"minDelay"`
		TimelockAddresses map[mcms.ChainIdentifier]common.Address `json:"timelockAddresses"`
	}{
		Operation:         m.Operation,
		MinDelay:          m.MinDelay,
		TimelockAddresses: m.TimelockAddresses,
	})
	if err != nil {
		return nil, err
	}

	// Merge the JSON objects
	finalJSON, err := mergeJSON(mcmsProposalBytes, transactionsBytes)
	if err != nil {
		return nil, err
	}
	finalJSON, err = mergeJSON(finalJSON, mcmsWithTimelockFieldsBytes)
	if err != nil {
		return nil, err
	}

	return finalJSON, nil
}

func (m *MCMSWithTimelockProposal) UnmarshalJSON(data []byte) error {
	// Unmarshal Transactions field from MCMSWithTimelockProposal
	transactionsFields := struct {
		Transactions []BatchChainOperation `json:"transactions"`
	}{}

	if err := json.Unmarshal(data, &transactionsFields); err != nil {
		return err
	}
	m.Transactions = transactionsFields.Transactions

	// Create a map to remove the "transactions" field from the data before unmarshalling into MCMSProposal
	var jsonData map[string]any
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}

	// Remove the "transactions" field from the map
	delete(jsonData, "transactions")

	// Marshal the modified map back into JSON
	modifiedData, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	// Now unmarshal the modified data into MCMSProposal (without Transactions field)
	if err := json.Unmarshal(modifiedData, &m.MCMSProposal); err != nil {
		return err
	}

	// Unmarshal the remaining fields specific to MCMSWithTimelockProposal
	mcmsWithTimelockFields := struct {
		Operation         TimelockOperation                       `json:"operation"`
		MinDelay          string                                  `json:"minDelay"`
		TimelockAddresses map[mcms.ChainIdentifier]common.Address `json:"timelockAddresses"`
	}{}

	if err := json.Unmarshal(data, &mcmsWithTimelockFields); err != nil {
		return err
	}

	// Assign the remaining fields to MCMSWithTimelockProposal
	m.Operation = mcmsWithTimelockFields.Operation
	m.MinDelay = mcmsWithTimelockFields.MinDelay
	m.TimelockAddresses = mcmsWithTimelockFields.TimelockAddresses

	return nil
}

func (m *MCMSWithTimelockProposal) Validate() error {
	if m.Version == "" {
		return &errors.InvalidVersionError{
			ReceivedVersion: m.Version,
		}
	}

	// Validate all chains in transactions have an entry in chain metadata
	for _, t := range m.Transactions {
		if _, ok := m.ChainMetadata[t.ChainIdentifier]; !ok {
			return &errors.MissingChainDetailsError{
				ChainIdentifier: uint64(t.ChainIdentifier),
				Parameter:       "chain metadata",
			}
		}
		for _, op := range t.Batch {
			// Chain specific validations.
			if err := mcms.ValidateAdditionalFields(op.AdditionalFields, t.ChainIdentifier); err != nil {
				return err
			}
		}
	}

	if err := timeLockProposalValidateBasic(*m); err != nil {
		return err
	}

	switch m.Operation {
	case types.Schedule, types.Cancel, types.Bypass:
		// NOOP
	default:
		return &errors.InvalidTimelockOperationError{
			ReceivedTimelockOperation: string(m.Operation),
		}
	}

	// Validate the delay is a valid duration but is only required
	// for Schedule operations
	if m.Operation == types.Schedule {
		if _, err := time.ParseDuration(m.MinDelay); err != nil {
			return err
		}
	}

	return nil
}

func (m *MCMSWithTimelockProposal) ToExecutor(sim bool) (*mcms.Executor, error) {
	// Convert the proposal to an MCMS only proposal
	mcmOnly, errToMcms := m.toMCMSOnlyProposal()
	if errToMcms != nil {
		return nil, errToMcms
	}

	return mcmOnly.ToExecutor(sim)
}

func (m *MCMSWithTimelockProposal) toMCMSOnlyProposal() (mcms.MCMSProposal, error) {
	mcmOnly := m.MCMSProposal

	// Start predecessor map with all chains pointing to the zero hash
	predecessorMap := make(map[mcmsTypes.ChainIdentifier]common.Hash)
	for chain := range m.ChainMetadata {
		predecessorMap[chain] = ZERO_HASH
	}

	// Convert chain metadata
	mcmOnly.ChainMetadata = make(map[mcmsTypes.ChainIdentifier]mcms.ChainMetadata)
	for chain, metadata := range m.ChainMetadata {
		mcmOnly.ChainMetadata[chain] = mcms.ChainMetadata{
			StartingOpCount: metadata.StartingOpCount,
			MCMAddress:      metadata.MCMAddress,
		}
	}

	// Convert transactions into timelock wrapped transactions using the helper function
	for _, t := range m.Transactions {
		timelockAddress := m.TimelockAddresses[t.ChainIdentifier]
		predecessor := predecessorMap[t.ChainIdentifier]

		chainOp, operationId, err := ToChainOperation(t, timelockAddress, m.MinDelay, m.Operation, predecessor)
		if err != nil {
			return mcms.MCMSProposal{}, err
		}

		// Append the converted operation to the MCMS only proposal
		mcmOnly.Transactions = append(mcmOnly.Transactions, chainOp)

		// Update predecessor for the chain
		predecessorMap[t.ChainIdentifier] = operationId
	}

	return mcmOnly, nil
}

func (m *MCMSWithTimelockProposal) AddSignature(signature mcms.Signature) {
	m.Signatures = append(m.Signatures, signature)
}

func mergeJSON(json1, json2 []byte) ([]byte, error) {
	var map1, map2 map[string]any

	// Unmarshal both JSON objects into maps
	if err := json.Unmarshal(json1, &map1); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(json2, &map2); err != nil {
		return nil, err
	}

	// Merge map2 into map1
	for key, value := range map2 {
		map1[key] = value
	}

	// Marshal the merged result back into JSON
	return json.Marshal(map1)
}
