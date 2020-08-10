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
	gitprovider "github.com/fluxcd/go-git-providers"
)

func newRepositoryClient(ctx *clientContext, repo *gitprovider.Repository) *RepositoryClient {
	return &RepositoryClient{
		clientContext: ctx,
		repo:          repo,
		teams: &RepositoryTeamAccessClient{
			clientContext: ctx,
			info:          repo.RepositoryInfo,
		},
		keys: &DeployKeyClient{
			clientContext: ctx,
			info:          repo.RepositoryInfo,
		},
	}
}

// RepositoryClient implements the gitprovider.RepositoryClient interface
var _ gitprovider.RepositoryClient = &RepositoryClient{}

// RepositoryClient operates on a given/specific repository
type RepositoryClient struct {
	*clientContext
	repo *gitprovider.Repository

	teams *RepositoryTeamAccessClient
	keys  *DeployKeyClient
}

// TeamAccess returns a client for operating on the teams that have access to this specific repository
func (c *RepositoryClient) TeamAccess() gitprovider.RepositoryTeamAccessClient {
	return c.teams
}

// Credentials gives access to manipulating credentials for accessing this specific repository
func (c *RepositoryClient) DeployKeys() gitprovider.DeployKeyClient {
	return c.keys
}
