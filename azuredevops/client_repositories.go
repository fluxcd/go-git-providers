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
	"errors"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
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
		return nil, handleHTTPError(err)
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

// Create creates a repository for the given organization, with the data and options.
// ErrAlreadyExists will be returned if the resource already exists.
func (c *RepositoriesClient) Create(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (gitprovider.OrgRepository, error) {
	if err := validateRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}
	gitRepositoryToCreate := git.GitRepositoryCreateOptions{
		Name: &ref.RepositoryName,
	}
	CreateRepositoryArgs := git.CreateRepositoryArgs{
		GitRepositoryToCreate: &gitRepositoryToCreate,
		Project:               &ref.Organization,
	}

	// Create a git repository in a team project.
	apiObj, err := c.g.CreateRepository(ctx, CreateRepositoryArgs)
	if err != nil {
		return nil, handleHTTPError(err)
	}
	return newRepository(c.clientContext, *apiObj, ref), nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *RepositoriesClient) Reconcile(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (gitprovider.OrgRepository, bool, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	// For Azure devops this enforces a default branch name of "main" which shouldn't be the case
	// if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
	// 	return nil, false, err
	// }

	actual, err := c.Get(ctx, ref)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			resp, err := c.Create(ctx, ref, req, toCreateOpts(opts...)...)
			return resp, true, err
		}

		// Unexpected path, Get should succeed or return NotFound
		return nil, false, err
	}

	// Run generic reconciliation
	actionTaken, err := reconcileRepository(ctx, actual, req)
	return actual, actionTaken, err
}

func reconcileRepository(ctx context.Context, actual gitprovider.OrgRepository, req gitprovider.RepositoryInfo) (bool, error) {
	// If the desired matches the actual state, just return the actual state
	if req.Equals(actual.Get()) {
		return false, nil
	}
	// Populate the desired state to the current-actual object
	if err := actual.Set(req); err != nil {
		return false, err
	}
	// Apply the desired state by running Update
	return true, actual.Update(ctx)
}

func toCreateOpts(opts ...gitprovider.RepositoryReconcileOption) []gitprovider.RepositoryCreateOption {
	// Convert RepositoryReconcileOption => RepositoryCreateOption
	createOpts := make([]gitprovider.RepositoryCreateOption, 0, len(opts))
	for _, opt := range opts {
		createOpts = append(createOpts, opt)
	}
	return createOpts
}
