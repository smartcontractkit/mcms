package stellar

import (
	"fmt"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"
)

func TestDecodeContractErrorFrames(t *testing.T) {
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

	frames, err := DecodeContractErrorFrames([]string{
		firstEvent,
		secondEvent,
	})
	require.NoError(t, err)
	require.Len(t, frames, 2)

	firstAddress, err := diagnosticContractID(&firstContractID)
	require.NoError(t, err)

	secondAddress, err := diagnosticContractID(&secondContractID)
	require.NoError(t, err)

	require.Equal(t, ContractError{
		ContractID: firstAddress,
		Code:       12,
	}, frames[0])

	require.Equal(t, ContractError{
		ContractID: secondAddress,
		Code:       32,
	}, frames[1])
}

func TestDecodeContractErrorFrames_ErrorInData(t *testing.T) {
	t.Parallel()

	contractID := testContractID(3)
	event := encodeContractErrorDataEvent(
		t,
		contractID,
		45,
	)

	frames, err := DecodeContractErrorFrames([]string{event})
	require.NoError(t, err)
	require.Len(t, frames, 1)

	address, err := diagnosticContractID(&contractID)
	require.NoError(t, err)

	require.Equal(t, ContractError{
		ContractID: address,
		Code:       45,
	}, frames[0])
}

func TestDecodeContractErrorFrames_IgnoresEventWithoutError(
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
		xdr.ScVal{Type: xdr.ScValTypeScvVoid},
	)

	encoded, err := xdr.MarshalBase64(event)
	require.NoError(t, err)

	frames, err := DecodeContractErrorFrames([]string{encoded})
	require.NoError(t, err)
	require.Empty(t, frames)
}

func TestDecodeContractErrorFrames_PreservesValidFramesWhenOneEventIsMalformed(
	t *testing.T,
) {
	t.Parallel()

	contractID := testContractID(4)
	validEvent := encodeContractErrorEvent(
		t,
		contractID,
		12,
	)

	frames, err := DecodeContractErrorFrames([]string{
		"not-valid-base64-xdr",
		validEvent,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "decode diagnostic event 0")
	require.Len(t, frames, 1)
	require.Equal(t, uint32(12), frames[0].Code)
}

func TestDecodeContractErrorFrames_NoEvents(t *testing.T) {
	t.Parallel()

	frames, err := DecodeContractErrorFrames(nil)
	require.NoError(t, err)
	require.Empty(t, frames)
}

func TestDecodeSimulationErrorFrames_UnwrapsSimulationError(
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
		Message:             "Error(Contract, #12)",
		DiagnosticEventsXDR: []string{event},
	}

	wrapped := fmt.Errorf(
		"stellar mcms set_config: %w",
		simulationErr,
	)

	frames, err := DecodeSimulationErrorFrames(wrapped)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	require.Equal(t, uint32(12), frames[0].Code)
}

func TestDecodeSimulationErrorFrames_RejectsUnrelatedError(
	t *testing.T,
) {
	t.Parallel()

	frames, err := DecodeSimulationErrorFrames(
		fmt.Errorf("unrelated failure"),
	)

	require.Error(t, err)
	require.ErrorContains(
		t,
		err,
		"does not contain a Stellar SimulationError",
	)
	require.Empty(t, frames)
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
		xdr.ScVal{Type: xdr.ScValTypeScvVoid},
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

func encodeNonContractErrorEvent(t *testing.T) string {
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
		xdr.ScVal{Type: xdr.ScValTypeScvVoid},
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

func TestDecodeContractErrorFrames_RealOutOfBoundsGroupFixture(
	t *testing.T,
) {
	t.Parallel()

	events := []string{
		// Paste the real EventsXDR values here.
	}

	frames, err := DecodeContractErrorFrames(events)
	require.NoError(t, err)

	for _, frame := range frames {
		t.Logf(
			"contract=%s code=%d",
			frame.ContractID,
			frame.Code,
		)
	}

	require.NotEmpty(t, frames)
	require.Equal(t, uint32(12), frames[len(frames)-1].Code)
}
