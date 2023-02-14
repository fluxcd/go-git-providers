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

func (c *CommitClient) ListPage(ctx context.Context, branch string, perPage int, page int) ([]gitprovider.Commit, error) {
	//TODO implement me
	panic("implement me")
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

	commitsList, err := c.g.GetRefs(ctx, git.GetRefsArgs{
		RepositoryId: &repositoryId,
		Project:      &projectId,
	})
	latestCommitTreeSHA := commitsList.Value[0].ObjectId

	// create the commit now
	refArgs := []git.GitRefUpdate{{
		Name:        &ref,
		OldObjectId: latestCommitTreeSHA,
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
