/*
Copyright 2021 The Flux CD contributors.

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

package azuredevops

import (
	"context"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

// PullRequestClient implements the gitprovider.PullRequestClient interface.

var _ gitprovider.PullRequestClient = &PullRequestClient{}

// PullRequestClient operates on the pull requests for a specific repository.
type PullRequestClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

func (c *PullRequestClient) List(ctx context.Context) ([]gitprovider.PullRequest, error) {
	repositoryId := c.ref.GetRepository()
	projectId := c.ref.GetIdentity()
	prOpts := git.GetPullRequestsArgs{
		RepositoryId:   &repositoryId,
		Project:        &projectId,
		SearchCriteria: &git.GitPullRequestSearchCriteria{},
	}
	prs, err := c.g.GetPullRequests(ctx, prOpts)
	if err != nil {
		return nil, err
	}

	requests := make([]gitprovider.PullRequest, len(*prs))

	for idx, pr := range *prs {
		requests[idx] = newPullRequest(c.clientContext, &pr)
	}

	return requests, nil
}

// Create creates a new pull request.
func (c *PullRequestClient) Create(ctx context.Context, title, branch, baseBranch, description string) (gitprovider.PullRequest, error) {
	repositoryId := c.ref.GetRepository()
	projectId := c.ref.GetIdentity()
	ref := "refs/heads/" + branch
	refBaseBranch := "refs/heads/" + baseBranch

	pullRequestToCreate := git.GitPullRequest{
		Description:   &description,
		SourceRefName: &ref,
		TargetRefName: &refBaseBranch,
		Title:         &title,
	}
	prOpts := git.CreatePullRequestArgs{
		GitPullRequestToCreate: &pullRequestToCreate,
		RepositoryId:           &repositoryId,
		Project:                &projectId,
	}
	pr, err := c.g.CreatePullRequest(ctx, prOpts)
	if err != nil {
		return nil, err
	}

	return newPullRequest(c.clientContext, pr), nil
}

func (c *PullRequestClient) Edit(ctx context.Context, number int, opts gitprovider.EditOptions) (gitprovider.PullRequest, error) {
	editPR := &git.GitPullRequest{}
	editPR.Title = opts.Title
	repositoryId := c.ref.GetRepository()
	projectId := c.ref.GetIdentity()
	editedPR, err := c.g.UpdatePullRequest(ctx, git.UpdatePullRequestArgs{
		GitPullRequestToUpdate: editPR,
		Project:                &projectId,
		RepositoryId:           &repositoryId,
		PullRequestId:          &number,
	})
	if err != nil {
		return nil, err
	}
	return newPullRequest(c.clientContext, editedPR), nil
}

func (c *PullRequestClient) Get(ctx context.Context, number int) (gitprovider.PullRequest, error) {
	projectId := c.ref.GetIdentity()
	pr, err := c.g.GetPullRequestById(ctx, git.GetPullRequestByIdArgs{
		Project:       &projectId,
		PullRequestId: &number,
	})
	if err != nil {
		return nil, err
	}

	return newPullRequest(c.clientContext, pr), nil
}

func (c *PullRequestClient) Merge(ctx context.Context, number int, mergeMethod gitprovider.MergeMethod, message string) error {
	//Request a git merge operation. Currently Azure Devops supports merging only 2 commits.
	repositoryId := c.ref.GetRepository()
	projectId := c.ref.GetIdentity()
	pullRequest, err := c.Get(ctx, number)
	if err != nil {
		return handleHTTPError(err)
	}
	//Get the last commit id from the pull request
	//Merge the pull request
	_, prError := c.g.UpdatePullRequest(ctx,
		git.UpdatePullRequestArgs{
			GitPullRequestToUpdate: &git.GitPullRequest{
				Description: &message,
				LastMergeSourceCommit: &git.GitCommitRef{
					CommitId: pullRequest.APIObject().(*git.GitPullRequest).LastMergeSourceCommit.CommitId,
				},
				Status: &git.PullRequestStatusValues.Completed,
				CompletionOptions: &git.GitPullRequestCompletionOptions{
					MergeStrategy: (*git.GitPullRequestMergeStrategy)(&mergeMethod),
				},
			},
			Project:       &projectId,
			PullRequestId: &number,
			RepositoryId:  &repositoryId,
		})
	if prError != nil {
		return handleHTTPError(prError)
	}
	return nil
}
