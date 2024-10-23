package timelock

import (
	"math/big"
	"time"

	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/pkg/errors"
	owner "github.com/smartcontractkit/mcms/pkg/gethwrappers"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms"
)

var ZERO_HASH = common.Hash{}

type TimelockOperation string

const (
	Schedule TimelockOperation = "schedule"
	Cancel   TimelockOperation = "cancel"
	Bypass   TimelockOperation = "bypass"
)

type MCMSWithTimelockProposal struct {
	mcms.MCMSProposal

	Operation TimelockOperation `json:"operation"` // Always 'schedule', 'cancel', or 'bypass'

	// i.e. 1d, 1w, 1m, 1y
	MinDelay string `json:"minDelay"`

	TimelockAddresses map[mcms.ChainIdentifier]common.Address `json:"timelockAddresses"`

	// Overridden: Operations to be executed after wrapping in a timelock
	Transactions []BatchChainOperation `json:"transactions"`
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
	chainMetadata map[mcms.ChainIdentifier]mcms.ChainMetadata,
	timelockAddresses map[mcms.ChainIdentifier]common.Address,
	description string,
	transactions []BatchChainOperation,
	operation TimelockOperation,
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
func (p *MCMSWithTimelockProposal) MarshalJSON() ([]byte, error) {
	// First, marshal the Transactions field from MCMSWithTimelockProposal
	transactionsBytes, err := json.Marshal(struct {
		Transactions []BatchChainOperation `json:"transactions"`
	}{
		Transactions: p.Transactions,
	})
	if err != nil {
		return nil, err
	}

	// Then, marshal the embedded MCMSProposal directly
	mcmsProposalBytes, err := json.Marshal(p.MCMSProposal)
	if err != nil {
		return nil, err
	}

	// Finally, marshal the remaining fields specific to MCMSWithTimelockProposal
	mcmsWithTimelockFieldsBytes, err := json.Marshal(struct {
		Operation         TimelockOperation                       `json:"operation"`
		MinDelay          string                                  `json:"minDelay"`
		TimelockAddresses map[mcms.ChainIdentifier]common.Address `json:"timelockAddresses"`
	}{
		Operation:         p.Operation,
		MinDelay:          p.MinDelay,
		TimelockAddresses: p.TimelockAddresses,
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

func (p *MCMSWithTimelockProposal) UnmarshalJSON(data []byte) error {

	// Unmarshal Transactions field from MCMSWithTimelockProposal
	transactionsFields := struct {
		Transactions []BatchChainOperation `json:"transactions"`
	}{}

	if err := json.Unmarshal(data, &transactionsFields); err != nil {
		return err
	}
	p.Transactions = transactionsFields.Transactions

	// Then, unmarshal into the embedded MCMSProposal directly
	if err := json.Unmarshal(data, &p.MCMSProposal); err != nil {
		return err
	}
	// This field is overridden in MCMSWithTimelockProposal
	p.MCMSProposal.Transactions = nil

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
	p.Operation = mcmsWithTimelockFields.Operation
	p.MinDelay = mcmsWithTimelockFields.MinDelay
	p.TimelockAddresses = mcmsWithTimelockFields.TimelockAddresses
	p.Transactions = transactionsFields.Transactions

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
	}

	if err := timeLockProposalValidateBasic(*m); err != nil {
		return err
	}

	switch m.Operation {
	case Schedule, Cancel, Bypass:
		// NOOP
	default:
		return &errors.InvalidTimelockOperationError{
			ReceivedTimelockOperation: string(m.Operation),
		}
	}

	// Validate the delay is a valid duration but is only required
	// for Schedule operations
	if m.Operation == Schedule {
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
	predecessorMap := make(map[mcms.ChainIdentifier]common.Hash)
	for chain := range m.ChainMetadata {
		predecessorMap[chain] = ZERO_HASH
	}

	// Convert chain metadata
	mcmOnly.ChainMetadata = make(map[mcms.ChainIdentifier]mcms.ChainMetadata)
	for chain, metadata := range m.ChainMetadata {
		mcmOnly.ChainMetadata[chain] = mcms.ChainMetadata{
			StartingOpCount: metadata.StartingOpCount,
			MCMAddress:      metadata.MCMAddress,
		}
	}

	// Convert transactions into timelock wrapped transactions
	for _, t := range m.Transactions {
		calls := make([]owner.RBACTimelockCall, 0)
		tags := make([]string, 0)
		for _, op := range t.Batch {
			calls = append(calls, owner.RBACTimelockCall{
				Target: op.To,
				Data:   op.Data,
				Value:  op.Value,
			})
			tags = append(tags, op.Tags...)
		}
		predecessor := predecessorMap[t.ChainIdentifier]
		salt := ZERO_HASH
		delay, _ := time.ParseDuration(m.MinDelay)

		abi, errAbi := owner.RBACTimelockMetaData.GetAbi()
		if errAbi != nil {
			return mcms.MCMSProposal{}, errAbi
		}

		operationId, errHash := hashOperationBatch(calls, predecessor, salt)
		if errHash != nil {
			return mcms.MCMSProposal{}, errHash
		}

		// Encode the data based on the operation
		var data []byte
		var err error
		switch m.Operation {
		case Schedule:
			data, err = abi.Pack("scheduleBatch", calls, predecessor, salt, big.NewInt(int64(delay.Seconds())))
			if err != nil {
				return mcms.MCMSProposal{}, err
			}
		case Cancel:
			data, err = abi.Pack("cancel", operationId)
			if err != nil {
				return mcms.MCMSProposal{}, err
			}
		case Bypass:
			data, err = abi.Pack("bypasserExecuteBatch", calls)
			if err != nil {
				return mcms.MCMSProposal{}, err
			}
		default:
			return mcms.MCMSProposal{}, &errors.InvalidTimelockOperationError{
				ReceivedTimelockOperation: string(m.Operation),
			}
		}

		mcmOnly.Transactions = append(mcmOnly.Transactions, mcms.ChainOperation{
			ChainIdentifier: t.ChainIdentifier,
			Operation: mcms.Operation{
				To:           m.TimelockAddresses[t.ChainIdentifier],
				Data:         data,
				Value:        big.NewInt(0), // TODO: is this right?
				ContractType: "RBACTimelock",
				Tags:         tags,
			},
		})

		predecessorMap[t.ChainIdentifier] = operationId
	}

	return mcmOnly, nil
}

func (m *MCMSWithTimelockProposal) AddSignature(signature mcms.Signature) {
	m.Signatures = append(m.Signatures, signature)
}

// hashOperationBatch replicates the hash calculation from Solidity
// TODO: see if there's an easier way to do this using the gethwrappers
func hashOperationBatch(calls []owner.RBACTimelockCall, predecessor, salt [32]byte) (common.Hash, error) {
	const abi = `[{"components":[{"internalType":"address","name":"target","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"internalType":"struct Call[]","name":"calls","type":"tuple[]"},{"internalType":"bytes32","name":"predecessor","type":"bytes32"},{"internalType":"bytes32","name":"salt","type":"bytes32"}]`
	encoded, err := mcms.ABIEncode(abi, calls, predecessor, salt)
	if err != nil {
		return common.Hash{}, err
	}

	// Return the hash as a [32]byte array
	return crypto.Keccak256Hash(encoded), nil
}

func mergeJSON(json1, json2 []byte) ([]byte, error) {
	var map1, map2 map[string]interface{}

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
