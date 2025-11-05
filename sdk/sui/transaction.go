package sui

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/block-vision/sui-go-sdk/models"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
)

func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	fields := AdditionalFields{}
	if err := json.Unmarshal(additionalFields, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal Sui additional fields: %w", err)
	}

	if err := fields.Validate(); err != nil {
		return err
	}

	return nil
}

func (af AdditionalFields) Validate() error {
	if len(af.ModuleName) <= 0 || len(af.ModuleName) > 64 {
		return errors.New("module name length must be between 1 and 64 characters")
	}
	if len(af.Function) <= 0 || len(af.Function) > 64 {
		return errors.New("function length must be between 1 and 64 characters")
	}

	return nil
}

func NewTransaction(moduleName, function string, to string, data []byte, contractType string, tags []string) (types.Transaction, error) {
	return NewTransactionWithStateObj(moduleName, function, to, data, contractType, tags, "", nil)
}

func newTransactionWithManyStateObj(moduleName, function string, to string, data []byte, contractType string, tags []string, stateObj string, typeArgs []string, internalStateObjects []string, internalTypeArgs [][]string) (types.Transaction, error) {
	additionalFields := AdditionalFields{
		ModuleName:           moduleName,
		Function:             function,
		StateObj:             stateObj,
		InternalStateObjects: internalStateObjects,
		TypeArgs:             typeArgs,
		InternalTypeArgs:     internalTypeArgs,
	}
	marshalledAdditionalFields, err := json.Marshal(additionalFields)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("failed to marshal additional fields: %w", err)
	}

	return types.Transaction{
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
		To:               to,
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
	}, nil
}

func NewTransactionWithStateObj(moduleName, function string, to string, data []byte, contractType string, tags []string, stateObj string, typeArgs []string) (types.Transaction, error) {
	return newTransactionWithManyStateObj(moduleName, function, to, data, contractType, tags, stateObj, typeArgs, nil, nil)
}

func NewTransactionWithUpgradeData(moduleName, function string, to string, data []byte, contractType string, tags []string, stateObj string, internalStateObjects []string, compiledModules [][]byte, dependencies []models.SuiAddress, packageToUpgrade string) (types.Transaction, error) {
	additionalFields := AdditionalFields{
		ModuleName:           moduleName,
		Function:             function,
		StateObj:             stateObj,
		InternalStateObjects: internalStateObjects,
		CompiledModules:      compiledModules,
		Dependencies:         dependencies,
		PackageToUpgrade:     packageToUpgrade,
	}
	marshalledAdditionalFields, err := json.Marshal(additionalFields)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("failed to marshal additional fields: %w", err)
	}

	return types.Transaction{
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
		To:               to,
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
	}, nil
}

// CreateUpgradeTransaction creates a transaction for upgrading a package through MCMS
func CreateUpgradeTransaction(compiledPackage bind.PackageArtifact, mcmsPackageID, depStateObj, registryObj, ownerCapObj, mcmsUserPackageId string) (types.Transaction, error) {
	upgradePolicy := uint8(0) // Compatible upgrade policy
	data, err := serializeAuthorizeUpgradeParams(ownerCapObj, depStateObj, upgradePolicy, compiledPackage.Digest, mcmsUserPackageId)
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
