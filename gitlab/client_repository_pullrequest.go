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

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// PullRequestClient implements the gitprovider.PullRequestClient interface.
var _ gitprovider.PullRequestClient = &PullRequestClient{}

// PullRequestClient operates on the pull requests for a specific repository.
type PullRequestClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// List lists all pull requests in the repository
func (c *PullRequestClient) List(_ context.Context) ([]gitprovider.PullRequest, error) {
	mrs, _, err := c.c.Client().MergeRequests.ListProjectMergeRequests(getRepoPath(c.ref), nil)
	if err != nil {
		return nil, err
	}

	requests := make([]gitprovider.PullRequest, len(mrs))

	for idx, basicMR := range mrs {
		mr, _, err := c.c.Client().MergeRequests.GetMergeRequest(getRepoPath(c.ref), basicMR.IID, nil)
		if err != nil {
			return nil, err
		}
		requests[idx] = newPullRequest(c.clientContext, mr)
	}

	return requests, nil
}

// Create creates a pull request with the given specifications.
func (c *PullRequestClient) Create(_ context.Context, title, branch, baseBranch, description string) (gitprovider.PullRequest, error) {

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

// Edit modifies an existing MR. Please refer to "EditOptions" for details on which data can be edited.
func (c *PullRequestClient) Edit(ctx context.Context, number int, opts gitprovider.EditOptions) (gitprovider.PullRequest, error) {
	mrUpdate := &gitlab.UpdateMergeRequestOptions{
		Title: opts.Title,
	}
	editedMR, _, err := c.c.Client().MergeRequests.UpdateMergeRequest(getRepoPath(c.ref), int64(number), mrUpdate)
	if err != nil {
		return nil, err
	}
	return newPullRequest(c.clientContext, editedMR), nil
}

// Get retrieves an existing pull request by number
func (c *PullRequestClient) Get(_ context.Context, number int) (gitprovider.PullRequest, error) {

	mr, _, err := c.c.Client().MergeRequests.GetMergeRequest(getRepoPath(c.ref), int64(number), &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return nil, err
	}

	return newPullRequest(c.clientContext, mr), nil
}

// Merge merges a pull request with the given specifications.
func (c *PullRequestClient) Merge(_ context.Context, number int, mergeMethod gitprovider.MergeMethod, message string) error {
	status, err := c.waitForMergeRequestToBeMergeable(number)
	if err != nil {
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

	_, _, err = c.c.Client().MergeRequests.AcceptMergeRequest(getRepoPath(c.ref), int64(number), amrOpts)
	if err != nil {
		return fmt.Errorf("failed to accept merge request with status '%s': %s", status, err)
	}

	return nil
}

func (c *PullRequestClient) waitForMergeRequestToBeMergeable(number int) (string, error) {
	// poll for merge status to reach "mergeable" state
	// docs: https://docs.gitlab.com/api/merge_requests/#merge-status
	currentStatus := "unknown"
	for retries := 0; retries < 10; retries++ {
		mr, _, err := c.c.Client().MergeRequests.GetMergeRequest(getRepoPath(c.ref), int64(number), &gitlab.GetMergeRequestsOptions{})
		if err != nil || mr.DetailedMergeStatus != "mergeable" {
			if mr != nil {
				currentStatus = mr.DetailedMergeStatus
			}
			time.Sleep(time.Second * 2)
			continue
		}

		return mr.DetailedMergeStatus, nil
	}

	return currentStatus, fmt.Errorf("merge status %s for pull request number: %d", currentStatus, number)
}
