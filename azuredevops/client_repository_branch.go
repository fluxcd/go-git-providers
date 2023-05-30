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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"
)

const (
	emptyObjectPlaceholder = "0000000000000000000000000000000000000000"
	gitRefPrefix           = "refs/heads/"
)

// BranchClient implements the gitprovider.BranchClient interface.
var _ gitprovider.BranchClient = &BranchClient{}

// BranchClient operates on the branch for a specific repository.
type BranchClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Create creates a branch with the given specifications.
func (b BranchClient) Create(ctx context.Context, branch, sha string) error {
	ref := gitRefPrefix + branch
	repositoryId := b.ref.GetRepository()
	project := b.ref.GetIdentity()
	oldObjectId := emptyObjectPlaceholder
	opts := []git.GitRefUpdate{
		{
			Name:        &ref,
			NewObjectId: &sha,
			OldObjectId: &oldObjectId,
		},
	}
	reference := git.UpdateRefsArgs{
		RefUpdates:   &opts,
		RepositoryId: &repositoryId,
		Project:      &project,
	}

	if _, err := b.g.UpdateRefs(ctx, reference); err != nil {
		return err
	}

	return nil
}
