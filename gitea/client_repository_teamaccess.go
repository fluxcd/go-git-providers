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
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// TeamAccessClient implements the gitprovider.TeamAccessClient interface.
var _ gitprovider.TeamAccessClient = &TeamAccessClient{}

// TeamAccessClient operates on the teams list for a specific repository.
type TeamAccessClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get a team within the specific organization.
//
// name may include slashes, but must not be an empty string.
// Teams are sub-groups in GitLab.
//
// ErrNotFound is returned if the resource does not exist.
//
// TeamAccess.APIObject will be nil, because there's no underlying Gitea struct.
func (c *TeamAccessClient) Get(ctx context.Context, name string) (gitprovider.TeamAccess, error) {
	// GET /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	permission, err := c.getTeamPermissions(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), name)
	if err != nil {
		return nil, err
	}

	return newTeamAccess(c, gitprovider.TeamAccessInfo{
		Name:       name,
		Permission: getProviderPermission(*permission),
	}), nil
}

// List lists the team access control list for this repository.
//
// List returns all available team access lists, using multiple paginated requests if needed.
func (c *TeamAccessClient) List(ctx context.Context) ([]gitprovider.TeamAccess, error) {
	// List all teams, using pagination. This does not contain information about the members
	apiObjs, err := c.listRepoTeams(ctx, c.ref.GetIdentity(), c.ref.GetRepository())
	if err != nil {
		return nil, err
	}

	teamAccess := make([]gitprovider.TeamAccess, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Get more detailed info about the team, we know that Slug is non-nil as of ListTeams.
		ta, err := c.Get(ctx, apiObj.Name)
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
func (c *TeamAccessClient) Create(ctx context.Context, req gitprovider.TeamAccessInfo) (gitprovider.TeamAccess, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, err
	}

	// PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	if err := c.addTeam(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), req.Name, *req.Permission); err != nil {
		return nil, err
	}

	return newTeamAccess(c, req), nil
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
	return actual, true, actual.Update(ctx)
}

// getTeamPermissions returns the permissions of the given team on the given repository.
func (c *TeamAccessClient) getTeamPermissions(_ context.Context, orgName, repo, teamName string) (*gitea.AccessMode, error) {
	apiObj, resp, err := c.c.CheckRepoTeam(orgName, repo, teamName)
	if err != nil {
		return nil, handleHTTPError(resp, err)
	}
	if apiObj == nil {
		return nil, fmt.Errorf("team %s not found in repository %s/%s", teamName, orgName, repo)
	}

	return &apiObj.Permission, nil
}

// listRepoTeams returns all teams of the given repository.
func (c *TeamAccessClient) listRepoTeams(ctx context.Context, orgName, repo string) ([]*gitea.Team, error) {
	teamObjs, resp, err := c.c.GetRepoTeams(orgName, repo)
	if err != nil {
		return nil, handleHTTPError(resp, err)
	}
	return teamObjs, nil
}

// addTeam adds the given team to the given repository.
// We don't support setting permissions for Gitea, so we ignore the permission parameter.
// see https://github.com/go-gitea/gitea/issues/14717
func (c *TeamAccessClient) addTeam(_ context.Context, orgName, repo, teamName string, permission gitprovider.RepositoryPermission) error {
	res, err := c.c.AddRepoTeam(orgName, repo, teamName)
	return handleHTTPError(res, err)
}

// removeTeam removes the given team from the given repository.
func (c *TeamAccessClient) removeTeam(_ context.Context, orgName, repo, teamName string) error {
	res, err := c.c.RemoveRepoTeam(orgName, repo, teamName)
	return handleHTTPError(res, err)
}
