package manifest

import (
	"encoding/json"
)

// GetVersion returns the version of the proposal manifest.
func GetVersion(data []byte) (string, error) {
	var meta BaseProposal

	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}

	return meta.Version, nil
}

// GetKind returns the kind of the proposal manifest.
func GetKind(data []byte) (string, error) {
	var meta BaseProposal

	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}

	return meta.Kind, nil
}
