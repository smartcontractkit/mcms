package ton

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/ccip/ccipsendexecutor"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/ccip/feequoter"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/ccip/offramp"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/ccip/onramp"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/ccip/router"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/jetton/minter"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/jetton/wallet"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/decoders/mcms/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/debug/lib"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// Map of TLBs keyed by contract type
// TODO: unify and move these definitions to smartcontractkit/chainlink-ton
var TLBsByContract = map[string]map[uint64]interface{}{
	// Jetton contract types
	"com.github.ton-blockchain.jetton-contract.contracts.jetton-wallet": wallet.TLBs,
	"com.github.ton-blockchain.jetton-contract.contracts.jetton-minter": minter.TLBs,
	// CCIP contract types
	"com.chainlink.ton.ccip.Router":           router.TLBs,
	"com.chainlink.ton.ccip.OnRamp":           onramp.TLBs,
	"com.chainlink.ton.ccip.OffRamp":          offramp.TLBs,
	"com.chainlink.ton.ccip.FeeQuoter":        feequoter.TLBs,
	"com.chainlink.ton.ccip.CCIPSendExecutor": ccipsendexecutor.TLBs,
	// MCMS contract types
	"com.chainlink.ton.lib.access.RBAC": rbac.TLBs,
	"com.chainlink.ton.mcms.MCMS":       mcms.TLBs,
	"com.chainlink.ton.mcms.Timelock":   timelock.TLBs,
}

type decoder struct{}

var _ sdk.Decoder = &decoder{}

func NewDecoder() sdk.Decoder {
	return &decoder{}
}

func (d *decoder) Decode(tx types.Transaction, contractInterfaces string) (sdk.DecodedOperation, error) {
	idTLBs := contractInterfaces
	tlbs, ok := TLBsByContract[idTLBs]
	if !ok {
		return nil, fmt.Errorf("decoding failed - unknown contract interface: %s", idTLBs)
	}

	datac, err := cell.FromBOC(tx.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid cell BOC data: %w", err)
	}

	// TODO: handle empty cell
	msgType, msgDecoded, err := lib.DecodeTLBValToJSON(datac, tlbs)
	if err != nil {
		return nil, fmt.Errorf("error while decoding message for contract %s: %w", idTLBs, err)
	}

	if msgType == "Cell" || msgType == "<nil>" { // on decoder fallback (not decoded)
		return nil, fmt.Errorf("failed to decode message for contract %s: %w", idTLBs, err)
	}

	// Extract the input keys and args (tree/map lvl 0)
	inputKeys := make([]string, 0)
	inputArgs := make([]any, 0)

	m, ok := msgDecoded.(map[string]interface{}) // JSON normalized
	if !ok {
		return nil, fmt.Errorf("failed to decode message for contract %s: %w", idTLBs, err)
	}

	// TODO: do we consider sorting these based on TL-B order? (decoded map is unsorted)
	for k, v := range m {
		inputKeys = append(inputKeys, k)
		inputArgs = append(inputArgs, v)
	}

	msgOpcode := uint64(0) // not exposed currently
	return NewDecodedOperation(idTLBs, msgType, msgOpcode, msgDecoded, inputKeys, inputArgs)
}
