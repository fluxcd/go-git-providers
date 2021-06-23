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

package stash

import (
	"context"

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
		return nil, err
	}
	// Cast to the generic []gitprovider.Commit
	commits := make([]gitprovider.Commit, 0, len(commitList))
	for _, commit := range commitList {
		commits = append(commits, commit)
	}
	return commits, nil
}

func (c *CommitClient) listPage(ctx context.Context, branch string, perPage, page int) ([]*commitType, error) {
	apiObjs, err := c.c.ListCommitsPage(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), branch, perPage, page)
	if err != nil {
		return nil, err
	}

	// Map the api object to our CommitType type
	keys := make([]*commitType, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		keys = append(keys, newCommit(c, apiObj))
	}
	return keys, nil // keys, nil
}

// Create creates a commit with the given specifications.
func (c *CommitClient) Create(ctx context.Context, branch string, message string, files []gitprovider.CommitFile) (gitprovider.Commit, error) {

	repo, err := c.c.GetRepository(ctx, c.ref.GetIdentity(), c.ref.GetRepository())
	if err != nil {
		return nil, err
	}

	user, err := c.c.GetUser(ctx, repo.SessionInfo.UserName)
	if err != nil {
		return nil, err
	}

	commitHashes, err := createCommit(ctx, c.clientContext, getRepoHTTPref(repo.Links.Clone), user.Name, user.EmailAddress, branch, message, files)
	if err != nil {
		return nil, err
	}

	commit, err := c.c.GetCommit(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), commitHashes.hash)
	if err != nil {
		return nil, err
	}

	return newCommit(c, commit), nil
}
