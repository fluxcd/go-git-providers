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
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// TreeClient implements the gitprovider.TreeClient interface.
var _ gitprovider.TreeClient = &TreeClient{}

// TreeClient operates on the trees in a specific repository.
type TreeClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns a single tree using the SHA1 value for that tree.
// uses https://docs.github.com/en/rest/git/trees#get-a-tree
func (c *TreeClient) Get(ctx context.Context, sha string, recursive bool) (*gitprovider.TreeInfo, error) {
	// GET /repos/{owner}/{repo}/git/trees
	repoName := c.ref.GetRepository()
	repoOwner := c.ref.GetIdentity()
	githubTree, _, err := c.c.Client().Git.GetTree(ctx, repoOwner, repoName, sha, recursive)
	if err != nil {
		return nil, err
	}

	treeEntries := make([]*gitprovider.TreeEntry, len(githubTree.Entries))
	for ind, treeEntry := range githubTree.Entries {
		size := 0
		if *treeEntry.Type != "tree" {
			size = *treeEntry.Size
		}
		treeEntries[ind] = &gitprovider.TreeEntry{
			Path: *treeEntry.Path,
			Mode: *treeEntry.Mode,
			Type: *treeEntry.Type,
			Size: size,
			SHA:  *treeEntry.SHA,
			URL:  *treeEntry.URL,
		}
	}

	treeInfo := gitprovider.TreeInfo{
		SHA:       *githubTree.SHA,
		Tree:      treeEntries,
		Truncated: *githubTree.Truncated,
	}

	return &treeInfo, nil

}

// List files (blob) in a tree givent the tree sha (path is not used with Github Tree client)
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
