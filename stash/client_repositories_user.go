/*
Copyright 2021 The Flux authors

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
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/hashicorp/go-multierror"
)

// UserRepositoriesClient implements the gitprovider.UserRepositoriesClient interface.
var _ gitprovider.UserRepositoriesClient = &UserRepositoriesClient{}

// UserRepositoriesClient operates on repositories the user has access to.
type UserRepositoriesClient struct {
	*clientContext
}

// Get returns the repository at the given path.
// ErrNotFound is returned if the resource does not exist.
func (c *UserRepositoriesClient) Get(ctx context.Context, ref gitprovider.UserRepositoryRef) (gitprovider.UserRepository, error) {
	// Make sure the UserRepositoryRef is valid
	if err := validateUserRepositoryRef(ref, c.host); err != nil {
		return nil, err
	}

	// Make sure the UserRef is valid
	if err := validateUserRef(ref.UserRef, c.host); err != nil {
		return nil, err
	}

	slug := ref.GetSlug()
	if slug == "" {
		// try with name
		slug = ref.GetRepository()
	}

	apiObj, err := c.client.Repositories.Get(ctx, addTilde(ref.UserLogin), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, gitprovider.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", addTilde(ref.UserLogin), slug, err)
	}

	// Validate the API objects
	if err := validateRepositoryAPI(apiObj); err != nil {
		return nil, err
	}

	ref.SetSlug(apiObj.Slug)

	return newUserRepository(c.clientContext, apiObj, ref), nil
}

// List all repositories for the given user.
// List returns all available repositories, using multiple paginated requests if needed.
func (c *UserRepositoriesClient) List(ctx context.Context, ref gitprovider.UserRef) ([]gitprovider.UserRepository, error) {
	// Make sure the UserRef is valid
	if err := validateUserRef(ref, c.host); err != nil {
		return nil, err
	}

	apiObjs, err := c.client.Repositories.All(ctx, addTilde(ref.UserLogin))
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories for %s: %w", addTilde(ref.UserLogin), err)
	}

	var errs error
	for _, apiObj := range apiObjs {
		if err := validateRepositoryAPI(apiObj); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return nil, errs
	}

	// Traverse the list, and return a list of UserRepository objects
	repos := make([]gitprovider.UserRepository, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		repoRef := gitprovider.UserRepositoryRef{
			UserRef:        ref,
			RepositoryName: apiObj.Name,
		}
		repoRef.SetSlug(apiObj.Slug)

		repos = append(repos, newUserRepository(c.clientContext, apiObj, repoRef))
	}
	return repos, nil
}

// Create creates a repository for the given organization, with the data and options
// ErrAlreadyExists will be returned if the resource already exists.
func (c *UserRepositoriesClient) Create(ctx context.Context,
	ref gitprovider.UserRepositoryRef,
	req gitprovider.RepositoryInfo,
	opts ...gitprovider.RepositoryCreateOption) (gitprovider.UserRepository, error) {
	// Make sure the RepositoryRef is valid
	if err := validateUserRepositoryRef(ref, c.host); err != nil {
		return nil, err
	}

	apiObj, err := createRepository(ctx, c.client, addTilde(ref.UserLogin), ref, req, opts...)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return nil, gitprovider.ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to create repository %s/%s: %w", addTilde(ref.UserLogin), ref.RepositoryName, err)
	}

	ref.SetSlug(apiObj.Slug)

	return newUserRepository(c.clientContext, apiObj, ref), nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
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
		return nil, false, fmt.Errorf("failed to reconcile repository %s/%s: %w", addTilde(ref.UserLogin), ref.RepositoryName, err)
	}

	actionTaken, err := c.reconcileRepository(ctx, actual, req)

	return actual, actionTaken, err
}

func (c *UserRepositoriesClient) reconcileRepository(ctx context.Context, actual gitprovider.UserRepository, req gitprovider.RepositoryInfo) (bool, error) {
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
	repo := actual.APIObject().(*Repository)
	ref := actual.Repository().(gitprovider.UserRepositoryRef)
	_, err := update(ctx, c.client, addTilde(ref.UserLogin), ref.GetSlug(), repo)

	if err != nil {
		return false, err
	}

	return true, nil
}

func validateUserAPI(apiObj *User) error {
	return validateAPIObject("Stash.User", func(validator validation.Validator) {
		if apiObj.Name == "" {
			validator.Required("Name")
		}
	})
}

// validateUserRepositoryRef makes sure the UserRepositoryRef is valid for GitHub's usage.
func validateUserRepositoryRef(ref gitprovider.UserRepositoryRef, expectedDomain string) error {
	// Make sure the RepositoryRef fields are valid
	if err := validation.ValidateTargets("UserRepositoryRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateUserRef makes sure the UserRef is valid for GitHub's usage.
func validateUserRef(ref gitprovider.UserRef, expectedDomain string) error {
	// Make sure the OrganizationRef fields are valid
	if err := validation.ValidateTargets("UserRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

func addTilde(userName string) string {
	if len(userName) > 0 && userName[0] == '~' {
		return userName
	}
	return fmt.Sprintf("~%s", userName)
}
