package stellar

import (
	"fmt"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"
)

func TestDecodeContractErrorChain(t *testing.T) {
	t.Parallel()

	firstContractID := testContractID(1)
	secondContractID := testContractID(2)

	firstEvent := encodeContractErrorEvent(
		t,
		firstContractID,
		12,
	)
	secondEvent := encodeContractErrorEvent(
		t,
		secondContractID,
		32,
	)

	errorChain, err := DecodeContractErrorChain([]string{
		firstEvent,
		secondEvent,
	})
	require.NoError(t, err)
	require.Len(t, errorChain, 2)

	firstAddress, err := diagnosticContractID(
		&firstContractID,
	)
	require.NoError(t, err)

	secondAddress, err := diagnosticContractID(
		&secondContractID,
	)
	require.NoError(t, err)

	require.Equal(t, ContractError{
		ContractID: firstAddress,
		Code:       12,
	}, errorChain[0])

	require.Equal(t, ContractError{
		ContractID: secondAddress,
		Code:       32,
	}, errorChain[1])
}

func TestDecodeContractErrorChain_ErrorInData(
	t *testing.T,
) {
	t.Parallel()

	contractID := testContractID(3)
	event := encodeContractErrorDataEvent(
		t,
		contractID,
		45,
	)

	errorChain, err := DecodeContractErrorChain(
		[]string{event},
	)
	require.NoError(t, err)
	require.Len(t, errorChain, 1)

	address, err := diagnosticContractID(&contractID)
	require.NoError(t, err)

	require.Equal(t, ContractError{
		ContractID: address,
		Code:       45,
	}, errorChain[0])
}

func TestDecodeContractErrorChain_IgnoresEventWithoutError(
	t *testing.T,
) {
	t.Parallel()

	event := diagnosticEvent(
		t,
		testContractID(6),
		[]xdr.ScVal{
			{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  new(xdr.ScSymbol("log")),
			},
		},
		xdr.ScVal{
			Type: xdr.ScValTypeScvVoid,
		},
	)

	encoded, err := xdr.MarshalBase64(event)
	require.NoError(t, err)

	errorChain, err := DecodeContractErrorChain(
		[]string{encoded},
	)
	require.NoError(t, err)
	require.Empty(t, errorChain)
}

func TestDecodeContractErrorChain_IgnoresNonContractError(
	t *testing.T,
) {
	t.Parallel()

	event := encodeNonContractErrorEvent(t)

	errorChain, err := DecodeContractErrorChain(
		[]string{event},
	)
	require.NoError(t, err)
	require.Empty(t, errorChain)
}

func TestDecodeContractErrorChain_PreservesValidErrorsWhenOneEventIsMalformed(
	t *testing.T,
) {
	t.Parallel()

	contractID := testContractID(4)
	validEvent := encodeContractErrorEvent(
		t,
		contractID,
		12,
	)

	errorChain, err := DecodeContractErrorChain([]string{
		"not-valid-base64-xdr",
		validEvent,
	})

	require.Error(t, err)
	require.ErrorContains(
		t,
		err,
		"decode diagnostic event 0",
	)
	require.Len(t, errorChain, 1)
	require.Equal(
		t,
		uint32(12),
		errorChain[0].Code,
	)
}

func TestDecodeContractErrorChain_NoEvents(
	t *testing.T,
) {
	t.Parallel()

	errorChain, err := DecodeContractErrorChain(nil)
	require.NoError(t, err)
	require.Empty(t, errorChain)
}

func TestDecodeSimulationErrorChain_UnwrapsSimulationError(
	t *testing.T,
) {
	t.Parallel()

	contractID := testContractID(5)
	event := encodeContractErrorEvent(
		t,
		contractID,
		12,
	)

	simulationErr := &SimulationError{
		Message: "Error(Contract, #12)",
		DiagnosticEventsXDR: []string{
			event,
		},
	}

	wrapped := fmt.Errorf(
		"stellar mcms set_config: %w",
		simulationErr,
	)

	errorChain, err := DecodeSimulationErrorChain(wrapped)
	require.NoError(t, err)
	require.Len(t, errorChain, 1)
	require.Equal(
		t,
		uint32(12),
		errorChain[0].Code,
	)
}

func TestDecodeSimulationErrorChain_RejectsUnrelatedError(
	t *testing.T,
) {
	t.Parallel()

	errorChain, err := DecodeSimulationErrorChain(
		fmt.Errorf("unrelated failure"),
	)

	require.Error(t, err)
	require.ErrorContains(
		t,
		err,
		"does not contain a Stellar SimulationError",
	)
	require.Empty(t, errorChain)
}

func TestNewExecutionError_NormalizesErrorChain(
	t *testing.T,
) {
	t.Parallel()

	rootContractID := testContractID(1)
	outerContractID := testContractID(2)

	rootEvent := encodeContractErrorEvent(
		t,
		rootContractID,
		12,
	)
	outerEvent := encodeContractErrorEvent(
		t,
		outerContractID,
		32,
	)

	rootAddress, err := diagnosticContractID(
		&rootContractID,
	)
	require.NoError(t, err)

	outerAddress, err := diagnosticContractID(
		&outerContractID,
	)
	require.NoError(t, err)

	simulationErr := &SimulationError{
		Message: "contract execution failed",
		DiagnosticEventsXDR: []string{
			rootEvent,
			outerEvent,
		},
	}

	result := newExecutionError(
		nil,
		simulationErr,
		map[string]contractKind{
			rootAddress:  contractKindMCMS,
			outerAddress: contractKindTimelock,
		},
	)

	var executionErr *ExecutionError
	require.ErrorAs(t, result, &executionErr)
	require.Len(t, executionErr.ErrorChain, 2)

	require.Equal(t, ContractError{
		ContractID: outerAddress,
		Code:       32,
		Name:       "CallReverted",
	}, executionErr.ErrorChain[0])

	require.Equal(t, ContractError{
		ContractID: rootAddress,
		Code:       12,
		Name:       "OutOfBoundsGroup",
	}, executionErr.ErrorChain[1])

	require.Equal(
		t,
		executionErr.ErrorChain[0],
		*executionErr.OuterError(),
	)
	require.Equal(
		t,
		executionErr.ErrorChain[1],
		*executionErr.RootCause(),
	)
}

func testContractID(lastByte byte) xdr.ContractId {
	var contractID xdr.ContractId
	contractID[len(contractID)-1] = lastByte

	return contractID
}

func encodeContractErrorEvent(
	t *testing.T,
	contractID xdr.ContractId,
	code uint32,
) string {
	t.Helper()

	errorValue := contractErrorScVal(t, code)

	event := diagnosticEvent(
		t,
		contractID,
		[]xdr.ScVal{errorValue},
		xdr.ScVal{
			Type: xdr.ScValTypeScvVoid,
		},
	)

	encoded, err := xdr.MarshalBase64(event)
	require.NoError(t, err)

	return encoded
}

func encodeContractErrorDataEvent(
	t *testing.T,
	contractID xdr.ContractId,
	code uint32,
) string {
	t.Helper()

	event := diagnosticEvent(
		t,
		contractID,
		nil,
		contractErrorScVal(t, code),
	)

	encoded, err := xdr.MarshalBase64(event)
	require.NoError(t, err)

	return encoded
}

func encodeNonContractErrorEvent(
	t *testing.T,
) string {
	t.Helper()

	scError, err := xdr.NewScError(
		xdr.ScErrorTypeSceContext,
		xdr.ScErrorCodeScecInvalidAction,
	)
	require.NoError(t, err)

	errorValue, err := xdr.NewScVal(
		xdr.ScValTypeScvError,
		scError,
	)
	require.NoError(t, err)

	event := diagnosticEvent(
		t,
		testContractID(6),
		[]xdr.ScVal{errorValue},
		xdr.ScVal{
			Type: xdr.ScValTypeScvVoid,
		},
	)

	encoded, err := xdr.MarshalBase64(event)
	require.NoError(t, err)

	return encoded
}

func contractErrorScVal(
	t *testing.T,
	code uint32,
) xdr.ScVal {
	t.Helper()

	scError, err := xdr.NewScError(
		xdr.ScErrorTypeSceContract,
		xdr.Uint32(code),
	)
	require.NoError(t, err)

	value, err := xdr.NewScVal(
		xdr.ScValTypeScvError,
		scError,
	)
	require.NoError(t, err)

	return value
}

func diagnosticEvent(
	t *testing.T,
	contractID xdr.ContractId,
	topics []xdr.ScVal,
	data xdr.ScVal,
) xdr.DiagnosticEvent {
	t.Helper()

	body, err := xdr.NewContractEventBody(
		0,
		xdr.ContractEventV0{
			Topics: topics,
			Data:   data,
		},
	)
	require.NoError(t, err)

	return xdr.DiagnosticEvent{
		InSuccessfulContractCall: false,
		Event: xdr.ContractEvent{
			ContractId: &contractID,
			Type:       xdr.ContractEventTypeDiagnostic,
			Body:       body,
		},
	}
}
