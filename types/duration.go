package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration wraps time.Duration with support for JSON encoding.
type Duration struct {
	time.Duration
}

// NewDuration wraps a time.Duration with a Duration.
func NewDuration(d time.Duration) Duration {
	return Duration{Duration: d}
}

// ParseDuration parses a duration string in the time.Duration format.
func ParseDuration(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return Duration{}, err
	}

	return NewDuration(d), nil
}

// MustParseDuration parses a duration string in the time.Duration format.
// Panics if the string is invalid.
//
// Useful for tests, but should be avoided in production code.
func MustParseDuration(s string) Duration {
	d, err := ParseDuration(s)
	if err != nil {
		panic(err)
	}

	return d
}

// String returns a string representing the duration in the form "72h3m0.5s".
func (d Duration) String() string {
	return d.Duration.String()
}

// MarshalJSON marshals the duration into JSON bytes and implements the json.Marshaler interface.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON unmarshals the duration from JSON bytes and implements the json.Unmarshaler
// interface.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case string:
		var err error
		if d.Duration, err = time.ParseDuration(value); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("invalid duration type: %T", v)
	}
}
