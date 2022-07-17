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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v42/github"
)

// TreeClient implements the gitprovider.TreeClient interface.
var _ gitprovider.TreeClient = &TreeClient{}

// TreeClient operates on the trees in a specific repository.
type TreeClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Create creates,updates,deletes a tree
func (c *TreeClient) Create(ctx context.Context, tree *gitprovider.TreeInfo) (*gitprovider.TreeInfo, error) {
	repoName := c.ref.GetRepository()
	repoOwner := c.ref.GetIdentity()

	treeEntries := make([]*github.TreeEntry, 0)
	for _, treeEntry := range tree.Tree {
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: &treeEntry.Path,
			Mode: &treeEntry.Mode,
			Type: &treeEntry.Type,
			Size: &treeEntry.Size,
			SHA:  &treeEntry.SHA,
			URL:  &treeEntry.URL,
		})
	}
	githubTree, _, err := c.c.Client().Git.CreateTree(ctx, repoOwner, repoName, tree.SHA, treeEntries)
	if err != nil {
		return nil, err
	}

	responseTreeEntries := make([]*gitprovider.TreeEntry, 0)
	for _, responseTreeEntry := range githubTree.Entries {
		size := 0
		if *responseTreeEntry.Type != "tree" {
			size = *responseTreeEntry.Size
		}
		responseTreeEntries = append(responseTreeEntries, &gitprovider.TreeEntry{
			Path: *responseTreeEntry.Path,
			Mode: *responseTreeEntry.Mode,
			Type: *responseTreeEntry.Type,
			Size: size,
			SHA:  *responseTreeEntry.SHA,
			URL:  *responseTreeEntry.URL,
		})
	}

	responseTreeInfo := gitprovider.TreeInfo{
		SHA:       *githubTree.SHA,
		Tree:      responseTreeEntries,
		Truncated: *githubTree.Truncated,
	}

	return &responseTreeInfo, nil
}

// Get returns a tree
func (c *TreeClient) Get(ctx context.Context, sha string, recursive bool) (*gitprovider.TreeInfo, error) {
	// GET /repos/{owner}/{repo}/git/trees
	repoName := c.ref.GetRepository()
	repoOwner := c.ref.GetIdentity()
	githubTree, _, err := c.c.Client().Git.GetTree(ctx, repoOwner, repoName, sha, true)
	if err != nil {
		return nil, err
	}

	treeEntries := make([]*gitprovider.TreeEntry, 0)
	for _, treeEntry := range githubTree.Entries {
		size := 0
		if *treeEntry.Type != "tree" {
			size = *treeEntry.Size
		}
		treeEntries = append(treeEntries, &gitprovider.TreeEntry{
			Path: *treeEntry.Path,
			Mode: *treeEntry.Mode,
			Type: *treeEntry.Type,
			Size: size,
			SHA:  *treeEntry.SHA,
			URL:  *treeEntry.URL,
		})
	}

	treeInfo := gitprovider.TreeInfo{
		SHA:       *githubTree.SHA,
		Tree:      treeEntries,
		Truncated: *githubTree.Truncated,
	}

	return &treeInfo, nil

}

// List files (blob) in a tree
func (c *TreeClient) List(ctx context.Context, sha string, recursive bool) ([]*gitprovider.TreeEntry, error) {
	treeInfo, err := c.Get(ctx, sha, recursive)
	if err != nil {
		return nil, err
	}
	treeEntries := make([]*gitprovider.TreeEntry, 0)
	for _, treeEntry := range treeInfo.Tree {
		if treeEntry.Type == "blob" {
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

	return treeEntries, nil
}

func createTreeEntry(githubTreeEntry github.TreeEntry) *gitprovider.TreeEntry {
	newTreeEntry := gitprovider.TreeEntry{
		Path: *githubTreeEntry.Path,
		Mode: *githubTreeEntry.Mode,
		Type: *githubTreeEntry.Type,
		Size: *githubTreeEntry.Size,
		SHA:  *githubTreeEntry.SHA,
		URL:  *githubTreeEntry.URL,
	}
	return &newTreeEntry
}
