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

// UserRepositoriesClient implements the gitprovider.UserRepositoriesClient interface
var _ gitprovider.UserRepositoriesClient = &UserRepositoriesClient{}

// UserRepositoriesClient operates on repositories the user has access to
type UserRepositoriesClient struct {
	*clientContext
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *UserRepositoriesClient) Get(ctx context.Context, ref gitprovider.UserRepositoryRef) (gitprovider.UserRepository, error) {
	// Make sure the UserRepositoryRef is valid
	if err := validateUserRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}
	apiObj, err := getRepository(c.c, ctx, ref)
	if err != nil {
		return nil, err
	}
	return newUserRepository(c.clientContext, apiObj, ref), nil
}

// List all repositories in the given organization.
//
// List returns all available repositories, using multiple paginated requests if needed.
func (c *UserRepositoriesClient) List(ctx context.Context, ref gitprovider.UserRef) ([]gitprovider.UserRepository, error) {
	// Make sure the UserRef is valid
	if err := validateUserRef(ref, c.domain); err != nil {
		return nil, err
	}

	// Get all of the user's repositories using pagination.
	var apiObjs []*github.Repository
	opts := &github.RepositoryListOptions{}
	err := allPages(&opts.ListOptions, func() (*github.Response, error) {
		// GET /users/{username}/repos
		pageObjs, resp, listErr := c.c.Repositories.List(ctx, ref.GetIdentity(), opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, handleHTTPError(err)
	}

	// Traverse the list, and return a list of UserRepository objects
	repos := make([]gitprovider.UserRepository, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Make sure name isn't nil
		if apiObj.Name == nil {
			return nil, fmt.Errorf("didn't expect name to be nil for repo: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
		}
		repos = append(repos, newUserRepository(c.clientContext, apiObj, gitprovider.UserRepositoryRef{
			UserRef:        ref,
			RepositoryName: *apiObj.Name,
		}))
	}
	return repos, nil
}

// Create creates a repository for the given organization, with the data and options
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *UserRepositoriesClient) Create(ctx context.Context, ref gitprovider.UserRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (gitprovider.UserRepository, error) {
	// Make sure the RepositoryRef is valid
	if err := validateUserRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}

	apiObj, err := createRepository(c.c, ctx, ref, "", req, opts...)
	if err != nil {
		return nil, err
	}
	return newUserRepository(c.clientContext, apiObj, ref), nil
}

// Reconcile makes sure req is the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *UserRepositoriesClient) Reconcile(ctx context.Context, ref gitprovider.UserRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (gitprovider.UserRepository, bool, error) {
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
	actionTaken, err := genericReconcile(ctx, actual, req)
	return actual, actionTaken, err
}