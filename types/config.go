package types //nolint:revive

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

var ErrInvalidConfig = errors.New("invalid MCMS config")

// Config is a struct that holds all the configuration for the owner contracts
type Config struct {
	// Quorum is the minimum number of signers required to reach consensus. Quorum can be reached
	// by a ensuring that the sum of signers and group signers that have signed is greater than or
	// equal to the quorum.
	Quorum uint8 `json:"quorum"`

	// Signers is a list of all single signers in the config
	Signers []common.Address `json:"signers"`

	// GroupSigners is a list of all group signers. This is a recursive structure where each group
	// signer can have its own signers and group signers.
	GroupSigners []Config `json:"groupSigners"`
}

// AllSigners returns a deduplicated list of all individual signers recursively.
func (c *Config) AllSigners() []common.Address {
	seen := make(map[common.Address]struct{})
	var signers []common.Address

	var collect func(cfg *Config)
	collect = func(cfg *Config) {
		for _, signer := range cfg.Signers {
			if _, ok := seen[signer]; !ok {
				seen[signer] = struct{}{}
				signers = append(signers, signer)
			}
		}
		for i := range cfg.GroupSigners {
			collect(&cfg.GroupSigners[i])
		}
	}

	collect(c)

	return signers
}

// NewConfig returns a new config with the given quorum, signers and group signers and ensures it
// is valid.
func NewConfig(quorum uint8, signers []common.Address, groupSigners []Config) (Config, error) {
	config := Config{
		Quorum:       quorum,
		Signers:      signers,
		GroupSigners: groupSigners,
	}

	if err := config.Validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// Validate checks if the config is valid, recursively checking all group signers configs.
func (c *Config) Validate() error {
	if c.Quorum == 0 {
		return fmt.Errorf("%w: Quorum must be greater than 0", ErrInvalidConfig)
	}

	if len(c.Signers) == 0 && len(c.GroupSigners) == 0 {
		return fmt.Errorf("%w: Config must have at least one signer or group", ErrInvalidConfig)
	}

	if (len(c.Signers) + len(c.GroupSigners)) < int(c.Quorum) {
		return fmt.Errorf("%w: Quorum must be less than or equal to the number of signers and groups", ErrInvalidConfig)
	}

	for _, groupSigner := range c.GroupSigners {
		if err := groupSigner.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Equals checks if two configs are equal, recursively checking all group signers configs.
func (c *Config) Equals(other *Config) bool {
	if c.Quorum != other.Quorum {
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

// GetAllSigners returns all signers in the config and all group signers.
func (c *Config) GetAllSigners() []common.Address {
	signers := make([]common.Address, 0)
	signers = append(signers, c.Signers...)

	for _, groupSigner := range c.GroupSigners {
		signers = append(signers, groupSigner.GetAllSigners()...)
	}

	return signers
}

// CanSetRoot checks if the recovered signers have reached consensus to set the root.
func (c *Config) CanSetRoot(recoveredSigners []common.Address) (bool, error) {
	allSigners := c.GetAllSigners()
	for _, recoveredSigner := range recoveredSigners {
		if !slices.Contains(allSigners, recoveredSigner) {
			// Q: We can't import tha mcms main package here. Should we move every implementation out of types package?
			return false, fmt.Errorf("recovered signer %s is not a valid signer in the MCMS proposal", recoveredSigner)
		}
	}

	return c.isGroupAtConsensus(recoveredSigners), nil
}

// isGroupAtConsensus checks if the recovered signers are at consensus for the group.
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

// unorderedArrayEquals checks if two arrays are equal regardless of order.
func unorderedArrayEquals[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	countMap := make(map[T]int)

	// Count occurrences in the first slice
	for _, elem := range a {
		countMap[elem]++
	}

	// Subtract occurrences using the second slice
	for _, elem := range b {
		if countMap[elem] == 0 {
			return false
		}
		countMap[elem]--
	}

	// If slices are equal, all counts should be zero
	return true
}
