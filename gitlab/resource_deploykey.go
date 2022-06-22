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
	"reflect"

	"github.com/xanzy/go-gitlab"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func newDeployKey(c *DeployKeyClient, key *gitlab.ProjectDeployKey) *deployKey {
	return &deployKey{
		k:       *key,
		c:       c,
		canpush: key.CanPush,
	}
}

var _ gitprovider.DeployKey = &deployKey{}

type deployKey struct {
	k       gitlab.ProjectDeployKey
	c       *DeployKeyClient
	canpush bool
}

func (dk *deployKey) Get() gitprovider.DeployKeyInfo {
	return deployKeyFromAPI(&dk.k)
}

func (dk *deployKey) Set(info gitprovider.DeployKeyInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	deployKeyInfoToAPIObj(&info, &dk.k)
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
	return dk.createIntoSelf()
}

// Delete deletes a deploy key from the repository.
//
// ErrNotFound is returned if the resource does not exist.
func (dk *deployKey) Delete(_ context.Context) error {
	// We can use the same DeployKey ID that we got from the GET calls. Make sure it's non-nil.
	// This _should never_ happen, but just check for it anyways to avoid panicing.
	if dk.k.ID == 0 {
		return fmt.Errorf("didn't expect ID to be 0: %w", gitprovider.ErrUnexpectedEvent)
	}

	return dk.c.c.DeleteKey(getRepoPath(dk.c.ref), dk.k.ID)
}

// Reconcile makes sure the desired state in this object (called "req" here) becomes
// the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// The internal API object will be overridden with the received server data if actionTaken == true.
func (dk *deployKey) Reconcile(ctx context.Context) (bool, error) {
	actual, err := dk.c.get(dk.k.Title)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			return true, dk.createIntoSelf()
		}

		// Unexpected path, Get should succeed or return NotFound
		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newGitlabKeySpec(&dk.k)
	actualSpec := newGitlabKeySpec(&actual.k)

	// If the desired matches the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}
	// If desired and actual state mis-match, update
	return true, dk.Update(ctx)
}

func (dk *deployKey) createIntoSelf() error {
	// POST /repos/{owner}/{repo}/keys
	apiObj, err := dk.c.c.CreateKey(getRepoPath(dk.c.ref), &dk.k)
	if err != nil {
		return err
	}
	dk.k = *apiObj
	return nil
}

func validateDeployKeyAPI(apiObj *gitlab.ProjectDeployKey) error {
	return validateAPIObject("GitLab.Key", func(validator validation.Validator) {
		if apiObj.Title == "" {
			validator.Required("Title")
		}
		if apiObj.Key == "" {
			validator.Required("Key")
		}
	})
}

func deployKeyFromAPI(apiObj *gitlab.ProjectDeployKey) gitprovider.DeployKeyInfo {
	return gitprovider.DeployKeyInfo{
		Name: apiObj.Title,
		Key:  []byte(apiObj.Key),
	}
}

func deployKeyToAPI(info *gitprovider.DeployKeyInfo) *gitlab.ProjectDeployKey {
	k := &gitlab.ProjectDeployKey{}
	deployKeyInfoToAPIObj(info, k)
	return k
}

func deployKeyInfoToAPIObj(info *gitprovider.DeployKeyInfo, apiObj *gitlab.ProjectDeployKey) {
	// Required fields, we assume info is validated, and hence these are set
	apiObj.Title = info.Name
	apiObj.Key = string(info.Key)
	// optional fields
	derefedBool := false
	if info.ReadOnly != nil {
		if *info.ReadOnly {
			apiObj.CanPush = derefedBool
		} else {
			derefedBool = true
			apiObj.CanPush = derefedBool
		}
	}
}

// This function copies over the fields that are part of create request of a deploy
// i.e. the desired spec of the deploy key. This allows us to separate "spec" from "status" fields.
func newGitlabKeySpec(key *gitlab.ProjectDeployKey) *gitlabKeySpec {
	return &gitlabKeySpec{
		&gitlab.ProjectDeployKey{
			// Create-specific parameters
			Title:   key.Title,
			Key:     key.Key,
			CanPush: key.CanPush,
		},
	}
}

type gitlabKeySpec struct {
	*gitlab.ProjectDeployKey
}

func (s *gitlabKeySpec) Equals(other *gitlabKeySpec) bool {
	return reflect.DeepEqual(s, other)
}
