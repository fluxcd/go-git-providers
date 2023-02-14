/*
Copyright 2023 The Flux CD contributors.

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
	"fmt"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"
)

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
	repositoryId := c.ref.GetRepository()
	projectId := c.ref.GetIdentity()
	branchRetrieved := git.GitVersionDescriptor{
		Version: &branch,
	}
	searchCriteria := git.GitQueryCommitsCriteria{
		Top:         &page,
		ItemVersion: &branchRetrieved,
	}
	apiObjs, err := c.g.GetCommits(ctx, git.GetCommitsArgs{
		RepositoryId:   &repositoryId,
		Project:        &projectId,
		SearchCriteria: &searchCriteria,
	})
	if err != nil {
		return nil, err
	}

	// Map the api object to our CommitType type
	keys := make([]*commitType, 0, len(*apiObjs))
	keys = append(keys, newCommit(c, &git.GitPush{
		Commits: apiObjs,
	}))

	return keys, nil
}

func (c *CommitClient) Create(ctx context.Context, branch string, message string, files []gitprovider.CommitFile) (gitprovider.Commit, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files added")
	}
	repositoryId := c.ref.GetRepository()
	projectId := c.ref.GetIdentity()
	ref := "refs/heads/" + branch

	var change []interface{}
	var all []interface{}

	for _, file := range files {
		all = append([]interface{}{git.Change{
			ChangeType: &git.VersionControlChangeTypeValues.Add,
			Item: map[string]interface{}{
				"path": file.Path,
			},
			NewContent: &git.ItemContent{
				Content:     file.Content,
				ContentType: &git.ItemContentTypeValues.RawText,
			},
			SourceServerItem: nil,
			Url:              nil,
		}}, change...)
	}

	commits := []git.GitCommitRef{
		{
			Changes: &all,
			Comment: &message,
		}}

	// get latest commit from branch

	commitsList, err := c.ListPage(ctx, branch, 0, 1)
	if err != nil {
		return nil, err
	}

	latestCommitTreeSHA := commitsList[0].Get().Sha

	// create the commit now
	refArgs := []git.GitRefUpdate{{
		Name:        &ref,
		OldObjectId: &latestCommitTreeSHA,
	},
	}

	opts := git.CreatePushArgs{
		Push: &git.GitPush{
			Commits:    &commits,
			RefUpdates: &refArgs,
		},
		RepositoryId: &repositoryId,
		Project:      &projectId,
	}
	commit, err := c.g.CreatePush(ctx, opts)
	if err != nil {
		return nil, err
	}

	return newCommit(c, commit), nil
}
