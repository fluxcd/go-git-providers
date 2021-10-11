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

// CommitClient implements the gitprovider.CommitClient interface.
var _ gitprovider.CommitClient = &CommitClient{}

// CommitClient operates on the commits for a specific repository.
type CommitClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// ListPage lists repository commits of the given page and page size.
func (c *CommitClient) ListPage(ctx context.Context, branch string, perPage, page int) ([]gitprovider.Commit, error) {
	commitList, err := c.listPage(ctx, branch, perPage, page)
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}
	// Cast to the generic []gitprovider.Commit
	commits := make([]gitprovider.Commit, 0, len(commitList))
	for _, commit := range commitList {
		commits = append(commits, commit)
	}
	return commits, nil
}

func (c *CommitClient) listPage(ctx context.Context, branch string, perPage, page int) ([]*commitType, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	apiObjs, err := c.client.Commits.ListPage(ctx, projectKey, repoSlug, branch, perPage, page)
	if err != nil {
		return nil, err
	}

	// Map the api object to our CommitType type
	commits := make([]*commitType, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		commits = append(commits, newCommit(apiObj))
	}
	return commits, nil
}

// Create creates a commit with the given specifications.
func (c *CommitClient) Create(ctx context.Context, branch string, message string, files []gitprovider.CommitFile) (gitprovider.Commit, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	repo, err := c.client.Repositories.Get(ctx, projectKey, repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", projectKey, repoSlug, err)
	}

	user, err := c.client.Users.Get(ctx, repo.Session.UserName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", repo.Session.UserName, err)
	}

	url := getRepoHTTPref(repo.Links.Clone)
	r, dir, err := c.client.Git.CloneRepository(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository %s: %w", url, err)
	}

	f := make([]CommitFile, 0, len(files))
	for _, file := range files {
		f = append(f, CommitFile{Path: file.Path, Content: file.Content})
	}
	commit, err := NewCommit(
		WithAuthor(&CommitAuthor{
			Name:  user.Name,
			Email: user.EmailAddress,
		}),
		WithMessage(message),
		WithURL(url),
		WithFiles(f))

	result, err := c.client.Git.CreateCommit(ctx, dir, r, branch, commit)
	if err != nil {
		return nil, fmt.Errorf("failed to create commit: %w", err)
	}

	err = c.client.Git.Push(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("failed to push commit: %w", err)
	}

	sha, err := c.client.Commits.Get(ctx, projectKey, repoSlug, result.SHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit %s: %w", result.SHA, err)
	}

	err = c.client.Git.Cleanup(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup repository: %w", err)
	}

	return newCommit(sha), nil
}
