package sui

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressLen(t *testing.T) {
	t.Parallel()

	// Test that the address length constant is correct
	assert.Equal(t, 32, AddressLen)

	// Test that Address type has the correct size
	var addr Address
	assert.Equal(t, AddressLen, len(addr))
	assert.Equal(t, 32, len(addr))
}

func TestAddressFromHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid full address with 0x prefix",
			input:    "0x0000000000000000000000000000000000000000000000000000000000000001",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:     "valid full address with 0X prefix",
			input:    "0X0000000000000000000000000000000000000000000000000000000000000001",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:     "valid full address without prefix",
			input:    "0000000000000000000000000000000000000000000000000000000000000001",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:     "short address with 0x prefix - gets padded",
			input:    "0x1",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:     "short address without prefix - gets padded",
			input:    "1",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:     "odd length address gets padded with leading zero",
			input:    "0x123",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 35},
		},
		{
			name:     "maximum valid address",
			input:    "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expected: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			name:     "zero address",
			input:    "0x0",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "empty string after prefix removal",
			input:    "0x",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "mixed case hex",
			input:    "0xAbCdEf",
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xAB, 0xCD, 0xEF},
		},
		{
			name:        "invalid hex characters",
			input:       "0xgg",
			expectError: true,
			errorMsg:    "invalid byte",
		},
		{
			name:        "address too long",
			input:       "0x1" + strings.Repeat("0", 65), // 66 hex chars = 33 bytes
			expectError: true,
			errorMsg:    "address length exceeds 32 bytes",
		},
		{
			name:        "non-hex characters",
			input:       "0xhello",
			expectError: true,
			errorMsg:    "invalid byte",
		},
		{
			name:        "invalid prefix",
			input:       "1x123",
			expectError: true,
			errorMsg:    "invalid byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			addr, err := AddressFromHex(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, addr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, addr)
				assert.Equal(t, tt.expected, addr.Bytes())
				assert.Equal(t, AddressLen, len(addr.Bytes()))
			}
		})
	}
}

func TestAddress_Bytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  Address
		expected []byte
	}{
		{
			name:     "zero address",
			address:  Address{},
			expected: make([]byte, 32),
		},
		{
			name:     "address with single byte",
			address:  Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:     "maximum address",
			address:  Address{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			expected: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			name:     "mixed pattern address",
			address:  Address{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
			expected: []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.address.Bytes()
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, AddressLen, len(result))

			// Verify that modifying the returned slice doesn't affect the original address
			originalAddr := tt.address
			result[0] = ^result[0] // flip all bits in the first byte
			assert.Equal(t, originalAddr, tt.address, "original address should not be modified")
		})
	}
}

func TestAddress_Hex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  Address
		expected string
	}{
		{
			name:     "zero address",
			address:  Address{},
			expected: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:     "address with single byte",
			address:  Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			expected: "0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			name:     "maximum address",
			address:  Address{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			expected: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			name:     "mixed pattern address",
			address:  Address{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
			expected: "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899",
		},
		{
			name:     "address with leading zeros",
			address:  Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x12, 0x34, 0x56, 0x78},
			expected: "0000000000000000000000000000000000000000000000000000000012345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.address.Hex()
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, 64, len(result), "hex string should always be 64 characters (32 bytes * 2)")

			// Verify the result is valid hex
			_, err := hex.DecodeString(result)
			assert.NoError(t, err, "result should be valid hex")
		})
	}
}

func TestAddressIntegration(t *testing.T) {
	t.Parallel()

	// Test comprehensive functionality
	testHex := "0x123456789abcdef0fedcba9876543210"

	// Parse from hex
	addr, err := AddressFromHex(testHex)
	require.NoError(t, err)

	// Get bytes
	bytes := addr.Bytes()
	assert.Equal(t, AddressLen, len(bytes))

	// Get hex (without 0x prefix)
	hex := addr.Hex()
	assert.Equal(t, 64, len(hex))

	// Verify round-trip consistency
	addr2, err := AddressFromHex("0x" + hex)
	require.NoError(t, err)
	assert.Equal(t, *addr, *addr2)

	// Verify bytes equality
	assert.Equal(t, bytes, addr2.Bytes())

	// Test that the address maintains its structure
	assert.Equal(t, AddressLen, len(*addr))

	// Test zero value
	var zeroAddr Address
	assert.Equal(t, make([]byte, AddressLen), zeroAddr.Bytes())
	assert.Equal(t, strings.Repeat("0", 64), zeroAddr.Hex())
}
