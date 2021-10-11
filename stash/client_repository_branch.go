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
)

// BranchClient implements the gitprovider.BranchClient interface.
var _ gitprovider.BranchClient = &BranchClient{}

// BranchClient operates on the branch for a specific repository.
type BranchClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Create creates a branch with the given specifications.
func (c *BranchClient) Create(ctx context.Context, branch, sha string) error {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	repo, err := c.client.Repositories.Get(ctx, projectKey, repoSlug)
	if err != nil {
		return fmt.Errorf("failed to get repository %s/%s: %w", projectKey, repoSlug, err)
	}

	user, err := c.client.Users.Get(ctx, repo.Session.UserName)
	if err != nil {
		return fmt.Errorf("failed to get user %s: %w", repo.Session.UserName, err)
	}

	url := getRepoHTTPref(repo.Links.Clone)

	r, dir, err := c.client.Git.CloneRepository(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	err = c.client.Git.CreateBranch(branch, r, sha)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	commit, err := NewCommit(
		WithAuthor(&CommitAuthor{
			Name:  user.Name,
			Email: user.EmailAddress,
		}),
		WithMessage("Create branch"),
		WithURL(url))

	_, err = c.client.Git.CreateCommit(ctx, dir, r, "", commit)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	err = c.client.Git.Push(ctx, r)
	if err != nil {
		return fmt.Errorf("failed to push commit: %w", err)
	}

	err = c.client.Git.Cleanup(dir)
	if err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	return nil
}

func (c *BranchClient) getDefault(ctx context.Context) (string, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	b, err := c.client.Branches.Default(ctx, projectKey, repoSlug)
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	return b.DisplayID, nil

}
