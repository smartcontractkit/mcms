package mcms

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewRemoteClient_RequiresParser(t *testing.T) {
	t.Parallel()

	_, err := NewRemoteClient(http.DefaultClient, nil)
	require.EqualError(t, err, "response parser is required")
}

func TestRemoteClient_Execute_DefaultsToPOSTAndSetsJSONContentType(t *testing.T) {
	t.Parallel()

	const customHeaderValue = "test-token"
	const testBody = `{"hello":"world"}`

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, customHeaderValue, r.Header.Get("X-Test-Header"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]string
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		require.Equal(t, "world", body["hello"])

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(testBody))
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	serverURL, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := NewRemoteClient(http.DefaultClient, &capturingParser{})
	require.NoError(t, err)

	resp, err := client.Execute(context.Background(), RemoteSignRequest{
		URL:  serverURL,
		Body: map[string]string{"hello": "world"},
		Headers: http.Header{
			"X-Test-Header": []string{customHeaderValue},
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.JSONEq(t, testBody, string(resp.Body))
}

func TestRemoteClient_Execute_PropagatesNetworkErrors(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}

	client, err := NewRemoteClient(httpClient, &capturingParser{})
	require.NoError(t, err)

	serverURL, err := url.Parse("https://remote-signer.test")
	require.NoError(t, err)

	_, err = client.Execute(context.Background(), RemoteSignRequest{URL: serverURL})
	require.ErrorContains(t, err, "execute remote signing request")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRemoteClient_Sign_DelegatesParsing(t *testing.T) {
	t.Parallel()

	parser := &capturingParser{
		returnedSig: types.Signature{
			R: common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
			S: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
			V: 27,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte(`{"backend":"response"}`))
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	serverURL, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := NewRemoteClient(http.DefaultClient, parser)
	require.NoError(t, err)

	sig, err := client.Sign(context.Background(), RemoteSignRequest{URL: serverURL})
	require.NoError(t, err)
	require.Equal(t, parser.returnedSig, sig)
	require.Equal(t, http.StatusAccepted, parser.receivedStatusCode)
	require.Equal(t, `{"backend":"response"}`, string(parser.receivedBody))
}

func TestRemoteClient_Sign_PropagatesParserError(t *testing.T) {
	t.Parallel()

	parser := &capturingParser{returnedErr: errors.New("parse failed")}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"ok":true}`))
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	serverURL, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := NewRemoteClient(http.DefaultClient, parser)
	require.NoError(t, err)

	_, err = client.Sign(context.Background(), RemoteSignRequest{URL: serverURL})
	require.EqualError(t, err, "parse failed")
}

func TestRequestRemoteSignatureAndAppend(t *testing.T) {
	t.Parallel()

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"r":"0x1111111111111111111111111111111111111111111111111111111111111111",
			"s":"0x2222222222222222222222222222222222222222222222222222222222222222",
			"v":27,
			"signer_address":"0x0000000000000000000000000000000000000000"
		}`))
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	serverURL, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	parser := &capturingParser{
		returnedSig: types.Signature{
			R: common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
			S: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
			V: 27,
		},
	}
	client, err := NewRemoteClient(http.DefaultClient, parser)
	require.NoError(t, err)

	signable, err := newTestSignableForRemoteFlow()
	require.NoError(t, err)
	require.Len(t, signable.proposal.Signatures, 0)

	sig, err := RequestRemoteSignatureAndAppend(context.Background(), signable, client, RemoteSignRequest{
		URL:  serverURL,
		Body: map[string]any{"request": "payload"},
	})
	require.NoError(t, err)
	require.Equal(t, uint8(27), sig.V)
	require.Len(t, signable.proposal.Signatures, 1)
	require.Equal(t, sig, signable.proposal.Signatures[0])
}

func newTestSignableForRemoteFlow() (*Signable, error) {
	proposal := &Proposal{
		BaseProposal: BaseProposal{
			Version:              "v1",
			Kind:                 types.KindProposal,
			ValidUntil:           uint32(time.Now().Add(10 * time.Minute).Unix()),
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain1Selector: {
					StartingOpCount: 0,
					MCMAddress:      "0x1111111111111111111111111111111111111111",
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: evm.NewTransaction(
					common.HexToAddress("0x2222222222222222222222222222222222222222"),
					[]byte{0x01},
					big.NewInt(0),
					"TestContract",
					[]string{"test"},
				),
			},
		},
	}

	return NewSignable(proposal, nil)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type capturingParser struct {
	receivedStatusCode int
	receivedBody       []byte
	returnedSig        types.Signature
	returnedErr        error
}

func (p *capturingParser) ParseResponse(statusCode int, headers http.Header, body []byte) (types.Signature, error) {
	p.receivedStatusCode = statusCode
	p.receivedBody = append([]byte(nil), body...)
	if p.returnedErr != nil {
		return types.Signature{}, p.returnedErr
	}
	return p.returnedSig, nil
}
