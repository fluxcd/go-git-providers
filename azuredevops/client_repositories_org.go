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

package azuredevops

import (
	"context"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// UserRepositoriesClient implements the gitprovider.UserRepositoriesClient interface.
var _ gitprovider.OrgRepositoriesClient = &RepositoriesClient{}

// UserRepositoriesClient operates on repositories the user has access to.

type RepositoriesClient struct {
	*clientContext
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *RepositoriesClient) Get(ctx context.Context, ref gitprovider.OrgRepositoryRef) (gitprovider.OrgRepository, error) {
	// GET /repos/{owner}/{repo}
	apiObj, err := c.c.GetRepo(ctx, ref.GetIdentity(), ref.GetRepository())
	if err != nil {
		return nil, err
	}
	return newUserRepository(c.clientContext, apiObj, ref), nil
}

func (c *RepositoriesClient) List(ctx context.Context, o gitprovider.OrganizationRef) ([]gitprovider.OrgRepository, error) {
	//TODO implement me
	panic("implement me")
}

func (c *RepositoriesClient) Create(ctx context.Context, r gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (gitprovider.OrgRepository, error) {
	//TODO implement me
	panic("implement me")
}

func (c *RepositoriesClient) Reconcile(ctx context.Context, r gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (resp gitprovider.OrgRepository, actionTaken bool, err error) {
	//TODO implement me
	panic("implement me")
}
