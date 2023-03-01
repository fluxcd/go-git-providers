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

// OrgRepositoriesClient implements the gitprovider.OrgRepositoriesClient interface.
var _ gitprovider.OrgRepositoriesClient = &RepositoriesClient{}

// OrgRepositoriesClient operates on repositories the user has access to.

type RepositoriesClient struct {
	*clientContext
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *RepositoriesClient) Get(ctx context.Context, ref gitprovider.OrgRepositoryRef) (gitprovider.OrgRepository, error) {
	// GET /repos/{owner}/{repo}
	opts := git.GetRepositoryArgs{RepositoryId: &ref.RepositoryName, Project: &ref.Organization}
	apiObj, err := c.g.GetRepository(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newRepository(c.clientContext, *apiObj, ref), nil
}

func (c *RepositoriesClient) List(ctx context.Context, ref gitprovider.OrganizationRef) ([]gitprovider.OrgRepository, error) {

	// Make sure the OrganizationRef is valid
	if err := validateOrganizationRef(ref, c.domain); err != nil {
		return nil, err
	}

	opts := git.GetRepositoriesArgs{Project: &ref.Organization}
	apiObjs, err := c.g.GetRepositories(ctx, opts)

	if err != nil {
		return nil, err
	}

	// Traverse the list, and return a list of UserRepository objects
	repos := make([]gitprovider.OrgRepository, 0, len(*apiObjs))
	for _, apiObj := range *apiObjs {
		// apiObj is already validated at ListUserRepos
		repos = append(repos, newRepository(c.clientContext, apiObj, gitprovider.OrgRepositoryRef{
			OrganizationRef: ref,
			RepositoryName:  *apiObj.Name,
		}))
	}
	return repos, nil
}

func (c *RepositoriesClient) Create(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (gitprovider.OrgRepository, error) {

	// if err := validateRepositoryRef(ref, c.domain); err != nil {
	// 	return nil, err
	// }

	// gitRepositoryToCreate := git.GitRepositoryCreateOptions{}
	// CreateRepositoryArgs := git.CreateRepositoryArgs{
	// 	git.GitRepositoryToCreate: &gitRepositoryToCreate,
	// 	Project:                   &ref.Organization,
	// 	Info:                      &req,
	// }

	// // Create a git repository in a team project.
	// apiObj, err := c.g.CreateRepository(ctx, CreateRepositoryArgs)
	// if err != nil {
	// 	return nil, err
	// }
	// return newOrgRepository(c.clientContext, apiObj, ref), nil
}

func (c *RepositoriesClient) Reconcile(ctx context.Context, r gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (resp gitprovider.OrgRepository, actionTaken bool, err error) {
	//TODO implement me
	panic("implement me")
}
