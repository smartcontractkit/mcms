package stellar

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stretchr/testify/require"
)

func FundStellarKey(t *testing.T, nodeURL string, stellarSigner *keypair.Full) {
	friendbotURL := strings.TrimRight(nodeURL, "/") +
		"/friendbot?addr=" +
		stellarSigner.Address()

	req, reqErr := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		friendbotURL,
		nil,
	)
	require.NoError(t, reqErr, "Failed to create Stellar friendbot request")

	resp, reqErr := http.DefaultClient.Do(req)
	require.NoError(t, reqErr, "Failed to fund Stellar account with friendbot")
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	require.NoError(t, readErr, "Failed to read Stellar friendbot response")

	require.Less(
		t,
		resp.StatusCode,
		http.StatusMultipleChoices,
		"Failed to fund Stellar account: status=%s body=%s",
		resp.Status,
		string(body),
	)
}
