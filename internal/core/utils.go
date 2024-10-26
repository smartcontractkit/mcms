package core

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cast"
)

const FilePermissionsUserOnly = 0600

func TransformHashes(hashes []common.Hash) [][32]byte {
	m := make([][32]byte, len(hashes))
	for i, h := range hashes {
		m[i] = [32]byte(h)
	}

	return m
}

// FromFile generic function to read a file and unmarshal its contents into the provided struct
func FromFile(filePath string, out any) error {
	// Load file from path
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Unmarshal JSON into the provided struct
	err = json.Unmarshal(fileBytes, out)
	if err != nil {
		return err
	}

	return nil
}

// WriteProposalToFile writes a proposal to the provided file path
func WriteProposalToFile(proposal any, filePath string) error {
	proposalBytes, err := json.Marshal(proposal)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, proposalBytes, FilePermissionsUserOnly)
	if err != nil {
		return err
	}

	return nil
}

// SafeCastIntToUint32 safely converts an int to uint32 using cast and checks for overflow
func SafeCastIntToUint32(value int) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("value %d exceeds uint32 range", value)
	}

	return cast.ToUint32E(value)
}

// SafeCastInt64ToUint32 safely converts an int64 to uint32 and checks for overflow and negative values
func SafeCastInt64ToUint32(value int64) (uint32, error) {
	// Check if the value is negative
	if value < 0 {
		return 0, fmt.Errorf("value %d is negative and cannot be converted to uint32", value)
	}

	// Check if the value exceeds the maximum value of uint32
	if value > int64(math.MaxUint32) {
		return 0, fmt.Errorf("value %d exceeds uint32 range", value)
	}

	// Use cast to convert value to uint32 safely
	return cast.ToUint32E(value)
}

// SafeCastUint64ToUint8 safely converts an int to uint8 using cast and checks for overflow
func SafeCastUint64ToUint8(value uint64) (uint8, error) {
	if value > math.MaxUint8 {
		return 0, fmt.Errorf("value %d exceeds uint8 range", value)
	}

	return cast.ToUint8E(value)
}

// SafeCastUint64ToUint32 safely converts an int to uint32 using cast and checks for overflow
func SafeCastUint64ToUint32(value uint64) (uint32, error) {
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("value %d exceeds uint32 range", value)
	}

	return cast.ToUint32E(value)
}
