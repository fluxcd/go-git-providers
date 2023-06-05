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

func newTeamAccess(c *TeamAccessClient, ta gitprovider.TeamAccessInfo) *teamAccess {
	return &teamAccess{
		ta: ta,
		c:  c,
	}
}

var _ gitprovider.TeamAccess = &teamAccess{}

type teamAccess struct {
	ta gitprovider.TeamAccessInfo
	c  *TeamAccessClient
}

func (ta *teamAccess) Get() gitprovider.TeamAccessInfo {
	return ta.ta
}

func (ta *teamAccess) Set(info gitprovider.TeamAccessInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	ta.ta = info
	return nil
}

func (ta *teamAccess) APIObject() interface{} {
	return nil
}

func (ta *teamAccess) Repository() gitprovider.RepositoryRef {
	return ta.c.ref
}

// Delete removes the given team from the repo's team access control list.
//
// ErrNotFound is returned if the resource does not exist.
func (ta *teamAccess) Delete(ctx context.Context) error {
	// DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	return ta.c.c.RemoveTeam(ctx, ta.c.ref.GetIdentity(), ta.c.ref.GetRepository(), ta.ta.Name)
}

func (ta *teamAccess) Update(ctx context.Context) error {
	// Update the actual state to be the desired state
	// by issuing a Create, which uses a PUT underneath.
	resp, err := ta.c.Create(ctx, ta.Get())
	if err != nil {
		return err
	}
	return ta.Set(resp.Get())
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (ta *teamAccess) Reconcile(ctx context.Context) (bool, error) {
	req := ta.Get()
	actual, err := ta.c.Get(ctx, req.Name)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			resp, err := ta.c.Create(ctx, req)
			if err != nil {
				return true, err
			}
			return true, ta.Set(resp.Get())
		}

		// Unexpected path, Get should succeed or return NotFound
		return false, err
	}

	// If the desired matches the actual state, just return the actual state
	if req.Equals(actual.Get()) {
		return false, nil
	}

	return true, ta.Update(ctx)
}

func getProviderPermission(accessMode gitea.AccessMode) (permission *gitprovider.RepositoryPermission) {
	switch accessMode {
	case gitea.AccessModeOwner, gitea.AccessModeAdmin:
		p := gitprovider.RepositoryPermissionAdmin
		permission = &p
	case gitea.AccessModeWrite:
		p := gitprovider.RepositoryPermissionPush
		permission = &p
	case gitea.AccessModeRead:
		p := gitprovider.RepositoryPermissionPull
		permission = &p
	default:
		p := gitprovider.RepositoryPermissionPull
		permission = &p
	}
	return permission
}
