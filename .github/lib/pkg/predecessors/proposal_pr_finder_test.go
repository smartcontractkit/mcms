package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/google/go-github/v71/github"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils/predecessors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/evm"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
)

func TestGetPRViews(t *testing.T) {
	t.Parallel()

	domain := "d"
	env := "e"
	filename := path.Join("domains", domain, env, "proposals", "p.json")

	// Start=10, Ops=3 -> End=13
	chain := mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)
	raw := makeTimelockProposalBytes(t, chain, "0xABC", 10)
	rawB64 := base64.StdEncoding.EncodeToString(raw)

	mux := http.NewServeMux()

	// 1) Raw bytes
	mux.HandleFunc("/raw/10", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(raw)
	})

	// 2) PR metadata (include head.sha so your code can call GetContents with ref=sha)
	mux.HandleFunc("/repos/owner/repo/pulls/10", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"number":     10,
			"state":      "open",
			"title":      "dummy",
			"created_at": time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
			"head": map[string]any{
				"ref": "branch",
				"sha": "abc123", // <-- important
				"repo": map[string]any{
					"name":      "repo",
					"full_name": "owner/repo",
					"owner":     map[string]any{"login": "owner"},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"repo": map[string]any{
					"name":      "repo",
					"full_name": "owner/repo",
					"owner":     map[string]any{"login": "owner"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 3) PR files (ListFiles). Keep providing raw_url for the happy path.
	mux.HandleFunc("/repos/owner/repo/pulls/10/files", func(w http.ResponseWriter, r *http.Request) {
		resp := []map[string]any{
			{
				"filename": filename,
				"raw_url":  "http://" + r.Host + "/raw/10",
				// optional extras:
				"status":    "modified",
				"additions": 1,
				"deletions": 0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 4) Contents API fallback used by your code when raw_url is missing or for private repos:
	//    GET /repos/owner/repo/contents/domains/d/e/proposals/p.json?ref=abc123
	mux.HandleFunc("/repos/owner/repo/contents/"+filename, func(w http.ResponseWriter, r *http.Request) {
		// you can assert r.URL.Query().Get("ref") == "abc123" if you want
		resp := map[string]any{
			"type":         "file",
			"name":         "p.json",
			"path":         filename,
			"download_url": "http://" + r.Host + "/raw/10",
			"content":      rawB64,
			"encoding":     "base64",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := ghClientToServer(t, srv)

	finder := &GithubProposalPRFinder{
		lggr: logger.Test(t),
		cldCtx: predecessors.CLDContext{
			Owner:       "owner",
			Name:        "repo",
			Domain:      domain,
			Environment: env,
		},
		client: client,
	}

	created := time.Now().Add(-1 * time.Hour)
	issue := &github.Issue{
		Number:    github.Ptr(10),
		CreatedAt: &github.Timestamp{Time: created},
	}

	views := finder.GetProposalPRViews(t.Context(), []*github.Issue{issue})
	require.Len(t, views, 1)
	require.Equal(t, predecessors.PRNum(10), views[0].Number)
	require.Equal(t, created.UTC().Truncate(time.Second), views[0].CreatedAt.UTC().Truncate(time.Second))

	v := views[0].ProposalData[chain]
	require.Equal(t, "0xABC", v.MCMAddress)
	require.Equal(t, uint64(10), v.StartingOpCount)
	require.Equal(t, uint64(3), v.OpsCount)
}

func TestFindPredecessorPRs(t *testing.T) {
	t.Parallel()

	domain := "d"
	env := "e"
	filename := path.Join("domains", domain, env, "proposals", "p.json")
	chain := mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)

	// predecessor proposal: Start=20, Ops=3
	predBytes := makeTimelockProposalBytes(t, chain, "0xABC", 20)
	predBytesB64 := base64.StdEncoding.EncodeToString(predBytes)

	prNumber := 11
	headSHA := "abc123"

	// ---- Handlers (register BEFORE starting server) ----
	mux := http.NewServeMux()

	// 1) Search issues -> returns PR #11
	mux.HandleFunc("/search/issues", func(w http.ResponseWriter, r *http.Request) {
		// Minimal Search API shape
		resp := map[string]any{
			"total_count":        1,
			"incomplete_results": false,
			"items": []map[string]any{
				{
					"number":     prNumber,
					"created_at": time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
					"state":      "open",
					"pull_request": map[string]any{
						"url": "http://" + r.Host + fmt.Sprintf("/repos/owner/repo/pulls/%d", prNumber),
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 2) PR metadata
	mux.HandleFunc(fmt.Sprintf("/repos/owner/repo/pulls/%d", prNumber), func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"number":     prNumber,
			"state":      "open",
			"title":      "pred",
			"created_at": time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
			"head": map[string]any{
				"ref": "branch",
				"sha": headSHA,
				"repo": map[string]any{
					"name":      "repo",
					"full_name": "owner/repo",
					"owner":     map[string]any{"login": "owner"},
					"private":   false,
				},
			},
			"base": map[string]any{
				"ref": "main",
				"repo": map[string]any{
					"name":      "repo",
					"full_name": "owner/repo",
					"owner":     map[string]any{"login": "owner"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 3) PR files (ListFiles)
	mux.HandleFunc(fmt.Sprintf("/repos/owner/repo/pulls/%d/files", prNumber), func(w http.ResponseWriter, r *http.Request) {
		resp := []map[string]any{
			{
				"filename":  filename,
				"raw_url":   "http://" + r.Host + fmt.Sprintf("/raw/%d", prNumber),
				"status":    "added",
				"additions": 1,
				"deletions": 0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 4) Raw proposal
	mux.HandleFunc(fmt.Sprintf("/raw/%d", prNumber), func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(predBytes)
	})

	// 5) Contents API
	mux.HandleFunc("/repos/owner/repo/contents/"+filename, func(w http.ResponseWriter, r *http.Request) {
		// You can assert r.URL.Query().Get("ref") == headSHA if desired
		resp := map[string]any{
			"type":         "file",
			"name":         "p.json",
			"path":         filename,
			"download_url": "http://" + r.Host + fmt.Sprintf("/raw/%d", prNumber),
			"content":      predBytesB64,
			"encoding":     "base64",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := ghClientToServer(t, srv)

	prFinder := &GithubProposalPRFinder{
		lggr: logger.Test(t),
		cldCtx: predecessors.CLDContext{
			Owner:       "owner",
			Name:        "repo",
			Domain:      domain,
			Environment: env,
		},
		client: client,
	}

	// new PR view: baseline lower than predecessor; same chain & MCM
	newData := predecessors.ProposalsOpData{
		chain: predecessors.McmOpData{
			MCMAddress:      "0xabc", // case-insensitive match with "0xABC"
			StartingOpCount: 10,
			OpsCount:        1,
		},
	}
	newView := predecessors.PRView{
		Number:       predecessors.PRNum(-1),
		CreatedAt:    time.Now(),
		ProposalData: newData,
	}

	t.Run("success: standard scenario ", func(t *testing.T) {
		t.Parallel()
		preds, err := prFinder.FindPredecessors(t.Context(), newView, []predecessors.PRNum{})
		require.NoError(t, err)
		require.Len(t, preds, 1)
		require.Equal(t, predecessors.PRNum(prNumber), preds[0].Number)
	})

	t.Run("success: exclude merged PRs", func(t *testing.T) {
		t.Parallel()
		preds, err := prFinder.FindPredecessors(t.Context(), newView, []predecessors.PRNum{11})
		require.NoError(t, err)
		require.Empty(t, preds)
	})
}

func TestNewGithubProposalPRFinder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lggr logger.Logger
	}{
		{
			name: "with nil logger",
			lggr: nil,
		},
		{
			name: "with test logger",
			lggr: logger.Test(t), // replace with the correct helper for your logger
		},
	}

	for _, tt := range tests {
		// capture range var
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := github.NewClient(nil)
			var cldCtx predecessors.CLDContext // use a real init if needed

			got := NewGithubProposalPRFinder(tt.lggr, client, cldCtx)

			require.NotNil(t, got)
			require.Equal(t, tt.lggr, got.lggr)
			require.Same(t, client, got.client)
			require.Equal(t, cldCtx, got.cldCtx)
		})
	}
}

// ghClientToServer wires a go-github client to the provided httptest.Server.
func ghClientToServer(t *testing.T, srv *httptest.Server) *github.Client {
	t.Helper()
	baseURL, _ := url.Parse(srv.URL + "/")
	cli := github.NewClient(nil)
	cli.BaseURL = baseURL
	cli.UploadURL = baseURL

	return cli
}

// makeTimelockProposalBytes builds a valid MCMS Timelock proposal and returns JSON bytes.
func makeTimelockProposalBytes(t *testing.T, chain mcmstypes.ChainSelector, mcmAddr string, startOpCount uint64) []byte {
	t.Helper()
	prop, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(time.Now().Add(24*time.Hour).Unix())). //nolint:gosec // test code, overflow acceptable
		SetDescription("test").
		AddTimelockAddress(chain, mcmAddr).
		AddChainMetadata(chain, mcmstypes.ChainMetadata{
			StartingOpCount: startOpCount,
			MCMAddress:      mcmAddr,
		}).
		AddOperation(mcmstypes.BatchOperation{
			ChainSelector: chain,
			Transactions: []mcmstypes.Transaction{
				evm.NewTransaction([20]byte{}, []byte{}, big.NewInt(0), "noop", nil),
			},
		}).
		AddOperation(mcmstypes.BatchOperation{
			ChainSelector: chain,
			Transactions: []mcmstypes.Transaction{
				evm.NewTransaction([20]byte{}, []byte{}, big.NewInt(0), "noop", nil),
			},
		}).
		AddOperation(mcmstypes.BatchOperation{
			ChainSelector: chain,
			Transactions: []mcmstypes.Transaction{
				evm.NewTransaction([20]byte{}, []byte{}, big.NewInt(0), "noop", nil),
			},
		}).
		SetAction(mcmstypes.TimelockActionSchedule).
		SetDelay(mcmstypes.NewDuration(time.Second)).
		Build()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, mcms.WriteTimelockProposal(&buf, prop))

	return buf.Bytes()
}
