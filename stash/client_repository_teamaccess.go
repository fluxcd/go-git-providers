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
	"errors"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const (
	stashPermissionProjectRead  = "PROJECT_READ"
	stashPermissionProjectWrite = "PROJECT_WRITE"
	stashPermissionProjectAdmin = "PROJECT_ADMIN"
)

// TeamAccessClient implements the gitprovider.TeamAccessClient interface.
var _ gitprovider.TeamAccessClient = &TeamAccessClient{}

// TeamAccessClient operates on the teams list for a specific repository.
type TeamAccessClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get a team's access permission for a given repository.
// Teams are groups in Stash.
// ErrNotFound is returned if the resource does not exist.
func (c *TeamAccessClient) Get(ctx context.Context, name string) (gitprovider.TeamAccess, error) {
	projectKey, repoSlug := getStashRefs(c.ref)
	// Repo level permissions
	repoPerm, err := c.client.Repositories.GetRepositoryGroupPermission(
		ctx,
		projectKey,
		repoSlug,
		name)

	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("failed to get repository team access: %w", err)
		}
	}

	// Try to get both project and repo level permissions, then return the highest one
	orgPerm, err := c.client.Projects.GetProjectGroupPermission(ctx,
		projectKey,
		name)

	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("failed to get organisation team access: %w", err)
		}
	}

	// If both are not found, return ErrNotFound
	if repoPerm == nil && orgPerm == nil {
		return nil, ErrNotFound
	}

	// Use a set to avoid duplicates
	perMap := make(map[string]bool)
	if repoPerm != nil {
		perMap[repoPerm.Permission] = true
	}

	if orgPerm != nil {
		// Get the project level permissions and figure the repo level permissions
		switch orgPerm.Permission {
		case stashPermissionProjectRead:
			perMap[stashPermissionRead] = true
		case stashPermissionProjectWrite:
			perMap[stashPermissionWrite] = true
		case stashPermissionProjectAdmin:
			perMap[stashPermissionAdmin] = true
		}
	}

	// Get the highest permission level
	permLevel, err := getGitProviderPermission(getStashPermissionFromMap(perMap))
	if err != nil {
		return nil, err
	}

	return newTeamAccess(c, gitprovider.TeamAccessInfo{
		Name:       name,
		Permission: permLevel,
	}), nil
}

// List lists the team access control list for this repository.
// List returns all available team access lists, using multiple paginated requests if needed.
func (c *TeamAccessClient) List(ctx context.Context) ([]gitprovider.TeamAccess, error) {
	projectKey, repoSlug := getStashRefs(c.ref)
	// Init a set of team access permissions
	namePermissions := make(map[string][]string)

	// Repo level permissions
	repoPerms, err := c.client.Repositories.AllGroupsPermission(ctx, projectKey, repoSlug)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("failed to get repository teams access: %w", err)
		}
	}

	// project level permissions
	orgPerms, err := c.client.Projects.AllGroupsPermission(ctx, projectKey)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("failed to get organisation teams access: %w", err)
		}
	}

	// If both are not found, return ErrNotFound
	if repoPerms == nil && orgPerms == nil {
		return nil, ErrNotFound
	}

	// Add repo level permissions to the set
	if repoPerms != nil && len(repoPerms) > 0 {
		for _, repoPerm := range repoPerms {
			namePermissions[repoPerm.Group.Name] = append(namePermissions[repoPerm.Group.Name], repoPerm.Permission)
		}
	}

	if orgPerms != nil && len(orgPerms) > 0 {
		// Add project level permissions to the set
		for _, orgPerm := range orgPerms {
			switch orgPerm.Permission {
			case stashPermissionProjectRead:
				namePermissions[orgPerm.Group.Name] = append(namePermissions[orgPerm.Group.Name], stashPermissionRead)
			case stashPermissionProjectWrite:
				namePermissions[orgPerm.Group.Name] = append(namePermissions[orgPerm.Group.Name], stashPermissionWrite)
			case stashPermissionProjectAdmin:
				namePermissions[orgPerm.Group.Name] = append(namePermissions[orgPerm.Group.Name], stashPermissionAdmin)
			}
		}
	}

	teamsAccess := make([]gitprovider.TeamAccess, 0, len(namePermissions))

	for k, v := range namePermissions {
		perMap := make(map[string]bool)
		for _, perm := range v {
			perMap[perm] = true
		}

		permLevel, err := getGitProviderPermission(getStashPermissionFromMap(perMap))
		if err != nil {
			return nil, err
		}

		n := newTeamAccess(c, gitprovider.TeamAccessInfo{
			Name:       k,
			Permission: permLevel,
		})

		teamsAccess = append(teamsAccess, n)

	}

	return teamsAccess, nil
}

// Create adds a given team to the repo's team access control list.
// The team shall exist in Stash.
// ErrAlreadyExists will be returned if the resource already exists.
func (c *TeamAccessClient) Create(ctx context.Context, team gitprovider.TeamAccessInfo) (gitprovider.TeamAccess, error) {
	projectKey, repoSlug := getStashRefs(c.ref)
	permission, err := getStashPermission(*team.Permission)
	if err != nil {
		return nil, err
	}

	type group struct {
		Name string "json:\"name,omitempty\""
	}

	permGroup := &RepositoryGroupPermission{
		Group: group{
			Name: team.Name,
		},
		Permission: permission,
	}

	err = c.client.Repositories.UpdateRepositoryGroupPermission(ctx,
		projectKey, repoSlug, permGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to update repository team access: %w", err)
	}

	// Shall fing the group in the admin level
	teamCreated, err := c.Get(ctx, team.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get team access: %w", err)
	}

	return teamCreated, nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *TeamAccessClient) Reconcile(ctx context.Context,
	req gitprovider.TeamAccessInfo,
) (gitprovider.TeamAccess, bool, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, false, err
	}

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

	// Update the actual state to be the desired state
	// by issuing a Create, which uses a PUT underneath.
	_, err = c.Create(ctx, actual.Get())
	if err != nil {
		return actual, false, err
	}

	return actual, true, nil
}
