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
	"strings"

	"code.gitea.io/sdk/gitea"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// TreeClient implements the gitprovider.TreeClient interface.
var _ gitprovider.TreeClient = &TreeClient{}

// TreeClient operates on the trees in a specific repository.
type TreeClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns a tree
func (c *TreeClient) Get(ctx context.Context, sha string, recursive bool) (*gitprovider.TreeInfo, error) {
	tree, resp, err := c.c.GetTrees(c.ref.GetIdentity(), c.ref.GetRepository(), gitea.ListTreeOptions{
		Ref:       sha,
		Recursive: recursive,
	})
	if err != nil {
		return nil, handleHTTPError(resp, err)
	}

	treeEntries := make([]*gitprovider.TreeEntry, len(tree.Entries))
	for ind, treeEntry := range tree.Entries {
		var size int64
		if treeEntry.Type != "tree" {
			size = treeEntry.Size
		}
		treeEntries[ind] = &gitprovider.TreeEntry{
			Path: treeEntry.Path,
			Mode: treeEntry.Mode,
			Type: treeEntry.Type,
			Size: int(size),
			SHA:  treeEntry.SHA,
			URL:  treeEntry.URL,
		}
	}

	treeInfo := gitprovider.TreeInfo{
		SHA:       tree.SHA,
		Tree:      treeEntries,
		Truncated: tree.Truncated,
	}

	return &treeInfo, nil
}

// List files (blob) in a tree, sha is represented by the branch name
func (c *TreeClient) List(ctx context.Context, sha string, path string, recursive bool) ([]*gitprovider.TreeEntry, error) {
	treeInfo, err := c.Get(ctx, sha, recursive)
	if err != nil {
		return nil, err
	}
	treeEntries := make([]*gitprovider.TreeEntry, 0)
	for _, treeEntry := range treeInfo.Tree {
		if treeEntry.Type == "blob" {
			if path == "" || (path != "" && strings.HasPrefix(treeEntry.Path, path)) {
				treeEntries = append(treeEntries, &gitprovider.TreeEntry{
					Path: treeEntry.Path,
					Mode: treeEntry.Mode,
					Type: treeEntry.Type,
					Size: treeEntry.Size,
					SHA:  treeEntry.SHA,
					URL:  treeEntry.URL,
				})
			}
		}
	}

	return treeEntries, nil
}
