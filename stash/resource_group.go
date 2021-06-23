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
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func newGroupMembers(ctx *clientContext, apiObj *team, ref gitprovider.OrganizationRef) *team {
	return &team{
		users: apiObj.users,
		info:  gitprovider.TeamInfo{},
		ref:   ref,
	}
}

var _ gitprovider.Team = &groupMembers{}

type groupMembers struct {
	*clientContext

	m   GroupMembers
	ref gitprovider.OrganizationRef
}

func (g *groupMembers) Get() gitprovider.TeamInfo {
	return teamFromAPI(&g.m)
}

func (g *groupMembers) APIObject() interface{} {
	return &g.m
}

func (g *groupMembers) Organization() gitprovider.OrganizationRef {
	return g.ref
}

func teamFromAPI(apiObj *GroupMembers) gitprovider.TeamInfo {
	return gitprovider.TeamInfo{
		Name:    apiObj.GroupName,
		Members: getGroupMemberNames(apiObj.Users),
	}
}

func validateGroupAPI(apiObj *Group) error {
	return validateAPIObject("Stash.Group", func(validator validation.Validator) {
		if apiObj.Name == "" {
			validator.Required("Name")
		}
	})
}

func validateProjectGroupPermissionAPI(apiObj *ProjectGroupPermission) error {
	return validateAPIObject("Stash.ProjectGroupPermission", func(validator validation.Validator) {
		if apiObj.Group.Name == "" {
			validator.Required("Name")
		}
	})
}

func validateUserAPI(apiObj *User) error {
	return validateAPIObject("Stash.User", func(validator validation.Validator) {
		if apiObj.Name == "" {
			validator.Required("Name")
		}
	})
}

func validateRepositoryGroupPermissionAPI(apiObj *RepositoryGroupPermission) error {
	return validateAPIObject("Stash.GroupPermission", func(validator validation.Validator) {
		if apiObj.Group.Name == "" {
			validator.Required("Name")
		}
		if apiObj.Permission == "" {
			validator.Required("Permission")
		}
	})
}
