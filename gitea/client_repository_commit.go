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

package gitea

import (
	"context"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

var giteaNewFileMode = "100644"
var giteaBlobTypeFile = "blob"

// CommitClient implements the gitprovider.CommitClient interface.
var _ gitprovider.CommitClient = &CommitClient{}

// CommitClient operates on the commits for a specific repository.
type CommitClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// ListPage lists all repository commits of the given page and page size.
// ListPage returns all available repository commits
// using multiple paginated requests if needed.
func (c *CommitClient) ListPage(ctx context.Context, branch string, perPage, page int) ([]gitprovider.Commit, error) {
	dks, err := c.listPage(ctx, branch, perPage, page)
	if err != nil {
		return nil, err
	}
	// Cast to the generic []gitprovider.Commit
	commits := make([]gitprovider.Commit, 0, len(dks))
	for _, dk := range dks {
		commits = append(commits, dk)
	}
	return commits, nil
}

func (c *CommitClient) listPage(ctx context.Context, branch string, perPage, page int) ([]*commitType, error) {
	// GET /repos/{owner}/{repo}/commits
	apiObjs, err := c.c.ListCommitsPage(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), branch, perPage, page)
	if err != nil {
		return nil, err
	}

	// Map the api object to our CommitType type
	keys := make([]*commitType, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		keys = append(keys, newCommit(c, apiObj))
	}

	return keys, nil
}

// Create creates a commit with the given specifications.
func (c *CommitClient) Create(ctx context.Context, branch string, message string, files []gitprovider.CommitFile) (gitprovider.Commit, error) {

	if len(files) == 0 {
		return nil, fmt.Errorf("no files added")
	}

	// TODO: fix workaround
	// Create temp branch
	// Add files to the branch
	// Sqash merge the temp branch to target branch
	// Remove temp branch
	// make changes locally and push a single commit
	// opts := gitea.ListCommitOptions{}
	// _, _, _ := c.c.Client().ListRepoCommits(c.ref.GetIdentity(), c.ref.GetRepository(), opts)
	// Gitea api currently doesn't yet support creating commits with multiple files/patches
	// see issue https://github.com/go-gitea/gitea/issues/14619#
	return nil, fmt.Errorf("gitea doesn't yet support creating commits: %w", gitprovider.ErrNoProviderSupport)
}
