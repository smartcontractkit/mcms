package canton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
	cselectors "github.com/smartcontractkit/chain-selectors"

	cantontypes "github.com/noders-team/go-daml/pkg/types"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Executor = &Executor{}

type Executor struct {
	*Encoder
	*Inspector
	client *client.DamlBindingClient
	userId string
	party  string
}

func NewExecutor(encoder *Encoder, inspector *Inspector, client *client.DamlBindingClient, userId string, party string) (*Executor, error) {
	return &Executor{
		Encoder:   encoder,
		Inspector: inspector,
		client:    client,
		userId:    userId,
		party:     party,
	}, nil
}

func (e Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	// Extract Canton-specific operation fields from AdditionalFields
	var cantonOpFields AdditionalFields
	if len(op.Transaction.AdditionalFields) > 0 {
		if err := json.Unmarshal(op.Transaction.AdditionalFields, &cantonOpFields); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal operation additional fields: %w", err)
		}
	}

	// Validate required Canton fields
	if cantonOpFields.TargetInstanceId == "" {
		return types.TransactionResult{}, errors.New("targetInstanceId is required in operation additional fields")
	}
	if cantonOpFields.FunctionName == "" {
		return types.TransactionResult{}, errors.New("functionName is required in operation additional fields")
	}
	if cantonOpFields.TargetCid == "" {
		return types.TransactionResult{}, errors.New("targetCid is required in operation additional fields")
	}

	// Extract metadata fields for chainId and multisigId
	var metadataFields struct {
		ChainId    int64  `json:"chainId"`
		MultisigId string `json:"multisigId"`
	}
	if len(metadata.AdditionalFields) > 0 {
		if err := json.Unmarshal(metadata.AdditionalFields, &metadataFields); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal metadata additional fields: %w", err)
		}
	}

	// Build Canton Op struct
	cantonOp := mcms.Op{
		ChainId:          cantontypes.INT64(metadataFields.ChainId),
		MultisigId:       cantontypes.TEXT(metadataFields.MultisigId),
		Nonce:            cantontypes.INT64(nonce),
		TargetInstanceId: cantontypes.TEXT(cantonOpFields.TargetInstanceId),
		FunctionName:     cantontypes.TEXT(cantonOpFields.FunctionName),
		OperationData:    cantontypes.TEXT(cantonOpFields.OperationData),
	}

	// Convert proof to Canton TEXT array
	opProof := make([]cantontypes.TEXT, len(proof))
	for i, p := range proof {
		opProof[i] = cantontypes.TEXT(hex.EncodeToString(p[:]))
	}

	// Convert contract IDs
	contractIds := make([]cantontypes.CONTRACT_ID, len(cantonOpFields.ContractIds))
	for i, cid := range cantonOpFields.ContractIds {
		contractIds[i] = cantontypes.CONTRACT_ID(cid)
	}

	// Build exercise command using generated bindings
	mcmsContract := mcms.MCMS{}
	var exerciseCmd *model.ExerciseCommand
	// Use different input struct depending on whether the operation is targeting the MCMS contract itself or another contract
	if cantonOpFields.TargetInstanceId == "self" {
		input := mcms.ExecuteMcmsOp{
			Submitter: cantontypes.PARTY(e.party),
			Op:        cantonOp,
			OpProof:   opProof,
		}
		exerciseCmd = mcmsContract.ExecuteMcmsOp(metadata.MCMAddress, input)
	} else {
		input := mcms.ExecuteOp{
			Submitter:   cantontypes.PARTY(e.party),
			TargetCid:   cantontypes.CONTRACT_ID(cantonOpFields.TargetCid),
			Op:          cantonOp,
			OpProof:     opProof,
			ContractIds: contractIds,
		}
		exerciseCmd = mcmsContract.ExecuteOp(metadata.MCMAddress, input)

	}

	// List known packages to find the package ID for mcms
	ListKnownPackagesResp, err := e.client.PackageMng.ListKnownPackages(ctx)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to list known packages: %w", err)
	}

	var mcmsPkgID string
	for _, p := range ListKnownPackagesResp {
		if strings.Contains(strings.ToLower(p.Name), "mcms") {
			mcmsPkgID = p.PackageID
			break
		}
	}
	if mcmsPkgID == "" {
		return types.TransactionResult{}, fmt.Errorf("failed to find mcms package")
	}

	commandID := uuid.Must(uuid.NewUUID()).String()
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: "mcms-execute-op",
			UserID:     e.userId,
			CommandID:  commandID,
			ActAs:      []string{e.party},
			Commands: []*model.Command{{
				Command: &model.ExerciseCommand{
					TemplateID: fmt.Sprintf("%s:%s:%s", mcmsPkgID, "MCMS.Main", "MCMS"),
					ContractID: exerciseCmd.ContractID,
					Choice:     exerciseCmd.Choice,
					Arguments:  exerciseCmd.Arguments,
				},
			}},
		},
	}

	submitResp, err := e.client.CommandService.SubmitAndWaitForTransaction(ctx, cmds)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute operation: %w", err)
	}

	// Extract NEW MCMS CID from Created event
	newMCMSContractID := ""
	newMCMSTemplateID := ""
	for _, ev := range submitResp.Transaction.Events {
		if ev.Created == nil {
			continue
		}
		normalized := NormalizeTemplateKey(ev.Created.TemplateID)
		if normalized == MCMSTemplateKey {
			newMCMSContractID = ev.Created.ContractID
			newMCMSTemplateID = ev.Created.TemplateID
			break
		}
	}

	if newMCMSContractID == "" {
		return types.TransactionResult{}, fmt.Errorf("execute-op tx had no Created MCMS event; refusing to continue with old CID=%s", metadata.MCMAddress)
	}

	return types.TransactionResult{
		Hash:        commandID,
		ChainFamily: cselectors.FamilyCanton,
		RawData: map[string]any{
			"NewMCMSContractID": newMCMSContractID,
			"NewMCMSTemplateID": newMCMSTemplateID,
			"RawTx":             submitResp,
		},
	}, nil
}

func (e Executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	// Calculate the hash to sign according to Canton's expectations, and extract signers from it
	rootHex := hex.EncodeToString(root[:])
	validUntilHexForSigning := strings.Repeat("0", 64) // TODO: Remove, Canton placeholder (64 zeros)
	concatenated := rootHex + validUntilHexForSigning

	innerData, err := hex.DecodeString(concatenated)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to decode hex for signing: %w", err)
	}
	innerHash := crypto.Keccak256(innerData)

	// Apply EIP-191 prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	prefixedData := append(prefix, innerHash...)
	cantonSignedHash := crypto.Keccak256Hash(prefixedData)

	// Convert signatures to Canton RawSignature array
	signatures := make([]mcms.RawSignature, len(sortedSignatures))
	for i, sig := range sortedSignatures {
		pubKey, err := sig.RecoverPublicKey(cantonSignedHash)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to recover public key for signature %d: %w", i, err)
		}

		// Convert public key to hex string
		pubkeyHex := hex.EncodeToString(crypto.FromECDSAPub(pubKey))
		signatures[i] = mcms.RawSignature{
			PublicKey: cantontypes.TEXT(pubkeyHex),
			R:         cantontypes.TEXT(hex.EncodeToString(sig.R[:])),
			S:         cantontypes.TEXT(hex.EncodeToString(sig.S[:])),
		}
	}

	// Extract root metadata from ChainMetadata.AdditionalFields
	var rootMetadata mcms.RootMetadata
	if len(metadata.AdditionalFields) > 0 {
		var additionalFields map[string]interface{}
		if err := json.Unmarshal(metadata.AdditionalFields, &additionalFields); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}

		// Extract fields with type assertions
		if chainId, ok := additionalFields["chainId"].(float64); ok {
			rootMetadata.ChainId = cantontypes.INT64(int64(chainId))
		}
		if multisigId, ok := additionalFields["multisigId"].(string); ok {
			rootMetadata.MultisigId = cantontypes.TEXT(multisigId)
		}
		if preOpCount, ok := additionalFields["preOpCount"].(float64); ok {
			rootMetadata.PreOpCount = cantontypes.INT64(int64(preOpCount))
		}
		if postOpCount, ok := additionalFields["postOpCount"].(float64); ok {
			rootMetadata.PostOpCount = cantontypes.INT64(int64(postOpCount))
		}
		if overridePreviousRoot, ok := additionalFields["overridePreviousRoot"].(bool); ok {
			rootMetadata.OverridePreviousRoot = cantontypes.BOOL(overridePreviousRoot)
		}
	}

	// Convert proof to Canton TEXT array
	metadataProof := make([]cantontypes.TEXT, len(proof))
	for i, p := range proof {
		metadataProof[i] = cantontypes.TEXT(hex.EncodeToString(p[:]))
	}

	validUntilTime := time.Unix(time.Unix(int64(validUntil), 0).UnixMicro(), 0)
	input := mcms.SetRoot{
		Submitter:     cantontypes.PARTY(e.party),
		NewRoot:       cantontypes.TEXT(rootHex),
		ValidUntil:    cantontypes.TIMESTAMP(validUntilTime),
		Metadata:      rootMetadata,
		MetadataProof: metadataProof,
		Signatures:    signatures,
	}

	// Build exercise command using generated bindings
	mcmsContract := mcms.MCMS{}
	exerciseCmd := mcmsContract.SetRoot(metadata.MCMAddress, input)

	ListKnownPackagesResp, err := e.client.PackageMng.ListKnownPackages(ctx)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to list known packages: %w", err)
	}

	var mcmsPkgID string
	for _, p := range ListKnownPackagesResp {
		if strings.Contains(strings.ToLower(p.Name), "mcms") {
			mcmsPkgID = p.PackageID
			break
		}
	}
	if mcmsPkgID == "" {
		return types.TransactionResult{}, fmt.Errorf("failed to find mcms package")
	}

	commandID := uuid.Must(uuid.NewUUID()).String()
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: "mcms-set-root",
			UserID:     e.userId,
			CommandID:  commandID,
			ActAs:      []string{e.party},
			Commands: []*model.Command{{
				Command: &model.ExerciseCommand{
					TemplateID: fmt.Sprintf("%s:%s:%s", mcmsPkgID, "MCMS.Main", "MCMS"),
					ContractID: exerciseCmd.ContractID,
					Choice:     exerciseCmd.Choice,
					Arguments:  exerciseCmd.Arguments,
				},
			}},
		},
	}

	submitResp, err := e.client.CommandService.SubmitAndWaitForTransaction(ctx, cmds)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set root: %w", err)
	}

	// Extract NEW MCMS CID from Created event
	newMCMSContractID := ""
	newMCMSTemplateID := ""
	for _, ev := range submitResp.Transaction.Events {
		if ev.Created == nil {
			continue
		}
		normalized := NormalizeTemplateKey(ev.Created.TemplateID)
		if normalized == MCMSTemplateKey {
			newMCMSContractID = ev.Created.ContractID
			newMCMSTemplateID = ev.Created.TemplateID
			break
		}
	}

	if newMCMSContractID == "" {
		return types.TransactionResult{}, fmt.Errorf("set-root tx had no Created MCMS event; refusing to continue with old CID=%s", metadata.MCMAddress)
	}

	return types.TransactionResult{
		Hash:        commandID,
		ChainFamily: cselectors.FamilyCanton,
		RawData: map[string]any{
			"NewMCMSContractID": newMCMSContractID,
			"NewMCMSTemplateID": newMCMSTemplateID,
			"RawTx":             submitResp,
		},
	}, nil
}
