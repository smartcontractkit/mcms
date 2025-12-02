package ton

import (
	"crypto/rand"
	"math/big"
)

func RandomQueryID() (uint64, error) {
	_max := big.NewInt(1 << 62)
	nBig, err := rand.Int(rand.Reader, _max)
	if err != nil {
		return 0, err
	}
	return nBig.Uint64(), nil
}
