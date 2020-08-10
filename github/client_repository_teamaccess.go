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

// RepositoryTeamAccessClient implements the gitprovider.RepositoryTeamAccessClient interface
var _ gitprovider.RepositoryTeamAccessClient = &RepositoryTeamAccessClient{}

// RepositoryTeamAccessClient operates on the teams list for a specific repository
type RepositoryTeamAccessClient struct {
	*clientContext
	info gitprovider.RepositoryInfo
}

func (c *RepositoryTeamAccessClient) Get(ctx context.Context, teamName string) (gitprovider.TeamAccess, error) {
	// Disallow operating teams on an user account
	if err := c.disallowUserAccount(); err != nil {
		return nil, err
	}

	// GET /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	teamInfo, _, err := c.c.Teams.IsTeamRepoBySlug(ctx, c.info.GetIdentity(), teamName, c.info.GetIdentity(), c.info.GetRepository())
	if err != nil {
		return nil, handleHTTPError(err)
	}

	ta := gitprovider.TeamAccessInfo{
		Name: teamName,
	}

	if teamInfo.Permissions != nil {
		ta.Permission = getPermissionFromMap(*teamInfo.Permissions)
	} // TODO: Handle teamInfo.Permissions == nil?

	return c.wrap(ta), nil
}

// List lists the team access control list for this repository.
//
// List returns all available team access lists, using multiple paginated requests if needed.
func (c *RepositoryTeamAccessClient) List(ctx context.Context) ([]gitprovider.TeamAccess, error) {
	// Disallow operating teams on an user account
	if err := c.disallowUserAccount(); err != nil {
		return nil, err
	}

	// List all teams, using pagination. This does not contain information about the members
	apiObjs := []*github.Team{}
	opts := &github.ListOptions{}
	err := allPages(opts, func() (*github.Response, error) {
		// GET /repos/{owner}/{repo}/teams
		pageObjs, resp, listErr := c.c.Repositories.ListTeams(ctx, c.info.GetIdentity(), c.info.GetRepository(), opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, handleHTTPError(err)
	}

	teamAccess := make([]gitprovider.TeamAccess, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// TODO: Handle this better
		if apiObj.Slug == nil {
			continue
		}

		// Get more detailed info about the team
		ta, err := c.Get(ctx, *apiObj.Slug)
		if err != nil {
			return nil, err
		}
		teamAccess = append(teamAccess, ta)
	}

	return teamAccess, nil
}

// Create adds a given team to the repo's team access control list.
//
// ErrAlreadyExists will be returned if the resource already exists.
func (c *RepositoryTeamAccessClient) Create(ctx context.Context, req gitprovider.TeamAccessInfo) (gitprovider.TeamAccess, error) {
	// Disallow operating teams on an user account
	if err := c.disallowUserAccount(); err != nil {
		return nil, err
	}
	// Validate the request and default
	if err := req.ValidateCreate(); err != nil {
		return nil, err
	}
	req.Default()

	opts := &github.TeamAddTeamRepoOptions{
		Permission: string(*req.Permission),
	}
	// PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	_, err := c.c.Teams.AddTeamRepoBySlug(ctx, c.info.GetIdentity(), req.Name, c.info.GetIdentity(), c.info.GetRepository(), opts)
	if err != nil {
		return nil, handleHTTPError(err)
	}

	return c.wrap(req), nil
}

func (c *RepositoryTeamAccessClient) wrap(ta gitprovider.TeamAccessInfo) *teamAccess {
	return &teamAccess{
		ta: ta,
		c:  c,
	}
}

var _ gitprovider.TeamAccess = &teamAccess{}

type teamAccess struct {
	ta gitprovider.TeamAccessInfo
	c  *RepositoryTeamAccessClient
}

func (ta *teamAccess) Get() gitprovider.TeamAccessInfo {
	return ta.ta
}

func (ta *teamAccess) Set(info gitprovider.TeamAccessInfo) error {
	ta.ta = info
	return nil
}

func (ta *teamAccess) APIObject() interface{} {
	return nil
}

func (ta *teamAccess) Repository() gitprovider.RepositoryRef {
	return ta.c.info
}

// Delete removes the given team from the repo's team access control list.
//
// ErrNotFound is returned if the resource does not exist.
func (ta *teamAccess) Delete(ctx context.Context) error {
	/*// Validate the request
	if err := req.ValidateDelete(); err != nil {
		return err
	}*/

	// DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	_, err := ta.c.c.Teams.RemoveTeamRepoBySlug(ctx, ta.c.info.GetIdentity(), ta.ta.Name, ta.c.info.GetIdentity(), ta.c.info.GetRepository())
	if err != nil {
		return handleHTTPError(err)
	}

	return nil
}

func (ta *teamAccess) Update(ctx context.Context) error {
	req := ta.Get()
	// Update the actual state to be the desired state
	// by issuing a Create, which uses a PUT underneath.
	resp, err := ta.c.Create(ctx, req)
	ta.Set(resp.Get())
	return err
}

// Reconcile makes sure req is the actual state in the backing Git provider.
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
			ta.Set(resp.Get())
			return true, err
		}

		// Unexpected path, Get should succeed or return NotFound
		return false, err
	}

	// If the desired matches the actual state, just return the actual state
	if reflect.DeepEqual(req, actual.Get()) {
		return false, nil
	}

	return true, ta.Update(ctx)
}

func (ta *teamAccess) ValidateDelete() error { return nil } // TODO consider removing this from the interface
func (ta *teamAccess) ValidateUpdate() error { return nil } // TODO what to do here, should we call in .Update()?

func (c *RepositoryTeamAccessClient) disallowUserAccount() error {
	switch c.info.GetType() {
	case gitprovider.IdentityTypeOrganization:
		return nil
	case gitprovider.IdentityTypeSuborganization:
		return fmt.Errorf("suborganizations aren't supported by GitHub: %w", gitprovider.ErrProviderNoSupport)
	case gitprovider.IdentityTypeUser:
		return fmt.Errorf("cannot manage teams for a personal repository: %w", gitprovider.ErrInvalidArgument)
	default:
		return fmt.Errorf("unrecognized reporef type %q: %w", c.info.GetType(), gitprovider.ErrInvalidArgument)
	}
}

var permissionPriority = map[gitprovider.RepositoryPermission]int{
	gitprovider.RepositoryPermissionPull:     1,
	gitprovider.RepositoryPermissionTriage:   2,
	gitprovider.RepositoryPermissionPush:     3,
	gitprovider.RepositoryPermissionMaintain: 4,
	gitprovider.RepositoryPermissionAdmin:    5,
}

func getPermissionFromMap(permissionMap map[string]bool) (permission *gitprovider.RepositoryPermission) {
	lastPriority := 0
	for key, ok := range permissionMap {
		if ok {
			p := gitprovider.RepositoryPermission(key)
			priority, ok := permissionPriority[p]
			if ok && priority > lastPriority {
				permission = &p
				lastPriority = priority
			}
		}
	}
	return
}
