package stellar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stretchr/testify/require"
)

const (
	friendbotReadyTimeout = 2 * time.Minute
	friendbotRetryDelay   = time.Second
	friendbotHTTPTimeout  = 10 * time.Second
)

type friendbotHTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *friendbotHTTPError) Error() string {
	return fmt.Sprintf(
		"Friendbot returned status=%s body=%s",
		e.Status,
		e.Body,
	)
}

func FundStellarKey(
	t *testing.T,
	rpcURL string,
	stellarSigner *keypair.Full,
) {
	t.Helper()

	friendbotURL, err := buildFriendbotURL(
		rpcURL,
		stellarSigner.Address(),
	)
	require.NoError(t, err)

	t.Logf(
		"Waiting for Stellar Friendbot: %s",
		friendbotURL,
	)

	ctx, cancel := context.WithTimeout(
		t.Context(),
		friendbotReadyTimeout,
	)
	defer cancel()

	client := &http.Client{
		Timeout: friendbotHTTPTimeout,
	}

	var lastErr error

	for attempt := 1; ; attempt++ {
		lastErr = fundStellarAccount(
			ctx,
			client,
			friendbotURL,
		)
		if lastErr == nil {
			t.Logf(
				"Funded Stellar account %s after %d attempt(s)",
				stellarSigner.Address(),
				attempt,
			)

			return
		}

		if !isRetryableFriendbotError(lastErr) {
			require.NoError(
				t,
				lastErr,
				"Friendbot request failed permanently: url=%s",
				friendbotURL,
			)

			return
		}

		t.Logf(
			"Friendbot is not ready (attempt %d): %v",
			attempt,
			lastErr,
		)

		select {
		case <-ctx.Done():
			require.FailNowf(
				t,
				"Friendbot did not become ready",
				"url=%s timeout=%s last_error=%v",
				friendbotURL,
				friendbotReadyTimeout,
				lastErr,
			)

			return

		case <-time.After(friendbotRetryDelay):
		}
	}
}

func fundStellarAccount(
	ctx context.Context,
	client *http.Client,
	friendbotURL string,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		friendbotURL,
		nil,
	)
	if err != nil {
		return fmt.Errorf(
			"create Stellar Friendbot request: %w",
			err,
		)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf(
			"send Stellar Friendbot request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(resp.Body, 1024*1024),
	)
	if err != nil {
		return fmt.Errorf(
			"read Stellar Friendbot response: %w",
			err,
		)
	}

	if resp.StatusCode >= http.StatusOK &&
		resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	return &friendbotHTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(body),
	}
}

func isRetryableFriendbotError(err error) bool {
	var httpErr *friendbotHTTPError
	if !errors.As(err, &httpErr) {
		// Connection failures are normally transient during startup.
		return true
	}

	switch httpErr.StatusCode {
	case http.StatusNotFound,
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true

	default:
		return httpErr.StatusCode >= 500
	}
}

func buildFriendbotURL(
	rpcURL string,
	address string,
) (string, error) {
	parsed, err := url.Parse(rpcURL)
	if err != nil {
		return "", fmt.Errorf(
			"parse Stellar RPC URL: %w",
			err,
		)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf(
			"invalid Stellar RPC URL %q",
			rpcURL,
		)
	}

	// Quickstart exposes RPC at /rpc and Friendbot at /friendbot.
	path := strings.TrimRight(parsed.Path, "/")
	path = strings.TrimSuffix(path, "/rpc")
	parsed.Path = strings.TrimRight(path, "/") + "/friendbot"

	query := parsed.Query()
	query.Set("addr", address)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}
