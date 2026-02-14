//go:build e2e

package canton

import (
	"sync"
	"testing"

	"github.com/smartcontractkit/chainlink-canton/contracts"
	"github.com/smartcontractkit/chainlink-canton/integration-tests/testhelpers"
	"github.com/stretchr/testify/require"
)

var (
	sharedEnv     *SharedCantonEnvironment
	sharedEnvOnce sync.Once
)

type SharedCantonEnvironment struct {
	Env        testhelpers.TestEnvironment
	PackageIDs []string
}

func GetSharedEnvironment(t *testing.T) *SharedCantonEnvironment {
	sharedEnvOnce.Do(func() {
		t.Log("Initializing shared Canton test environment...")

		env := testhelpers.NewTestEnvironment(t, testhelpers.WithNumberOfParticipants(1))

		t.Log("Uploading MCMS DAR (once for all suites)...")
		mcmsDar, err := contracts.GetDar(contracts.MCMS, contracts.CurrentVersion)
		require.NoError(t, err)

		packageIDs, err := testhelpers.UploadDARstoMultipleParticipants(
			t.Context(),
			[][]byte{mcmsDar},
			env.Participant(1),
		)
		require.NoError(t, err)

		sharedEnv = &SharedCantonEnvironment{
			Env:        env,
			PackageIDs: packageIDs,
		}
	})
	return sharedEnv
}
