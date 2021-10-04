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

package gitlab

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/xanzy/go-gitlab"
)

// mergeStatusChecking indicates that gitlab has not yet asynchronously updated the merge status for a merge request
const mergeStatusChecking = "checking"

// PullRequestClient implements the gitprovider.PullRequestClient interface.
var _ gitprovider.PullRequestClient = &PullRequestClient{}

// PullRequestClient operates on the pull requests for a specific repository.
type PullRequestClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// List lists all pull requests in the repository
func (c *PullRequestClient) List(ctx context.Context) ([]gitprovider.PullRequest, error) {
	mrs, _, err := c.c.Client().MergeRequests.ListMergeRequests(nil)
	if err != nil {
		return nil, err
	}

	requests := make([]gitprovider.PullRequest, len(mrs))

	for idx, mr := range mrs {
		requests[idx] = newPullRequest(c.clientContext, mr)
	}

	return requests, nil
}

// Create creates a pull request with the given specifications.
func (c *PullRequestClient) Create(ctx context.Context, title, branch, baseBranch, description string) (gitprovider.PullRequest, error) {

	prOpts := &gitlab.CreateMergeRequestOptions{
		Title:        &title,
		SourceBranch: &branch,
		TargetBranch: &baseBranch,
		Description:  &description,
	}

	mr, _, err := c.c.Client().MergeRequests.CreateMergeRequest(getRepoPath(c.ref), prOpts)
	if err != nil {
		return nil, err
	}

	return newPullRequest(c.clientContext, mr), nil
}

// Get retrieves an existing pull request by number
func (c *PullRequestClient) Get(ctx context.Context, number int) (gitprovider.PullRequest, error) {

	mr, _, err := c.c.Client().MergeRequests.GetMergeRequest(getRepoPath(c.ref), number, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return nil, err
	}

	return newPullRequest(c.clientContext, mr), nil
}

// Merge merges a pull request with the given specifications.
func (c *PullRequestClient) Merge(ctx context.Context, number int, mergeMethod gitprovider.MergeMethod, message string) error {
	if err := c.waitForMergeRequestToBeMergeable(number); err != nil {
		return err
	}

	var squash bool

	var mergeCommitMessage *string
	var squashCommitMessage *string

	switch mergeMethod {
	case gitprovider.MergeMethodSquash:
		squashCommitMessage = &message
		squash = true
	case gitprovider.MergeMethodMerge:
		mergeCommitMessage = &message
	default:
		return fmt.Errorf("unknown merge method: %s", mergeMethod)
	}

	amrOpts := &gitlab.AcceptMergeRequestOptions{
		MergeCommitMessage:        mergeCommitMessage,
		SquashCommitMessage:       squashCommitMessage,
		Squash:                    &squash,
		ShouldRemoveSourceBranch:  nil,
		MergeWhenPipelineSucceeds: nil,
		SHA:                       nil,
	}

	_, _, err := c.c.Client().MergeRequests.AcceptMergeRequest(getRepoPath(c.ref), number, amrOpts)
	if err != nil {
		return err
	}

	return nil
}

func (c *PullRequestClient) waitForMergeRequestToBeMergeable(number int) error {
	// gitlab says to poll for merge status
	for retries := 0; retries < 10; retries++ {
		mr, _, err := c.c.Client().MergeRequests.GetMergeRequest(getRepoPath(c.ref), number, &gitlab.GetMergeRequestsOptions{})
		if err != nil || mr.MergeStatus == mergeStatusChecking {
			time.Sleep(time.Second * 2)
			continue
		}

		return nil
	}

	return fmt.Errorf("merge status unavailable for pull request number: %d", number)
}
