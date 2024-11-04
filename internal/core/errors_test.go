package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err      error
		expected string
	}{
		{&InvalidChainIDError{ReceivedChainID: 1}, "invalid chain ID: 1"},
		{&InvalidChainIDError{ReceivedChainID: 0}, "invalid chain ID: 0"},
		{ErrEmptyDescription, "invalid empty description"},
		{&InvalidDelayError{ReceivedDelay: "1"}, "invalid delay: 1"},
		{&InvalidProposalTypeError{ReceivedProposalType: "type"}, "invalid proposal type: type"},
		{&InvalidTimelockOperationError{ReceivedTimelockOperation: "operation"}, "invalid timelock operation: operation"},
		{&InvalidValidUntilError{ReceivedValidUntil: 1}, "invalid valid until: 1"},
		{&InvalidValidUntilError{ReceivedValidUntil: 0}, "invalid valid until: 0"},
		{&InvalidVersionError{ReceivedVersion: "version"}, "invalid version: version"},
		{&MissingChainDetailsError{ChainIdentifier: 1, Parameter: "parameter"}, "missing parameter for chain 1"},
		{ErrNoChainMetadata, "no chain metadata"},
		{ErrNoTransactions, "no transactions"},
		{ErrNoTransactionsInBatch, "no transactions in batch"},
		{&InvalidSignatureError{
			RecoveredAddress: common.HexToAddress("0x2"),
		}, "invalid signature: received signature for address 0x0000000000000000000000000000000000000002 is not a signer on the MCMS contract"},
		{&InvalidMCMSConfigError{Reason: "reason"}, "invalid MCMS config: reason"},
		{&TooManySignersError{NumSigners: 1}, "too many signers: 1 max number is 255"},
	}

	for _, test := range tests {
		got := test.err.Error()
		if got != test.expected {
			assert.Equal(t, test.expected, test.err.Error())
		}
	}
}
