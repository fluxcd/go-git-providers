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

package gitea

import (
	"context"

	"code.gitea.io/sdk/gitea"
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
// Creating a branch from a commit is noy supported by Gitea.
// see: https://github.com/go-gitea/gitea/issues/22139
func (c *BranchClient) Create(ctx context.Context, branch, sha string) error {

	// Doesn't seem to support specific sha?
	opts := gitea.CreateBranchOption{
		BranchName:    branch,
		OldBranchName: sha,
	}
	err := opts.Validate()
	if err != nil {
		return err
	}
	if _, _, err := c.c.CreateBranch(c.ref.GetIdentity(), c.ref.GetRepository(), opts); err != nil {
		return err
	}

	return nil
}
