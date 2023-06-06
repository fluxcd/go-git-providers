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

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// TeamsClient implements the gitprovider.TeamsClient interface.
var _ gitprovider.TeamsClient = &TeamsClient{}

// TeamsClient handles teams organization-wide.
type TeamsClient struct {
	*clientContext
	ref gitprovider.OrganizationRef
}

// Get a team within the specific organization.
//
// teamName may include slashes, to point to e.g. subgroups in GitLab.
// teamName must not be an empty string.
//
// ErrNotFound is returned if the resource does not exist.
func (c *TeamsClient) Get(ctx context.Context, teamName string) (gitprovider.Team, error) {
	// GET /orgs/{org}/teams/{team_slug}/members
	apiObjs, err := c.listOrgTeamMembers(c.ref.Organization, teamName)
	if err != nil {
		return nil, err
	}

	// Collect a list of the members' names.ÒÒ Login is validated to be non-nil in ListOrgTeamMembers.
	logins := make([]string, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Login is validated to be non-nil in ListOrgTeamMembers
		logins = append(logins, apiObj.UserName)
	}

	return &team{
		users: apiObjs,
		info: gitprovider.TeamInfo{
			Name:    teamName,
			Members: logins,
		},
		ref: c.ref,
	}, nil
}

// List all teams (recursively, in terms of subgroups) within the specific organization.
//
// List returns all available organizations, using multiple paginated requests if needed.
func (c *TeamsClient) List(ctx context.Context) ([]gitprovider.Team, error) {
	// GET /orgs/{org}/teams
	apiObjs, err := c.listOrgTeams(c.ref.Organization)
	if err != nil {
		return nil, err
	}

	// Use .Get() to get detailed information about each member
	teams := make([]gitprovider.Team, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Get detailed information about individual teams (including members).
		// Slug is validated to be non-nil in ListOrgTeams.
		team, err := c.Get(ctx, apiObj.Name)
		if err != nil {
			return nil, err
		}

		teams = append(teams, team)
	}

	return teams, nil
}

// listOrgTeamMembers returns all of current team members of the given team.
func (c *TeamsClient) listOrgTeamMembers(orgName, teamName string) ([]*gitea.User, error) {
	teams, err := c.listOrgTeams(orgName)
	if err != nil {
		return nil, err
	}
	apiObjs := []*gitea.User{}
	opts := gitea.ListTeamMembersOptions{}
	for _, team := range teams {
		if team.Name == teamName {
			err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
				pageObjs, resp, listErr := c.c.ListTeamMembers(team.ID, gitea.ListTeamMembersOptions{})
				if len(pageObjs) > 0 {
					apiObjs = append(apiObjs, pageObjs...)
					return resp, listErr
				}
				return nil, nil
			})
			if err != nil {
				return nil, err
			}
			return apiObjs, nil
		}
	}

	return nil, gitprovider.ErrNotFound
}

// listOrgTeams returns all teams of the given organization the user has access to.
func (c *TeamsClient) listOrgTeams(orgName string) ([]*gitea.Team, error) {
	opts := gitea.ListTeamsOptions{}
	apiObjs := []*gitea.Team{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /orgs/{org}/teams"
		pageObjs, resp, listErr := c.c.ListOrgTeams(orgName, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

var _ gitprovider.Team = &team{}

type team struct {
	users []*gitea.User
	info  gitprovider.TeamInfo
	ref   gitprovider.OrganizationRef
}

func (t *team) Get() gitprovider.TeamInfo {
	return t.info
}

func (t *team) APIObject() interface{} {
	return t.users
}

func (t *team) Organization() gitprovider.OrganizationRef {
	return t.ref
}
