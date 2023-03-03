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

package gitlab

import (
	"context"
	"errors"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/xanzy/go-gitlab"
)

// DeployTokenClient implements the gitprovider.DeployTokenClient interface.
var _ gitprovider.DeployTokenClient = &DeployTokenClient{}

// DeployTokenClient operates on the access deploy token list for a specific repository.
type DeployTokenClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get returns the repository at the given path.
//
// ErrNotFound is returned if the resource does not exist.
func (c *DeployTokenClient) Get(_ context.Context, deployTokenName string) (gitprovider.DeployToken, error) {
	return c.get(deployTokenName)
}

func (c *DeployTokenClient) get(deployTokenName string) (*deployToken, error) {
	deployTokens, err := c.list()
	if err != nil {
		return nil, err
	}
	// Loop through deploy tokens once we find one with the right name
	for _, dk := range deployTokens {
		if dk.k.Name == deployTokenName {
			return dk, nil
		}
	}
	return nil, gitprovider.ErrNotFound
}

// List lists all repository deploy tokens of the given deploy token type.
//
// List returns all available repository deploy tokens for the given type,
// using multiple paginated requests if needed.
func (c *DeployTokenClient) List(_ context.Context) ([]gitprovider.DeployToken, error) {
	dks, err := c.list()
	if err != nil {
		return nil, err
	}
	// Cast to the generic []gitprovider.DeployToken
	tokens := make([]gitprovider.DeployToken, 0, len(dks))
	for _, dk := range dks {
		tokens = append(tokens, dk)
	}
	return tokens, nil
}

func (c *DeployTokenClient) list() ([]*deployToken, error) {
	// GET /repos/{owner}/{repo}/tokens
	apiObjs, err := c.c.ListTokens(getRepoPath(c.ref))
	if err != nil {
		return nil, err
	}

	// Map the api object to our DeployToken type
	tokens := make([]*deployToken, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// apiObj is already validated at ListTokens
		tokens = append(tokens, newDeployToken(c, apiObj))
	}

	return tokens, nil
}

// Create creates a deploy token with the given specifications.
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *DeployTokenClient) Create(_ context.Context, req gitprovider.DeployTokenInfo) (gitprovider.DeployToken, error) {
	apiObj, err := createDeployToken(c.c, c.ref, req)
	if err != nil {
		return nil, err
	}
	return newDeployToken(c, apiObj), nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *DeployTokenClient) Reconcile(ctx context.Context, req gitprovider.DeployTokenInfo) (gitprovider.DeployToken, bool, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, false, err
	}

	// Get the token with the desired name
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

	actionTaken, err := actual.Reconcile(ctx)
	if err != nil {
		return nil, false, err
	}

	return actual, actionTaken, nil
	//
	// // If the desired matches the actual state, just return the actual state
	// if req.Equals(actual.Get()) {
	// 	return actual, false, nil
	// }
	//
	// // Populate the desired state to the current-actual object
	// if err := actual.Set(req); err != nil {
	// 	return actual, false, err
	// }
	// // Apply the desired state by running Update
	// return actual, true, actual.Update(ctx)
}

func createDeployToken(c gitlabClient, ref gitprovider.RepositoryRef, req gitprovider.DeployTokenInfo) (*gitlab.DeployToken, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, err
	}

	return c.CreateToken(fmt.Sprintf("%s/%s", ref.GetIdentity(), ref.GetRepository()), deployTokenToAPI(&req))
}
