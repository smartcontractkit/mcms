package sui

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/block-vision/sui-go-sdk/models"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"

	"github.com/smartcontractkit/mcms/types"
)

// CreateUpgradeTransaction creates a transaction for upgrading a package through MCMS
func CreateUpgradeTransaction(compiledPackage bind.PackageArtifact, mcmsPackageID, depStateObj, registryObj, mcmsUserPackageId string) (types.Transaction, error) {
	upgradePolicy := uint8(0) // Compatible upgrade policy
	data, err := serializeAuthorizeUpgradeParams(upgradePolicy, compiledPackage.Digest, mcmsUserPackageId)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("serializing authorize upgrade params: %w", err)
	}

	// Convert modules from base64 strings to raw bytes
	moduleBytes := make([][]byte, len(compiledPackage.Modules))
	for i, moduleBase64 := range compiledPackage.Modules {
		decoded, err := base64.StdEncoding.DecodeString(moduleBase64)
		if err != nil {
			return types.Transaction{}, fmt.Errorf("decoding module %d: %w", i, err)
		}
		moduleBytes[i] = decoded
	}

	depAddresses := make([]models.SuiAddress, len(compiledPackage.Dependencies))
	for i, dep := range compiledPackage.Dependencies {
		depAddresses[i] = models.SuiAddress(dep)
	}

	// Create transaction targeting mcms_deployer::authorize_upgrade with upgrade data
	return NewTransactionWithUpgradeData(
		"mcms_deployer",             // Module name
		"authorize_upgrade",         // Function name
		mcmsPackageID,               // Package ID (mcms_deployer is in MCMS package)
		data,                        // BCS-encoded parameters
		"MCMS",                      // Contract type
		[]string{"upgrade", "mcms"}, // Tags
		depStateObj,                 // Main state object (DeployerState)
		[]string{registryObj},       // Internal objects (Registry for validation)
		moduleBytes,                 // Compiled modules for upgrade
		depAddresses,                // Dependencies for upgrade
		mcmsUserPackageId,           // Package being upgraded
	)
}

// serializeAuthorizeUpgradeParams serializes parameters for `mcms_deployer::authorize_upgrade`
func serializeAuthorizeUpgradeParams(policy uint8, digest []byte, packageAddress string) ([]byte, error) {
	// The authorize_upgrade function expects:
	// - policy: u8
	// - digest: vector<u8>
	// - package_address: address

	// Convert package address to bytes for BCS serialization
	packageAddrBytes, err := hex.DecodeString(strings.TrimPrefix(packageAddress, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode package address: %w", err)
	}

	// Use proper BCS serialization like the existing codebase
	return bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.U8(policy)                   // u8 policy
		ser.WriteBytes(digest)           // vector<u8> digest
		ser.FixedBytes(packageAddrBytes) // address (32-byte fixed bytes)
	})
}
