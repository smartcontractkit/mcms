package stellar

import (
	"errors"
	"fmt"

	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// DecodeSimulationErrorFrames finds a SimulationError anywhere in err's
// wrapped error chain and extracts all Soroban contract errors from its
// diagnostic events.
func DecodeSimulationErrorFrames(err error) ([]ContractError, error) {
	if err == nil {
		return nil, nil
	}

	var simulationErr *SimulationError
	if !errors.As(err, &simulationErr) {
		return nil, fmt.Errorf("error does not contain a Stellar SimulationError: %w", err)
	}

	return DecodeContractErrorFrames(
		simulationErr.DiagnosticEventsXDR,
	)
}

// DecodeContractErrorFrames extracts Soroban contract errors from
// base64-encoded DiagnosticEvent XDR values.
func DecodeContractErrorFrames(eventsXDR []string) ([]ContractError, error) {
	frames := make([]ContractError, 0)
	decodeErrors := make([]error, 0)

	for i, encoded := range eventsXDR {
		var event xdr.DiagnosticEvent
		if err := xdr.SafeUnmarshalBase64(encoded, &event); err != nil {
			decodeErrors = append(decodeErrors, fmt.Errorf("decode diagnostic event %d: %w", i, err))
			continue
		}

		eventFrames, err := contractErrorsFromDiagnosticEvent(event)
		if err != nil {
			decodeErrors = append(decodeErrors,
				fmt.Errorf("inspect diagnostic event %d: %w", i, err),
			)
		}

		frames = append(frames, eventFrames...)
	}

	return frames, errors.Join(decodeErrors...)
}

func contractErrorsFromDiagnosticEvent(event xdr.DiagnosticEvent) ([]ContractError, error) {
	body, ok := event.Event.Body.GetV0()
	if !ok {
		return nil, fmt.Errorf(
			"unsupported contract event body version %d",
			event.Event.Body.V,
		)
	}

	contractID, err := diagnosticContractID(event.Event.ContractId)
	if err != nil {
		return nil, err
	}

	codes := make([]uint32, 0)

	// Soroban error diagnostics commonly place the ScError in a topic,
	// but inspect both topics and data because the representation may vary
	// between host/protocol versions.
	for _, topic := range body.Topics {
		topicCodes, topicErr := contractErrorCodesFromScVal(topic, 0)
		if topicErr != nil {
			return nil, fmt.Errorf("inspect diagnostic topic: %w", topicErr)
		}

		codes = append(codes, topicCodes...)
	}

	dataCodes, dataErr := contractErrorCodesFromScVal(body.Data, 0)
	if dataErr != nil {
		return nil, fmt.Errorf("inspect diagnostic data: %w", dataErr)
	}

	codes = append(codes, dataCodes...)

	frames := make([]ContractError, 0, len(codes))
	for _, code := range codes {
		frames = append(frames, ContractError{
			ContractID: contractID,
			Code:       code,
		})
	}

	return frames, nil
}

// contractErrorCodesFromScVal recursively searches an ScVal for contract
// errors. It handles direct errors as well as errors nested in vectors/maps.
func contractErrorCodesFromScVal(value xdr.ScVal, depth int) ([]uint32, error) {
	if depth > maxScValErrorDecodeDepth {
		return nil, fmt.Errorf(
			"ScVal nesting exceeds maximum depth %d",
			maxScValErrorDecodeDepth,
		)
	}

	if scError, ok := value.GetError(); ok {
		contractCode, isContractError := scError.GetContractCode()
		if !isContractError {
			return nil, nil
		}

		return []uint32{uint32(contractCode)}, nil
	}

	if vector, ok := value.GetVec(); ok && vector != nil {
		codes := make([]uint32, 0)

		for _, item := range *vector {
			itemCodes, err := contractErrorCodesFromScVal(
				item,
				depth+1,
			)
			if err != nil {
				return nil, err
			}

			codes = append(codes, itemCodes...)
		}

		return codes, nil
	}

	if scMap, ok := value.GetMap(); ok && scMap != nil {
		codes := make([]uint32, 0)

		for _, entry := range *scMap {
			keyCodes, err := contractErrorCodesFromScVal(
				entry.Key,
				depth+1,
			)
			if err != nil {
				return nil, err
			}

			valueCodes, err := contractErrorCodesFromScVal(
				entry.Val,
				depth+1,
			)
			if err != nil {
				return nil, err
			}

			codes = append(codes, keyCodes...)
			codes = append(codes, valueCodes...)
		}

		return codes, nil
	}

	return nil, nil
}

func diagnosticContractID(
	contractID *xdr.ContractId,
) (string, error) {
	if contractID == nil {
		return "", nil
	}

	encoded, err := strkey.Encode(
		strkey.VersionByteContract,
		contractID[:],
	)
	if err != nil {
		return "", fmt.Errorf("encode diagnostic contract ID: %w", err)
	}

	return encoded, nil
}
