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
	"reflect"

	gitprovider "github.com/fluxcd/go-git-providers"
	"github.com/google/go-github/v32/github"
)

func newDeployKey(c *DeployKeyClient, key *github.Key) *deployKey {
	return &deployKey{
		k: *key,
		c: c,
	}
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
	// TODO
	dk.k.Title = &info.Name
	return nil
}

func (dk *deployKey) APIObject() interface{} {
	return &dk.k
}

func (dk *deployKey) Repository() gitprovider.RepositoryRef {
	return dk.c.ref
}

// Update will apply the desired state in this object to the server.
// Only set fields will be respected (i.e. PATCH behaviour).
// In order to apply changes to this object, use the .Set({Resource}Info) error
// function, or cast .APIObject() to a pointer to the provider-specific type
// and set custom fields there.
//
// ErrNotFound is returned if the resource does not exist.
//
// The internal API object will be overridden with the received server data.
func (dk *deployKey) Update(ctx context.Context) error {
	// Delete the old key and recreate
	if err := dk.Delete(ctx); err != nil {
		return err
	}
	return dk.createIntoSelf(ctx)
}

// Delete deletes a deploy key from the repository.
//
// ErrNotFound is returned if the resource does not exist.
func (dk *deployKey) Delete(ctx context.Context) error {
	// We can use the same DeployKey ID that we got from the GET calls. Make sure it's non-nil
	// This _should never_ happen, but just check for it anyways to avoid panicing
	if dk.k.ID == nil {
		return fmt.Errorf("didn't expect ID to be nil: %w", gitprovider.ErrUnexpectedEvent)
	}

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
	actual, err := dk.c.Get(ctx, *dk.k.Key)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			return true, dk.createIntoSelf(ctx)
		}

		// Unexpected path, Get should succeed or return NotFound
		return false, err
	}

	// This should never (tm) fail, but just to make sure, return an error and don't panic
	actualKey, ok := actual.(*deployKey)
	if !ok {
		return false, fmt.Errorf("expected to be able to cast actual to *deployKey: %w", gitprovider.ErrUnexpectedEvent)
	}

	// If the desired matches the actual state, do nothing
	if reflect.DeepEqual(dk.k, actualKey.k) {
		return false, nil
	}
	// If desired and actual state mis-match, update
	return true, dk.Update(ctx)
}

func (dk *deployKey) createIntoSelf(ctx context.Context) error {
	apiObj, err := createDeployKeyData(dk.c.c, ctx, dk.c.ref, &dk.k)
	if err != nil {
		return err
	}
	dk.k = *apiObj // TODO: VALIDATE HERE?
	return nil
}

func deployKeyFromAPI(apiObj *github.Key) gitprovider.DeployKeyInfo {
	return gitprovider.DeployKeyInfo{
		Name:     *apiObj.Title,
		Key:      []byte(*apiObj.Key),
		ReadOnly: apiObj.ReadOnly,
	}
}

func deployKeyToAPI(info *gitprovider.DeployKeyInfo) *github.Key {
	return &github.Key{
		Title:    gitprovider.StringVar(info.Name),
		Key:      gitprovider.StringVar(string(info.Key)),
		ReadOnly: info.ReadOnly,
	}
}