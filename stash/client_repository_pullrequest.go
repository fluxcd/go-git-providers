/*
Copyright 2021 The Flux authors

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

package stash

import (
	"context"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

// PullRequestClient implements the gitprovider.PullRequestClient interface.
var _ gitprovider.PullRequestClient = &PullRequestClient{}

// PullRequestClient operates on the pull requests for a specific repository.
type PullRequestClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns the pull request with the given number.
func (c *PullRequestClient) Get(ctx context.Context, number int) (gitprovider.PullRequest, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	pr, err := c.client.PullRequests.Get(ctx, projectKey, repoSlug, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}
	return newPullRequest(pr), nil

}

// List returns all pull requests for the given repository.
func (c *PullRequestClient) List(ctx context.Context) ([]gitprovider.PullRequest, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	apiObjs, err := c.client.PullRequests.All(ctx, projectKey, repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	// Traverse the list, and return a list of OrgRepository objects
	prs := make([]gitprovider.PullRequest, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		prs = append(prs, newPullRequest(apiObj))
	}

	return prs, nil

}

// Merge merges the pull request.
// Stash does not support message and merge strategy options for pull requests automatic merges.
func (c *PullRequestClient) Merge(ctx context.Context, number int, _ gitprovider.MergeMethod, _ string) error {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	// Get the pull request first
	pr, err := c.client.PullRequests.Get(ctx, projectKey, repoSlug, number)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	// Merge the pull request
	_, err = c.client.PullRequests.Merge(ctx, projectKey, repoSlug, pr.ID, pr.Version)
	if err != nil {
		return err
	}

	return nil

}

// Create creates a pull request with the given specifications.
func (c *PullRequestClient) Create(ctx context.Context, title, branch, baseBranch, description string) (gitprovider.PullRequest, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	pr := &CreatePullRequest{
		Title:       title,
		Description: description,
		State:       "OPEN",
		Open:        true,
		Closed:      false,
		Locked:      false,
		ToRef: Ref{
			ID: fmt.Sprintf("refs/heads/%s", baseBranch),
			Repository: Repository{
				Slug:    repoSlug,
				Project: Project{Key: projectKey},
			},
		},
		FromRef: Ref{
			ID: fmt.Sprintf("refs/heads/%s", branch),
			Repository: Repository{
				Slug:    repoSlug,
				Project: Project{Key: projectKey},
			},
		},
	}

	created, err := c.client.PullRequests.Create(ctx, projectKey, repoSlug, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}
	return newPullRequest(created), nil
}

func (c *PullRequestClient) Edit(ctx context.Context, number int, opts gitprovider.EditOptions) (gitprovider.PullRequest, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}
	pr := PullRequest{}
	if opts.Title != nil {
		pr.Title = *opts.Title
	}
	edited, err := c.client.PullRequests.Update(ctx, projectKey, repoSlug, &pr)
	if err != nil {
		return nil, fmt.Errorf("failed to edit pull request: %w", err)
	}

	return newPullRequest(edited), nil
}

func validatePullRequestsAPI(apiObj *PullRequest) error {
	return validateAPIObject("Stash.PullRequest", func(validator validation.Validator) {
		// Make sure there is a version and a title
		if apiObj.Version == 0 || apiObj.Title == "" {
			validator.Required("ID")
		}
	})
}
