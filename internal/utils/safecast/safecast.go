// Package safecast implements functions to safely cast types to avoid panics

package safecast

import (
	"fmt"
	"math"

	"github.com/spf13/cast"
)

// IntToUint8 safely converts an int to uint8 using cast and checks for overflow
func IntToUint8(value int) (uint8, error) {
	if value < 0 || value > math.MaxUint8 {
		return 0, fmt.Errorf("value %d exceeds uint8 range", value)
	}

	return cast.ToUint8E(value)
}

// Uint64ToUint8 safely converts an int64 to uint8 using cast and checks for overflow
func Uint64ToUint8(value uint64) (uint8, error) {
	if value > math.MaxUint8 {
		return 0, fmt.Errorf("value %d exceeds uint8 range", value)
	}

	return cast.ToUint8E(value)
}

// Uint64ToUint32 safely converts an uint64 to uint32 using cast and checks for overflow
func Uint64ToUint32(value uint64) (uint32, error) {
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("value %d exceeds uint32 range", value)
	}

	return cast.ToUint32E(value)
}

// Int64ToUint32 safely converts an Int64 to uint32 using cast and checks for overflow
func Int64ToUint32(value int64) (uint32, error) {
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("value %d exceeds uint32 range", value)
	}

	return cast.ToUint32E(value)
}

// Uint64ToInt64 safely converts a uint64 to int64 using cast and checks for overflow
func Uint64ToInt64(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("value %d exceeds int64 range", value)
	}

	return cast.ToInt64E(value)
}
