package stellar

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stretchr/testify/require"
)

func FundStellarKey(
	t *testing.T,
	friendbotURL string,
	signer *keypair.Full,
) {
	t.Helper()

	require.NotEmpty(t, friendbotURL, "Stellar Friendbot URL is empty")
	require.NotNil(t, signer, "Stellar signer is nil")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	deadline := time.Now().Add(3 * time.Minute)

	var lastErr error

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			fmt.Sprintf(
				"%s?addr=%s",
				friendbotURL,
				url.QueryEscape(signer.Address()),
			),
			nil,
		)
		require.NoError(t, err)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			closeErr := resp.Body.Close()

			if readErr != nil {
				lastErr = readErr
			} else if closeErr != nil {
				lastErr = closeErr
			} else if resp.StatusCode >= http.StatusOK &&
				resp.StatusCode < http.StatusMultipleChoices {
				return
			} else {
				lastErr = fmt.Errorf(
					"status=%s body=%s",
					resp.Status,
					string(body),
				)
			}
		}

		select {
		case <-t.Context().Done():
			require.NoError(
				t,
				t.Context().Err(),
				"context cancelled while waiting for Stellar Friendbot",
			)
		case <-time.After(5 * time.Second):
		}
	}

	require.FailNowf(
		t,
		"Failed to fund Stellar account",
		"Friendbot did not become ready: %v",
		lastErr,
	)
}
