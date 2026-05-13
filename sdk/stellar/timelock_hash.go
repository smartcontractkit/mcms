package stellar

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"
)

// HashOperationBatch matches Soroban RBACTimelock
// hash_operation_batch_internal in chainlink-stellar/contracts/timelock/src/lib.rs:
//
//	call_hash_i = keccak256(to_i || keccak256(data_i))
//	id = keccak256(n_calls_u256_be || call_hash_0 || … || call_hash_n || predecessor || salt)
func HashOperationBatch(calls []timelockbindings.Call, predecessor, salt common.Hash) common.Hash {
	var buf []byte

	n := uint64(len(calls))
	var nWord [32]byte
	binary.BigEndian.PutUint64(nWord[24:32], n)
	buf = append(buf, nWord[:]...)

	for _, c := range calls {
		h := hashSingleCall(c)
		buf = append(buf, h[:]...)
	}

	buf = append(buf, predecessor[:]...)
	buf = append(buf, salt[:]...)

	return crypto.Keccak256Hash(buf)
}

func hashSingleCall(c timelockbindings.Call) common.Hash {
	dataHash := crypto.Keccak256Hash(c.Data)
	var concat [64]byte
	copy(concat[:32], c.To[:])
	copy(concat[32:], dataHash[:])

	return crypto.Keccak256Hash(concat[:])
}
