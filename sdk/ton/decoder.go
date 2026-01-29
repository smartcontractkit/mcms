package ton

import (
	"fmt"

	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/codec"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
)

type decoder struct {
	// Map of contract type to TL-B definitions (type -> opcode -> TL-B struct)
	TypeToTLBMap map[string]tvm.TLBMap
}

var _ sdk.Decoder = &decoder{}

func NewDecoder(tlbs map[string]tvm.TLBMap) sdk.Decoder {
	return &decoder{
		TypeToTLBMap: tlbs,
	}
}

func (d *decoder) Decode(tx types.Transaction, contractInterfaces string) (sdk.DecodedOperation, error) {
	contractType := contractInterfaces
	tlbs, ok := d.TypeToTLBMap[contractType]
	if !ok {
		return nil, fmt.Errorf("decoding failed - unknown contract interface: %s", contractType)
	}

	datac, err := cell.FromBOC(tx.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid cell BOC data: %w", err)
	}

	// Handle message with no body - empty cell
	isEmpty := datac.RefsNum() == 0 && datac.BitsSize() == 0
	if isEmpty {
		return NewDecodedOperation(contractType, "", 0, map[string]any{}, []string{}, []any{})
	}

	msgType, msgDecoded, err := codec.DecodeTLBValToJSON(datac, tlbs)
	if err != nil {
		return nil, fmt.Errorf("error while JSON decoding message (cell) for contract %s: %w", contractType, err)
	}

	if msgType == "Cell" || msgType == "<nil>" { // on decoder fallback (not decoded)
		return nil, fmt.Errorf("failed to decode message for contract %s: %w", contractType, err)
	}

	// Extract the input keys and args (tree/map lvl 0)
	keys, err := codec.DecodeTLBStructKeys(datac, tlbs)
	if err != nil {
		return nil, fmt.Errorf("error while (struct) decoding message (cell) for contract %s: %w", contractType, err)
	}
	inputKeys := make([]string, len(keys))
	inputArgs := make([]any, len(keys))

	m, ok := msgDecoded.(map[string]any) // JSON normalized
	if !ok {
		return nil, fmt.Errorf("failed to cast as map %s: %w", contractType, err)
	}

	// Notice: sorting keys based on TL-B order (decoded map is unsorted)
	for i, k := range keys {
		inputKeys[i] = k
		inputArgs[i] = m[k]
	}

	msgOpcode, err := tvm.ExtractOpcode(datac)
	if err != nil {
		return nil, fmt.Errorf("failed to extract opcode: %w", err)
	}

	return NewDecodedOperation(contractType, msgType, msgOpcode, msgDecoded, inputKeys, inputArgs)
}
