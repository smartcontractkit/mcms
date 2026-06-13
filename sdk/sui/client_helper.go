package sui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cslclient "github.com/smartcontractkit/chainlink-sui/relayer/client"
)

const defaultSuiGrpcToken = "test"
const defaultSuiGrpcTimeout = 30 * time.Second
const defaultSuiGrpcMaxConcurrentRequests = 50

// NewBindingsClientFromNodeURL creates a gRPC-backed bindings client from an HTTP RPC URL.
// Local Sui nodes expose gRPC on the same host:port as JSON-RPC.
func NewBindingsClientFromNodeURL(log logger.Logger, nodeURL string, grpcToken string) (cslclient.BindingsClient, error) {
	grpcTarget, err := grpcTargetFromNodeURL(nodeURL)
	if err != nil {
		return nil, err
	}
	if grpcToken == "" {
		grpcToken = defaultSuiGrpcToken
	}

	return cslclient.NewPTBClient(log, cslclient.PTBClientConfig{
		GrpcTarget:            grpcTarget,
		GrpcToken:             grpcToken,
		TransactionTimeout:    defaultSuiGrpcTimeout,
		MaxConcurrentRequests: defaultSuiGrpcMaxConcurrentRequests,
		DefaultRequestType:    cslclient.WaitForEffectsCert,
	})
}

func grpcTargetFromNodeURL(nodeURL string) (string, error) {
	u, err := url.Parse(nodeURL)
	if err != nil {
		return "", fmt.Errorf("parse node URL %q: %w", nodeURL, err)
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		return "", fmt.Errorf("node URL %q has no host", nodeURL)
	}
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		default:
			port = "9000"
		}
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return fmt.Sprintf("[%s]:%s", host, port), nil
	}

	return fmt.Sprintf("%s:%s", host, port), nil
}
