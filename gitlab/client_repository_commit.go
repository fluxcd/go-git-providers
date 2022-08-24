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
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/xanzy/go-gitlab"
)

// CommitClient implements the gitprovider.CommitClient interface.
var _ gitprovider.CommitClient = &CommitClient{}

// CommitClient operates on the commits for a specific repository.
type CommitClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// ListPage lists repository commits of the given page and page size.
func (c *CommitClient) ListPage(_ context.Context, branch string, perPage, page int) ([]gitprovider.Commit, error) {
	dks, err := c.listPage(branch, perPage, page)
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

// Get a list of repository commits in a project.
func (c *CommitClient) listPage(branch string, perPage, page int) ([]*commitType, error) {
	// GET /projects/{id}/repository/commits
	p := getRepoPath(c.ref)
	p = strings.ReplaceAll(p, "/", "%2F")
	apiObjs, err := c.c.ListCommitsPage(p, branch, perPage, page)
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
func (c *CommitClient) Create(_ context.Context, branch string, message string, files []gitprovider.CommitFile) (gitprovider.Commit, error) {

	if len(files) == 0 {
		return nil, fmt.Errorf("no files added")
	}

	commitActions := make([]*gitlab.CommitActionOptions, 0)
	for _, file := range files {
		fileAction := gitlab.FileCreate
		if file.Content == nil {
			fileAction = gitlab.FileDelete
		}

		commitActions = append(commitActions, &gitlab.CommitActionOptions{
			Action:   &fileAction,
			FilePath: file.Path,
			Content:  file.Content,
		})
	}

	opts := &gitlab.CreateCommitOptions{
		Branch:        &branch,
		CommitMessage: &message,
		Actions:       commitActions,
	}

	commit, _, err := c.c.Client().Commits.CreateCommit(getRepoPath(c.ref), opts)
	if err != nil {
		return nil, err
	}

	return newCommit(c, commit), nil
}
