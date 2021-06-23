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
	"errors"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// OrgRepositoriesClient implements the gitprovider.OrgRepositoriesClient interface.
var _ gitprovider.OrgRepositoriesClient = &OrgRepositoriesClient{}

// OrgRepositoriesClient operates on repositories the user has access to.
type OrgRepositoriesClient struct {
	*clientContext
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *OrgRepositoriesClient) Get(ctx context.Context, ref gitprovider.OrgRepositoryRef) (gitprovider.OrgRepository, error) {
	// Make sure the OrgRepositoryRef is valid
	if err := validateOrgRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}

	c.log.V(2).Info("getting repository", "org", ref.GetIdentity(), "repo", ref.GetRepository())

	apiObj, err := c.c.GetRepository(ctx, ref.GetIdentity(), ref.GetRepository())
	if err != nil {
		return nil, err
	}
	return newProjectRepository(c.clientContext, apiObj, ref), nil
}

// List all repositories in the given organization.
//
// List returns all available repositories, using multiple paginated requests if needed.
func (c *OrgRepositoriesClient) List(ctx context.Context, ref gitprovider.OrganizationRef) ([]gitprovider.OrgRepository, error) {
	// Make sure the OrganizationRef is valid
	if err := validateOrganizationRef(ref, c.domain); err != nil {
		return nil, err
	}

	apiObjs, err := c.c.ListRepositories(ctx, ref.Organization)
	if err != nil {
		c.clientContext.log.V(1).Error(err, "failed to list repositories")
		return nil, err
	}

	// Traverse the list, and return a list of OrgRepository objects
	repos := make([]gitprovider.OrgRepository, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// apiObj is already validated at ListOrgRepos
		repos = append(repos, newProjectRepository(c.clientContext, apiObj, gitprovider.OrgRepositoryRef{
			OrganizationRef: ref,
			RepositoryName:  apiObj.Name,
		}))
	}
	return repos, nil
}

// Create creates a repository for the given organization, with the data and options.
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *OrgRepositoriesClient) Create(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (gitprovider.OrgRepository, error) {
	// Make sure the RepositoryRef is valid
	if err := validateOrgRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}

	apiObj, err := createRepository(ctx, c.c, *c.clientContext, ref, ref.Organization, req, opts...)
	if err != nil {
		return nil, err
	}
	return newProjectRepository(c.clientContext, apiObj, ref), nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *OrgRepositoriesClient) Reconcile(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (gitprovider.OrgRepository, bool, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, false, err
	}

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
	actionTaken, err := reconcileRepository(ctx, actual, req)
	return actual, actionTaken, err
}

//nolint
func createRepository(ctx context.Context, c stashClient, cx clientContext, ref gitprovider.RepositoryRef, projectName string, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (*Repository, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, err
	}

	// Assemble the options struct based on the given options
	o, err := gitprovider.MakeRepositoryCreateOptions(opts...)
	if err != nil {
		return nil, err
	}
	// Convert to the API object and apply the options
	data := repositoryToAPI(&req, ref)
	if err != nil {
		return nil, err
	}

	repo, err := c.CreateRepository(ctx, projectName, data)
	if err != nil {
		return nil, err
	}

	if o.AutoInit != nil && *(o.AutoInit) {
		user, err := c.GetUser(ctx, repo.SessionInfo.UserName)
		if err != nil {
			return nil, err
		}

		readmeContents := fmt.Sprintf("# %s\n%s", repo.Name, repo.Description)
		if err := createInitialCommit(ctx, c, &cx, getRepoHTTPref(repo.Links.Clone), user.Name, user.EmailAddress, readmeContents, *o.LicenseTemplate); err != nil {
			if e := c.DeleteRepository(ctx, projectName, repo.Name); e != nil {
				return nil, e
			}
			return nil, err
		}
	}

	return repo, nil
}

func getRepoHTTPref(clones []Clone) string {
	for _, clone := range clones {
		if clone.Name == "http" {
			return clone.Href
		}
	}
	return "no http ref found"
}

func reconcileRepository(ctx context.Context, actual gitprovider.UserRepository, req gitprovider.RepositoryInfo) (bool, error) {
	// If the desired matches the actual state, just return the actual state
	new := actual.Get()
	if req.Equals(new) {
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
