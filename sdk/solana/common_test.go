package solana

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

var (
	testProgramID = solana.MustPublicKeyFromBase58("6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX")
	testPDASeed   = PDASeed{'t', 'e', 's', 't', '-', 'm', 'c', 'm'}
	testRoot      = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
)

func Test_FindSignerPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindSignerPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG")))
}

func Test_FindConfigPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindConfigPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("CiPYshUKNDV9i4p4MLaqXSRqYWtnMtW6b1aYjh4Lw9nP")))
}

func Test_getConfigSignersPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindConfigSignersPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("EZJdMB7TCRcSTP6KMp1HPnzwNtW6wvqKXHLAZh1Jn81w")))
}

func Test_FindRootMetadataPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindRootMetadataPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("H45XH8Z1zpcLUHLLQzUwEgB1s3mZQcRvCYfcHriRcMxN")))
}

func Test_FindExpiringRootAndOpCountPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindExpiringRootAndOpCountPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("7nh2qGybwNRxL3zKpiSUzk2yc9CjCb5MhrB61B98hYZu")))
}

func Test_FindRootSignaturesPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindRootSignaturesPDA(testProgramID, testPDASeed, testRoot, 1735689600)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("528jBx5Mn1EPt4vG47CRkr1zhj8QVfSMvfvBZksZdrHr")))
}

func Test_FindSeenSignedHashesPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindSeenSignedHashesPDA(testProgramID, testPDASeed, testRoot, 1735689600)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("FxPYSHG9tm35T43zpAuVDdNY8uMPQfaaVBftxVrLyXVq")))
}
