package stellar

// MCMS-Stellar numeric constants (ABI packing; see chainlink-stellar contracts/mcms/src/abi_encoding.rs).
const (
	abiWordBytes = 32

	uint40BitWidth     = 40
	uint40MaxExclusive = uint64(1) << uint40BitWidth
	// Low 40 bits of a uint64 in big-endian occupy the last 5 bytes.
	uint40TailBytes = 5
	uint64ByteLen   = 8

	stellarContractIDBytes = abiWordBytes
	// Network / contract ids are 32-byte hashes; hex form without 0x is 64 characters.
	stellarChainHexCharLen = stellarContractIDBytes * 2

	stellarOpStaticWordCount = 6
	// Byte offset of dynamic `data` from start of StellarOp head (6 words × 32 bytes).
	stellarOpDataABIByteOffset = stellarOpStaticWordCount * abiWordBytes

	hexRadix = 16
	// Bits representable in StellarOp.value / metadata words as uint256.
	uint256BitWidth = 256

	hexPrefixLen = 2 // "0x" / "0X"

	uint32ByteLen = 4

	// Solidity ABI: address is 20 bytes right-aligned in a 32-byte word (same padding as Stellar MCMS signers).
	evmAddressABIWordLeadingZeroBytes = 12
)

// Domain separators — must match chainlink-stellar contracts/mcms/src/constants.rs
// (keccak256 of the ASCII strings below).

var (
	// domainOpStellar = keccak256("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_STELLAR")
	domainOpStellar = [32]byte{
		0x12, 0xcd, 0xc8, 0x8e, 0x33, 0xb5, 0x9a, 0x3a, 0x5a, 0x9f, 0xe3, 0x07, 0x2e, 0x0b, 0xab, 0x63,
		0xee, 0x3d, 0xb8, 0x88, 0xaf, 0x2c, 0xdb, 0x10, 0xbc, 0x93, 0x34, 0x56, 0x88, 0x05, 0x8d, 0x16,
	}
	// domainMetaStellar = keccak256("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_STELLAR")
	domainMetaStellar = [32]byte{
		0xde, 0x51, 0xf2, 0xd6, 0x7b, 0xb4, 0x89, 0x5d, 0x0d, 0xd1, 0xf3, 0x6a, 0xdb, 0x04, 0x42, 0x27,
		0xaa, 0x7b, 0x76, 0x4d, 0x4e, 0x52, 0x4d, 0x6b, 0x0d, 0x70, 0x04, 0x72, 0x27, 0x28, 0xfd, 0xa0,
	}
)
