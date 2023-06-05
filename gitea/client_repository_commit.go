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

	"code.gitea.io/sdk/gitea"
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
	apiObjs, err := c.c.ListCommits(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), branch, perPage, page)
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
// This method creates a commit with a single file.
// TODO: fix when gitea supports creating commits with multiple files
func (c *CommitClient) Create(ctx context.Context, branch string, message string, files []gitprovider.CommitFile) (gitprovider.Commit, error) {
	if len(files) == 0 || len(files) > 1 {
		return nil, fmt.Errorf("no files added")
	}

	resp, err := c.c.CreateCommits(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), *files[0].Path, &gitea.CreateFileOptions{
		Content: *files[0].Content,
		FileOptions: gitea.FileOptions{
			Message:    message,
			BranchName: branch,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create commit: %w", err)
	}

	commit := &gitea.Commit{
		HTMLURL: resp.Commit.HTMLURL,
		Author: &gitea.User{
			UserName: resp.Commit.Author.Name,
			Email:    resp.Commit.Author.Email,
		},
		Committer: &gitea.User{
			UserName: resp.Commit.Committer.Name,
			Email:    resp.Commit.Committer.Email,
		},
		Parents: resp.Commit.Parents,
	}
	commit.CommitMeta = &resp.Commit.CommitMeta

	return newCommit(c, commit), nil
}
