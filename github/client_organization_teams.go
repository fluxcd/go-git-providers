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
	"fmt"

	"github.com/google/go-github/v32/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// TeamsClient implements the gitprovider.TeamsClient interface
var _ gitprovider.TeamsClient = &TeamsClient{}

// TeamsClient handles teams organization-wide
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
	apiObjs := []*github.User{}
	opts := &github.TeamListTeamMembersOptions{}
	err := allPages(&opts.ListOptions, func() (*github.Response, error) {
		// GET /orgs/{org}/teams/{team_slug}/members
		pageObjs, resp, listErr := c.c.Teams.ListTeamMembersBySlug(ctx, c.ref.Organization, teamName, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}

	logins := make([]string, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Make sure login isn't nil
		if apiObj.Login == nil {
			return nil, fmt.Errorf("didn't expect login to be nil for user: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
		}
		logins = append(logins, *apiObj.Login)
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

// List all teams (recursively, in terms of subgroups) within the specific organization
//
// List returns all available organizations, using multiple paginated requests if needed.
func (c *TeamsClient) List(ctx context.Context) ([]gitprovider.Team, error) {
	// List all teams, using pagination. This does not contain information about the members
	apiObjs := []*github.Team{}
	opts := &github.ListOptions{}
	err := allPages(opts, func() (*github.Response, error) {
		// GET /orgs/{org}/teams
		pageObjs, resp, listErr := c.c.Teams.ListTeams(ctx, c.ref.Organization, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}

	// Use .Get() to get detailed information about each member
	teams := make([]gitprovider.Team, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Make sure name isn't nil
		if apiObj.Slug == nil {
			return nil, fmt.Errorf("didn't expect slug to be nil for team: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
		}

		// Get information about individual teams
		team, err := c.Get(ctx, *apiObj.Slug)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, nil
}

var _ gitprovider.Team = &team{}

type team struct {
	users []*github.User
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
