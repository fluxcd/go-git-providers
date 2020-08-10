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

	gitprovider "github.com/fluxcd/go-git-providers"
	"github.com/google/go-github/v32/github"
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
		keys = append(keys, c.wrap(apiObj))
	}

	return keys, nil
}

// Create creates a deploy key with the given specifications.
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *DeployKeyClient) Create(ctx context.Context, req gitprovider.DeployKeyInfo) (resp gitprovider.DeployKey, err error) {
	apiObj, err := c.create(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.wrap(apiObj), nil
}

func (c *DeployKeyClient) wrap(key *github.Key) *deployKey {
	return &deployKey{
		k: *key,
		c: c,
	}
}

func (c *DeployKeyClient) create(ctx context.Context, req gitprovider.DeployKeyInfo) (*github.Key, error) {
	// Validate the create request and default
	if err := req.ValidateInfo(); err != nil {
		return nil, err
	}
	req.Default()

	// POST /repos/{owner}/{repo}/keys
	apiObj, _, err := c.c.Repositories.CreateKey(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), &github.Key{
		Title:    gitprovider.StringVar(req.Name),
		Key:      gitprovider.StringVar(string(req.Key)),
		ReadOnly: req.ReadOnly,
	})
	if err != nil {
		return nil, handleHTTPError(err)
	}
	return apiObj, nil
}

var _ gitprovider.DeployKey = &deployKey{}

type deployKey struct {
	k github.Key
	c *DeployKeyClient
}

func (dk *deployKey) Get() gitprovider.DeployKeyInfo {
	return deployKeyFromAPI(&dk.k)
}

func (dk *deployKey) Set(info gitprovider.DeployKeyInfo) error {
	dk.k.Title = &info.Name
	return nil
}

func (dk *deployKey) APIObject() interface{} {
	return &dk.k
}

func (dk *deployKey) Repository() gitprovider.RepositoryRef {
	return dk.c.ref
}

// Delete deletes a deploy key from the repository.
//
// ErrNotFound is returned if the resource does not exist.
func (dk *deployKey) Delete(ctx context.Context) error {
	// We can use the same DeployKey ID that we got from the GET calls

	// DELETE /repos/{owner}/{repo}/keys/{key_id}
	_, err := dk.c.c.Repositories.DeleteKey(ctx, dk.c.ref.GetIdentity(), dk.c.ref.GetRepository(), *dk.k.ID)
	if err != nil {
		return handleHTTPError(err)
	}

	return nil
}

// Reconcile makes sure req is the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// resp will contain any updated information given by the server; hence it is encouraged
// to stop using req after this call, and use resp instead.
func (dk *deployKey) Reconcile(ctx context.Context) (bool, error) {
	req := dk.Get()
	actual, err := dk.c.Get(ctx, *dk.k.Key)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			return true, dk.createIntoSelf(ctx, req)
		}

		// Unexpected path, Get should succeed or return NotFound
		return false, err
	}

	// If the desired matches the actual state, just return the actual state
	if reflect.DeepEqual(req, actual.Get()) {
		return false, nil
	}

	// Delete the old key and recreate
	if err := dk.Delete(ctx); err != nil {
		return true, err
	}
	return true, dk.createIntoSelf(ctx, req)
}

func (dk *deployKey) createIntoSelf(ctx context.Context, req gitprovider.DeployKeyInfo) error {
	apiObj, err := dk.c.create(ctx, req)
	if err != nil {
		return err
	}
	dk.k = *apiObj // VALIDATE HERE?
	return nil
}

func (dk *deployKey) ValidateDelete() error { return nil } // TODO consider removing this from the interface

func deployKeyFromAPI(apiObj *github.Key) gitprovider.DeployKeyInfo {
	return gitprovider.DeployKeyInfo{
		Name:     *apiObj.Title,
		Key:      []byte(*apiObj.Key),
		ReadOnly: apiObj.ReadOnly,
	}
}
