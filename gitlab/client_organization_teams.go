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

package gitlab

import (
	"context"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"gitlab.com/gitlab-org/api/client-go"
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
	apiObjs, err := c.c.ListGroupMembers(ctx, c.ref.Organization)
	if err != nil {
		return nil, err
	}

	// Collect a list of the members' names. Login is validated to be non-nil in ListOrgTeamMembers.
	logins := make([]string, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Login is validated to be non-nil in ListOrgTeamMembers
		logins = append(logins, apiObj.Username)
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
	subgroups, err := c.c.ListSubgroups(ctx, c.ref.Organization)
	if err != nil {
		return nil, err
	}

	teams := make([]gitprovider.Team, 0, len(subgroups))
	for _, subgroup := range subgroups {
		team, err := c.Get(ctx, subgroup.Name)
		if err != nil {
			return nil, err
		}

		teams = append(teams, team)
	}

	return teams, nil
}

var _ gitprovider.Team = &team{}

type team struct {
	users []*gitlab.GroupMember
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
