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
	gostash "github.com/fluxcd/go-git-providers/go-stash"
)

// TeamsClient implements the gitprovider.TeamsClient interface.
var _ gitprovider.TeamsClient = &TeamsClient{}

// TeamsClient handles teams organization-wide.
type TeamsClient struct {
	*clientContext
	ref gitprovider.OrganizationRef
}

func getGroupMemberSlugs(users []*gostash.User) []string {
	slugs := make([]string, len(users))
	for i, user := range users {
		// We rely on slugs here as it is used for login
		slugs[i] = user.Slug
	}
	return slugs
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

	team := &Team{
		ref:   c.ref,
		users: users,
	}

	team.info = gitprovider.TeamInfo{
		Name: teamName,
		// We rely on slugs here as it is used for login
		Members: getGroupMemberSlugs(team.users),
	}

	return team, nil
}

// List teams (stash groups).
//
// ErrNotFound is returned if the resource does not exist.
func (c *TeamsClient) List(ctx context.Context) ([]gitprovider.Team, error) {
	// Retrieve all groups for a given project
	// pagination happens in ListProjectGroups
	apiObjs, err := c.c.ListProjectGroups(ctx, c.ref.Organization)
	if err != nil {
		return nil, err
	}

	teams := make([]gitprovider.Team, len(apiObjs))
	for i, apiObj := range apiObjs {
		// Get detailed information about individual teams (including members).
		// Slug is validated to be non-nil in ListGroupMembers.
		team, err := c.Get(ctx, apiObj.Group.Name)
		if err != nil {
			return nil, err
		}

		teams[i] = team
	}

	return teams, nil
}
