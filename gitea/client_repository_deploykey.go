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

	"code.gitea.io/sdk/gitea"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// DeployKeyClient implements the gitprovider.DeployKeyClient interface.
var _ gitprovider.DeployKeyClient = &DeployKeyClient{}

// DeployKeyClient operates on the access deploy key list for a specific repository.
type DeployKeyClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *DeployKeyClient) Get(ctx context.Context, name string) (gitprovider.DeployKey, error) {
	return c.get(ctx, name)
}

func (c *DeployKeyClient) get(ctx context.Context, name string) (*deployKey, error) {
	deployKeys, err := c.list(ctx)
	if err != nil {
		return nil, err
	}
	// Loop through deploy keys once we find one with the right name
	for _, dk := range deployKeys {
		if dk.k.Title == name {
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
	dks, err := c.list(ctx)
	if err != nil {
		return nil, err
	}
	// Cast to the generic []gitprovider.DeployKey
	keys := make([]gitprovider.DeployKey, 0, len(dks))
	for _, dk := range dks {
		keys = append(keys, dk)
	}
	return keys, nil
}

func (c *DeployKeyClient) list(ctx context.Context) ([]*deployKey, error) {
	// GET /repos/{owner}/{repo}/keys
	apiObjs, err := c.listKeys(c.ref.GetIdentity(), c.ref.GetRepository())
	if err != nil {
		return nil, err
	}

	// Map the api object to our DeployKey type
	keys := make([]*deployKey, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// apiObj is already validated at ListKeys
		keys = append(keys, newDeployKey(c, apiObj))
	}

	return keys, nil
}

// Create creates a deploy key with the given specifications.
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *DeployKeyClient) Create(ctx context.Context, req gitprovider.DeployKeyInfo) (gitprovider.DeployKey, error) {
	apiObj, err := c.createDeployKey(ctx, c.ref, req)
	if err != nil {
		return nil, err
	}
	return newDeployKey(c, apiObj), nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
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
		return nil, false, err
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
	return actual, true, actual.Update(ctx)
}

// listKeys returns all deploy keys of the given repository.
func (c *DeployKeyClient) listKeys(owner, repo string) ([]*gitea.DeployKey, error) {
	opts := gitea.ListDeployKeysOptions{}
	apiObjs := []*gitea.DeployKey{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /repos/{owner}/{repo}/keys"
		pageObjs, resp, listErr := c.c.ListDeployKeys(owner, repo, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	for _, apiObj := range apiObjs {
		if err := validateDeployKeyAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

// deleteKey deletes the given deploy key from the given repository.
func (c *DeployKeyClient) deleteKey(owner, repo string, id int64) error {
	res, err := c.c.DeleteDeployKey(owner, repo, id)
	return handleHTTPError(res, err)
}

func (c *DeployKeyClient) createDeployKey(ctx context.Context, ref gitprovider.RepositoryRef, req gitprovider.DeployKeyInfo) (*gitea.DeployKey, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, err
	}
	// POST /repos/{owner}/{repo}/keys
	return c.createKey(ref.GetIdentity(), ref.GetRepository(), deployKeyToAPI(&req))
}

// createKey creates a new deploy key for the given repository.
func (c *DeployKeyClient) createKey(owner, repo string, req *gitea.DeployKey) (*gitea.DeployKey, error) {
	opts := gitea.CreateKeyOption{Title: req.Title, Key: req.Key, ReadOnly: req.ReadOnly}
	apiObj, resp, err := c.c.CreateDeployKey(owner, repo, opts)
	if err != nil {
		return nil, handleHTTPError(resp, err)
	}
	if err := validateDeployKeyAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}
