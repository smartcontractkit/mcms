package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{&InvalidChainIDError{ReceivedChainID: 1}, "invalid chain ID: 1"},
		{&InvalidChainIDError{ReceivedChainID: 0}, "invalid chain ID: 0"},
		{&EmptyDescriptionError{}, "invalid empty description"},
		{&InvalidMinDelayError{ReceivedMinDelay: "1"}, "invalid min delay: 1"},
		{&InvalidMinDelayError{ReceivedMinDelay: "0"}, "invalid min delay: 0"},
		{&InvalidProposalTypeError{ReceivedProposalType: "type"}, "invalid proposal type: type"},
		{&InvalidTimelockOperationError{ReceivedTimelockOperation: "operation"}, "invalid timelock operation: operation"},
		{&InvalidValidUntilError{ReceivedValidUntil: 1}, "invalid valid until: 1"},
		{&InvalidValidUntilError{ReceivedValidUntil: 0}, "invalid valid until: 0"},
		{&InvalidVersionError{ReceivedVersion: "version"}, "invalid version: version"},
		{&MissingChainDetailsError{ChainIdentifier: 1, Parameter: "parameter"}, "missing parameter for chain 1"},
		{&MissingChainClientError{ChainIdentifier: 1}, "missing chain client for chain 1"},
		{&NoChainMetadataError{}, "no chain metadata"},
		{&NoTransactionsError{}, "no transactions"},
		{&NoTransactionsInBatchError{}, "no transactions in batch"},
		{&InvalidSignatureError{
			ChainIdentifier:  1,
			MCMSAddress:      common.HexToAddress("0x1"),
			RecoveredAddress: common.HexToAddress("0x2"),
		}, "invalid signature: received signature for address 0x0000000000000000000000000000000000000002 is not a signer on MCMS 0x0000000000000000000000000000000000000001 on chain 1"},
		{&InvalidMCMSConfigError{Reason: "reason"}, "invalid MCMS config: reason"},
		{&QuorumNotMetError{ChainIdentifier: 1}, "quorum not met for chain 1"},
		{&InconsistentConfigsError{ChainIdentifierA: 1, ChainIdentifierB: 2}, "inconsistent configs for chains 1 and 2"},
		{&TooManySignersError{NumSigners: 1}, "too many signers: 1 max number is 255"},
	}

	for _, test := range tests {
		got := test.err.Error()
		if got != test.expected {
			assert.Equal(t, test.expected, test.err.Error())
		}
	}
}
