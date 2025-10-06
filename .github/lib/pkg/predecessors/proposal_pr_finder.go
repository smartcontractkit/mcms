package github

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-github/v71/github"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils/predecessors"
	"github.com/smartcontractkit/mcms"
)

// ProposalPRFinder defines methods to find open proposal PRs and their predecessors.
type ProposalPRFinder interface {
	FindOpenPRs(ctx context.Context, title string) ([]*github.Issue, error)
	FindPredecessors(ctx context.Context, newPRViewData predecessors.PRView, excludePRs []predecessors.PRNum) ([]predecessors.PRView, error)
	GetProposalPRViews(ctx context.Context, proposalPRs []*github.Issue) []predecessors.PRView
}

// GithubProposalPRFinder implements ProposalPRFinder using GitHub API.
type GithubProposalPRFinder struct {
	cldCtx predecessors.CLDContext
	client *github.Client
	lggr   logger.Logger
}

var _ ProposalPRFinder = (*GithubProposalPRFinder)(nil)

func NewGithubProposalPRFinder(lggr logger.Logger, client *github.Client, cldCtx predecessors.CLDContext) *GithubProposalPRFinder {
	return &GithubProposalPRFinder{
		lggr:   lggr,
		client: client,
		cldCtx: cldCtx,
	}
}

// FindOpenPRs filter for open mcms proposals in the given domain/environment using Search API.
func (f *GithubProposalPRFinder) FindOpenPRs(ctx context.Context, title string) ([]*github.Issue, error) {
	cldCtx := f.cldCtx
	client := f.client
	q := fmt.Sprintf(`repo:%s/%s is:pr is:open in:title "%s" label:SIGNED,proposal,CREATED,PARTIALLY_SIGNED,PENDING_SIGNATURES -label:WAITING_FOR_TIMELOCK,executed`,
		cldCtx.Owner, cldCtx.Name, title)

	opts := &github.SearchOptions{
		Sort:        "created",
		Order:       "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var out []*github.Issue
	for {
		res, resp, err := client.Search.Issues(ctx, q, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, res.Issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return out, nil
}

// FindPredecessors returns the open proposal PRs sorted by most recently created that have dependencies on
// the same mcms addresses as the current proposal, or nil if none found.
func (f *GithubProposalPRFinder) FindPredecessors(
	ctx context.Context,
	newPRViewData predecessors.PRView,
	excludePRs []predecessors.PRNum,
) ([]predecessors.PRView, error) {
	cldCtx := predecessors.CLDContext{
		Owner:       f.cldCtx.Owner,
		Name:        f.cldCtx.Name,
		Domain:      f.cldCtx.Domain,
		Environment: f.cldCtx.Environment,
		QueueID:     "", // ignore queue for predecessor search
	}
	lggr := f.lggr

	proposalPRs, err := f.FindOpenPRs(ctx, ProposalPRTitle("", cldCtx))
	if err != nil {
		return nil, fmt.Errorf("search open proposal PRs: %w", err)
	}
	if len(proposalPRs) == 0 {
		lggr.Warnf("No matching open proposal PR found.")
		return nil, nil
	}

	prViews := f.GetProposalPRViews(ctx, proposalPRs)
	prViews = filterSlice(prViews, func(prView predecessors.PRView, _ int) bool { return !slices.Contains(excludePRs, prView.Number) })
	prViews = append(prViews, newPRViewData) // include the new PR

	// Build graph to get predecessors
	prsGraph, err := predecessors.BuildPRDependencyGraph(prViews)
	if err != nil {
		return nil, fmt.Errorf("build PR dependency graph: %w", err)
	}
	preds := prsGraph.Nodes[newPRViewData.Number].Pred
	predViews := make([]predecessors.PRView, 0, len(preds))
	for _, p := range preds {
		if v, ok := prsGraph.Nodes[p]; ok {
			predViews = append(predViews, v.PR)
		}
	}

	return predViews, nil
}

// GetProposalPRViews fetches PR details and proposal op count data for the given issues.
func (f *GithubProposalPRFinder) GetProposalPRViews(
	ctx context.Context,
	proposalPRs []*github.Issue,
) []predecessors.PRView {
	out := make([]predecessors.PRView, 0, len(proposalPRs))
	lggr := f.lggr
	for _, issue := range proposalPRs {
		number := issue.GetNumber()
		createdAt := issue.GetCreatedAt()

		head, err := f.getPRHeadInfo(ctx, number)
		if err != nil {
			// getPRHeadInfo already logs a warning with context
			continue
		}

		proposal, proposalContent, proposalFilename, proposalData, found := f.findProposalDataForPR(ctx, number, head)
		if !found {
			lggr.Infof("PR#%d has no proposal files; skipping.", number)
			continue
		}

		out = append(out, predecessors.PRView{
			Number:           predecessors.PRNum(number),
			Body:             issue.GetBody(),
			CreatedAt:        createdAt.Time,
			Head:             head,
			Proposal:         proposal,
			ProposalData:     proposalData,
			ProposalFilename: proposalFilename,
			ProposalContent:  proposalContent,
		})
	}

	return out
}

// -- Internal helpers --
// getProposalOpData gets op counts from mcms.TimelockProposal
func getProposalOpData(ctx context.Context, proposal *mcms.TimelockProposal) (predecessors.ProposalsOpData, error) {
	// Use conversion-aware counts
	counts, err := proposal.OperationCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("converted operation counts: %w", err)
	}

	data := make(predecessors.ProposalsOpData, len(proposal.ChainMetadatas()))
	for chain, meta := range proposal.ChainMetadatas() {
		data[chain] = predecessors.McmOpData{
			MCMAddress:      strings.TrimSpace(meta.MCMAddress),
			StartingOpCount: meta.StartingOpCount,
			OpsCount:        counts[chain],
		}
	}

	return data, nil
}

// getPRHeadInfo fetches the PR and extracts the head owner/repo/SHA.
// It logs context-rich warnings and returns an error when data is missing.
func (f *GithubProposalPRFinder) getPRHeadInfo(
	ctx context.Context,
	number int,
) (predecessors.PRHead, error) {
	client := f.client
	cldCtx := f.cldCtx
	lggr := f.lggr
	pr, _, err := client.PullRequests.Get(ctx, cldCtx.Owner, cldCtx.Name, number)
	if err != nil {
		lggr.Warnf("PullRequests.Get failed for PR#%d: %v", number, err)
		return predecessors.PRHead{}, err
	}
	if pr.GetHead() == nil || pr.GetHead().GetRepo() == nil || pr.GetHead().GetRepo().GetOwner() == nil {
		lggr.Warnf("PR#%d missing head repo info", number)
		return predecessors.PRHead{}, fmt.Errorf("missing head repo info for PR#%d", number)
	}

	head := predecessors.PRHead{
		Owner: pr.GetHead().GetRepo().GetOwner().GetLogin(),
		Repo:  pr.GetHead().GetRepo().GetName(),
		SHA:   pr.GetHead().GetSHA(),
		Ref:   pr.GetHead().GetRef(),
	}
	lggr.Infof("[DEBUG] PR#%d head: %s/%s @ %s (%s)", number, head.Owner, head.Repo, head.SHA, head.Ref)

	return head, nil
}

// findProposalDataForPR scans PR files (with pagination) and returns the first parsed proposal's data.
func (f *GithubProposalPRFinder) findProposalDataForPR(
	ctx context.Context,
	number int,
	head predecessors.PRHead,
) (*mcms.TimelockProposal, string, string, predecessors.ProposalsOpData, bool) {
	lggr := f.lggr
	cldCtx := f.cldCtx
	lggr.Debugf("inspecting PR#%d for proposal files", number)

	var proposal *mcms.TimelockProposal
	var proposalContent string
	var proposalFilename string
	var parsedProposal predecessors.ProposalsOpData // declare outside the closure

	handleFile := func(commitFile *github.CommitFile) (stop bool) {
		filename := commitFile.GetFilename()
		lggr.Debugf("file: %s (status=%s, additions=%d, deletions=%d)",
			filename, commitFile.GetStatus(), commitFile.GetAdditions(), commitFile.GetDeletions())

		if !proposalutils.MatchesProposalPath(cldCtx.Domain, cldCtx.Environment, filename) {
			lggr.Debugf("skip (path mismatch): %s", filename)
			return false
		}
		lggr.Debugf("candidate proposal file: %s", filename)

		content, err := f.fetchContentAtRef(ctx, head, filename)
		if err != nil {
			// fetchContentAtRef logs details
			return false
		}

		proposal, err = mcms.NewTimelockProposal(bytes.NewReader([]byte(content)))
		if err != nil {
			return false
		}

		opData, perr := getProposalOpData(ctx, proposal)
		if perr != nil {
			lggr.Warnf("parse proposal failed for %s in PR#%d: %v", filename, number, perr)
			return false
		}

		lggr.Debugf("using proposal file: %s", filename)
		proposalContent = content
		parsedProposal = opData
		proposalFilename = filename

		return true // stop iterating after first valid proposal
	}

	found := f.iterPRFiles(ctx, number, handleFile)

	return proposal, proposalContent, proposalFilename, parsedProposal, found
}

// iterPRFiles iterates all files in a PR (handling pagination) and calls fn for each.
// If fn returns true, iteration stops early and the function returns true.
func (f *GithubProposalPRFinder) iterPRFiles(
	ctx context.Context,
	number int,
	fn func(*github.CommitFile) (stop bool),
) bool {
	opts := &github.ListOptions{PerPage: 100}
	cldCtx := f.cldCtx
	lggr := f.lggr
	for {
		fs, resp, err := f.client.PullRequests.ListFiles(ctx, cldCtx.Owner, cldCtx.Name, number, opts)
		if err != nil {
			lggr.Warnf("ListFiles failed for PR#%d: %v", number, err)
			return false
		}
		for _, f := range fs {
			if fn(f) {
				return true
			}
		}
		// go-github v71: Response has field NextPage (int), no getter.
		if resp == nil || resp.NextPage == 0 {
			return false
		}
		opts.Page = resp.NextPage
	}
}

// fetchContentAtRef uses the Contents API to load file contents from a specific ref in (possibly forked) head repo.
func (f *GithubProposalPRFinder) fetchContentAtRef(
	ctx context.Context,
	head predecessors.PRHead,
	path string,
) (string, error) {
	client := f.client
	lggr := f.lggr
	rcOpts := &github.RepositoryContentGetOptions{Ref: head.SHA}
	fileContent, _, _, err := client.Repositories.GetContents(ctx, head.Owner, head.Repo, path, rcOpts)
	if err != nil {
		lggr.Warnf("GetContents failed for %s (ref=%s, repo=%s/%s): %v", path, head.SHA, head.Owner, head.Repo, err)
		return "", err
	}
	str, err := fileContent.GetContent()
	if err != nil {
		lggr.Warnf("GetContent decode failed for %s (ref=%s, repo=%s/%s): %v", path, head.SHA, head.Owner, head.Repo, err)
		return "", err
	}

	return str, nil
}

// ProposalPRTitle constructs a PR title for the given domain/environment/queue/file.
func ProposalPRTitle(fileName string, cldCtx predecessors.CLDContext) string {
	title := "Proposal for " + cldCtx.Domain + " - " + cldCtx.Environment
	if cldCtx.QueueID != "" {
		title += " - queue:" + cldCtx.QueueID
	}
	if fileName != "" {
		title += ": " + fileName
	}

	return title
}

// filterSlice iterates over elements of collection, returning an array of all elements predicate returns truthy for.
func filterSlice[V any](collection []V, predicate func(V, int) bool) []V {
	result := make([]V, 0, len(collection))

	for i, item := range collection {
		if predicate(item, i) {
			result = append(result, item)
		}
	}

	return result
}
