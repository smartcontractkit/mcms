package stellar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

// TimelockConverter converts timelock proposal batches into Soroban MCMS operations whose
// transaction.to is the timelock contract and transaction.data is Soroban invoke payload bytes
// (ScVec of Symbol + args) as consumed by MCMS execute / timelock decode_invoke_payload.
type TimelockConverter struct{}

// NewTimelockConverter returns a TimelockConverter for Stellar / Soroban RBACTimelock.
func NewTimelockConverter() *TimelockConverter {
	return &TimelockConverter{}
}

// TimelockProposalAdditionalFields are JSON fields on types.ChainMetadata.AdditionalFields for
// Stellar timelock proposals. The address must hold the corresponding on-chain role when the
// timelock entrypoint runs (first argument to schedule_batch / cancel / bypasser_execute_batch).
//
// For [sdk.TimelockExecutor] / [sdk.TimelockConfigurer] wiring, set timelockExecutor (execute_batch
// caller) and timelockAdmin (update_delay caller) respectively.
type TimelockProposalAdditionalFields struct {
	TimelockProposer  string `json:"timelockProposer,omitempty"`
	TimelockCanceller string `json:"timelockCanceller,omitempty"`
	TimelockBypasser  string `json:"timelockBypasser,omitempty"`
	TimelockExecutor  string `json:"timelockExecutor,omitempty"`
	TimelockAdmin     string `json:"timelockAdmin,omitempty"`
}

// ParseTimelockProposalAdditionalFields unmarshals Stellar timelock-related additional metadata.
func ParseTimelockProposalAdditionalFields(raw json.RawMessage) (TimelockProposalAdditionalFields, error) {
	var z TimelockProposalAdditionalFields
	if len(raw) == 0 {
		return z, fmt.Errorf("stellar timelock: chain metadata additionalFields is required")
	}
	if err := json.Unmarshal(raw, &z); err != nil {
		return z, fmt.Errorf("stellar timelock: additionalFields: %w", err)
	}

	return z, nil
}

func callsFromBatchOperation(bop types.BatchOperation) ([]timelockbindings.Call, error) {
	out := make([]timelockbindings.Call, 0, len(bop.Transactions))
	for _, tx := range bop.Transactions {
		to, err := ParseContractID(tx.To)
		if err != nil {
			return nil, fmt.Errorf("batch transaction to: %w", err)
		}

		out = append(out, timelockbindings.Call{
			To:   to,
			Data: tx.Data,
		})
	}

	return out, nil
}

func (t TimelockConverter) ConvertBatchToChainOperations(
	ctx context.Context,
	metadata types.ChainMetadata,
	batchOp types.BatchOperation,
	timelockAddress string,
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	_ = ctx
	_ = mcmAddress

	if _, err := ParseContractID(timelockAddress); err != nil {
		return nil, common.Hash{}, fmt.Errorf("timelock address: %w", err)
	}

	if _, err := ParseContractID(mcmAddress); err != nil {
		return nil, common.Hash{}, fmt.Errorf("mcm address: %w", err)
	}

	af, err := ParseTimelockProposalAdditionalFields(metadata.AdditionalFields)
	if err != nil {
		return nil, common.Hash{}, err
	}

	pred := predecessor
	if action == types.TimelockActionBypass {
		pred = common.Hash{}
	}

	calls, err := callsFromBatchOperation(batchOp)
	if err != nil {
		return nil, common.Hash{}, err
	}

	operationID := HashOperationBatch(calls, pred, salt)

	tags := make([]string, 0)
	for _, tx := range batchOp.Transactions {
		tags = append(tags, tx.Tags...)
	}

	var data []byte

	switch action {
	case types.TimelockActionSchedule:
		caller := af.TimelockProposer
		if caller == "" {
			return nil, common.Hash{}, fmt.Errorf("stellar timelock: timelockProposer is required in metadata.additionalFields")
		}

		callsVal, err := timelockbindings.Calls{Inner: calls}.ToScVal()
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("calls to ScVal: %w", err)
		}

		data, err = sorobanInvokePayloadBytes("schedule_batch",
			scval.AddressToScVal(caller),
			callsVal,
			scval.Bytes32ToScVal(pred),
			scval.Bytes32ToScVal(salt),
			scval.Uint64ToScVal(uint64(delay.Seconds())),
		)
		if err != nil {
			return nil, common.Hash{}, err
		}

	case types.TimelockActionCancel:
		caller := af.TimelockCanceller
		if caller == "" {
			return nil, common.Hash{}, fmt.Errorf("stellar timelock: timelockCanceller is required in metadata.additionalFields")
		}

		data, err = sorobanInvokePayloadBytes("cancel",
			scval.AddressToScVal(caller),
			scval.Bytes32ToScVal(operationID),
		)
		if err != nil {
			return nil, common.Hash{}, err
		}

	case types.TimelockActionBypass:
		caller := af.TimelockBypasser
		if caller == "" {
			return nil, common.Hash{}, fmt.Errorf("stellar timelock: timelockBypasser is required in metadata.additionalFields")
		}

		callsVal, err := timelockbindings.Calls{Inner: calls}.ToScVal()
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("calls to ScVal: %w", err)
		}

		data, err = sorobanInvokePayloadBytes("bypasser_execute_batch",
			scval.AddressToScVal(caller),
			callsVal,
		)
		if err != nil {
			return nil, common.Hash{}, err
		}

	default:
		return nil, common.Hash{}, fmt.Errorf("invalid timelock action: %s", action)
	}

	additional := json.RawMessage([]byte("{}"))

	op := types.Operation{
		ChainSelector: batchOp.ChainSelector,
		Transaction: types.Transaction{
			OperationMetadata: types.OperationMetadata{
				ContractType: "RBACTimelock",
				Tags:         tags,
			},
			To:               timelockAddress,
			Data:             data,
			AdditionalFields: additional,
		},
	}

	return []types.Operation{op}, operationID, nil
}

// OperationID returns the Soroban timelock operation id for the batch (same as on-chain
// hash_operation_batch). For bypass actions predecessor is treated as zero before hashing,
// matching schedule vs bypass semantics on-chain.
func OperationID(
	batchOp types.BatchOperation,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) (common.Hash, error) {
	calls, err := callsFromBatchOperation(batchOp)
	if err != nil {
		return common.Hash{}, err
	}

	pred := predecessor
	if action == types.TimelockActionBypass {
		pred = common.Hash{}
	}

	return HashOperationBatch(calls, pred, salt), nil
}
