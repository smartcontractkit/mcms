package ton

import (
	"crypto/rand"
	"math/big"
)

func RandomQueryID() (uint64, error) {
	max := big.NewInt(1 << 62)
	nBig, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return nBig.Uint64(), nil
}
