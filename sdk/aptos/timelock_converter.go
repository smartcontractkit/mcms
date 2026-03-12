package aptos

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"

	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	curse_mcms "github.com/smartcontractkit/chainlink-aptos/bindings/curse_mcms"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

// timelockEncoder is the subset of encoder methods needed for timelock conversion.
// Both mcms.MCMSEncoder and curse_mcms.CurseMCMSEncoder satisfy this interface.
type timelockEncoder interface {
	TimelockScheduleBatch(targets []aptos.AccountAddress, moduleNames []string, functionNames []string, datas [][]byte, predecessor []byte, salt []byte, delay uint64) (bind.ModuleInformation, string, []aptos.TypeTag, [][]byte, error)
	TimelockBypasserExecuteBatch(targets []aptos.AccountAddress, moduleNames []string, functionNames []string, datas [][]byte) (bind.ModuleInformation, string, []aptos.TypeTag, [][]byte, error)
	TimelockCancel(id []byte) (bind.ModuleInformation, string, []aptos.TypeTag, [][]byte, error)
}

type TimelockConverter struct {
	encoderFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) timelockEncoder
}

func NewTimelockConverter() *TimelockConverter {
	return &TimelockConverter{
		encoderFn: func(address aptos.AccountAddress, client aptos.AptosRpcClient) timelockEncoder {
			return mcms.Bind(address, client).MCMS().Encoder()
		},
	}
}

func NewCurseTimelockConverter() *TimelockConverter {
	return &TimelockConverter{
		encoderFn: func(address aptos.AccountAddress, client aptos.AptosRpcClient) timelockEncoder {
			return curse_mcms.Bind(address, client).CurseMCMS().Encoder()
		},
	}
}

func (t *TimelockConverter) ConvertBatchToChainOperations(
	_ context.Context,
	_ types.ChainMetadata,
	bop types.BatchOperation,
	_ string,
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	mcmsAddress, mcmsErr := hexToAddress(mcmAddress)
	if mcmsErr != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to parse MCMS address %q: %w", mcmAddress, mcmsErr)
	}
	encoder := t.encoderFn(mcmsAddress, nil)

	targets := make([]aptos.AccountAddress, len(bop.Transactions))
	moduleNames := make([]string, len(bop.Transactions))
	functionNames := make([]string, len(bop.Transactions))
	datas := make([][]byte, len(bop.Transactions))
	tags := make([]string, 0, len(bop.Transactions))

	for i, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}

		toAddress, err := hexToAddress(tx.To)
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to parse To address %q: %w", tx.To, err)
		}

		targets[i] = toAddress
		moduleNames[i] = additionalFields.ModuleName
		functionNames[i] = additionalFields.Function
		datas[i] = tx.Data
		tags = append(tags, tx.Tags...)
	}

	operationID, err := HashOperationBatch(targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes())
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to compute hash of batch operation: %w", err)
	}

	var (
		module   bind.ModuleInformation
		function string
		args     [][]byte
	)
	switch action {
	case types.TimelockActionSchedule:
		module, function, _, args, err = encoder.TimelockScheduleBatch(targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes(), uint64(delay.Seconds()))
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to encode timelock_schedule_batch: %w", err)
		}
	case types.TimelockActionBypass:
		module, function, _, args, err = encoder.TimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to encode timelock_bypasser_execute_batch: %w", err)
		}
	case types.TimelockActionCancel:
		module, function, _, args, err = encoder.TimelockCancel(operationID[:])
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to encode timelock_cancel: %w", err)
		}
	}

	tx, err := NewTransaction(
		module.PackageName,
		module.ModuleName,
		function,
		mcmsAddress,
		ArgsToData(args),
		"MCMSTimelock",
		tags,
	)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	op := types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction:   tx,
	}

	return []types.Operation{op}, operationID, nil
}

func HashOperationBatch(targets []aptos.AccountAddress, moduleNames, functionNames []string, datas [][]byte, predecessor, salt []byte) (common.Hash, error) {
	ser := bcs.Serializer{}
	//nolint:gosec
	ser.Uleb128(uint32(len(targets)))
	for i, target := range targets {
		moduleName := moduleNames[i]
		functionName := functionNames[i]
		data := datas[i]

		ser.Struct(&target)
		ser.WriteString(moduleName)
		ser.WriteString(functionName)
		ser.WriteBytes(data)
	}
	ser.FixedBytes(predecessor)
	ser.FixedBytes(salt)

	if err := ser.Error(); err != nil {
		return common.Hash{}, fmt.Errorf("failed to serialize batch operation: %w", err)
	}

	return crypto.Keccak256Hash(ser.ToBytes()), nil
}
