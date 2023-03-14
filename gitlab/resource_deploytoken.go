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

package gitlab

import (
	"context"
	"fmt"
	"reflect"

	"github.com/xanzy/go-gitlab"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func newDeployToken(c *DeployTokenClient, token *gitlab.DeployToken) *deployToken {
	return &deployToken{
		k: *token,
		c: c,
	}
}

var _ gitprovider.DeployToken = &deployToken{}

type deployToken struct {
	k gitlab.DeployToken
	c *DeployTokenClient
}

func (dk *deployToken) Get() gitprovider.DeployTokenInfo {
	return deployTokenFromAPI(&dk.k)
}

func (dk *deployToken) Set(info gitprovider.DeployTokenInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	deployTokenInfoToAPIObj(&info, &dk.k)
	return nil
}

func (dk *deployToken) APIObject() interface{} {
	return &dk.k
}

func (dk *deployToken) Repository() gitprovider.RepositoryRef {
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
func (dk *deployToken) Update(ctx context.Context) error {
	// Delete the old token and recreate
	if err := dk.Delete(ctx); err != nil {
		return err
	}
	return dk.createIntoSelf()
}

// Delete deletes a deploy token from the repository.
//
// ErrNotFound is returned if the resource does not exist.
func (dk *deployToken) Delete(_ context.Context) error {
	// We can use the same DeployToken ID that we got from the GET calls. Make sure it's non-nil.
	// This _should never_ happen, but just check for it anyways to avoid panicing.
	if dk.k.ID == 0 {
		return fmt.Errorf("didn't expect ID to be 0: %w", gitprovider.ErrUnexpectedEvent)
	}

	return dk.c.c.DeleteToken(getRepoPath(dk.c.ref), dk.k.ID)
}

// Reconcile makes sure the desired state in this object (called "req" here) becomes
// the actual state in the backing Git provider.
//
// The deploy token cannot be retrieved from the GitLab,
// consequently we have to re-create the deploy token in every reconcile call.
//
// The internal API object will be overridden with the received server data if actionTaken == true.
func (dk *deployToken) Reconcile(ctx context.Context) (bool, error) {
	// Always update for now.
	// The problem here is that there is no way to retrieve the secret token value
	// from GitLab again, so we'll never be sure if it actually matches what we have
	// in this deployToken, so lets just update.
	return true, dk.Update(ctx)
}

func (dk *deployToken) createIntoSelf() error {
	// POST /repos/{owner}/{repo}/tokens
	apiObj, err := dk.c.c.CreateToken(getRepoPath(dk.c.ref), &dk.k)
	if err != nil {
		return err
	}
	dk.k = *apiObj
	return nil
}

func validateDeployTokenAPI(apiObj *gitlab.DeployToken) error {
	return validateAPIObject("GitLab.Token", func(validator validation.Validator) {
		if apiObj.Name == "" {
			validator.Required("Name")
		}
		if apiObj.Username == "" {
			validator.Required("Username")
		}
	})
}

func deployTokenFromAPI(apiObj *gitlab.DeployToken) gitprovider.DeployTokenInfo {
	return gitprovider.DeployTokenInfo{
		Name:     apiObj.Name,
		Username: apiObj.Username,
		Token:    apiObj.Token,
	}
}

func deployTokenToAPI(info *gitprovider.DeployTokenInfo) *gitlab.DeployToken {
	k := &gitlab.DeployToken{}
	deployTokenInfoToAPIObj(info, k)
	return k
}

func deployTokenInfoToAPIObj(info *gitprovider.DeployTokenInfo, apiObj *gitlab.DeployToken) {
	// Required fields, we assume info is validated, and hence these are set
	apiObj.Name = info.Name
	apiObj.Username = info.Username
}

// This function copies over the fields that are part of create request of a deploy
// i.e. the desired spec of the deploy token. This allows us to separate "spec" from "status" fields.
func newGitlabTokenSpec(token *gitlab.DeployToken) *gitlabTokenSpec {
	return &gitlabTokenSpec{
		&gitlab.DeployToken{
			// Create-specific parameters
			Name: token.Name,
		},
	}
}

type gitlabTokenSpec struct {
	*gitlab.DeployToken
}

func (s *gitlabTokenSpec) Equals(other *gitlabTokenSpec) bool {
	return reflect.DeepEqual(s, other)
}
