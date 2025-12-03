package ton

import (
	"crypto/rand"
	"math"
	"math/big"
)

// TODO: move as tvm.SizeUINT160
const SizeUINT160 = 160
const SizeUINT256 = 256

func RandomQueryID() (uint64, error) {
	_max := new(big.Int).SetUint64(math.MaxUint64)
	nBig, err := rand.Int(rand.Reader, _max)
	if err != nil {
		return 0, err
	}

	return nBig.Uint64(), nil
}
