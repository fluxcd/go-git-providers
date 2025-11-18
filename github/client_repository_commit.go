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
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v75/github"
)

var githubNewFileMode = "100644"
var githubBlobTypeFile = "blob"

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

	treeEntries := make([]*github.TreeEntry, 0)
	for _, file := range files {
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path:    file.Path,
			Mode:    &githubNewFileMode,
			Type:    &githubBlobTypeFile,
			Content: file.Content,
		})
	}

	commits, err := c.ListPage(ctx, branch, 1, 0)
	if err != nil {
		return nil, err
	}

	latestCommitTreeSHA := commits[0].Get().TreeSha

	tree, _, err := c.c.Client().Git.CreateTree(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), latestCommitTreeSHA, treeEntries)
	if err != nil {
		return nil, err
	}

	latestCommitSHA := commits[0].Get().Sha
	nCommit, _, err := c.c.Client().Git.CreateCommit(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), github.Commit{
		Message: &message,
		Tree:    tree,
		Parents: []*github.Commit{
			{
				SHA: &latestCommitSHA,
			},
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	if nCommit.SHA == nil {
		// The UpdateRef API requires a SHA.
		return nil, fmt.Errorf("created commit has no SHA")
	}

	ref := "refs/heads/" + branch
	updateRef := github.UpdateRef{
		SHA:   *nCommit.SHA,
		Force: github.Ptr(true),
	}

	if _, _, err := c.c.Client().Git.UpdateRef(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), ref, updateRef); err != nil {
		return nil, err
	}

	return newCommit(c, nCommit), nil
}
