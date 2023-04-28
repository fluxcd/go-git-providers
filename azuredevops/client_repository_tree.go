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

// TreeClient implements the gitprovider.TreeClient interface.
var _ gitprovider.TreeClient = &TreeClient{}

// TreeClient operates on the trees in a specific repository.
type TreeClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns a tree object for a given SHA.
func (c *TreeClient) Get(ctx context.Context, sha string, recursive bool) (*gitprovider.TreeInfo, error) {
	repoName := c.ref.GetRepository()
	projectName := c.ref.GetIdentity()
	opts := git.GetTreeArgs{
		RepositoryId: &repoName,
		Project:      &projectName,
		Sha1:         &sha,
		Recursive:    &recursive,
	}
	apiObj, err := c.g.GetTree(ctx, opts)
	if err != nil {
		return nil, err
	}
	treeEntries := make([]*gitprovider.TreeEntry, len(*apiObj.TreeEntries))
	for ind, treeEntry := range *apiObj.TreeEntries {
		size := 0
		if *treeEntry.GitObjectType != "tree" {
			size = int(*treeEntry.Size)
		}
		treeEntries[ind] = &gitprovider.TreeEntry{
			Path: *treeEntry.RelativePath,
			Mode: *treeEntry.Mode,
			Type: string(*treeEntry.GitObjectType),
			Size: size,
			SHA:  *treeEntry.ObjectId,
			URL:  *treeEntry.Url,
		}
	}
	treeInfo := gitprovider.TreeInfo{
		SHA:  *apiObj.ObjectId,
		Tree: treeEntries,
	}
	return &treeInfo, nil
}

// List files (blob) in a tree
func (c *TreeClient) List(ctx context.Context, sha string, path string, recursive bool) ([]*gitprovider.TreeEntry, error) {
	repoName := c.ref.GetRepository()
	projectName := c.ref.GetIdentity()
	opts := git.GetTreeArgs{
		RepositoryId: &repoName,
		Project:      &projectName,
		Sha1:         &sha,
		Recursive:    &recursive,
	}
	apiObj, err := c.g.GetTree(ctx, opts)
	if err != nil {
		return nil, err
	}
	treeEntries := make([]*gitprovider.TreeEntry, 0, len(*apiObj.TreeEntries))
	for _, treeEntry := range *apiObj.TreeEntries {
		if *treeEntry.GitObjectType == "blob" {
			treeEntries = append(treeEntries, &gitprovider.TreeEntry{
				Path: *treeEntry.RelativePath,
				Mode: *treeEntry.Mode,
				Type: string(*treeEntry.GitObjectType),
				Size: int(*treeEntry.Size),
				SHA:  *treeEntry.ObjectId,
				URL:  *treeEntry.Url,
			})
		}
	}

	return treeEntries, nil
}
