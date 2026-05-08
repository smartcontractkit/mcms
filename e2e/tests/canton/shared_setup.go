//go:build e2e

package canton

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/smartcontractkit/chainlink-canton/contracts"
	"github.com/smartcontractkit/chainlink-canton/testhelpers"
	"github.com/stretchr/testify/require"

	mcmstypes "github.com/smartcontractkit/mcms/types"
)

var (
	sharedEnv     *SharedCantonEnvironment
	sharedEnvOnce sync.Once
	errSharedEnv  error
)

type SharedCantonEnvironment struct {
	Env             testhelpers.TestEnvironment
	PackageIDs      []string
	ChainSelector   mcmstypes.ChainSelector
	SubmittingParty string
}

func GetSharedEnvironment(t *testing.T) *SharedCantonEnvironment {
	t.Helper()

	sharedEnvOnce.Do(func() {
		t.Log("Initializing shared Canton test environment...")

		env := testhelpers.NewTestEnvironment(t, testhelpers.WithNumberOfParticipants(1))
		participant := env.Chain.Participants[0]

		t.Logf("Allocating a submitting party...")
		// Create a separate submitting party and grant the participant's user actAs rights for that party
		submittingParty := testhelpers.AllocateParty(t, participant, fmt.Sprintf("submittingParty-%s", uuid.NewString()[:8]))
		testhelpers.GrantCanActAs(t, participant, submittingParty)
		t.Logf("Allocated submitting party %s", submittingParty)

		t.Log("Uploading MCMS and MCMSTest DARs (once for all suites)...")
		mcmsDar, err := contracts.GetDar(contracts.MCMS, contracts.CurrentVersion)
		if err != nil {
			errSharedEnv = err
			return
		}

		mcmsTestDar, err := contracts.GetDar(contracts.MCMSTest, contracts.CurrentVersion)
		if err != nil {
			errSharedEnv = err
			return
		}

		packageIDs, err := testhelpers.UploadDARstoMultipleParticipants(
			t.Context(),
			[][]byte{mcmsDar, mcmsTestDar},
			participant,
		)
		if err != nil {
			errSharedEnv = err
			return
		}

		sharedEnv = &SharedCantonEnvironment{
			Env:             env,
			PackageIDs:      packageIDs,
			ChainSelector:   mcmstypes.ChainSelector(env.Chain.ChainSelector()),
			SubmittingParty: submittingParty,
		}
	})

	require.NoError(t, errSharedEnv, "failed to initialize shared environment")
	require.NotNil(t, sharedEnv, "shared environment is nil")

	return sharedEnv
}
