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

package stash

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"

	"encoding/hex"
	"math/rand"
)

func newDeployKey(c *DeployKeyClient, key *DeployKey) *deployKey {
	return &deployKey{
		k: *key,
		c: c,
	}
}

var _ gitprovider.DeployKey = &deployKey{}

type deployKey struct {
	k DeployKey
	c *DeployKeyClient
}

func (dk *deployKey) Get() gitprovider.DeployKeyInfo {
	return deployKeyFromAPI(&dk.k)
}

func (dk *deployKey) Set(info gitprovider.DeployKeyInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	dk.k.Key.Text = string(info.Key)
	setKeyName(&info)
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
	return dk.createIntoSelf(ctx)
}

// Delete deletes a deploy key from the repository.
//
// ErrNotFound is returned if the resource does not exist.
func (dk *deployKey) Delete(ctx context.Context) error {
	return dk.c.c.DeleteKey(ctx, dk.k.Repository.Project.Name, dk.k.Repository.Name, dk.k.Key.ID)
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
	actual, err := dk.c.get(ctx, dk.k.Key.Label)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			return true, dk.createIntoSelf(ctx)
		}

		// Unexpected path, Get should succeed or return NotFound
		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newStashKeySpec(&dk.k)
	actualSpec := newStashKeySpec(&actual.k)

	// If the desired matches the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}

	// If desired and actual state mis-match, update
	return true, dk.Update(ctx)
}

func (dk *deployKey) createIntoSelf(ctx context.Context) error {
	// POST /repos/{owner}/{repo}/keys
	apiObj, err := dk.c.c.CreateKey(ctx, &dk.k)
	if err != nil {
		return err
	}
	dk.k = *apiObj
	return nil
}

func validateDeployKeyAPI(apiObj *DeployKey) error {
	// ToDo
	return nil
}

func deployKeyFromAPI(apiObj *DeployKey) gitprovider.DeployKeyInfo {
	deRefBool := apiObj.Permission == "REPO_READ"
	return gitprovider.DeployKeyInfo{
		Name:     apiObj.Key.Label,
		Key:      []byte(apiObj.Key.Text),
		ReadOnly: &deRefBool,
	}
}

func deployKeyToAPI(ref gitprovider.RepositoryRef, info *gitprovider.DeployKeyInfo) *DeployKey {
	k := &DeployKey{
		Key: Key{
			Text: string(info.Key),
		},
		Repository: Repository{
			Name: ref.GetRepository(),
			Project: Project{
				Name: ref.GetIdentity(),
			},
		},
	}
	setKeyName(info)
	deployKeyInfoToAPIObj(info, k)
	return k
}

func setKeyName(info *gitprovider.DeployKeyInfo) {
	keyFields := strings.Split(string(info.Key), " ")
	if len(keyFields) < 3 && len(info.Name) == 0 {
		info.Name = randName()
		return
	}
	if len(info.Name) > 0 {
		info.Key = []byte(fmt.Sprintf("%s %s %s", keyFields[0], keyFields[1], info.Name))
		return
	}
	if len(keyFields) == 3 && len(info.Name) == 0 {
		info.Name = keyFields[2]
		return
	}
}

func randName() string {
	b := make([]byte, 4) //equals 8 characters
	rand.Read(b)
	return hex.EncodeToString(b)
}

func deployKeyInfoToAPIObj(info *gitprovider.DeployKeyInfo, apiObj *DeployKey) {
	if info.ReadOnly != nil {
		if *info.ReadOnly {
			apiObj.Permission = "REPO_READ"
		} else {
			apiObj.Permission = "REPO_WRITE"
		}
	}
	apiObj.Key.Label = info.Name
}

// This function copies over the fields that are part of create request of a deploy
// i.e. the desired spec of the deploy key. This allows us to separate "spec" from "status" fields.
func newStashKeySpec(key *DeployKey) *stashKeySpec {
	return &stashKeySpec{
		&DeployKey{
			Key: Key{
				ID:    key.Key.ID,
				Label: key.Key.Label,
				Text:  key.Key.Text,
			},
			Repository: Repository{
				Name: key.Repository.Name,
				Project: Project{
					Name: key.Repository.Project.Name,
				},
			},
		},
	}
}

type stashKeySpec struct {
	*DeployKey
}

func (s *stashKeySpec) Equals(other *stashKeySpec) bool {
	return reflect.DeepEqual(s, other)
}
