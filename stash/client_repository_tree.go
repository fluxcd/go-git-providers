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

package stash

import (
	"context"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
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
	return nil, fmt.Errorf("error creaing tree %s. not implemented in stash yet", tree.SHA)

}

// Get returns a tree
func (c *TreeClient) Get(ctx context.Context, sha string, recursive bool) (*gitprovider.TreeInfo, error) {
	return nil, fmt.Errorf("error getting tree %s. not implemented in stash yet", sha)

}

// List files (blob) in a tree
func (c *TreeClient) List(ctx context.Context, sha string, path string, recursive bool) ([]*gitprovider.TreeEntry, error) {
	return nil, fmt.Errorf("error listing tree items %s. not implemented in stash yet", sha)
}
