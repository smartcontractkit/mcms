package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		json1       []byte
		json2       []byte
		expected    []byte
		expectError bool
	}{
		{
			json1:    []byte(`{"a": 1, "b": 2}`),
			json2:    []byte(`{"b": 3, "c": 4}`),
			expected: []byte(`{"a":1,"b":3,"c":4}`), // json2 should overwrite json1's "b"
		},
		{
			json1:    []byte(`{"a": 1}`),
			json2:    []byte(`{"b": 2}`),
			expected: []byte(`{"a":1,"b":2}`), // json1 and json2 have no overlapping keys
		},
		{
			json1:       []byte(`{"a": 1}`),
			json2:       []byte(`invalid json`),
			expectError: true, // json2 is invalid JSON
		},
		{
			json1:       []byte(`invalid json`),
			json2:       []byte(`{"b": 2}`),
			expectError: true, // json1 is invalid JSON
		},
	}

	for _, test := range tests {
		result, err := Merge(test.json1, test.json2)

		if test.expectError {
			if err == nil {
				t.Errorf("expected an error but got none for json1: %s and json2: %s", test.json1, test.json2)
			}
		} else {
			assert.Equal(t, test.expected, result)
		}
	}
}
