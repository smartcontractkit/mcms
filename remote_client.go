package mcms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/smartcontractkit/mcms/types"
)

// ResponseParser converts backend-specific HTTP responses into an MCMS signature.
// Implementations define success and failure semantics from status code + body.
type ResponseParser interface {
	ParseResponse(statusCode int, headers http.Header, body []byte) (types.Signature, error)
}

// RemoteSignRequest defines a transport-level request to a remote signer.
// Callers are responsible for constructing the auth payload and headers.
type RemoteSignRequest struct {
	Method  string
	URL     *url.URL
	Headers http.Header
	Body    any
}

// RemoteHTTPResponse contains the raw HTTP response from a remote signer call.
type RemoteHTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// RemoteClient executes HTTP requests and delegates response interpretation to a parser.
type RemoteClient struct {
	httpClient *http.Client
	parser     ResponseParser
}

// NewRemoteClient constructs a remote client with a parser-defined response contract.
func NewRemoteClient(httpClient *http.Client, parser ResponseParser) (*RemoteClient, error) {
	if parser == nil {
		return nil, errors.New("response parser is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &RemoteClient{
		httpClient: httpClient,
		parser:     parser,
	}, nil
}

// Execute sends a remote signing request and returns the raw status/body response.
func (c *RemoteClient) Execute(ctx context.Context, req RemoteSignRequest) (RemoteHTTPResponse, error) {
	if req.URL == nil {
		return RemoteHTTPResponse{}, errors.New("remote signer URL is required")
	}

	body, err := marshalHTTPBody(req.Body)
	if err != nil {
		return RemoteHTTPResponse{}, err
	}

	method := req.Method
	if method == "" {
		method = http.MethodPost
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL.String(), bytes.NewReader(body))
	if err != nil {
		return RemoteHTTPResponse{}, fmt.Errorf("create remote signing request: %w", err)
	}

	httpReq.Header = cloneHeaders(req.Headers)
	if body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return RemoteHTTPResponse{}, fmt.Errorf("execute remote signing request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return RemoteHTTPResponse{}, fmt.Errorf("read remote signing response: %w", err)
	}

	return RemoteHTTPResponse{
		StatusCode: httpResp.StatusCode,
		Body:       respBody,
		Headers:    httpResp.Header.Clone(),
	}, nil
}

// Sign executes the request and parses the response into a signature.
func (c *RemoteClient) Sign(ctx context.Context, req RemoteSignRequest) (types.Signature, error) {
	resp, err := c.Execute(ctx, req)
	if err != nil {
		return types.Signature{}, err
	}

	return c.parser.ParseResponse(resp.StatusCode, resp.Headers, resp.Body)
}

// RequestRemoteSignatureAndAppend validates the proposal, requests a remote signature,
// and appends that signature to the signable proposal.
func RequestRemoteSignatureAndAppend(
	ctx context.Context,
	signable *Signable,
	client *RemoteClient,
	req RemoteSignRequest,
) (types.Signature, error) {
	if signable == nil {
		return types.Signature{}, errors.New("signable is required")
	}
	if client == nil {
		return types.Signature{}, errors.New("remote client is required")
	}

	if err := signable.proposal.Validate(); err != nil { //nolint:contextcheck // Proposal.Validate has no context param; lookups use Background internally.
		return types.Signature{}, fmt.Errorf("validate proposal: %w", err)
	}

	sig, err := client.Sign(ctx, req)
	if err != nil {
		return types.Signature{}, fmt.Errorf("request remote signature: %w", err)
	}

	signable.proposal.AppendSignature(sig)

	return sig, nil
}

func marshalHTTPBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	switch typedBody := body.(type) {
	case []byte:
		return typedBody, nil
	case json.RawMessage:
		return []byte(typedBody), nil
	default:
		marshaledBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal HTTP body: %w", err)
		}

		return marshaledBody, nil
	}
}

func cloneHeaders(headers http.Header) http.Header {
	if headers == nil {
		return make(http.Header)
	}

	return headers.Clone()
}
