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
	"fmt"
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
	// update by calling client
	apiObj, err := dk.c.update(ctx, dk.Repository(), dk.Get())
	if err != nil {
		// Log the error and return it
		dk.c.log.V(1).Error(err, "failed to update deploy key", "org", dk.Repository().GetIdentity(), "repo", dk.Repository().GetRepository())
		return err
	}
	dk.k = *apiObj
	return nil
}

// Delete deletes a deploy key from the repository.
// ErrNotFound is returned if the resource does not exist.
func (dk *deployKey) Delete(ctx context.Context) error {
	return dk.c.delete(ctx, dk.Repository(), dk.Get())
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
	_, actionTaken, err := dk.c.Reconcile(ctx, deployKeyFromAPI(&dk.k))

	if err != nil {
		// Log the error and return it
		dk.c.log.V(1).Error(err, "failed to reconcile deploy key",
			"org", dk.Repository().GetIdentity(),
			"repo", dk.Repository().GetRepository(),
			"actionTaken", actionTaken)
		return actionTaken, err
	}

	return actionTaken, nil
}

func setKeyName(info *gitprovider.DeployKeyInfo) {
	keyFields := strings.Split(string(info.Key), " ")
	if info.Name != "" {
		info.Key = []byte(fmt.Sprintf("%s %s %s", keyFields[0], keyFields[1], info.Name))
		return
	}

	if len(keyFields) < 3 && info.Name == "" {
		info.Name = randName()
		return
	}

	if len(keyFields) == 3 && info.Name == "" {
		info.Name = keyFields[2]
		return
	}
}

func randName() string {
	b := make([]byte, 4) //equals 8 characters
	rand.Read(b)
	return hex.EncodeToString(b)
}
