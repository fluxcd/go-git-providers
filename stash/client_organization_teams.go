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

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// TeamsClient implements the gitprovider.TeamsClient interface.
var _ gitprovider.TeamsClient = &TeamsClient{}

// TeamsClient handles teams organization-wide.
type TeamsClient struct {
	*clientContext
	ref gitprovider.OrganizationRef
}

func getGroupMemberNames(users []*User) []string {
	names := make([]string, len(users))
	for index, user := range users {
		names[index] = user.Name
	}
	return names
}

// Get a team (stash group).
// teamName must not be an empty string.
//
// ErrNotFound is returned if the resource does not exist.
func (c *TeamsClient) Get(ctx context.Context, teamName string) (gitprovider.Team, error) {
	users, err := c.c.ListGroupMembers(ctx, teamName)
	if err != nil {
		return nil, err
	}
	team := &team{
		ref:   c.ref,
		users: users,
	}
	team.info = gitprovider.TeamInfo{
		Name:    teamName,
		Members: getGroupMemberNames(team.users),
	}

	return team, nil
}

// List teams (stash groups).
// teamName must not be an empty string.
//
// ErrNotFound is returned if the resource does not exist.
func (c *TeamsClient) List(ctx context.Context) ([]gitprovider.Team, error) {
	apiObjs, err := c.c.ListProjectGroups(ctx, c.ref.Organization)
	if err != nil {
		return nil, err
	}

	// Use .Get() to get detailed information about each member
	teams := make([]gitprovider.Team, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Get detailed information about individual teams (including members).
		// Slug is validated to be non-nil in ListOrgTeams.
		team, err := c.Get(ctx, apiObj.Group.Name)
		if err != nil {
			return nil, err
		}

		teams = append(teams, team)
	}

	return teams, nil
}

var _ gitprovider.Team = &team{}

type team struct {
	users []*User
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
