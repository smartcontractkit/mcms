//go:build e2e

package stellare2e

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/stretchr/testify/suite"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
)

// SmokeSuite is iteration-1 Stellar e2e: Soroban RPC reachability only (no MCMS deploy).
// Extend with contract deploy + MCMS flows in a later iteration.
type SmokeSuite struct {
	suite.Suite
	e2e.TestSetup
}

func (s *SmokeSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
	s.Require().NotEmpty(s.StellarRPCURL, "CTF_CONFIGS must include stellar_config (see e2e/config.stellar.toml)")
}

func (s *SmokeSuite) TestSorobanRPCGetHealth() {
	const body = `{"jsonrpc":"2.0","id":1,"method":"getHealth"}`
	resp, err := http.Post(s.StellarRPCURL, "application/json", bytes.NewReader([]byte(body)))
	s.Require().NoError(err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, "body=%s", string(raw))
	s.Require().Contains(string(raw), `"result"`, "body=%s", string(raw))
}

func (s *SmokeSuite) TestSorobanRPCGetLatestLedger() {
	body := `{"jsonrpc":"2.0","id":2,"method":"getLatestLedger","params":{}}`
	resp, err := http.Post(s.StellarRPCURL, "application/json", strings.NewReader(body))
	s.Require().NoError(err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, "body=%s", string(raw))
	s.Require().Contains(string(raw), `"result"`, "body=%s", string(raw))
}
