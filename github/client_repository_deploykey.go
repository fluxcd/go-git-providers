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

// DeployKeyClient implements the gitprovider.DeployKeyClient interface
var _ gitprovider.DeployKeyClient = &DeployKeyClient{}

// DeployKeyClient operates on the access deploy key list for a specific repository
type DeployKeyClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *DeployKeyClient) Get(ctx context.Context, name string) (gitprovider.DeployKey, error) {

	deployKeys, err := c.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, dk := range deployKeys {
		if dk.Get().Name == name {
			return dk, nil
		}
	}

	return nil, gitprovider.ErrNotFound
}

// List lists all repository deploy keys of the given deploy key type.
//
// List returns all available repository deploy keys for the given type,
// using multiple paginated requests if needed.
func (c *DeployKeyClient) List(ctx context.Context) ([]gitprovider.DeployKey, error) {
	// List all keys, using pagination.
	apiObjs := []*github.Key{}
	opts := &github.ListOptions{}
	err := allPages(opts, func() (*github.Response, error) {
		// GET /repos/{owner}/{repo}/keys
		pageObjs, resp, listErr := c.c.Repositories.ListKeys(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, handleHTTPError(err)
	}

	// Map the api object to our DeployKey type
	keys := make([]gitprovider.DeployKey, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		k, err := newDeployKey(c, apiObj)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}

	return keys, nil
}

// Create creates a deploy key with the given specifications.
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *DeployKeyClient) Create(ctx context.Context, req gitprovider.DeployKeyInfo) (gitprovider.DeployKey, error) {
	apiObj, err := createDeployKey(c.c, ctx, c.ref, req)
	if err != nil {
		return nil, err
	}
	return newDeployKey(c, apiObj)
}

// Reconcile makes sure req is the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *DeployKeyClient) Reconcile(ctx context.Context, req gitprovider.DeployKeyInfo) (gitprovider.DeployKey, bool, error) {
	// First thing, validate the request
	if err := req.ValidateInfo(); err != nil {
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
		return nil, false, err
	}

	// If the desired matches the actual state, just return the actual state
	if reflect.DeepEqual(req, actual.Get()) {
		return actual, false, nil
	}

	// Populate the desired state to the current-actual object
	if err := actual.Set(req); err != nil {
		return actual, false, err
	}
	// Apply the desired state by running Update
	return actual, true, actual.Update(ctx)
}

func createDeployKey(c *github.Client, ctx context.Context, ref gitprovider.RepositoryRef, req gitprovider.DeployKeyInfo) (*github.Key, error) {
	// Validate the create request and default
	if err := req.ValidateInfo(); err != nil {
		return nil, err
	}
	req.Default()

	return createDeployKeyData(c, ctx, ref, deployKeyToAPI(&req))
}

func createDeployKeyData(c *github.Client, ctx context.Context, ref gitprovider.RepositoryRef, data *github.Key) (*github.Key, error) {
	// POST /repos/{owner}/{repo}/keys
	apiObj, _, err := c.Repositories.CreateKey(ctx, ref.GetIdentity(), ref.GetRepository(), data)
	return apiObj, handleHTTPError(err)
}
