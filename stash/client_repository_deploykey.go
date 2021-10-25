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
	"github.com/hashicorp/go-multierror"
)

// DeployKeyClient implements the gitprovider.DeployKeyClient interface.
var _ gitprovider.DeployKeyClient = &DeployKeyClient{}

// DeployKeyClient operates on the access deploy key list for a specific repository.
type DeployKeyClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns the key with the given name.
// name is internally converted to a label.
// ErrNotFound is returned if the resource does not exist.
func (c *DeployKeyClient) Get(ctx context.Context, name string) (gitprovider.DeployKey, error) {
	key, err := c.get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get deploy key %q: %w", name, err)
	}
	return newDeployKey(c, key), nil
}

func (c *DeployKeyClient) get(ctx context.Context, name string) (*DeployKey, error) {
	deployKeys, err := c.list(ctx)
	if err != nil {
		return nil, err
	}
	// Loop through deploy keys once we find one with the right name
	for _, dk := range deployKeys {
		if dk.Label == name {
			return dk, nil
		}
	}
	return nil, gitprovider.ErrNotFound
}

// List lists all repository deploy keys.
// List returns all available repository deploy keys for the given type,
// using multiple paginated requests if needed.
func (c *DeployKeyClient) List(ctx context.Context) ([]gitprovider.DeployKey, error) {
	apiObjs, err := c.list(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list deploy keys: %w", err)
	}
	// Cast to the generic []gitprovider.DeployKey
	keys := make([]gitprovider.DeployKey, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		keys = append(keys, newDeployKey(c, apiObj))
	}
	return keys, nil
}

func (c *DeployKeyClient) list(ctx context.Context) ([]*DeployKey, error) {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	apiObjs, err := c.client.DeployKeys.All(ctx, projectKey, repoSlug)
	if err != nil {
		return nil, err
	}

	var errs error
	for _, apiObj := range apiObjs {
		if err := validateDeployKeyAPI(apiObj); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return nil, errs
	}
	return apiObjs, nil
}

// Create creates a deploy key with the given specifications.
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *DeployKeyClient) Create(ctx context.Context, req gitprovider.DeployKeyInfo) (gitprovider.DeployKey, error) {
	apiObj, err := createDeployKey(ctx, c, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create deploy key: %w", err)
	}

	return newDeployKey(c, apiObj), nil
}

func createDeployKey(ctx context.Context, c *DeployKeyClient, req gitprovider.DeployKeyInfo) (*DeployKey, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, err
	}

	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	apiObj, err := c.client.DeployKeys.Create(ctx, deployKeyToAPI(projectKey, repoSlug, &req))
	if err != nil {
		return nil, err
	}

	return apiObj, nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *DeployKeyClient) Reconcile(ctx context.Context, req gitprovider.DeployKeyInfo) (gitprovider.DeployKey, bool, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, false, err
	}

	// Get the key with the desired name
	actual, err := c.Get(ctx, req.Name)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			resp, err := c.Create(ctx, req)
			return resp, true, err
		}

		// Unexpected path, Get should succeed or return NotFound
		return nil, false, fmt.Errorf("failed to reconcile deploy key %q: %w", req.Name, err)
	}

	// If the desired matches the actual state, just return the actual state
	if req.Equals(actual.Get()) {
		return actual, false, nil
	}

	// Populate the desired state to the current-actual object
	if err := actual.Set(req); err != nil {
		return actual, false, err
	}
	// Apply the desired state by running Update
	_, err = c.update(ctx, actual.Repository(), actual.Get())
	if err != nil {
		return actual, false, fmt.Errorf("failed to update deploy key %q: %w", req.Name, err)
	}
	return actual, true, nil
}

// update will apply the desired state in this object to the server.
// ErrNotFound is returned if the resource does not exist.
func (c *DeployKeyClient) update(ctx context.Context, ref gitprovider.RepositoryRef, req gitprovider.DeployKeyInfo) (*DeployKey, error) {
	// Delete the old key and recreate
	if err := c.delete(ctx, ref, req); err != nil {
		return nil, err
	}

	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	// if yes, we need to add a tilde to the user login and use it as the project key
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	apiObj, err := c.client.DeployKeys.Create(ctx, deployKeyToAPI(projectKey, repoSlug, &req))
	if err != nil {
		return nil, err
	}

	return apiObj, nil
}

func (c *DeployKeyClient) delete(ctx context.Context, ref gitprovider.RepositoryRef, req gitprovider.DeployKeyInfo) error {
	projectKey, repoSlug := getStashRefs(c.ref)

	// check if it is a user repository
	if r, ok := c.ref.(gitprovider.UserRepositoryRef); ok {
		projectKey = addTilde(r.UserLogin)
	}

	key := deployKeyToAPI(projectKey, repoSlug, &req)
	// Delete the old key
	if err := c.client.DeployKeys.Delete(ctx, key.Project.Key, key.Repository.Slug, key.Key.ID); err != nil {
		return fmt.Errorf("failed to delete deploy key %q: %w", req.Name, err)
	}

	return nil
}

func deployKeyInfoToAPIObj(info *gitprovider.DeployKeyInfo, apiObj *DeployKey) {
	if info.ReadOnly != nil {
		if *info.ReadOnly {
			apiObj.Permission = stashPermissionRead
		} else {
			apiObj.Permission = stashPermissionWrite
		}
	}
	apiObj.Key.Label = info.Name
}

func deployKeyToAPI(orgKey, repoSlug string, info *gitprovider.DeployKeyInfo) *DeployKey {
	k := &DeployKey{
		Key: Key{
			Text: string(info.Key),
		},
		Repository: Repository{
			Slug: repoSlug,
			Project: Project{
				Key: orgKey,
			},
		},
	}
	setKeyName(info)
	deployKeyInfoToAPIObj(info, k)
	return k
}

// TO DO: Implement this
func validateDeployKeyAPI(apiObj *DeployKey) error {
	return nil
}

func deployKeyFromAPI(apiObj *DeployKey) gitprovider.DeployKeyInfo {
	deRefBool := apiObj.Permission == stashPermissionRead
	return gitprovider.DeployKeyInfo{
		Name:     apiObj.Key.Label,
		Key:      []byte(apiObj.Key.Text),
		ReadOnly: &deRefBool,
	}
}
