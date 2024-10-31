package core

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/common"
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
