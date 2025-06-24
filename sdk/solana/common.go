package solana

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/fees"
)

const (
	// FIXME: should we reuse these from sdk/evm/utils or duplicate them here?
	SignatureVOffset    = 27
	SignatureVThreshold = 2
)

func FindSignerPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_signer"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindConfigPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindConfigSignersPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config_signers"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindRootMetadataPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_metadata"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindExpiringRootAndOpCountPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("expiring_root_and_op_count"), pdaSeed[:]}
	return findPDA(programID, seeds)
}

func FindRootSignaturesPDA(
	programID solana.PublicKey, msigID PDASeed, root common.Hash, validUntil uint32, authority solana.PublicKey,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_signatures"), msigID[:], root[:], validUntilBytes(validUntil), authority[:]}
	return findPDA(programID, seeds)
}

func FindSeenSignedHashesPDA(
	programID solana.PublicKey, msigID PDASeed, root common.Hash, validUntil uint32,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("seen_signed_hashes"), msigID[:], root[:], validUntilBytes(validUntil)}
	return findPDA(programID, seeds)
}

func FindTimelockConfigPDA(
	programID solana.PublicKey, timelockID PDASeed,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_config"), timelockID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockOperationPDA(
	programID solana.PublicKey, timelockID PDASeed, opID [32]byte,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_operation"), timelockID[:], opID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockBypasserOperationPDA(
	programID solana.PublicKey, timelockID PDASeed, opID [32]byte,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_bypasser_operation"), timelockID[:], opID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockSignerPDA(
	programID solana.PublicKey, timelockID PDASeed,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_signer"), timelockID[:]}
	return findPDA(programID, seeds)
}

func findPDA(programID solana.PublicKey, seeds [][]byte) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find %s pda: %w", string(seeds[0]), err)
	}

	return pda, nil
}

func validUntilBytes(validUntil uint32) []byte {
	const uint32Size = 4
	vuBytes := make([]byte, uint32Size)
	binary.LittleEndian.PutUint32(vuBytes, validUntil)

	return vuBytes
}

type mcmInstructionBuilder interface {
	ValidateAndBuild() (*mcm.Instruction, error)
}

type timelockInstructionBuilder interface {
	ValidateAndBuild() (*timelock.Instruction, error)
}

func validateAndBuildSolanaInstruction(instructionBuilder any) (solana.Instruction, error) {
	var err error
	var builtInstruction solana.Instruction

	switch builder := instructionBuilder.(type) {
	case mcmInstructionBuilder:
		builtInstruction, err = builder.ValidateAndBuild()
		if err != nil {
			return nil, fmt.Errorf("unable to validate and build instruction: %w", err)
		}
	case timelockInstructionBuilder:
		builtInstruction, err = builder.ValidateAndBuild()
		if err != nil {
			return nil, fmt.Errorf("unable to validate and build instruction: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported instruction builder: %T", instructionBuilder)
	}

	return builtInstruction, nil
}

type SendAndConfirmFn func(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	builder any,
	commitmentType rpc.CommitmentType,
	opts ...sendTransactionOption,
) (string, *rpc.GetTransactionResult, error)

type SendAndConfirmInstructionsFn func(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructions []solana.Instruction,
	commitmentType rpc.CommitmentType,
	opts ...sendTransactionOption,
) (string, *rpc.GetTransactionResult, error)

// sendAndConfirm contains the default logic for sending and confirming instructions.
func sendAndConfirm(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructionBuilder any,
	commitmentType rpc.CommitmentType,
	opts ...sendTransactionOption,
) (string, *rpc.GetTransactionResult, error) {
	instruction, err := validateAndBuildSolanaInstruction(instructionBuilder)
	if err != nil {
		return "", nil, fmt.Errorf("unable to validate and build instruction: %w", err)
	}

	return sendAndConfirmInstructions(ctx, client, auth, []solana.Instruction{instruction}, commitmentType, opts...)
}

// sendAndConfirm contains the default logic for sending and confirming instructions.
func sendAndConfirmInstructions(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructions []solana.Instruction,
	commitmentType rpc.CommitmentType,
	opts ...sendTransactionOption,
) (string, *rpc.GetTransactionResult, error) {
	result, err := sendTransaction(ctx, client, instructions, auth, commitmentType, opts...)
	if err != nil {
		return "", nil, fmt.Errorf("unable to send instruction: %w", err)
	}
	if result.Transaction == nil {
		return "", nil, fmt.Errorf("nil transaction in instruction result")
	}

	transaction, err := result.Transaction.GetTransaction()
	if err != nil {
		return "", nil, fmt.Errorf("unable to get transaction from instruction result: %w", err)
	}

	return transaction.Signatures[0].String(), result, nil
}

func chunkIndexes(numItems int, chunkSize int) [][2]int {
	indexes := make([][2]int, 0)

	for i := 0; i < numItems; i += chunkSize {
		end := i + chunkSize
		if end > numItems {
			end = numItems
		}
		indexes = append(indexes, [2]int{i, end})
	}

	return indexes
}

type sendTransactionOptions struct {
	retries          int
	delay            time.Duration
	skipPreflight    bool
	computeUnitLimit fees.ComputeUnitLimit
	computeUnitPrice fees.ComputeUnitPrice
}

var defaultSendTransactionOptions = func() *sendTransactionOptions {
	retries := getenv("MCMS_SOLANA_MAX_RETRIES", 500, strconv.Atoi) //nolint:mnd
	delay := getenv("MCMS_SOLANA_RETRY_DELAY", 50, strconv.Atoi)    //nolint:mnd
	skipPreflight := getenv("MCMS_SOLANA_SKIP_PREFLIGHT", false, strconv.ParseBool)
	computeUnitPrice := getenv("MCMS_SOLANA_COMPUTE_UNIT_PRICE", 0, strconv.Atoi)

	// FIXME: should default ot 0 like computeUnitPrice above
	// right now we're always setting the compute unit limit to the max; this
	// consumes more gas than we needed. We should either set the compute unit limit
	// based on the output of a simulation, or do not set the limit initially, then
	// handle the "compute unit limits" exceeded error and automatically retry with
	// a higher limit
	// computeUnitLimit := getenv("MCMS_SOLANA_COMPUTE_UNIT_LIMIT", 0, strconv.Atoi)
	computeUnitLimit := getenv("MCMS_SOLANA_COMPUTE_UNIT_LIMIT", computebudget.MAX_COMPUTE_UNIT_LIMIT, strconv.Atoi)

	return &sendTransactionOptions{
		retries:          retries,
		delay:            time.Millisecond * time.Duration(delay),
		skipPreflight:    skipPreflight,
		computeUnitPrice: fees.ComputeUnitPrice(computeUnitPrice),
		computeUnitLimit: fees.ComputeUnitLimit(computeUnitLimit),
	}
}

type sendTransactionOption func(*sendTransactionOptions)

func WithRetries(retries int) sendTransactionOption {
	return func(opts *sendTransactionOptions) { opts.retries = retries }
}

func WithDelay(delay time.Duration) sendTransactionOption {
	return func(opts *sendTransactionOptions) {
		opts.delay = delay
	}
}

func WithSkipPreflight(skipPreflight bool) sendTransactionOption {
	return func(opts *sendTransactionOptions) {
		opts.skipPreflight = skipPreflight
	}
}

func WithComputeUnitPrice(price fees.ComputeUnitPrice) sendTransactionOption {
	return func(opts *sendTransactionOptions) {
		opts.computeUnitPrice = price
	}
}

func WithComputeUnitLimit(limit fees.ComputeUnitLimit) sendTransactionOption {
	return func(opts *sendTransactionOptions) {
		opts.computeUnitLimit = limit
	}
}

func sendTransaction(
	ctx context.Context,
	rpcClient *rpc.Client,
	instructions []solana.Instruction,
	signerAndPayer solana.PrivateKey,
	commitment rpc.CommitmentType,
	opts ...sendTransactionOption,
) (*rpc.GetTransactionResult, error) {
	var errBlockHash error
	var hashRes *rpc.GetLatestBlockhashResult
	logger := logFromContext(ctx)

	sendTransactionOptions := defaultSendTransactionOptions()
	for _, opt := range opts {
		opt(sendTransactionOptions)
	}

	for range sendTransactionOptions.retries {
		hashRes, errBlockHash = rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
		if errBlockHash != nil {
			logger.Infof("GetLatestBlockhash error:", errBlockHash)
			time.Sleep(sendTransactionOptions.delay)

			continue
		}

		break
	}
	if errBlockHash != nil {
		logger.Infof("GetLatestBlockhash error after retries:", errBlockHash)
		return nil, errBlockHash
	}

	tx, err := solana.NewTransaction(instructions, hashRes.Value.Blockhash, solana.TransactionPayer(signerAndPayer.PublicKey()))
	if err != nil {
		return nil, err
	}

	if sendTransactionOptions.computeUnitPrice > 0 {
		err = fees.SetComputeUnitPrice(tx, sendTransactionOptions.computeUnitPrice)
		if err != nil {
			return nil, fmt.Errorf("failed to set compute unit price: %w", err)
		}
	}
	if sendTransactionOptions.computeUnitLimit > 0 {
		err = fees.SetComputeUnitLimit(tx, sendTransactionOptions.computeUnitLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to set compute unit limit: %w", err)
		}
	}

	// build signers map
	signers := map[solana.PublicKey]solana.PrivateKey{signerAndPayer.PublicKey(): signerAndPayer}
	_, err = tx.Sign(func(pub solana.PublicKey) *solana.PrivateKey {
		priv, ok := signers[pub]
		if !ok {
			logger.Infof("ERROR: Missing signer private key for %s\n", pub)
		}

		return &priv
	})
	if err != nil {
		return nil, err
	}

	var txsig solana.Signature
	for range sendTransactionOptions.retries {
		txOpts := rpc.TransactionOpts{SkipPreflight: sendTransactionOptions.skipPreflight, PreflightCommitment: commitment}
		txsig, err = rpcClient.SendTransactionWithOpts(ctx, tx, txOpts)
		if err != nil {
			logger.Infof("Error sending transaction:", err)
			time.Sleep(sendTransactionOptions.delay)

			continue
		}

		break
	}
	// If tx failed with rpc error, we should not retry as confirmation will never happen
	if err != nil {
		return nil, err
	}

	var txStatus rpc.ConfirmationStatusType
	count := 0
	for txStatus != rpc.ConfirmationStatusConfirmed && txStatus != rpc.ConfirmationStatusFinalized {
		if count > sendTransactionOptions.retries {
			return nil, fmt.Errorf("unable to find transaction within timeout (sig: %v)", txsig)
		}
		count++
		statusRes, sigErr := rpcClient.GetSignatureStatuses(ctx, true, txsig)
		if sigErr != nil {
			logger.Infof("GetSignaturesStatuses error: %v", sigErr)
			time.Sleep(sendTransactionOptions.delay)

			continue
		}
		if statusRes != nil && len(statusRes.Value) > 0 && statusRes.Value[0] != nil {
			txStatus = statusRes.Value[0].ConfirmationStatus
		}
		time.Sleep(sendTransactionOptions.delay)
	}

	v := uint64(0)
	var errGetTx error
	var transactionRes *rpc.GetTransactionResult
	txOpts := &rpc.GetTransactionOpts{Commitment: commitment, MaxSupportedTransactionVersion: &v}
	for range sendTransactionOptions.retries {
		transactionRes, err = rpcClient.GetTransaction(ctx, txsig, txOpts)
		if err != nil {
			logger.Infof("GetTransaction error:", err)
			time.Sleep(sendTransactionOptions.delay)

			continue
		}

		break
	}

	return transactionRes, errGetTx
}

func getenv[T any](key string, defaultValue T, converter func(string) (T, error)) T {
	value, found := os.LookupEnv(key)
	if !found || value == "" {
		return defaultValue
	}

	convertedValue, err := converter(value)
	if err != nil {
		return defaultValue
	}

	return convertedValue
}

type Logger interface {
	Infof(template string, args ...any)
}

type contextLoggerValueT string

const ContextLoggerValue = contextLoggerValueT("mcms-logger")

func logFromContext(ctx context.Context) Logger {
	value := ctx.Value(ContextLoggerValue)
	logger, ok := value.(Logger)
	if !ok {
		logger = zap.Must(zap.NewProduction()).Sugar()
	}

	return logger
}
