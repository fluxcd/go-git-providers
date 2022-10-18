/*
Copyright 2020 The Flux CD contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"context"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v47/github"
)

// PullRequestClient implements the gitprovider.PullRequestClient interface.
var _ gitprovider.PullRequestClient = &PullRequestClient{}

// PullRequestClient operates on the pull requests for a specific repository.
type PullRequestClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// List lists all pull requests in the repository
func (c *PullRequestClient) List(ctx context.Context) ([]gitprovider.PullRequest, error) {
	prs, _, err := c.c.Client().PullRequests.List(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), nil)
	if err != nil {
		return nil, err
	}

	requests := make([]gitprovider.PullRequest, len(prs))

	for idx, pr := range prs {
		requests[idx] = newPullRequest(c.clientContext, pr)
	}

	return requests, nil
}

// Create creates a pull request with the given specifications.
func (c *PullRequestClient) Create(ctx context.Context, title, branch, baseBranch, description string) (gitprovider.PullRequest, error) {

	prOpts := &github.NewPullRequest{
		Title: &title,
		Head:  &branch,
		Base:  &baseBranch,
		Body:  &description,
	}

	pr, _, err := c.c.Client().PullRequests.Create(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), prOpts)
	if err != nil {
		return nil, err
	}

	return newPullRequest(c.clientContext, pr), nil
}

// Edit modifies an existing PR. Please refer to "EditOptions" for details on which data can be edited.
func (c *PullRequestClient) Edit(ctx context.Context, number int, opts gitprovider.EditOptions) (gitprovider.PullRequest, error) {
	editPR := &github.PullRequest{}
	editPR.Title = opts.Title
	editedPR, _, err := c.c.Client().PullRequests.Edit(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), number, editPR)
	if err != nil {
		return nil, err
	}
	return newPullRequest(c.clientContext, editedPR), nil
}

// Get retrieves an existing pull request by number
func (c *PullRequestClient) Get(ctx context.Context, number int) (gitprovider.PullRequest, error) {

	pr, _, err := c.c.Client().PullRequests.Get(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), number)
	if err != nil {
		return nil, err
	}

	return newPullRequest(c.clientContext, pr), nil
}

// Merge merges a pull request with the given specifications.
func (c *PullRequestClient) Merge(ctx context.Context, number int, mergeMethod gitprovider.MergeMethod, message string) error {

	prOpts := &github.PullRequestOptions{
		CommitTitle: "",
		SHA:         "",
		MergeMethod: string(mergeMethod),
	}

	_, _, err := c.c.Client().PullRequests.Merge(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), number, message, prOpts)
	if err != nil {
		return err
	}

	return nil
}
