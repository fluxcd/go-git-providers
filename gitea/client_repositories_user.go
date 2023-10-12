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

package gitea

import (
	"context"
	"errors"
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// UserRepositoriesClient implements the gitprovider.UserRepositoriesClient interface.
var _ gitprovider.UserRepositoriesClient = &UserRepositoriesClient{}

// UserRepositoriesClient operates on repositories the user has access to.
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
	// GET /repos/{owner}/{repo}
	apiObj, err := getRepo(c.c, ref.GetIdentity(), ref.GetRepository())
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

	// GET /users/{username}/repos
	apiObjs, err := c.listUserRepos(ref.UserLogin)
	if err != nil {
		return nil, err
	}

	// Traverse the list, and return a list of UserRepository objects
	repos := make([]gitprovider.UserRepository, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// apiObj is already validated at ListUserRepos
		repos = append(repos, newUserRepository(c.clientContext, apiObj, gitprovider.UserRepositoryRef{
			UserRef:        ref,
			RepositoryName: apiObj.Name,
		}))
	}
	return repos, nil
}

func (c *UserRepositoriesClient) listUserRepos(username string) ([]*gitea.Repository, error) {
	opts := gitea.ListReposOptions{}
	apiObjs := []*gitea.Repository{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /users/{username}/repos
		pageObjs, resp, listErr := c.c.ListUserRepos(username, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return validateRepositoryObjects(apiObjs)
}

// GetUserLogin returns the authenticated user
func (c *UserRepositoriesClient) GetUserLogin(ctx context.Context) (gitprovider.IdentityRef, error) {
	// GET /user
	user, _, err := c.c.GetMyUserInfo()
	if err != nil {
		return nil, err
	}
	return gitprovider.UserRef{
		Domain:    c.domain,
		UserLogin: user.UserName,
	}, nil
}

// Create creates a repository for the given organization, with the data and options
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *UserRepositoriesClient) Create(ctx context.Context,
	ref gitprovider.UserRepositoryRef,
	req gitprovider.RepositoryInfo,
	opts ...gitprovider.RepositoryCreateOption,
) (gitprovider.UserRepository, error) {
	// Make sure the RepositoryRef is valid
	if err := validateUserRepositoryRef(ref, c.domain); err != nil {
		return nil, err
	}

	// extra validation to ensure we don't create a project when the wrong owner
	// is passed in
	idRef, err := c.GetUserLogin(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get owner from API")
	}

	if ref.GetIdentity() != idRef.GetIdentity() {
		return nil, gitprovider.NewErrIncorrectUser(ref.GetIdentity())
	}

	apiObj, err := createRepository(ctx, c.c, ref, "", req, opts...)
	if err != nil {
		return nil, err
	}

	if apiObj.Owner == nil {
		return nil, fmt.Errorf("returned API object doesn't have an owner")
	}
	ref.UserLogin = apiObj.Owner.UserName

	return newUserRepository(c.clientContext, apiObj, ref), nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *UserRepositoriesClient) Reconcile(ctx context.Context, ref gitprovider.UserRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (gitprovider.UserRepository, bool, error) {
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

	// Run generic reconciliation
	actionTaken, err := reconcileRepository(ctx, actual, req)
	return actual, actionTaken, err
}
