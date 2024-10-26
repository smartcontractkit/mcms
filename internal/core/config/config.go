package config

import (
	"slices"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core"
)

// Config is a struct that holds all the configuration for the owner contracts
type Config struct {
	Quorum       uint8            `json:"quorum"`
	Signers      []common.Address `json:"signers"`
	GroupSigners []Config         `json:"groupSigners"`
}

func NewConfig(quorum uint8, signers []common.Address, groupSigners []Config) (*Config, error) {
	config := Config{
		Quorum:       quorum,
		Signers:      signers,
		GroupSigners: groupSigners,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.Quorum == 0 {
		return &core.InvalidMCMSConfigError{
			Reason: "Quorum must be greater than 0",
		}
	}

	if len(c.Signers) == 0 && len(c.GroupSigners) == 0 {
		return &core.InvalidMCMSConfigError{
			Reason: "Config must have at least one signer or group",
		}
	}

	if (len(c.Signers) + len(c.GroupSigners)) < int(c.Quorum) {
		return &core.InvalidMCMSConfigError{
			Reason: "Quorum must be less than or equal to the number of signers and groups",
		}
	}

	for _, groupSigner := range c.GroupSigners {
		if err := groupSigner.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) Equals(other *Config) bool {
	if c.Quorum != other.Quorum {
		return false
	}

	if len(c.Signers) != len(other.Signers) {
		return false
	}

	// Compare signers (order doesn't matter)
	if !unorderedArrayEquals(c.Signers, other.Signers) {
		return false
	}

	if len(c.GroupSigners) != len(other.GroupSigners) {
		return false
	}

	// Compare all group signers in first exist in second (order doesn't matter)
	// the reverse is not necessary because the lengths are already checked
	for i := range c.GroupSigners {
		found := false
		for j := range other.GroupSigners {
			if c.GroupSigners[i].Equals(&other.GroupSigners[j]) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func unorderedArrayEquals[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[T]struct{})
	bMap := make(map[T]struct{})

	for _, i := range a {
		aMap[i] = struct{}{}
	}

	for _, i := range b {
		bMap[i] = struct{}{}
	}

	for _, i := range a {
		if _, ok := bMap[i]; !ok {
			return false
		}
	}

	for _, i := range b {
		if _, ok := aMap[i]; !ok {
			return false
		}
	}

	return true
}

func (c *Config) GetAllSigners() []common.Address {
	signers := make([]common.Address, 0)
	signers = append(signers, c.Signers...)

	for _, groupSigner := range c.GroupSigners {
		signers = append(signers, groupSigner.GetAllSigners()...)
	}

	return signers
}

func (c *Config) CanSetRoot(recoveredSigners []common.Address) (bool, error) {
	allSigners := c.GetAllSigners()
	for _, recoveredSigner := range recoveredSigners {
		if !slices.Contains(allSigners, recoveredSigner) {
			return false, &core.InvalidSignatureError{
				RecoveredAddress: recoveredSigner,
			}
		}
	}

	return c.isGroupAtConsensus(recoveredSigners), nil
}

func (c *Config) isGroupAtConsensus(recoveredSigners []common.Address) bool {
	signerApprovalsInGroup := 0
	for _, signer := range c.Signers {
		for _, recoveredSigner := range recoveredSigners {
			if signer == recoveredSigner {
				signerApprovalsInGroup++
				break
			}
		}
	}

	groupApprovals := 0
	for _, groupSigner := range c.GroupSigners {
		if groupSigner.isGroupAtConsensus(recoveredSigners) {
			groupApprovals++
		}
	}

	return (signerApprovalsInGroup + groupApprovals) >= int(c.Quorum)
}
