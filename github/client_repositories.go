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
	"fmt"

	gitprovider "github.com/fluxcd/go-git-providers"
	"github.com/google/go-github/v32/github"
)

// RepositoriesClient implements the gitprovider.RepositoriesClient interface
var _ gitprovider.RepositoriesClient = &RepositoriesClient{}

// RepositoriesClient operates on repositories the user has access to
type RepositoriesClient struct {
	*clientContext
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *RepositoriesClient) Get(ctx context.Context, ref gitprovider.RepositoryRef) (*gitprovider.Repository, error) {
	// Make sure the RepositoryRef is valid
	if err := validateRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}

	// GET /repos/{owner}/{repo}
	apiObj, _, err := c.c.Repositories.Get(ctx, ref.GetIdentity(), ref.GetRepository())
	if err != nil {
		return nil, handleHTTPError(err)
	}

	return repositoryFromAPI(apiObj, ref), nil
}

// List all repositories in the given organization or user account.
//
// List returns all available repositories, using multiple paginated requests if needed.
func (c *RepositoriesClient) List(ctx context.Context, ref gitprovider.IdentityRef) ([]gitprovider.Repository, error) {
	// Make sure the IdentityRef is valid
	if err := validateIdentityRef(ref, c.domain); err != nil {
		return nil, err
	}
	// Get the name of the organization, or an empty string in the case of an user account
	orgName := resolveOrg(ref)

	var apiObjs []*github.Repository
	var err error
	if orgName == "" {
		opts := &github.RepositoryListOptions{}
		err = allPages(&opts.ListOptions, func() (*github.Response, error) {
			// GET /users/{username}/repos
			pageObjs, resp, listErr := c.c.Repositories.List(ctx, ref.GetIdentity(), opts)
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		})
	} else {
		opts := &github.RepositoryListByOrgOptions{}
		err = allPages(&opts.ListOptions, func() (*github.Response, error) {
			// GET /orgs/{org}/repos
			pageObjs, resp, listErr := c.c.Repositories.ListByOrg(ctx, orgName, opts)
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		})
	}
	if err != nil {
		return nil, handleHTTPError(err)
	}

	repos := make([]gitprovider.Repository, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Make sure name isn't nil
		if apiObj.Name == nil {
			return nil, fmt.Errorf("didn't expect name to be nil for repo: %+v: %w", apiObj, gitprovider.ErrUnexpectedEvent)
		}
		// Track the info of the repo and add the object to the list
		repoInfo := gitprovider.RepositoryInfo{
			IdentityRef:    ref,
			RepositoryName: *apiObj.Name,
		}
		repos = append(repos, *repositoryFromAPI(apiObj, repoInfo))
	}

	return repos, nil
}

// Create creates a repository at the given organization path, with the given URL-encoded name and options
//
// ErrAlreadyExists will be returned if the resource already exists.
//
// resp will contain any updated information given by the server; hence it is encouraged
// to stop using req after this call, and use resp instead.
func (c *RepositoriesClient) Create(ctx context.Context, req *gitprovider.Repository, opts ...gitprovider.RepositoryCreateOption) (resp *gitprovider.Repository, err error) {
	// Make sure the RepositoryRef is valid
	if err := validateRepositoryRef(req.RepositoryInfo, c.domain); err != nil {
		return nil, err
	}

	// Make sure the request is valid
	if err := req.ValidateCreate(); err != nil {
		return nil, err
	}
	// Assemble the options struct based on the given options
	o, err := gitprovider.MakeRepositoryCreateOptions(opts...)
	if err != nil {
		return nil, err
	}
	// Get the name of the organization, or an empty string in the case of an user account
	orgName := resolveOrg(req.IdentityRef)

	// Default the request object
	req.Default()
	// Convert to the API object and apply the options
	data := repositoryToAPI(req)
	applyRepoCreateOptions(data, o)

	// POST /user/repos or
	// POST /orgs/{org}/repos
	// depending on orgName
	apiObj, _, err := c.c.Repositories.Create(ctx, orgName, data)
	if err != nil {
		return nil, handleHTTPError(err)
	}

	return repositoryFromAPI(apiObj, req.RepositoryInfo), nil
}

// Update will update the desired state of the repository. Only set fields will be respected.
//
// ErrNotFound is returned if the resource does not exist.
//
// resp will contain any updated information given by the server; hence it is encouraged
// to stop using req after this call, and use resp instead.
func (c *RepositoriesClient) Update(ctx context.Context, req *gitprovider.Repository) (resp *gitprovider.Repository, err error) {
	// Make sure the RepositoryRef is valid
	if err := validateRepositoryRef(req.RepositoryInfo, c.domain); err != nil {
		return nil, err
	}
	// Make sure the request is valid
	if err := req.ValidateUpdate(); err != nil {
		return nil, err
	}

	// Convert to the API object and apply the options
	data := repositoryToAPI(req)

	// PATCH /repos/{owner}/{repo}
	apiObj, _, err := c.c.Repositories.Edit(ctx, req.GetIdentity(), req.GetRepository(), data)
	if err != nil {
		return nil, handleHTTPError(err)
	}

	return repositoryFromAPI(apiObj, req.RepositoryInfo), nil
}

// Reconcile makes sure req is the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// resp will contain any updated information given by the server; hence it is encouraged
// to stop using req after this call, and use resp instead.
func (c *RepositoriesClient) Reconcile(ctx context.Context, req *gitprovider.Repository, opts ...gitprovider.RepositoryReconcileOption) (resp *gitprovider.Repository, actionTaken bool, err error) {
	actual, err := c.Get(ctx, req.RepositoryInfo)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			// Convert RepositoryReconcileOption => RepositoryCreateOption
			createOpts := make([]gitprovider.RepositoryCreateOption, 0, len(opts))
			for _, opt := range opts {
				createOpts = append(createOpts, opt)
			}
			resp, err = c.Create(ctx, req, createOpts...)
			actionTaken = true
			return
		}

		// Unexpected path, Get should succeed or return NotFound
		return nil, false, err
	}

	// If the desired matches the actual state, just return the actual state
	if equals, _ := gitprovider.Equals(req, actual); equals {
		return actual, false, nil
	}

	// Update the actual state to be the desired state
	resp, err = c.Update(ctx, req)
	actionTaken = true
	return
}

// Delete removes the given repository.
//
// ErrNotFound is returned if the resource does not exist.
func (c *RepositoriesClient) Delete(ctx context.Context, ref gitprovider.RepositoryRef) error {
	_, err := c.c.Repositories.Delete(ctx, ref.GetIdentity(), ref.GetRepository())
	if err != nil {
		return handleHTTPError(err)
	}
	return nil
}
