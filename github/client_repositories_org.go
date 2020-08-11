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
	"errors"
	"reflect"

	"github.com/google/go-github/v32/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// OrgRepositoriesClient implements the gitprovider.OrgRepositoriesClient interface
var _ gitprovider.OrgRepositoriesClient = &OrgRepositoriesClient{}

// OrgRepositoriesClient operates on repositories the user has access to
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
	apiObj, err := getRepository(c.c, ctx, ref)
	if err != nil {
		return nil, err
	}
	return newOrgRepository(c.clientContext, apiObj, ref), nil
}

// List all repositories in the given organization.
//
// List returns all available repositories, using multiple paginated requests if needed.
func (c *OrgRepositoriesClient) List(ctx context.Context, ref gitprovider.OrganizationRef) ([]gitprovider.OrgRepository, error) {
	// Make sure the OrganizationRef is valid
	if err := validateOrganizationRef(ref, c.domain); err != nil {
		return nil, err
	}

	// Get all of the repositories in the organization using pagination.
	var apiObjs []*github.Repository
	opts := &github.RepositoryListByOrgOptions{}
	err := allPages(&opts.ListOptions, func() (*github.Response, error) {
		// GET /orgs/{org}/repos
		pageObjs, resp, listErr := c.c.Repositories.ListByOrg(ctx, ref.Organization, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, handleHTTPError(err)
	}

	// Traverse the list, and return a list of OrgRepository objects
	repos := make([]gitprovider.OrgRepository, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Make sure apiObj is valid
		if err := validateRepositoryAPI(apiObj); err != nil {
			return nil, err
		}

		repos = append(repos, newOrgRepository(c.clientContext, apiObj, gitprovider.OrgRepositoryRef{
			OrganizationRef: ref,
			RepositoryName:  *apiObj.Name,
		}))
	}
	return repos, nil
}

// Create creates a repository for the given organization, with the data and options
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *OrgRepositoriesClient) Create(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (gitprovider.OrgRepository, error) {
	// Make sure the RepositoryRef is valid
	if err := validateOrgRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}

	apiObj, err := createRepository(c.c, ctx, ref, ref.Organization, req, opts...)
	if err != nil {
		return nil, err
	}
	return newOrgRepository(c.clientContext, apiObj, ref), nil
}

// Reconcile makes sure req is the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *OrgRepositoriesClient) Reconcile(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (gitprovider.OrgRepository, bool, error) {
	// First thing, validate the request
	if err := req.ValidateInfo(); err != nil {
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
	// Run generic reconciliation
	actionTaken, err := reconcileRepository(ctx, actual, req)
	return actual, actionTaken, err
}

func getRepository(c *github.Client, ctx context.Context, ref gitprovider.RepositoryRef) (*github.Repository, error) {
	// GET /repos/{owner}/{repo}
	apiObj, _, err := c.Repositories.Get(ctx, ref.GetIdentity(), ref.GetRepository())
	return validateRepositoryAPIResp(apiObj, err)
}

func createRepository(c *github.Client, ctx context.Context, ref gitprovider.RepositoryRef, orgName string, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (*github.Repository, error) {
	// Make sure the request is valid
	if err := req.ValidateInfo(); err != nil {
		return nil, err
	}
	// Assemble the options struct based on the given options
	o, err := gitprovider.MakeRepositoryCreateOptions(opts...)
	if err != nil {
		return nil, err
	}
	// Default the request object
	req.Default()

	// Convert to the API object and apply the options
	data := repositoryToAPI(&req, ref)
	applyRepoCreateOptions(&data, o)

	return createRepositoryData(c, ctx, orgName, &data)
}

func createRepositoryData(c *github.Client, ctx context.Context, orgName string, data *github.Repository) (*github.Repository, error) {
	// POST /user/repos or
	// POST /orgs/{org}/repos
	// depending on orgName
	apiObj, _, err := c.Repositories.Create(ctx, orgName, data)
	return validateRepositoryAPIResp(apiObj, err)
}

func reconcileRepository(ctx context.Context, actual gitprovider.UserRepository, req gitprovider.RepositoryInfo) (actionTaken bool, err error) {
	// If the desired matches the actual state, just return the actual state
	if reflect.DeepEqual(req, actual.Get()) {
		return
	}

	// Populate the desired state to the current-actual object
	if err = actual.Set(req); err != nil {
		return
	}
	// Apply the desired state by running Update
	err = actual.Update(ctx)
	actionTaken = true
	return
}

func toCreateOpts(opts ...gitprovider.RepositoryReconcileOption) []gitprovider.RepositoryCreateOption {
	// Convert RepositoryReconcileOption => RepositoryCreateOption
	createOpts := make([]gitprovider.RepositoryCreateOption, 0, len(opts))
	for _, opt := range opts {
		createOpts = append(createOpts, opt)
	}
	return createOpts
}
